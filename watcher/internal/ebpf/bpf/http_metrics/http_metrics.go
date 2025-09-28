package http_metrics

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

// HTTPRequest represents an active HTTP request being tracked
type HTTPRequest struct {
	StartTime  uint64
	CgroupID   uint64
	PID        uint32
	StatusCode uint32
	Method     [16]byte
	URL        [256]byte
}

// HTTPMetrics represents aggregated HTTP metrics per container
type HTTPMetrics struct {
	RequestCount uint64
	ErrorCount   uint64
	LatencySum   uint64
	LatencyMax   uint64
	LatencyMin   uint64
	Status2xx    uint64
	Status3xx    uint64
	Status4xx    uint64
	Status5xx    uint64
	LastUpdate   uint64
}

// LatencyBucket represents a histogram bucket for latency distribution
type LatencyBucket struct {
	Count      uint64
	UpperBound uint64
}

// HTTPEvent represents a real-time HTTP event from the ring buffer
type HTTPEvent struct {
	Timestamp  uint64
	CgroupID   uint64
	PID        uint32
	StatusCode uint32
	LatencyUS  uint64
	Method     [16]byte
	URL        [64]byte
}

// HTTPMetricsCollector manages the HTTP metrics eBPF program
type HTTPMetricsCollector struct {
	objects       *http_metricsObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Event handlers
	onHTTPEvent func(*pb.HTTPMetrics)
}

// NewHTTPMetricsCollector creates a new HTTP metrics collector
func NewHTTPMetricsCollector(containerMap *ectx.ContainerMapper) (*HTTPMetricsCollector, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &http_metricsObjects{}
	if err := loadHttp_metricsObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load HTTP metrics eBPF objects: %w", err)
	}

	collector := &HTTPMetricsCollector{
		objects:      objects,
		containerMap: containerMap,
	}

	return collector, nil
}

// Start begins collecting HTTP metrics
func (hmc *HTTPMetricsCollector) Start(ctx context.Context) error {
	// Attach tracepoints for syscall-based monitoring
	if err := hmc.attachTracepoints(); err != nil {
		return fmt.Errorf("failed to attach tracepoints: %w", err)
	}

	// Set up ring buffer reader for real-time events
	if err := hmc.setupRingBuffer(ctx); err != nil {
		return fmt.Errorf("failed to setup ring buffer: %w", err)
	}

	log.Info().Msg("HTTP metrics collector started")
	return nil
}

// attachTracepoints attaches the eBPF programs to kernel tracepoints
func (hmc *HTTPMetricsCollector) attachTracepoints() error {
	// Attach to write syscall entry (HTTP request start)
	writeLink, err := link.Tracepoint("syscalls", "sys_enter_write", hmc.objects.TraceHttpRequestStart, nil)
	if err != nil {
		return fmt.Errorf("failed to attach write tracepoint: %w", err)
	}
	hmc.links = append(hmc.links, writeLink)

	// Attach to read syscall exit (HTTP response received)
	readLink, err := link.Tracepoint("syscalls", "sys_exit_read", hmc.objects.TraceHttpRequestEnd, nil)
	if err != nil {
		return fmt.Errorf("failed to attach read tracepoint: %w", err)
	}
	hmc.links = append(hmc.links, readLink)

	log.Info().Msg("HTTP metrics tracepoints attached")
	return nil
}

// AttachGoHTTPUprobes attaches uprobes to Go HTTP handlers
func (hmc *HTTPMetricsCollector) AttachGoHTTPUprobes(executablePath string) error {
	// Attach uprobe to Go HTTP handler function entry
	ex, err := link.OpenExecutable(executablePath)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to open executable")
		return err
	}

	uprobeLink, err := ex.Uprobe("net/http.(*ServeMux).ServeHTTP", hmc.objects.TraceGoHttpStart, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach Go HTTP uprobe")
		return err
	}
	hmc.links = append(hmc.links, uprobeLink)

	// Attach uretprobe to Go HTTP handler function exit
	uretprobeLink, err := ex.Uretprobe("net/http.(*ServeMux).ServeHTTP", hmc.objects.TraceGoHttpEnd, nil)
	if err != nil {
		log.Warn().Err(err).Str("path", executablePath).Msg("Failed to attach Go HTTP uretprobe")
		return err
	}
	hmc.links = append(hmc.links, uretprobeLink)

	log.Info().Str("path", executablePath).Msg("Go HTTP uprobes attached")
	return nil
}

// setupRingBuffer sets up the ring buffer for real-time events
func (hmc *HTTPMetricsCollector) setupRingBuffer(ctx context.Context) error {
	reader, err := ringbuf.NewReader(hmc.objects.HttpEvents)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	hmc.ringbufReader = reader

	// Start ring buffer processing
	go hmc.processRingBufferEvents(ctx)

	return nil
}

// processRingBufferEvents processes real-time HTTP events from the ring buffer
func (hmc *HTTPMetricsCollector) processRingBufferEvents(ctx context.Context) {
	defer hmc.ringbufReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			record, err := hmc.ringbufReader.Read()
			if err != nil {
				if err != ringbuf.ErrClosed {
					log.Error().Err(err).Msg("Failed to read from ring buffer")
				}
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(HTTPEvent{})) {
				log.Warn().Int("size", len(record.RawSample)).Msg("Invalid HTTP event size")
				continue
			}

			// Parse the event
			event := (*HTTPEvent)(unsafe.Pointer(&record.RawSample[0]))

			// Convert to protobuf and emit
			if httpMetrics := hmc.convertEventToMetrics(event); httpMetrics != nil && hmc.onHTTPEvent != nil {
				hmc.onHTTPEvent(httpMetrics)
			}
		}
	}
}

// Poll implements the CollectorProgram interface
func (hmc *HTTPMetricsCollector) Poll(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := hmc.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect HTTP metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (hmc *HTTPMetricsCollector) collectAndEmitMetrics() error {
	var nextKey uint64
	var metrics HTTPMetrics

	iter := hmc.objects.HttpMetricsMap.Iterate()
	for iter.Next(&nextKey, &metrics) {
		cgroupID := nextKey

		// Get container info for this cgroup
		containerInfo, exists := hmc.containerMap.GetContainerByCgroupID(cgroupID)
		if !exists {
			log.Debug().Uint64("cgroup_id", cgroupID).Msg("No container found for cgroup")
			continue
		}

		// Convert to protobuf
		pbMetrics := hmc.convertMetricsToProto(&metrics, containerInfo)
		if pbMetrics != nil && hmc.onHTTPEvent != nil {
			hmc.onHTTPEvent(pbMetrics)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate HTTP metrics map: %w", err)
	}

	return nil
}

// convertEventToMetrics converts a ring buffer event to protobuf metrics
func (hmc *HTTPMetricsCollector) convertEventToMetrics(event *HTTPEvent) *pb.HTTPMetrics {
	_, exists := hmc.containerMap.GetContainerByCgroupID(event.CgroupID)
	if !exists {
		return nil
	}

	// Create a single-event metrics message
	latencyHist := &pb.LatencyHistogram{
		Mean: float64(event.LatencyUS),
		Max:  float64(event.LatencyUS),
		Min:  float64(event.LatencyUS),
	}

	// Set percentiles (for single event, all percentiles are the same)
	latencyHist.P50 = float64(event.LatencyUS)
	latencyHist.P90 = float64(event.LatencyUS)
	latencyHist.P95 = float64(event.LatencyUS)
	latencyHist.P99 = float64(event.LatencyUS)
	latencyHist.P999 = float64(event.LatencyUS)

	statusCodes := make(map[uint32]uint64)
	statusCodes[event.StatusCode] = 1

	methods := make(map[string]uint64)
	methodStr := nullTerminatedBytesToString(event.Method[:])
	if methodStr != "" {
		methods[methodStr] = 1
	}

	now := time.Now()
	timestampProto := timestamppb.New(now)

	return &pb.HTTPMetrics{
		RequestCount:      1,
		ErrorCount:        hmc.getErrorCountForStatus(event.StatusCode),
		ErrorRate:         hmc.getErrorRateForStatus(event.StatusCode),
		Latency:           latencyHist,
		StatusCodes:       statusCodes,
		Methods:           methods,
		ActiveConnections: 0, // Not tracked in this event
		ConnectionErrors:  0, // Not tracked in this event
		WindowStart:       timestampProto,
		WindowEnd:         timestampProto,
	}
}

// convertMetricsToProto converts eBPF metrics to protobuf format
func (hmc *HTTPMetricsCollector) convertMetricsToProto(metrics *HTTPMetrics, containerInfo *ectx.ContainerInfo) *pb.HTTPMetrics {
	if metrics.RequestCount == 0 {
		return nil
	}

	// Calculate latency statistics
	avgLatency := float64(metrics.LatencySum) / float64(metrics.RequestCount)
	errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)

	latencyHist := &pb.LatencyHistogram{
		Mean: avgLatency,
		Max:  float64(metrics.LatencyMax),
		Min:  float64(metrics.LatencyMin),
		// TODO: Calculate actual percentiles from histogram buckets
		P50: avgLatency,
		P90: avgLatency * 1.2,
		P95: avgLatency * 1.5,
		P99: float64(metrics.LatencyMax),
	}

	// Status code distribution
	statusCodes := map[uint32]uint64{
		200: metrics.Status2xx,
		300: metrics.Status3xx,
		400: metrics.Status4xx,
		500: metrics.Status5xx,
	}

	now := time.Now()
	timestampProto := timestamppb.New(now)

	return &pb.HTTPMetrics{
		RequestCount:      metrics.RequestCount,
		ErrorCount:        metrics.ErrorCount,
		ErrorRate:         errorRate,
		Latency:           latencyHist,
		StatusCodes:       statusCodes,
		Methods:           make(map[string]uint64), // TODO: Track methods
		Endpoints:         []*pb.EndpointMetrics{}, // TODO: Implement endpoint tracking
		ActiveConnections: 0,                       // TODO: Track active connections
		ConnectionErrors:  0,                       // TODO: Track connection errors
		WindowStart:       timestampProto,
		WindowEnd:         timestampProto,
	}
}

// SetHTTPEventHandler sets the callback for HTTP events
func (hmc *HTTPMetricsCollector) SetHTTPEventHandler(handler func(*pb.HTTPMetrics)) {
	hmc.onHTTPEvent = handler
}

// Close releases all resources
func (hmc *HTTPMetricsCollector) Close() error {
	// Close ring buffer reader
	if hmc.ringbufReader != nil {
		hmc.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range hmc.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if hmc.objects != nil {
		hmc.objects.Close()
	}

	log.Info().Msg("HTTP metrics collector closed")
	return nil
}

// Helper functions

func (hmc *HTTPMetricsCollector) getErrorCountForStatus(statusCode uint32) uint64 {
	if statusCode >= 400 {
		return 1
	}
	return 0
}

func (hmc *HTTPMetricsCollector) getErrorRateForStatus(statusCode uint32) float64 {
	if statusCode >= 400 {
		return 1.0
	}
	return 0.0
}

func nullTerminatedBytesToString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
