package memory_tracker

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/rs/zerolog/log"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	ectx "github.com/opisvigilant/futura/watcher/internal/ebpf/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AllocInfo represents an active memory allocation
type AllocInfo struct {
	Size      uint64
	Timestamp uint64
	CgroupID  uint64
	PID       uint32
	StackID   uint32
}

// MemoryMetrics represents aggregated memory metrics per container
type MemoryMetrics struct {
	AllocCount      uint64
	FreeCount       uint64
	BytesAllocated  uint64
	BytesFreed      uint64
	NetAllocated    uint64
	SmallAllocs     uint64 // < 1KB
	MediumAllocs    uint64 // 1KB - 64KB
	LargeAllocs     uint64 // 64KB - 1MB
	HugeAllocs      uint64 // > 1MB
	PageFaults      uint64
	MajorPageFaults uint64
	LastUpdate      uint64
}

// LeakCandidate represents a potential memory leak
type LeakCandidate struct {
	Address   uint64
	Size      uint64
	AllocTime uint64
	StackID   uint32
	PID       uint32
}

// GCMetrics represents garbage collection metrics
type GCMetrics struct {
	GCCount        uint64
	GCTimeNS       uint64
	BytesCollected uint64
	LastGCTime     uint64
}

// MemoryEvent represents a real-time memory event
type MemoryEvent struct {
	Timestamp uint64
	CgroupID  uint64
	PID       uint32
	EventType uint32 // 0=alloc, 1=free, 2=leak_detected
	Address   uint64
	Size      uint64
	StackID   uint32
}

// MemoryTracker manages the memory tracking eBPF program
type MemoryTracker struct {
	objects       *memory_trackerObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Event handlers
	onMemoryMetrics func(*pb.MemoryPatterns)
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker(containerMap *ectx.ContainerMapper) (*MemoryTracker, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &memory_trackerObjects{}
	if err := loadMemory_trackerObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load memory tracker eBPF objects: %w", err)
	}

	tracker := &MemoryTracker{
		objects:      objects,
		containerMap: containerMap,
	}

	return tracker, nil
}

// Start begins memory tracking
func (mt *MemoryTracker) Start(ctx context.Context) error {
	// Attach kernel memory tracepoints
	if err := mt.attachKernelTracepoints(); err != nil {
		return fmt.Errorf("failed to attach kernel tracepoints: %w", err)
	}

	// Set up ring buffer reader
	if err := mt.setupRingBuffer(ctx); err != nil {
		return fmt.Errorf("failed to setup ring buffer: %w", err)
	}

	log.Info().Msg("Memory tracker started")
	return nil
}

// attachKernelTracepoints attaches eBPF programs to kernel tracepoints
func (mt *MemoryTracker) attachKernelTracepoints() error {
	// Attach to kmalloc tracepoint
	kmallocLink, err := link.Tracepoint("kmem", "kmalloc", mt.objects.TraceKmalloc, nil)
	if err != nil {
		return fmt.Errorf("failed to attach kmalloc tracepoint: %w", err)
	}
	mt.links = append(mt.links, kmallocLink)

	// Attach to kfree tracepoint
	kfreeLink, err := link.Tracepoint("kmem", "kfree", mt.objects.TraceKfree, nil)
	if err != nil {
		return fmt.Errorf("failed to attach kfree tracepoint: %w", err)
	}
	mt.links = append(mt.links, kfreeLink)

	// Note: Page fault tracking removed due to missing tracepoint in current kernel
	// Alternative: Could use software events or kprobes for page fault tracking

	log.Info().Msg("Memory tracking kernel tracepoints attached")
	return nil
}

// AttachUserSpaceUprobes attaches uprobes to user-space malloc/free
func (mt *MemoryTracker) AttachUserSpaceUprobes(executablePath string) error {
	ex, err := link.OpenExecutable(executablePath)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to open executable")
		return err
	}

	// Attach uprobe to malloc
	mallocUpLink, err := ex.Uprobe("malloc", mt.objects.TraceMalloc, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach malloc uprobe")
		return err
	}
	mt.links = append(mt.links, mallocUpLink)

	// Attach uretprobe to malloc
	mallocRetLink, err := ex.Uretprobe("malloc", mt.objects.TraceMallocRet, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach malloc uretprobe")
		return err
	}
	mt.links = append(mt.links, mallocRetLink)

	// Attach uprobe to free
	freeUpLink, err := ex.Uprobe("free", mt.objects.TraceFree, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach free uprobe")
		return err
	}
	mt.links = append(mt.links, freeUpLink)

	log.Info().Str("path", executablePath).Msg("Memory tracking user-space uprobes attached")
	return nil
}

// AttachGoGCUprobes attaches uprobes to Go garbage collector
func (mt *MemoryTracker) AttachGoGCUprobes(executablePath string) error {
	ex, err := link.OpenExecutable(executablePath)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to open executable")
		return err
	}

	// Attach uprobe to Go GC
	gcUpLink, err := ex.Uprobe("runtime.GC", mt.objects.TraceGoGcStart, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach Go GC uprobe")
		return err
	}
	mt.links = append(mt.links, gcUpLink)

	log.Info().Str("path", executablePath).Msg("Go GC tracking uprobes attached")
	return nil
}

// setupRingBuffer sets up the ring buffer for real-time events
func (mt *MemoryTracker) setupRingBuffer(ctx context.Context) error {
	reader, err := ringbuf.NewReader(mt.objects.MemoryEvents)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	mt.ringbufReader = reader

	// Start ring buffer processing
	go mt.processRingBufferEvents(ctx)

	return nil
}

// processRingBufferEvents processes real-time memory events
func (mt *MemoryTracker) processRingBufferEvents(ctx context.Context) {
	defer mt.ringbufReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			record, err := mt.ringbufReader.Read()
			if err != nil {
				if err != ringbuf.ErrClosed {
					log.Error().Err(err).Msg("Failed to read from memory events ring buffer")
				}
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(MemoryEvent{})) {
				log.Warn().Int("size", len(record.RawSample)).Msg("Invalid memory event size")
				continue
			}

			// Parse the event
			event := (*MemoryEvent)(unsafe.Pointer(&record.RawSample[0]))

			// Process the event
			mt.processMemoryEvent(event)
		}
	}
}

// processMemoryEvent processes a single memory event
func (mt *MemoryTracker) processMemoryEvent(event *MemoryEvent) {
	containerInfo, exists := mt.containerMap.GetContainerByCgroupID(event.CgroupID)
	if !exists {
		return
	}

	log.Debug().
		Str("container", containerInfo.ContainerID).
		Str("app", containerInfo.AppName).
		Uint32("event_type", event.EventType).
		Uint64("size", event.Size).
		Msg("Memory event")

	// Detect potential memory leaks
	if event.EventType == 0 && event.Size > 1024*1024 { // Large allocation
		mt.checkForMemoryLeak(event, containerInfo)
	}
}

// checkForMemoryLeak performs simple memory leak detection
func (mt *MemoryTracker) checkForMemoryLeak(event *MemoryEvent, containerInfo *ectx.ContainerInfo) {
	// Simple heuristic: large allocations that aren't freed within a reasonable time
	// In a production system, this would be more sophisticated
	log.Debug().
		Str("container", containerInfo.ContainerID).
		Uint64("size", event.Size).
		Msg("Large memory allocation detected - monitoring for potential leak")
}

// Poll implements the CollectorProgram interface
func (mt *MemoryTracker) Poll(ctx context.Context) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := mt.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect memory metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (mt *MemoryTracker) collectAndEmitMetrics() error {
	var nextKey uint64
	var metrics MemoryMetrics

	iter := mt.objects.MemoryMetricsMap.Iterate()
	for iter.Next(&nextKey, &metrics) {
		cgroupID := nextKey

		// Get container info for this cgroup
		containerInfo, exists := mt.containerMap.GetContainerByCgroupID(cgroupID)
		if !exists {
			log.Debug().Uint64("cgroup_id", cgroupID).Msg("No container found for cgroup")
			continue
		}

		// Convert to protobuf
		pbMetrics := mt.convertMetricsToProto(&metrics, containerInfo)
		if pbMetrics != nil && mt.onMemoryMetrics != nil {
			mt.onMemoryMetrics(pbMetrics)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate memory metrics map: %w", err)
	}

	return nil
}

// convertMetricsToProto converts eBPF metrics to protobuf format
func (mt *MemoryTracker) convertMetricsToProto(metrics *MemoryMetrics, containerInfo *ectx.ContainerInfo) *pb.MemoryPatterns {
	if metrics.AllocCount == 0 {
		return nil
	}

	// Calculate derived metrics
	var avgAllocSize float64
	if metrics.AllocCount > 0 {
		avgAllocSize = float64(metrics.BytesAllocated) / float64(metrics.AllocCount)
	}

	var fragmentationRatio float64
	if metrics.BytesAllocated > 0 {
		fragmentationRatio = float64(metrics.NetAllocated) / float64(metrics.BytesAllocated)
	}

	// Allocation size histogram
	allocSizes := &pb.AllocationSizeHistogram{
		SmallAllocs:  metrics.SmallAllocs,
		MediumAllocs: metrics.MediumAllocs,
		LargeAllocs:  metrics.LargeAllocs,
		HugeAllocs:   metrics.HugeAllocs,
		AvgAllocSize: avgAllocSize,
		MaxAllocSize: 0, // TODO: Track max allocation size
	}

	// Potential memory leaks
	var potentialLeaks []*pb.MemoryLeak
	// TODO: Extract leak candidates from eBPF map

	// GC metrics
	var gcMetrics *pb.GCMetrics
	if gc := mt.getGCMetrics(containerInfo); gc != nil {
		var gcOverhead float64
		if gc.GCTimeNS > 0 && metrics.LastUpdate > 0 {
			// Calculate GC overhead as percentage of total time
			totalTimeNS := metrics.LastUpdate - (metrics.LastUpdate - 60*1000000000) // Last 60 seconds
			if totalTimeNS > 0 {
				gcOverhead = float64(gc.GCTimeNS) / float64(totalTimeNS) * 100
			}
		}

		gcMetrics = &pb.GCMetrics{
			GcCount:           gc.GCCount,
			GcTimeUs:          gc.GCTimeNS / 1000,
			BytesCollected:    gc.BytesCollected,
			GcOverheadPercent: gcOverhead,
		}
	}

	now := time.Now()
	windowStart := now.Add(-15 * time.Second) // 15-second window

	return &pb.MemoryPatterns{
		AllocCount:         metrics.AllocCount,
		FreeCount:          metrics.FreeCount,
		BytesAllocated:     metrics.BytesAllocated,
		BytesFreed:         metrics.BytesFreed,
		NetAllocated:       metrics.NetAllocated,
		AllocSizes:         allocSizes,
		PageFaults:         metrics.PageFaults,
		MajorPageFaults:    metrics.MajorPageFaults,
		FragmentationRatio: fragmentationRatio,
		PotentialLeaks:     potentialLeaks,
		Gc:                 gcMetrics,
		WindowStart:        timestampFromTime(windowStart),
		WindowEnd:          timestampFromTime(now),
	}
}

// getGCMetrics retrieves GC metrics for a container
func (mt *MemoryTracker) getGCMetrics(containerInfo *ectx.ContainerInfo) *GCMetrics {
	var gcMetrics GCMetrics
	cgroupID := containerInfo.CgroupID

	if err := mt.objects.GcMetricsMap.Lookup(cgroupID, &gcMetrics); err != nil {
		return nil
	}

	return &gcMetrics
}

// SetMemoryMetricsHandler sets the callback for memory metrics
func (mt *MemoryTracker) SetMemoryMetricsHandler(handler func(*pb.MemoryPatterns)) {
	mt.onMemoryMetrics = handler
}

// Close releases all resources
func (mt *MemoryTracker) Close() error {
	// Close ring buffer reader
	if mt.ringbufReader != nil {
		mt.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range mt.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if mt.objects != nil {
		mt.objects.Close()
	}

	log.Info().Msg("Memory tracker closed")
	return nil
}

// Helper functions

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
