# Enhanced eBPF Application Metrics Collection

This implementation provides comprehensive application-level metrics collection using eBPF for Kubernetes containers, designed to feed high-quality data into your RL-based scaling engine.

## Quick Start

### 1. Generate Protocol Buffers

```bash
cd /Users/davideberdin/Documents/github/opisvigilant/futura
make proto-files
```

### 2. Generate eBPF Objects

```bash
cd watcher/internal/ebpf/bpf/http_metrics
go generate

cd ../memory_tracker
go generate

cd ../cpu_tracker
go generate
```

### 3. Build and Test

```bash
cd watcher
go build ./internal/ebpf/...
```

## Implementation Status

### ✅ Completed Components

#### **Container Context Mapping** (`context/container_mapping.go`)

- Real-time Kubernetes pod/container discovery
- cgroup ID ↔ container metadata mapping
- Process ID attribution to containers
- Application label extraction (app, version, service)

#### **HTTP/gRPC Metrics** (`bpf/http_metrics/`)

- Request latency tracking (P50, P95, P99)
- Error rate monitoring with status codes
- Real-time event streaming via ring buffers
- Support for both syscall tracepoints and Go uprobes

#### **Memory Allocation Tracking** (`bpf/memory_tracker/`)

- Kernel and user-space allocation monitoring
- Memory leak detection with stack traces
- Allocation size histograms
- Page fault tracking
- GC metrics for managed languages

#### **CPU Scheduling** (`bpf/cpu_tracker/`)

- Context switch tracking (voluntary vs involuntary)
- Thread lifecycle monitoring
- Basic CPU profiling support

#### **Telemetry Integration** (`telemetry_integration.go`)

- Channel-based metrics processing
- Real-time event handling
- External handler support for ClickHouse/Prometheus

### 🔧 Next Steps to Complete

#### **1. Generate Missing eBPF Objects**

```bash
# Run these commands to generate the missing Go bindings:
cd watcher/internal/ebpf/bpf/http_metrics && go generate
cd ../memory_tracker && go generate
cd ../cpu_tracker && go generate
```

#### **2. Fix Remaining Compilation Issues**

- The main remaining issues are missing generated eBPF objects (`*Objects`, `load*Objects` functions)
- These will be resolved once `go generate` is run

#### **3. Test with Sample Applications**

```go
// Example usage:
telemetryHandler := ebpf.NewTelemetryIntegration()
collector, err := ebpf.NewEbpfCollector(kubeClient, nodeName, telemetryHandler)
if err != nil {
    log.Fatal(err)
}

// Start collecting metrics
ctx := context.Background()
if err := collector.Start(ctx); err != nil {
    log.Fatal(err)
}

// Attach to specific Go applications
collector.AttachGoHTTPUprobes("/path/to/your/go/app")
```

## Architecture Overview

### Data Flow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   eBPF Programs │────┤ Container Mapper ├────┤ Telemetry Integ │
│                 │    │                  │    │                 │
│ • HTTP Metrics  │    │ • cgroup→pod     │    │ • Channel Proc  │
│ • Memory Track  │    │ • PID→container  │    │ • ClickHouse    │
│ • CPU Monitor   │    │ • K8s Metadata   │    │ • Real-time     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Key Benefits for RL Engine

#### **Rich Training Data**

- **Sub-second granularity** application metrics
- **Container-attributed** data with full Kubernetes context
- **Causal relationships** between scaling actions and performance
- **Multi-dimensional** metrics (latency, memory, CPU, network)

#### **Advanced Scaling Triggers**

- **Application-aware** scaling beyond basic CPU/memory
- **Predictive signals** like memory pressure and allocation patterns
- **Service communication** patterns for microservice optimization
- **Real-time anomaly detection**

#### **Production Benefits**

- **Minimal overhead** with efficient eBPF programs
- **Zero-configuration** automatic container discovery
- **Language-agnostic** support (Go, Java, Python, C/C++)
- **Kubernetes-native** integration

## Protocol Buffer Schema

### EBPFMetrics Message

```protobuf
message EBPFMetrics {
  // Container context
  string container_id = 1;
  string pod_name = 2;
  string namespace = 4;
  string app_name = 6;

  // Metric categories
  HTTPMetrics http = 10;
  MemoryPatterns memory_patterns = 11;
  CPUPatterns cpu_patterns = 12;
  NetworkFlow network_flow = 13;

  google.protobuf.Timestamp timestamp = 16;
}
```

## Troubleshooting

### Common Issues

#### **1. Missing eBPF Objects**

```
undefined: http_metricsObjects
```

**Solution**: Run `go generate` in each eBPF program directory

#### **2. Clang/LLVM Not Found**

```
clang: command not found
```

**Solution**: Install LLVM/Clang development tools:

```bash
# Ubuntu/Debian
sudo apt-get install clang llvm-dev libbpf-dev

# CentOS/RHEL
sudo yum install clang llvm-devel libbpf-devel
```

#### **3. Kernel Headers Missing**

```
vmlinux.h: No such file or directory
```

**Solution**: The `vmlinux.h` should be in `bpf/headers/` directory. If missing:

```bash
# Generate from running kernel
bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h
```

#### **4. Permission Denied**

```
failed to load eBPF program: operation not permitted
```

**Solution**: Ensure container has appropriate capabilities:

```yaml
securityContext:
  capabilities:
    add:
      - SYS_ADMIN
      - SYS_RESOURCE
  privileged: true
```

## Integration with Engine

### Sample Aggregation for RL

```go
// Aggregate metrics for RL consumption
aggregator := NewMetricsAggregator(60 * time.Second)

for metrics := range telemetryIntegration.GetHTTPMetricsChannel() {
    aggregator.ProcessEBPFMetrics(metrics)

    // Get features for RL model
    features := aggregator.GetAggregatedMetrics(appKey)

    // features["http_requests_per_second"]
    // features["http_error_rate"]
    // features["http_avg_latency_ms"]
    // features["memory_allocs_per_second"]
    // features["memory_net_allocated_mb"]
}
```

### ClickHouse Schema Example

```sql
CREATE TABLE ebpf_metrics (
    timestamp DateTime64(3),
    cluster_id String,
    namespace String,
    app_name String,
    container_id String,

    -- HTTP metrics
    http_requests_per_sec Float64,
    http_error_rate Float64,
    http_p95_latency_ms Float64,

    -- Memory metrics
    memory_allocs_per_sec Float64,
    memory_net_allocated_mb Float64,
    memory_page_faults_per_sec Float64,

    -- CPU metrics
    cpu_context_switches_per_sec Float64,
    cpu_utilization Float64

) ENGINE = MergeTree()
ORDER BY (timestamp, cluster_id, namespace, app_name);
```

## Future Enhancements

### Phase 2 Features

- **Network flow tracking** for service-to-service communication
- **Advanced language support** (Java JVM, Python, Node.js specific metrics)
- **Custom business metrics** via dynamic uprobe attachment
- **Security monitoring** with syscall anomaly detection

### Performance Optimizations

- **Adaptive sampling** based on cluster size
- **Map size optimization** with LRU eviction
- **Batch processing** for high-throughput environments
- **Custom aggregation** windows per application type

This implementation provides the foundation for extremely sophisticated, data-driven scaling decisions that go far beyond traditional CPU/memory-based approaches.
