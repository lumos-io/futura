package security_monitor

import (
	"context"
	"fmt"
	"strings"
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

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" security_monitor security_monitor.bpf.c -- -I../../../../headers

// SecurityMetrics represents security metrics per container
type SecurityMetrics struct {
	// Syscall monitoring
	TotalSyscalls                uint64
	SuspiciousSyscalls           uint64
	PrivilegeEscalationAttempts  uint64
	SyscallRateViolations        uint64

	// Resource violations
	MemoryViolations             uint64
	CPUViolations                uint64
	FDViolations                 uint64
	ProcessViolations            uint64

	// Network security
	SuspiciousConnections        uint64
	BlockedConnections           uint64
	PortScanAttempts             uint64

	// Process security
	NewProcesses                 uint64
	SuspiciousProcesses          uint64
	SetuidExecutions             uint64
	ContainerEscapeAttempts      uint64

	// File system security
	UnauthorizedFileAccess       uint64
	SensitiveFileAccess          uint64
	SystemFileModifications      uint64

	LastUpdate                   uint64
}

// SyscallStats represents syscall frequency statistics
type SyscallStats struct {
	Count         uint64
	LastSeen      uint64
	AvgFrequency  uint64
	IsSuspicious  uint8
}

// ProcessInfo represents tracked process information
type ProcessInfo struct {
	PID           uint32
	PPID          uint32
	UID           uint32
	GID           uint32
	StartTime     uint64
	Comm          [16]byte
	Filename      [256]byte
	IsSuspicious  uint8
	CgroupID      uint64
}

// SecurityEvent represents a real-time security event
type SecurityEvent struct {
	Timestamp    uint64
	CgroupID     uint64
	PID          uint32
	EventType    uint32
	Severity     uint32
	Value1       uint64
	Value2       uint64
	Description  [256]byte
}

// SecurityMonitor manages the security monitoring eBPF program
type SecurityMonitor struct {
	objects       *security_monitorObjects
	links         []link.Link
	ringbufReader *ringbuf.Reader
	containerMap  *ectx.ContainerMapper

	// Baseline tracking for anomaly detection
	syscallBaselines map[string]uint64
	startTime        time.Time

	// Event handlers
	onSecurityMetrics func(*pb.SecurityMetrics)
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(containerMap *ectx.ContainerMapper) (*SecurityMonitor, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock: %w", err)
	}

	objects := &security_monitorObjects{}
	if err := loadSecurity_monitorObjects(objects, nil); err != nil {
		return nil, fmt.Errorf("failed to load security monitor eBPF objects: %w", err)
	}

	monitor := &SecurityMonitor{
		objects:          objects,
		containerMap:     containerMap,
		syscallBaselines: make(map[string]uint64),
		startTime:        time.Now(),
	}

	// Initialize security allowlists and sensitive paths
	if err := monitor.initializeSecurityPolicies(); err != nil {
		return nil, fmt.Errorf("failed to initialize security policies: %w", err)
	}

	return monitor, nil
}

// initializeSecurityPolicies sets up allowlists and sensitive file monitoring
func (sm *SecurityMonitor) initializeSecurityPolicies() error {
	// Initialize syscall allowlist with common safe syscalls
	safeSyscalls := []uint32{
		0,   // read
		1,   // write
		2,   // open
		3,   // close
		4,   // stat
		5,   // fstat
		6,   // lstat
		7,   // poll
		8,   // lseek
		9,   // mmap
		10,  // mprotect
		11,  // munmap
		12,  // brk
		13,  // rt_sigaction
		14,  // rt_sigprocmask
		15,  // rt_sigreturn
		16,  // ioctl
		17,  // pread64
		18,  // pwrite64
		19,  // readv
		20,  // writev
		// Add more safe syscalls as needed
	}

	allowed := uint8(1)
	for _, syscallNum := range safeSyscalls {
		if err := sm.objects.SyscallAllowlist.Put(syscallNum, allowed); err != nil {
			log.Warn().Uint32("syscall", syscallNum).Err(err).Msg("Failed to add syscall to allowlist")
		}
	}

	// Initialize sensitive file paths
	sensitivePaths := map[string]uint8{
		"/etc/passwd":         3, // Critical
		"/etc/shadow":         3, // Critical
		"/etc/sudoers":        3, // Critical
		"/etc/ssh/":           2, // High
		"/root/.ssh/":         2, // High
		"/proc/sys/":          2, // High
		"/sys/class/":         2, // High
		"/etc/crontab":        2, // High
		"/var/log/auth.log":   1, // Medium
		"/var/log/secure":     1, // Medium
	}

	for path, sensitivity := range sensitivePaths {
		pathHash := hashString(path)
		if err := sm.objects.SensitivePathsMap.Put(pathHash, sensitivity); err != nil {
			log.Warn().Str("path", path).Err(err).Msg("Failed to add sensitive path")
		}
	}

	return nil
}

// hashString creates a simple hash of a string
func hashString(s string) uint64 {
	hash := uint64(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}

// Start begins security monitoring
func (sm *SecurityMonitor) Start(ctx context.Context) error {
	// Attach raw tracepoints for syscall monitoring
	if err := sm.attachSyscallTracepoints(); err != nil {
		return fmt.Errorf("failed to attach syscall tracepoints: %w", err)
	}

	// Attach process monitoring tracepoints
	if err := sm.attachProcessTracepoints(); err != nil {
		return fmt.Errorf("failed to attach process tracepoints: %w", err)
	}

	// Attach network monitoring kprobes
	if err := sm.attachNetworkKprobes(); err != nil {
		log.Warn().Err(err).Msg("Failed to attach network kprobes, continuing without network monitoring")
	}

	// Attach memory/resource monitoring tracepoints
	if err := sm.attachResourceTracepoints(); err != nil {
		log.Warn().Err(err).Msg("Failed to attach resource tracepoints, continuing without resource monitoring")
	}

	// Set up ring buffer reader
	if err := sm.setupRingBuffer(ctx); err != nil {
		return fmt.Errorf("failed to setup ring buffer: %w", err)
	}

	log.Info().Msg("Security monitor started")
	return nil
}

// attachSyscallTracepoints attaches syscall monitoring tracepoints
func (sm *SecurityMonitor) attachSyscallTracepoints() error {
	// Attach to raw syscall tracepoint for comprehensive monitoring
	syscallLink, err := link.AttachRawTracepoint(link.RawTracepointOptions{
		Name:    "sys_enter",
		Program: sm.objects.TraceSysEnter,
	})
	if err != nil {
		return fmt.Errorf("failed to attach sys_enter tracepoint: %w", err)
	}
	sm.links = append(sm.links, syscallLink)

	log.Info().Msg("Syscall monitoring tracepoints attached")
	return nil
}

// attachProcessTracepoints attaches process monitoring tracepoints
func (sm *SecurityMonitor) attachProcessTracepoints() error {
	// Attach to process execution tracepoint
	execLink, err := link.Tracepoint("sched", "sched_process_exec", sm.objects.TraceProcessExec, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sched_process_exec tracepoint: %w", err)
	}
	sm.links = append(sm.links, execLink)

	// Attach to file access tracepoint
	fileLink, err := link.Tracepoint("syscalls", "sys_enter_openat", sm.objects.TraceFileAccess, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_enter_openat tracepoint: %w", err)
	}
	sm.links = append(sm.links, fileLink)

	log.Info().Msg("Process monitoring tracepoints attached")
	return nil
}

// attachNetworkKprobes attaches network security monitoring kprobes
func (sm *SecurityMonitor) attachNetworkKprobes() error {
	// Attach to TCP connection kprobe
	tcpLink, err := link.Kprobe("tcp_connect", sm.objects.TraceTcpConnect, nil)
	if err != nil {
		return fmt.Errorf("failed to attach tcp_connect kprobe: %w", err)
	}
	sm.links = append(sm.links, tcpLink)

	log.Info().Msg("Network monitoring kprobes attached")
	return nil
}

// attachResourceTracepoints attaches resource monitoring tracepoints
func (sm *SecurityMonitor) attachResourceTracepoints() error {
	// Attach to memory pressure tracepoint
	memoryLink, err := link.Tracepoint("vmscan", "mm_vmscan_memcg_softlimit_reclaim_begin", sm.objects.TraceMemoryPressure, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to attach memory pressure tracepoint")
	} else {
		sm.links = append(sm.links, memoryLink)
	}

	// Attach to OOM tracepoint
	oomLink, err := link.Tracepoint("oom", "oom_score_adj_update", sm.objects.TraceOomEvent, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to attach OOM tracepoint")
	} else {
		sm.links = append(sm.links, oomLink)
	}

	log.Info().Msg("Resource monitoring tracepoints attached")
	return nil
}

// setupRingBuffer sets up the ring buffer for real-time security events
func (sm *SecurityMonitor) setupRingBuffer(ctx context.Context) error {
	reader, err := ringbuf.NewReader(sm.objects.SecurityEvents)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	sm.ringbufReader = reader

	// Start ring buffer processing
	go sm.processRingBufferEvents(ctx)

	return nil
}

// processRingBufferEvents processes real-time security events
func (sm *SecurityMonitor) processRingBufferEvents(ctx context.Context) {
	defer sm.ringbufReader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			record, err := sm.ringbufReader.Read()
			if err != nil {
				if err != ringbuf.ErrClosed {
					log.Error().Err(err).Msg("Failed to read from security events ring buffer")
				}
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(SecurityEvent{})) {
				log.Warn().Int("size", len(record.RawSample)).Msg("Invalid security event size")
				continue
			}

			// Parse the event
			event := (*SecurityEvent)(unsafe.Pointer(&record.RawSample[0]))

			// Process the event
			sm.processSecurityEvent(event)
		}
	}
}

// processSecurityEvent processes a single security event
func (sm *SecurityMonitor) processSecurityEvent(event *SecurityEvent) {
	containerInfo, exists := sm.containerMap.GetContainerByCgroupID(event.CgroupID)
	if !exists {
		return
	}

	// Convert description to string
	description := string(event.Description[:])
	description = strings.TrimRight(description, "\x00")

	// Log security events based on severity
	logger := log.Debug()
	switch event.Severity {
	case 3: // Critical
		logger = log.Error()
	case 2: // High
		logger = log.Warn()
	case 1: // Medium
		logger = log.Info()
	default: // Low
		logger = log.Debug()
	}

	eventType := "unknown"
	switch event.EventType {
	case 0:
		eventType = "syscall_anomaly"
	case 1:
		eventType = "resource_violation"
	case 2:
		eventType = "network_anomaly"
	case 3:
		eventType = "process_anomaly"
	case 4:
		eventType = "file_anomaly"
	}

	logger.
		Str("container", containerInfo.ContainerID).
		Str("app", containerInfo.AppName).
		Str("event_type", eventType).
		Uint32("severity", event.Severity).
		Uint32("pid", event.PID).
		Str("description", description).
		Msg("Security event detected")
}

// Poll implements the CollectorProgram interface
func (sm *SecurityMonitor) Poll(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second) // Security metrics every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sm.collectAndEmitMetrics(); err != nil {
				log.Error().Err(err).Msg("Failed to collect security metrics")
			}
		}
	}
}

// collectAndEmitMetrics reads current metrics from eBPF maps and emits them
func (sm *SecurityMonitor) collectAndEmitMetrics() error {
	var nextKey uint64
	var metrics SecurityMetrics

	iter := sm.objects.SecurityMetricsMap.Iterate()
	for iter.Next(&nextKey, &metrics) {
		cgroupID := nextKey

		// Get container info for this cgroup
		containerInfo, exists := sm.containerMap.GetContainerByCgroupID(cgroupID)
		if !exists {
			log.Debug().Uint64("cgroup_id", cgroupID).Msg("No container found for cgroup")
			continue
		}

		// Convert to protobuf
		pbMetrics := sm.convertMetricsToProto(&metrics, containerInfo)
		if pbMetrics != nil && sm.onSecurityMetrics != nil {
			sm.onSecurityMetrics(pbMetrics)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate security metrics map: %w", err)
	}

	return nil
}

// convertMetricsToProto converts eBPF metrics to protobuf format
func (sm *SecurityMonitor) convertMetricsToProto(metrics *SecurityMetrics, containerInfo *ectx.ContainerInfo) *pb.SecurityMetrics {
	if metrics.TotalSyscalls == 0 && metrics.NewProcesses == 0 {
		return nil
	}

	now := time.Now()
	windowStart := now.Add(-30 * time.Second)

	// Calculate syscall rate
	windowSeconds := 30.0
	syscallRate := float64(metrics.TotalSyscalls) / windowSeconds
	highRateDetected := syscallRate > 1000.0 // Threshold for high syscall rate

	// Build syscall metrics
	syscallMetrics := &pb.SyscallMetrics{
		TotalSyscalls:                metrics.TotalSyscalls,
		SyscallCounts:                sm.getSyscallCounts(),
		Anomalies:                    []*pb.SyscallAnomaly{}, // TODO: Extract from eBPF maps
		PrivilegeEscalationAttempts:  metrics.PrivilegeEscalationAttempts,
		SuspiciousPatterns:           metrics.SuspiciousSyscalls,
		SyscallRatePerSecond:         syscallRate,
		HighSyscallRateDetected:      highRateDetected,
	}

	// Build resource violations
	resourceViolations := &pb.ResourceViolations{
		MemoryLimitViolations:        metrics.MemoryViolations,
		OomKills:                     0, // TODO: Extract from eBPF
		MemoryPressureEvents:         metrics.MemoryViolations,
		CpuThrottlingEvents:          metrics.CPUViolations,
		CpuQuotaViolations:           metrics.CPUViolations,
		FdLimitViolations:            metrics.FDViolations,
		ProcessLimitViolations:       metrics.ProcessViolations,
		NetworkBandwidthViolations:   0, // TODO: Implement
		ConnectionLimitViolations:    0, // TODO: Implement
		DiskQuotaViolations:          0, // TODO: Implement
		IopsLimitViolations:          0, // TODO: Implement
	}

	// Build network security metrics
	networkSecurity := &pb.NetworkSecurity{
		SuspiciousConnections:        metrics.SuspiciousConnections,
		PortScanningAttempts:         metrics.PortScanAttempts,
		BlockedOutboundConnections:   metrics.BlockedConnections,
		NetworkAnomalies:             []*pb.NetworkAnomaly{}, // TODO: Extract from eBPF
		ProtocolViolations:           0, // TODO: Implement
		BlockedProtocols:             map[string]uint64{},   // TODO: Implement
	}

	// Build process security metrics
	processSecurity := &pb.ProcessSecurity{
		NewProcesses:                 metrics.NewProcesses,
		SuspiciousProcesses:          metrics.SuspiciousProcesses,
		SuspiciousProcessList:        []*pb.SuspiciousProcess{}, // TODO: Extract from eBPF
		SetuidExecutions:             metrics.SetuidExecutions,
		ContainerEscapeAttempts:      metrics.ContainerEscapeAttempts,
	}

	// Build filesystem security metrics
	filesystemSecurity := &pb.FileSystemSecurity{
		UnauthorizedFileAccess:       metrics.UnauthorizedFileAccess,
		SensitiveFileAccess:          metrics.SensitiveFileAccess,
		SensitiveAccesses:            []*pb.SensitiveFileAccess{}, // TODO: Extract from eBPF
		SystemFileModifications:      metrics.SystemFileModifications,
	}

	return &pb.SecurityMetrics{
		Syscalls:            syscallMetrics,
		ResourceViolations:  resourceViolations,
		NetworkSecurity:     networkSecurity,
		ProcessSecurity:     processSecurity,
		FilesystemSecurity:  filesystemSecurity,
		WindowStart:         timestampFromTime(windowStart),
		WindowEnd:           timestampFromTime(now),
	}
}

// getSyscallCounts retrieves syscall frequency data from eBPF maps
func (sm *SecurityMonitor) getSyscallCounts() map[string]uint64 {
	syscallCounts := make(map[string]uint64)

	var nextKey uint32
	var stats SyscallStats

	iter := sm.objects.SyscallStatsMap.Iterate()
	for iter.Next(&nextKey, &stats) {
		syscallNum := nextKey
		// Convert syscall number to name (simplified)
		syscallName := fmt.Sprintf("syscall_%d", syscallNum)
		syscallCounts[syscallName] = stats.Count
	}

	return syscallCounts
}

// SetSecurityMetricsHandler sets the callback for security metrics
func (sm *SecurityMonitor) SetSecurityMetricsHandler(handler func(*pb.SecurityMetrics)) {
	sm.onSecurityMetrics = handler
}

// Close releases all resources
func (sm *SecurityMonitor) Close() error {
	// Close ring buffer reader
	if sm.ringbufReader != nil {
		sm.ringbufReader.Close()
	}

	// Detach all links
	for _, l := range sm.links {
		if err := l.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close eBPF link")
		}
	}

	// Close eBPF objects
	if sm.objects != nil {
		sm.objects.Close()
	}

	log.Info().Msg("Security monitor closed")
	return nil
}

// Helper functions

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}