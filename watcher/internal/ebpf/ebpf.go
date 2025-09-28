package ebpf

import (
	"context"
	"sync"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/http_metrics"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/memory_tracker"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/uprobe"
	ectx "github.com/opisvigilant/futura/watcher/internal/ebpf/context"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

// CollectorProgram defines a common interface for eBPF programs.
type CollectorProgram interface {
	// Poll runs the program until the context is canceled.
	Poll(ctx context.Context) error
	// Close releases resources.
	Close() error
}

// EBPFMetricsHandler handles eBPF metrics and sends them to telemetry
type EBPFMetricsHandler interface {
	HandleHTTPMetrics(*pb.HTTPMetrics)
	HandleMemoryMetrics(*pb.MemoryPatterns)
	HandleCPUMetrics(*pb.CPUPatterns)
	HandleNetworkMetrics(*pb.NetworkFlow)
}

type EbpfCollector struct {
	programs       []CollectorProgram
	containerMap   *ectx.ContainerMapper
	httpCollector  *http_metrics.HTTPMetricsCollector
	memoryTracker  *memory_tracker.MemoryTracker
	metricsHandler EBPFMetricsHandler
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewEbpfCollector(kubeClient kubernetes.Interface, nodeName string, metricsHandler EBPFMetricsHandler) (*EbpfCollector, error) {
	// Initialize container mapper
	containerMap := ectx.NewContainerMapper(kubeClient, nodeName)

	// Initialize HTTP metrics collector
	httpCollector, err := http_metrics.NewHTTPMetricsCollector(containerMap)
	if err != nil {
		return nil, err
	}

	// Initialize memory tracker
	memoryTracker, err := memory_tracker.NewMemoryTracker(containerMap)
	if err != nil {
		return nil, err
	}

	// Set up event handlers
	if metricsHandler != nil {
		httpCollector.SetHTTPEventHandler(metricsHandler.HandleHTTPMetrics)
		memoryTracker.SetMemoryMetricsHandler(metricsHandler.HandleMemoryMetrics)
	}

	// Keep the original uprobe for compatibility
	uprobeCollector := uprobe.NewUprobes()

	programs := []CollectorProgram{
		uprobeCollector,
		httpCollector,
		memoryTracker,
	}

	c := &EbpfCollector{
		programs:       programs,
		containerMap:   containerMap,
		httpCollector:  httpCollector,
		memoryTracker:  memoryTracker,
		metricsHandler: metricsHandler,
	}
	return c, nil
}

func (e *EbpfCollector) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	// Start container mapper
	if err := e.containerMap.Start(ctx); err != nil {
		return err
	}

	// Start HTTP collector
	if err := e.httpCollector.Start(ctx); err != nil {
		return err
	}

	// Start memory tracker
	if err := e.memoryTracker.Start(ctx); err != nil {
		return err
	}

	// Start all programs
	for _, prog := range e.programs {
		e.wg.Add(1)
		go func(p CollectorProgram) {
			defer e.wg.Done()
			if err := p.Poll(ctx); err != nil {
				log.Error().Err(err).Msg("program poll error")
			}
		}(prog)
	}

	log.Info().Msg("eBPF collector started with enhanced metrics")
	return nil
}

// SenderMetricsHandler integrates eBPF metrics with the sender channel
type SenderMetricsHandler struct {
	ebpfMetricsChan chan *pb.EBPFMetrics
	nodeName        string
	clusterID       string
}

// NewSenderMetricsHandler creates a handler that sends metrics to the sender's eBPF channel
func NewSenderMetricsHandler(ebpfMetricsChan chan *pb.EBPFMetrics, nodeName, clusterID string) *SenderMetricsHandler {
	return &SenderMetricsHandler{
		ebpfMetricsChan: ebpfMetricsChan,
		nodeName:        nodeName,
		clusterID:       clusterID,
	}
}

// HandleHTTPMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleHTTPMetrics(httpMetrics *pb.HTTPMetrics) {
	if httpMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:  s.nodeName,
		Http:      httpMetrics,
		// Container context will be enriched by HTTP collector
		// API key and metadata will be added by sender
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("HTTP eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping HTTP metrics")
	}
}

// HandleMemoryMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleMemoryMetrics(memoryMetrics *pb.MemoryPatterns) {
	if memoryMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:       s.nodeName,
		MemoryPatterns: memoryMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("Memory eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping memory metrics")
	}
}

// HandleCPUMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleCPUMetrics(cpuMetrics *pb.CPUPatterns) {
	if cpuMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:    s.nodeName,
		CpuPatterns: cpuMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("CPU eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping CPU metrics")
	}
}

// HandleNetworkMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleNetworkMetrics(networkMetrics *pb.NetworkFlow) {
	if networkMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:    s.nodeName,
		NetworkFlow: networkMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("Network eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping network metrics")
	}
}

// NewEbpfCollectorWithSender creates an eBPF collector integrated with the sender system
func NewEbpfCollectorWithSender(kubeClient kubernetes.Interface, nodeName, clusterID string, ebpfMetricsChan chan *pb.EBPFMetrics) (*EbpfCollector, error) {
	// Create sender handler that integrates with your existing sender
	senderHandler := NewSenderMetricsHandler(ebpfMetricsChan, nodeName, clusterID)

	// Create collector with the sender handler
	return NewEbpfCollector(kubeClient, nodeName, senderHandler)
}

// Stop all programs gracefully
func (e *EbpfCollector) Close() error {
	if e.cancel != nil {
		e.cancel()
	}

	e.wg.Wait()

	for _, prog := range e.programs {
		prog.Close()
	}
	return nil
}
