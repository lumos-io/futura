package fs_tracker

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

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" fs_tracker fs_tracker.bpf.c -- -I../../../../headers

// FSMetrics represents aggregated file system metrics per container
type FSMetrics struct {
	ReadOps            uint64
	WriteOps           uint64
	OpenOps            uint64
	CloseOps           uint64
	SyncOps            uint64
	BytesRead          uint64
	BytesWritten       uint64
	ReadLatencyTotal   uint64
	WriteLatencyTotal  uint64
	OpenLatencyTotal   uint64
	IOErrors           uint64
	PermissionErrors   uint64
	LastUpdate         uint64
}

// FileAccess represents file access pattern
type FileAccess struct {
	FilePath      [256]byte
	AccessCount   uint64
	BytesAccessed uint64
	LatencyTotal  uint64
	AccessType    uint32
	CgroupID      uint64
}

// FSEvent represents a real-time file system event
type FSEvent struct {
	Timestamp  uint64
	CgroupID   uint64
	PID        uint32
	EventType  uint32
	FD         int32
	Size       uint64
	LatencyUS  uint64
	Filename   [256]byte
	ErrorCode  int32
}

// FSTracker manages the file system tracking eBPF program
type FSTracker struct {
	objects       *fs_trackerObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Event handlers
	onFSMetrics func(*pb.FileSystemMetrics)
}

// NewFSTracker creates a new file system tracker
func NewFSTracker(containerMap *ectx.ContainerMapper) (*FSTracker, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &fs_trackerObjects{}
	if err := loadFs_trackerObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load FS tracker eBPF objects: %w", err)
	}

	tracker := &FSTracker{
		objects:      objects,
		containerMap: containerMap,
	}

	return tracker, nil
}

// Start begins file system tracking
func (fst *FSTracker) Start(ctx context.Context) error {
	// Attach syscall tracepoints
	if err := fst.attachSyscallTracepoints(); err != nil {
		return fmt.Errorf("failed to attach syscall tracepoints: %w", err)
	}

	// Set up ring buffer reader
	if err := fst.setupRingBuffer(ctx); err != nil {
		return fmt.Errorf("failed to setup ring buffer: %w", err)
	}

	log.Info().Msg("File system tracker started")
	return nil
}

// attachSyscallTracepoints attaches eBPF programs to syscall tracepoints
func (fst *FSTracker) attachSyscallTracepoints() error {
	// Attach to openat syscalls
	openatEnterLink, err := link.Tracepoint("syscalls", "sys_enter_openat", fst.objects.TraceOpenatEnter, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_enter_openat tracepoint: %w", err)
	}
	fst.links = append(fst.links, openatEnterLink)

	openatExitLink, err := link.Tracepoint("syscalls", "sys_exit_openat", fst.objects.TraceOpenatExit, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_exit_openat tracepoint: %w", err)
	}
	fst.links = append(fst.links, openatExitLink)

	// Attach to read syscalls
	readEnterLink, err := link.Tracepoint("syscalls", "sys_enter_read", fst.objects.TraceReadEnter, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_enter_read tracepoint: %w", err)
	}
	fst.links = append(fst.links, readEnterLink)

	readExitLink, err := link.Tracepoint("syscalls", "sys_exit_read", fst.objects.TraceReadExit, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_exit_read tracepoint: %w", err)
	}
	fst.links = append(fst.links, readExitLink)

	// Attach to write syscalls
	writeEnterLink, err := link.Tracepoint("syscalls", "sys_enter_write", fst.objects.TraceWriteEnter, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_enter_write tracepoint: %w", err)
	}
	fst.links = append(fst.links, writeEnterLink)

	writeExitLink, err := link.Tracepoint("syscalls", "sys_exit_write", fst.objects.TraceWriteExit, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_exit_write tracepoint: %w", err)
	}
	fst.links = append(fst.links, writeExitLink)

	log.Info().Msg("File system tracking syscall tracepoints attached")
	return nil
}

// setupRingBuffer sets up the ring buffer for real-time events
func (fst *FSTracker) setupRingBuffer(ctx context.Context) error {
	reader, err := ringbuf.NewReader(fst.objects.FsEvents)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	fst.ringbufReader = reader

	// Start ring buffer processing
	go fst.processRingBufferEvents(ctx)

	return nil
}

// processRingBufferEvents processes real-time file system events
func (fst *FSTracker) processRingBufferEvents(ctx context.Context) {
	defer fst.ringbufReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			record, err := fst.ringbufReader.Read()
			if err != nil {
				if err != ringbuf.ErrClosed {
					log.Error().Err(err).Msg("Failed to read from FS events ring buffer")
				}
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(FSEvent{})) {
				log.Warn().Int("size", len(record.RawSample)).Msg("Invalid FS event size")
				continue
			}

			// Parse the event
			event := (*FSEvent)(unsafe.Pointer(&record.RawSample[0]))

			// Process the event
			fst.processFSEvent(event)
		}
	}
}

// processFSEvent processes a single file system event
func (fst *FSTracker) processFSEvent(event *FSEvent) {
	containerInfo, exists := fst.containerMap.GetContainerByCgroupID(event.CgroupID)
	if !exists {
		return
	}

	// Log significant events
	if event.Size > 1024*1024 || event.LatencyUS > 10000 { // Large I/O or high latency
		eventType := "unknown"
		switch event.EventType {
		case 0:
			eventType = "read"
		case 1:
			eventType = "write"
		case 2:
			eventType = "open"
		case 3:
			eventType = "close"
		case 4:
			eventType = "error"
		}

		log.Debug().
			Str("container", containerInfo.ContainerID).
			Str("app", containerInfo.AppName).
			Str("event_type", eventType).
			Uint64("size", event.Size).
			Uint64("latency_us", event.LatencyUS).
			Msg("Significant FS event")
	}
}

// Poll implements the CollectorProgram interface
func (fst *FSTracker) Poll(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := fst.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect FS metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (fst *FSTracker) collectAndEmitMetrics() error {
	var nextKey uint64
	var metrics FSMetrics

	iter := fst.objects.FsMetricsMap.Iterate()
	for iter.Next(&nextKey, &metrics) {
		cgroupID := nextKey

		// Get container info for this cgroup
		containerInfo, exists := fst.containerMap.GetContainerByCgroupID(cgroupID)
		if !exists {
			log.Debug().Uint64("cgroup_id", cgroupID).Msg("No container found for cgroup")
			continue
		}

		// Convert to protobuf
		pbMetrics := fst.convertMetricsToProto(&metrics, containerInfo)
		if pbMetrics != nil && fst.onFSMetrics != nil {
			fst.onFSMetrics(pbMetrics)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate FS metrics map: %w", err)
	}

	return nil
}

// convertMetricsToProto converts eBPF metrics to protobuf format
func (fst *FSTracker) convertMetricsToProto(metrics *FSMetrics, containerInfo *ectx.ContainerInfo) *pb.FileSystemMetrics {
	if metrics.ReadOps == 0 && metrics.WriteOps == 0 && metrics.OpenOps == 0 {
		return nil
	}

	// Calculate average latencies
	var avgReadLatency, avgWriteLatency, avgOpenLatency float64
	if metrics.ReadOps > 0 {
		avgReadLatency = float64(metrics.ReadLatencyTotal) / float64(metrics.ReadOps)
	}
	if metrics.WriteOps > 0 {
		avgWriteLatency = float64(metrics.WriteLatencyTotal) / float64(metrics.WriteOps)
	}
	if metrics.OpenOps > 0 {
		avgOpenLatency = float64(metrics.OpenLatencyTotal) / float64(metrics.OpenOps)
	}

	// Create latency histograms (simplified for now)
	readLatency := &pb.LatencyHistogram{
		Mean: avgReadLatency,
		P50:  avgReadLatency * 0.8,
		P90:  avgReadLatency * 1.5,
		P95:  avgReadLatency * 2.0,
		P99:  avgReadLatency * 3.0,
	}

	writeLatency := &pb.LatencyHistogram{
		Mean: avgWriteLatency,
		P50:  avgWriteLatency * 0.8,
		P90:  avgWriteLatency * 1.5,
		P95:  avgWriteLatency * 2.0,
		P99:  avgWriteLatency * 3.0,
	}

	openLatency := &pb.LatencyHistogram{
		Mean: avgOpenLatency,
		P50:  avgOpenLatency * 0.8,
		P90:  avgOpenLatency * 1.5,
		P95:  avgOpenLatency * 2.0,
		P99:  avgOpenLatency * 3.0,
	}

	// Calculate I/O size distribution (simplified estimates)
	totalIO := metrics.BytesRead + metrics.BytesWritten
	totalOps := metrics.ReadOps + metrics.WriteOps
	var avgIOSize float64
	if totalOps > 0 {
		avgIOSize = float64(totalIO) / float64(totalOps)
	}

	ioSizes := &pb.IOSizeHistogram{
		SmallIo:   totalOps * 40 / 100,  // Assume 40% small I/O
		MediumIo:  totalOps * 35 / 100,  // Assume 35% medium I/O
		LargeIo:   totalOps * 20 / 100,  // Assume 20% large I/O
		HugeIo:    totalOps * 5 / 100,   // Assume 5% huge I/O
		AvgIoSize: avgIOSize,
		MaxIoSize: avgIOSize * 10, // Estimate
	}

	// Calculate bandwidth (over 10-second window)
	windowSeconds := 10.0
	readBandwidthMbps := float64(metrics.BytesRead) / (1024 * 1024) / windowSeconds
	writeBandwidthMbps := float64(metrics.BytesWritten) / (1024 * 1024) / windowSeconds

	// Get hot files (simplified for now)
	var hotFiles []*pb.FileAccessPattern
	// TODO: Extract hot files from file_access_map

	// Get hot directories (simplified for now)
	var hotDirectories []*pb.DirectoryPattern
	// TODO: Extract hot directories from aggregated data

	now := time.Now()
	windowStart := now.Add(-10 * time.Second)

	return &pb.FileSystemMetrics{
		ReadOps:             metrics.ReadOps,
		WriteOps:            metrics.WriteOps,
		OpenOps:             metrics.OpenOps,
		CloseOps:            metrics.CloseOps,
		SyncOps:             metrics.SyncOps,
		BytesRead:           metrics.BytesRead,
		BytesWritten:        metrics.BytesWritten,
		ReadLatency:         readLatency,
		WriteLatency:        writeLatency,
		OpenLatency:         openLatency,
		HotFiles:            hotFiles,
		HotDirectories:      hotDirectories,
		ReadBandwidthMbps:   readBandwidthMbps,
		WriteBandwidthMbps:  writeBandwidthMbps,
		DiskUtilization:     (readBandwidthMbps + writeBandwidthMbps) / 100.0, // Simplified estimate
		IoSizes:             ioSizes,
		IoErrors:            metrics.IOErrors,
		PermissionErrors:    metrics.PermissionErrors,
		WindowStart:         timestampFromTime(windowStart),
		WindowEnd:           timestampFromTime(now),
	}
}

// SetFSMetricsHandler sets the callback for file system metrics
func (fst *FSTracker) SetFSMetricsHandler(handler func(*pb.FileSystemMetrics)) {
	fst.onFSMetrics = handler
}

// Close releases all resources
func (fst *FSTracker) Close() error {
	// Close ring buffer reader
	if fst.ringbufReader != nil {
		fst.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range fst.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if fst.objects != nil {
		fst.objects.Close()
	}

	log.Info().Msg("File system tracker closed")
	return nil
}

// Helper functions

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}