package ebpf

import (
	"context"
	"sync"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/app_specific"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/cpu_tracker"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/fs_tracker"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/http_metrics"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/memory_tracker"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/network_flow"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/security_monitor"
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
	HandleFileSystemMetrics(*pb.FileSystemMetrics)
	HandleApplicationMetrics(*pb.ApplicationMetrics)
	HandleSecurityMetrics(*pb.SecurityMetrics)
}

type EbpfCollector struct {
	programs           []CollectorProgram
	containerMap       *ectx.ContainerMapper
	httpCollector      *http_metrics.HTTPMetricsCollector
	memoryTracker      *memory_tracker.MemoryTracker
	cpuTracker         *cpu_tracker.CPUTracker
	networkFlowTracker *network_flow.NetworkFlowTracker
	fsTracker          *fs_tracker.FSTracker
	appSpecificTracker *app_specific.AppSpecificTracker
	securityMonitor    *security_monitor.SecurityMonitor
	metricsHandler     EBPFMetricsHandler
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
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

	// Initialize CPU tracker
	cpuTracker, err := cpu_tracker.NewCPUTracker(containerMap)
	if err != nil {
		return nil, err
	}

	// Initialize network flow tracker
	networkFlowTracker, err := network_flow.NewNetworkFlowTracker(containerMap)
	if err != nil {
		return nil, err
	}

	// Initialize file system tracker
	fsTracker, err := fs_tracker.NewFSTracker(containerMap)
	if err != nil {
		return nil, err
	}

	// Initialize application-specific tracker
	appSpecificTracker, err := app_specific.NewAppSpecificTracker(containerMap)
	if err != nil {
		return nil, err
	}

	// Initialize security monitor
	securityMonitor, err := security_monitor.NewSecurityMonitor(containerMap)
	if err != nil {
		return nil, err
	}

	// Set up event handlers
	if metricsHandler != nil {
		httpCollector.SetHTTPEventHandler(metricsHandler.HandleHTTPMetrics)
		memoryTracker.SetMemoryMetricsHandler(metricsHandler.HandleMemoryMetrics)
		cpuTracker.SetCPUMetricsHandler(metricsHandler.HandleCPUMetrics)
		networkFlowTracker.SetNetworkMetricsHandler(metricsHandler.HandleNetworkMetrics)
		fsTracker.SetFSMetricsHandler(metricsHandler.HandleFileSystemMetrics)
		appSpecificTracker.SetApplicationMetricsHandler(metricsHandler.HandleApplicationMetrics)
		securityMonitor.SetSecurityMetricsHandler(metricsHandler.HandleSecurityMetrics)
	}

	// Keep the original uprobe for compatibility
	uprobeCollector := uprobe.NewUprobes()

	programs := []CollectorProgram{
		uprobeCollector,
		httpCollector,
		memoryTracker,
		cpuTracker,
		networkFlowTracker,
		fsTracker,
		appSpecificTracker,
		securityMonitor,
	}

	c := &EbpfCollector{
		programs:           programs,
		containerMap:       containerMap,
		httpCollector:      httpCollector,
		memoryTracker:      memoryTracker,
		cpuTracker:         cpuTracker,
		networkFlowTracker: networkFlowTracker,
		fsTracker:          fsTracker,
		appSpecificTracker: appSpecificTracker,
		securityMonitor:    securityMonitor,
		metricsHandler:     metricsHandler,
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

	// Start CPU tracker
	if err := e.cpuTracker.Start(ctx); err != nil {
		return err
	}

	// Start network flow tracker
	if err := e.networkFlowTracker.Start(ctx); err != nil {
		return err
	}

	// Start file system tracker
	if err := e.fsTracker.Start(ctx); err != nil {
		return err
	}

	// Start application-specific tracker
	if err := e.appSpecificTracker.Start(ctx); err != nil {
		return err
	}

	// Start security monitor
	if err := e.securityMonitor.Start(ctx); err != nil {
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
		NodeName: s.nodeName,
		Http:     httpMetrics,
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

// HandleFileSystemMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleFileSystemMetrics(fsMetrics *pb.FileSystemMetrics) {
	if fsMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:   s.nodeName,
		Filesystem: fsMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("File system eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping file system metrics")
	}
}

// HandleApplicationMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleApplicationMetrics(appMetrics *pb.ApplicationMetrics) {
	if appMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName:    s.nodeName,
		Application: appMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("Application eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping application metrics")
	}
}

// HandleSecurityMetrics implements EBPFMetricsHandler
func (s *SenderMetricsHandler) HandleSecurityMetrics(securityMetrics *pb.SecurityMetrics) {
	if securityMetrics == nil {
		return
	}

	ebpfMetrics := &pb.EBPFMetrics{
		NodeName: s.nodeName,
		Security: securityMetrics,
	}

	select {
	case s.ebpfMetricsChan <- ebpfMetrics:
		log.Debug().Msg("Security eBPF metrics sent to sender channel")
	default:
		log.Warn().Msg("eBPF metrics channel full, dropping security metrics")
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
