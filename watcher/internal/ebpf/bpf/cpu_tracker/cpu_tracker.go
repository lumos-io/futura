package cpu_tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/rs/zerolog/log"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	ectx "github.com/opisvigilant/futura/watcher/internal/ebpf/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CPUMetrics represents aggregated CPU metrics per container
type CPUMetrics struct {
	UserTimeUS         uint64
	SystemTimeUS       uint64
	IdleTimeUS         uint64
	CPUUtilization     float64
	ContextSwitches    uint64
	VoluntarySwitches  uint64
	InvoluntarySwitches uint64
	RunqueueLatencyUS  uint64
	ActiveThreads      uint32
	BlockedThreads     uint32
	LastUpdate         uint64
}

// CPUHotspot represents a CPU-intensive operation
type CPUHotspot struct {
	FunctionName    [64]byte
	Samples         uint64
	CPUPercent      float64
	StackTraceHash  [32]byte
}

// WakeupInfo represents process wakeup tracking
type WakeupInfo struct {
	PID       uint32
	WakeupTime uint64
}

// CPUTracker manages the CPU tracking eBPF program
type CPUTracker struct {
	objects       *cpu_trackerObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Event handlers
	onCPUMetrics func(*pb.CPUPatterns)
}

// NewCPUTracker creates a new CPU tracker
func NewCPUTracker(containerMap *ectx.ContainerMapper) (*CPUTracker, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &cpu_trackerObjects{}
	if err := loadCpu_trackerObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load CPU tracker eBPF objects: %w", err)
	}

	tracker := &CPUTracker{
		objects:      objects,
		containerMap: containerMap,
	}

	return tracker, nil
}

// Start begins CPU tracking
func (ct *CPUTracker) Start(ctx context.Context) error {
	// Attach scheduler tracepoints
	if err := ct.attachSchedulerTracepoints(); err != nil {
		return fmt.Errorf("failed to attach scheduler tracepoints: %w", err)
	}

	log.Info().Msg("CPU tracker started")
	return nil
}

// attachSchedulerTracepoints attaches eBPF programs to scheduler tracepoints
func (ct *CPUTracker) attachSchedulerTracepoints() error {
	// Attach to sched_switch tracepoint
	switchLink, err := link.Tracepoint("sched", "sched_switch", ct.objects.TraceSchedSwitch, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sched_switch tracepoint: %w", err)
	}
	ct.links = append(ct.links, switchLink)

	// Attach to sched_wakeup tracepoint
	wakeupLink, err := link.Tracepoint("sched", "sched_wakeup", ct.objects.TraceSchedWakeup, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sched_wakeup tracepoint: %w", err)
	}
	ct.links = append(ct.links, wakeupLink)

	log.Info().Msg("CPU tracking scheduler tracepoints attached")
	return nil
}

// Poll implements the CollectorProgram interface
func (ct *CPUTracker) Poll(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := ct.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect CPU metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (ct *CPUTracker) collectAndEmitMetrics() error {
	var nextKey uint64
	var metrics CPUMetrics

	iter := ct.objects.CpuMetricsMap.Iterate()
	for iter.Next(&nextKey, &metrics) {
		cgroupID := nextKey

		// Get container info for this cgroup
		containerInfo, exists := ct.containerMap.GetContainerByCgroupID(cgroupID)
		if !exists {
			log.Debug().Uint64("cgroup_id", cgroupID).Msg("No container found for cgroup")
			continue
		}

		// Convert to protobuf
		pbMetrics := ct.convertMetricsToProto(&metrics, containerInfo)
		if pbMetrics != nil && ct.onCPUMetrics != nil {
			ct.onCPUMetrics(pbMetrics)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate CPU metrics map: %w", err)
	}

	return nil
}

// convertMetricsToProto converts eBPF metrics to protobuf format
func (ct *CPUTracker) convertMetricsToProto(metrics *CPUMetrics, containerInfo *ectx.ContainerInfo) *pb.CPUPatterns {
	if metrics.ContextSwitches == 0 {
		return nil
	}

	// Calculate CPU utilization
	totalTime := metrics.UserTimeUS + metrics.SystemTimeUS + metrics.IdleTimeUS
	var cpuUtilization float64
	if totalTime > 0 {
		cpuUtilization = float64(metrics.UserTimeUS+metrics.SystemTimeUS) / float64(totalTime) * 100
	}

	// Get CPU hotspots
	hotspots := ct.getCPUHotspots(containerInfo)

	now := time.Now()
	windowStart := now.Add(-10 * time.Second) // 10-second window

	return &pb.CPUPatterns{
		UserTimeUs:          metrics.UserTimeUS,
		SystemTimeUs:        metrics.SystemTimeUS,
		IdleTimeUs:          metrics.IdleTimeUS,
		CpuUtilization:      cpuUtilization,
		ContextSwitches:     metrics.ContextSwitches,
		VoluntarySwitches:   metrics.VoluntarySwitches,
		InvoluntarySwitches: metrics.InvoluntarySwitches,
		RunqueueLatencyUs:   float64(metrics.RunqueueLatencyUS),
		ActiveThreads:       metrics.ActiveThreads,
		BlockedThreads:      metrics.BlockedThreads,
		Hotspots:            hotspots,
		WindowStart:         timestampFromTime(windowStart),
		WindowEnd:           timestampFromTime(now),
	}
}

// getCPUHotspots retrieves CPU hotspots for a container
func (ct *CPUTracker) getCPUHotspots(containerInfo *ectx.ContainerInfo) []*pb.CPUHotspot {
	var hotspots []*pb.CPUHotspot

	// TODO: Extract hotspots from eBPF map
	// This would iterate through a hotspots map and convert the data

	return hotspots
}

// SetCPUMetricsHandler sets the callback for CPU metrics
func (ct *CPUTracker) SetCPUMetricsHandler(handler func(*pb.CPUPatterns)) {
	ct.onCPUMetrics = handler
}

// Close releases all resources
func (ct *CPUTracker) Close() error {
	// Close ring buffer reader
	if ct.ringbufReader != nil {
		ct.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range ct.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if ct.objects != nil {
		ct.objects.Close()
	}

	log.Info().Msg("CPU tracker closed")
	return nil
}

// Helper functions

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}