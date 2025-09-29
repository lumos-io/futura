package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opisvigilant/futura/go-lib/stream"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	StoreEBPFMetricsTopic = "store.ebpf.metrics"
)

type EBPFFlattener struct {
	kc stream.Stream
}

func NewEBPFFlattener(kc stream.Stream) *EBPFFlattener {
	return &EBPFFlattener{
		kc: kc,
	}
}

type flatEBPFMetrics struct {
	OrganizationID uint32 `json:"organization_id"`
	ClusterID      int64  `json:"cluster_id"`
	K8SVersion     string `json:"k8s_version"`
	ReceivedAtUnix int64  `json:"received_at_unix"`
	TimestampUnix  int64  `json:"timestamp_unix"`

	// Container context
	ContainerID string `json:"container_id"`
	PodName     string `json:"pod_name"`
	PodUID      string `json:"pod_uid"`
	Namespace   string `json:"namespace"`
	NodeName    string `json:"node_name"`
	AppName     string `json:"app_name"`
	AppVersion  string `json:"app_version"`
	ServiceName string `json:"service_name"`
	CgroupID    uint64 `json:"cgroup_id"`

	// HTTP metrics
	HTTPRequestCount uint64  `json:"http_request_count,omitempty"`
	HTTPErrorCount   uint64  `json:"http_error_count,omitempty"`
	HTTPErrorRate    float64 `json:"http_error_rate,omitempty"`
	HTTPLatencyP50   float64 `json:"http_latency_p50,omitempty"`
	HTTPLatencyP95   float64 `json:"http_latency_p95,omitempty"`
	HTTPLatencyP99   float64 `json:"http_latency_p99,omitempty"`

	// Memory metrics
	MemoryAllocCount      uint64 `json:"memory_alloc_count,omitempty"`
	MemoryFreeCount       uint64 `json:"memory_free_count,omitempty"`
	MemoryBytesAllocated  uint64 `json:"memory_bytes_allocated,omitempty"`
	MemoryBytesFreed      uint64 `json:"memory_bytes_freed,omitempty"`
	MemoryNetAllocated    uint64 `json:"memory_net_allocated,omitempty"`
	MemoryPageFaults      uint64 `json:"memory_page_faults,omitempty"`
	MemoryMajorPageFaults uint64 `json:"memory_major_page_faults,omitempty"`
	MemoryGCCount         uint64 `json:"memory_gc_count,omitempty"`
	MemoryGCTimeUs        uint64 `json:"memory_gc_time_us,omitempty"`

	// CPU metrics
	CPUUserTimeUs          uint64  `json:"cpu_user_time_us,omitempty"`
	CPUSystemTimeUs        uint64  `json:"cpu_system_time_us,omitempty"`
	CPUUtilization         float64 `json:"cpu_utilization,omitempty"`
	CPUContextSwitches     uint64  `json:"cpu_context_switches,omitempty"`
	CPUVoluntarySwitches   uint64  `json:"cpu_voluntary_switches,omitempty"`
	CPUInvoluntarySwitches uint64  `json:"cpu_involuntary_switches,omitempty"`
	CPURunqueueLatencyUs   float64 `json:"cpu_runqueue_latency_us,omitempty"`
	CPUActiveThreads       uint32  `json:"cpu_active_threads,omitempty"`
	CPUBlockedThreads      uint32  `json:"cpu_blocked_threads,omitempty"`

	// Network metrics
	NetworkBytesSent              uint64  `json:"network_bytes_sent,omitempty"`
	NetworkBytesReceived          uint64  `json:"network_bytes_received,omitempty"`
	NetworkPacketsSent            uint64  `json:"network_packets_sent,omitempty"`
	NetworkPacketsReceived        uint64  `json:"network_packets_received,omitempty"`
	NetworkConnectionsEstablished uint64  `json:"network_connections_established,omitempty"`
	NetworkConnectionsClosed      uint64  `json:"network_connections_closed,omitempty"`
	NetworkConnectionFailures     uint64  `json:"network_connection_failures,omitempty"`
	NetworkAvgRTTUs               float64 `json:"network_avg_rtt_us,omitempty"`

	// File system metrics
	FSReadOps            uint64  `json:"fs_read_ops,omitempty"`
	FSWriteOps           uint64  `json:"fs_write_ops,omitempty"`
	FSBytesRead          uint64  `json:"fs_bytes_read,omitempty"`
	FSBytesWritten       uint64  `json:"fs_bytes_written,omitempty"`
	FSReadLatencyP95Us   float64 `json:"fs_read_latency_p95_us,omitempty"`
	FSWriteLatencyP95Us  float64 `json:"fs_write_latency_p95_us,omitempty"`
	FSReadBandwidthMbps  float64 `json:"fs_read_bandwidth_mbps,omitempty"`
	FSWriteBandwidthMbps float64 `json:"fs_write_bandwidth_mbps,omitempty"`

	// Application metrics
	AppDBQueryCount  uint64  `json:"app_db_query_count,omitempty"`
	AppDBSlowQueries uint64  `json:"app_db_slow_queries,omitempty"`
	AppCacheHitRate  float64 `json:"app_cache_hit_rate,omitempty"`
	AppGCCount       uint64  `json:"app_gc_count,omitempty"`
	AppGCTimeUs      uint64  `json:"app_gc_time_us,omitempty"`

	// Security metrics
	SecurityAnomalousCount            uint64 `json:"security_anomalous_count,omitempty"`
	SecurityPrivilegeEscalationCount  uint64 `json:"security_privilege_escalation_count,omitempty"`
	SecuritySuspiciousConnectionCount uint64 `json:"security_suspicious_connection_count,omitempty"`
	SecuritySuspiciousProcessCount    uint64 `json:"security_suspicious_process_count,omitempty"`
	SecurityUnauthorizedAccessCount   uint64 `json:"security_unauthorized_access_count,omitempty"`
}

func (ef *EBPFFlattener) Flatten(ctx context.Context, msg *pb.EBPFMetrics) error {
	if msg == nil {
		return fmt.Errorf("nil EBPFMetrics")
	}

	data := &flatEBPFMetrics{
		OrganizationID: safeUInt32Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) uint32 { return e.OrganizationId }),
		ClusterID:      safeInt64Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) int64 { return e.ClusterId }),
		K8SVersion:     safeStringPtr(msg.Enrichment, func(e *pb.EnrichmentMetadata) string { return e.K8SVersion }),
		ReceivedAtUnix: safeInt64Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) int64 { return e.ReceivedAtUnix }),
		TimestampUnix:  safeTimestampPtr(msg.Timestamp),

		// Container context
		ContainerID: msg.GetContainerId(),
		PodName:     msg.GetPodName(),
		PodUID:      msg.GetPodUid(),
		Namespace:   msg.GetNamespace(),
		NodeName:    msg.GetNodeName(),
		AppName:     msg.GetAppName(),
		AppVersion:  msg.GetAppVersion(),
		ServiceName: msg.GetServiceName(),
		CgroupID:    msg.GetCgroupId(),
	}

	// HTTP metrics
	if msg.Http != nil {
		data.HTTPRequestCount = msg.Http.RequestCount
		data.HTTPErrorCount = msg.Http.ErrorCount
		data.HTTPErrorRate = msg.Http.ErrorRate
		if msg.Http.Latency != nil {
			data.HTTPLatencyP50 = msg.Http.Latency.P50
			data.HTTPLatencyP95 = msg.Http.Latency.P95
			data.HTTPLatencyP99 = msg.Http.Latency.P99
		}
	}

	// Memory metrics
	if msg.MemoryPatterns != nil {
		data.MemoryAllocCount = msg.MemoryPatterns.AllocCount
		data.MemoryFreeCount = msg.MemoryPatterns.FreeCount
		data.MemoryBytesAllocated = msg.MemoryPatterns.BytesAllocated
		data.MemoryBytesFreed = msg.MemoryPatterns.BytesFreed
		data.MemoryNetAllocated = msg.MemoryPatterns.NetAllocated
		data.MemoryPageFaults = msg.MemoryPatterns.PageFaults
		data.MemoryMajorPageFaults = msg.MemoryPatterns.MajorPageFaults
		if msg.MemoryPatterns.Gc != nil {
			data.MemoryGCCount = msg.MemoryPatterns.Gc.GcCount
			data.MemoryGCTimeUs = msg.MemoryPatterns.Gc.GcTimeUs
		}
	}

	// CPU metrics
	if msg.CpuPatterns != nil {
		data.CPUUserTimeUs = msg.CpuPatterns.UserTimeUs
		data.CPUSystemTimeUs = msg.CpuPatterns.SystemTimeUs
		data.CPUUtilization = msg.CpuPatterns.CpuUtilization
		data.CPUContextSwitches = msg.CpuPatterns.ContextSwitches
		data.CPUVoluntarySwitches = msg.CpuPatterns.VoluntarySwitches
		data.CPUInvoluntarySwitches = msg.CpuPatterns.InvoluntarySwitches
		data.CPURunqueueLatencyUs = msg.CpuPatterns.RunqueueLatencyUs
		data.CPUActiveThreads = msg.CpuPatterns.ActiveThreads
		data.CPUBlockedThreads = msg.CpuPatterns.BlockedThreads
	}

	// Network metrics
	if msg.NetworkFlow != nil {
		data.NetworkBytesSent = msg.NetworkFlow.BytesSent
		data.NetworkBytesReceived = msg.NetworkFlow.BytesReceived
		data.NetworkPacketsSent = msg.NetworkFlow.PacketsSent
		data.NetworkPacketsReceived = msg.NetworkFlow.PacketsReceived
		data.NetworkConnectionsEstablished = msg.NetworkFlow.ConnectionsEstablished
		data.NetworkConnectionsClosed = msg.NetworkFlow.ConnectionsClosed
		data.NetworkConnectionFailures = msg.NetworkFlow.ConnectionFailures
		data.NetworkAvgRTTUs = msg.NetworkFlow.AvgRttUs
	}

	// File system metrics
	if msg.Filesystem != nil {
		data.FSReadOps = msg.Filesystem.ReadOps
		data.FSWriteOps = msg.Filesystem.WriteOps
		data.FSBytesRead = msg.Filesystem.BytesRead
		data.FSBytesWritten = msg.Filesystem.BytesWritten
		if msg.Filesystem.ReadLatency != nil {
			data.FSReadLatencyP95Us = msg.Filesystem.ReadLatency.P95
		}
		if msg.Filesystem.WriteLatency != nil {
			data.FSWriteLatencyP95Us = msg.Filesystem.WriteLatency.P95
		}
		data.FSReadBandwidthMbps = msg.Filesystem.ReadBandwidthMbps
		data.FSWriteBandwidthMbps = msg.Filesystem.WriteBandwidthMbps
	}

	// Application metrics
	if msg.Application != nil {
		if msg.Application.Database != nil {
			data.AppDBQueryCount = msg.Application.Database.QueryCount
			data.AppDBSlowQueries = msg.Application.Database.SlowQueries
		}
		if msg.Application.Cache != nil {
			data.AppCacheHitRate = msg.Application.Cache.HitRate
		}
		// Note: GC metrics are in MemoryPatterns.Gc, already extracted above
	}

	// Security metrics
	if msg.Security != nil {
		if msg.Security.Syscalls != nil {
			data.SecurityAnomalousCount = uint64(len(msg.Security.Syscalls.Anomalies))
			data.SecurityPrivilegeEscalationCount = msg.Security.Syscalls.PrivilegeEscalationAttempts
		}
		if msg.Security.NetworkSecurity != nil {
			data.SecuritySuspiciousConnectionCount = msg.Security.NetworkSecurity.SuspiciousConnections
		}
		if msg.Security.ProcessSecurity != nil {
			data.SecuritySuspiciousProcessCount = msg.Security.ProcessSecurity.SuspiciousProcesses
		}
		if msg.Security.FilesystemSecurity != nil {
			data.SecurityUnauthorizedAccessCount = msg.Security.FilesystemSecurity.UnauthorizedFileAccess
		}
	}

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal eBPF metrics: %w", err)
	}
	return ef.kc.Publish(ctx, StoreEBPFMetricsTopic, b)
}
