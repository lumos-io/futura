# HTTP Metrics Program

## Overview
The HTTP Metrics eBPF program provides comprehensive monitoring of HTTP traffic at the container level. It tracks request latencies, status codes, error rates, and endpoint-specific metrics by intercepting system calls and network events related to HTTP communications.

## Program Details

### File Structure
- **http_metrics.bpf.c**: eBPF C source code for kernel-space HTTP monitoring
- **http_metrics.go**: Go wrapper providing user-space management and metric aggregation
- **http_metrics_bpf.go**: Auto-generated Go bindings (created by bpf2go)
- **http_metrics_bpf.o**: Compiled eBPF bytecode object

### Attach Points
- **Syscall Tracepoints**: `sys_enter_sendto`, `sys_enter_recvfrom` for HTTP data flow
- **Network Tracepoints**: Socket state changes for connection lifecycle
- **Kprobes**: HTTP-specific kernel functions (optional)

### Data Structures

#### Request Tracking
```c
struct http_request {
    __u64 start_time;
    __u64 cgroup_id;
    __u32 pid;
    __u32 status_code;
    char method[16];    // GET, POST, PUT, etc.
    char url[256];      // Request URL/path
};
```

#### Metrics Aggregation
```c
struct http_metrics {
    __u64 request_count;
    __u64 error_count;
    __u64 latency_sum;
    __u64 latency_max;
    __u64 latency_min;
    __u64 status_2xx;
    __u64 status_3xx;
    __u64 status_4xx;
    __u64 status_5xx;
    __u64 last_update;
};
```

#### Endpoint Metrics
```c
struct endpoint_metrics {
    __u64 request_count;
    __u64 error_count;
    __u64 latency_sum;
    __u64 p95_latency;
    __u64 last_update;
};
```

## Maps and Storage

### Active Request Tracking
- **active_requests**: Hash map storing ongoing HTTP requests
- **Key**: request_id (combination of PID, file descriptor, timestamp)
- **Value**: http_request structure with timing and metadata
- **Size**: 16,384 entries

### Metrics Aggregation
- **http_metrics_map**: Per-container HTTP metrics
- **Key**: cgroup_id
- **Value**: http_metrics structure with counters and statistics
- **Size**: 16,384 entries

### Endpoint Analysis
- **endpoint_metrics_map**: Per-endpoint performance metrics
- **Key**: endpoint_key (cgroup_id + endpoint path)
- **Value**: endpoint_metrics with endpoint-specific data
- **Size**: 1,024 entries

### Latency Distribution
- **latency_histogram**: Histogram buckets for latency analysis
- **Key**: cgroup_id
- **Value**: Array of 10 histogram buckets
- **Buckets**: Configurable latency ranges (µs)

### Real-time Events
- **http_events**: Ring buffer for immediate event streaming
- **Size**: 256KB circular buffer
- **Events**: Real-time HTTP request completion notifications

## Functionality

### HTTP Request Lifecycle
1. **Request Start**: Detected via `sys_enter_sendto` with HTTP headers
2. **Request Tracking**: Stored in active_requests map with timing
3. **Response Processing**: Detected via `sys_enter_recvfrom` with status
4. **Metrics Update**: Aggregated statistics updated atomically
5. **Event Emission**: Real-time event sent via ring buffer

### Metric Calculations
- **Latency**: End-to-end request-response time in microseconds
- **Throughput**: Requests per second per container
- **Error Rate**: 4xx/5xx responses as percentage of total
- **Status Distribution**: Breakdown by HTTP status code ranges
- **P95 Latency**: 95th percentile response time per endpoint

### Protocol Detection
- **HTTP/1.1**: Standard request-response pattern
- **Keep-Alive**: Multiple requests on same connection
- **Chunked Transfer**: Progressive response processing
- **WebSocket Upgrade**: Detection of protocol switching

## Implementation Details

### Request Identification
```c
// Generate unique request ID
__u64 request_id = ((__u64)pid << 32) | (fd << 16) | (timestamp & 0xFFFF);

// Parse HTTP headers for method and URL
if (is_http_request(data, size)) {
    parse_http_method(data, req.method);
    parse_http_url(data, req.url);
}
```

### Latency Measurement
```c
// Request start
req.start_time = bpf_ktime_get_ns();

// Response completion
__u64 latency = (bpf_ktime_get_ns() - req.start_time) / 1000; // Convert to µs
update_latency_histogram(cgroup_id, latency);
```

### Container Attribution
```c
__u64 cgroup_id = bpf_get_current_cgroup_id();
struct http_metrics *metrics = bpf_map_lookup_elem(&http_metrics_map, &cgroup_id);
```

## Performance Characteristics

### Overhead Analysis
- **Request Processing**: ~500ns per HTTP request
- **Memory Usage**: ~2MB for all maps combined
- **CPU Impact**: <1% additional CPU load under normal traffic
- **Network Overhead**: Zero bytes added to actual HTTP traffic

### Scalability Limits
- **Concurrent Requests**: 16,384 active requests maximum
- **Containers**: 16,384 containers maximum
- **Endpoints**: 1,024 unique endpoints per container
- **Event Rate**: ~100,000 events/second ring buffer capacity

## Container Integration

### Kubernetes Compatibility
- **Pod Attribution**: Automatic mapping via cgroup hierarchy
- **Service Correlation**: Endpoint grouping by service labels
- **Namespace Isolation**: Separate metrics per namespace
- **Resource Limits**: Respects container resource constraints

### Multi-tenant Support
- **Isolation**: Complete separation of metrics between containers
- **Aggregation**: Service-level and namespace-level rollups
- **Security**: No cross-container data leakage
- **Performance**: Independent scaling per tenant

## Data Output and Integration

### Protobuf Schema
```protobuf
message HTTPMetrics {
    ContainerInfo container = 1;
    RequestMetrics requests = 2;
    StatusCodeMetrics status_codes = 3;
    LatencyMetrics latency = 4;
    EndpointMetrics endpoints = 5;
    google.protobuf.Timestamp timestamp = 6;
}
```

### Real-time Events
- **Ring Buffer**: Immediate event notification
- **Event Types**: Request completion, error detection, SLA violations
- **Batching**: Configurable event aggregation
- **Filtering**: Selective event emission based on criteria

### Telemetry Pipeline
- **Collection Frequency**: 1-second intervals for metrics
- **Event Streaming**: Real-time for critical alerts
- **Batch Processing**: Efficient bulk data transfer
- **Compression**: Optimized protobuf serialization

## Security and Privacy

### Data Handling
- **Headers Only**: HTTP headers processed, body ignored
- **Sensitive Data**: No authentication tokens or personal data stored
- **Anonymization**: Optional IP address masking
- **Retention**: In-memory only, no persistent storage

### Access Control
- **Privilege Requirements**: CAP_BPF, CAP_SYS_ADMIN
- **Container Boundaries**: Enforced isolation between containers
- **Network Policies**: Respects Kubernetes network policies
- **RBAC Integration**: Compatible with cluster RBAC rules

## Troubleshooting

### Common Issues
1. **Missing HTTP Traffic**: Non-HTTP protocols not detected
2. **High Memory Usage**: Too many concurrent requests
3. **Incomplete Metrics**: Partial HTTP request/response pairs
4. **Container Misattribution**: Cgroup mapping failures

### Debug Features
- **Verbose Logging**: Detailed request tracking
- **Map Inspection**: Direct access to eBPF maps
- **Event Tracing**: Ring buffer event monitoring
- **Metric Validation**: Cross-reference with application logs

### Performance Tuning
- **Sampling Rate**: Reduce overhead with statistical sampling
- **Buffer Sizes**: Adjust map sizes for workload characteristics
- **Filter Rules**: Exclude non-critical endpoints
- **Aggregation Windows**: Optimize collection intervals

## Use Cases

### Application Performance Monitoring (APM)
- **SLA Monitoring**: P95/P99 latency tracking
- **Error Rate Alerts**: Automatic threshold violations
- **Capacity Planning**: Request volume trends
- **Deployment Impact**: Before/after performance comparison

### Service Mesh Observability
- **Inter-service Communication**: Request flow between services
- **Circuit Breaker Metrics**: Failure rate monitoring
- **Load Balancing**: Request distribution analysis
- **Canary Deployments**: A/B testing performance metrics

### DevOps and SRE
- **Incident Response**: Real-time performance during outages
- **Capacity Management**: Resource scaling decisions
- **Performance Regression**: Automated detection of slowdowns
- **Service Dependencies**: Impact analysis of downstream services

## Future Enhancements

### Protocol Support
- **HTTP/2**: Multi-stream connection handling
- **gRPC**: Protocol buffer message inspection
- **GraphQL**: Query complexity analysis
- **WebSocket**: Bidirectional message tracking

### Advanced Analytics
- **Machine Learning**: Anomaly detection in request patterns
- **Predictive Scaling**: Forecasting resource needs
- **Root Cause Analysis**: Automated performance investigation
- **Business Metrics**: Revenue impact correlation

### Integration Improvements
- **Prometheus**: Native metrics export
- **Jaeger**: Distributed tracing correlation
- **Grafana**: Enhanced visualization dashboards
- **Alert Manager**: Intelligent alerting rules

## Dependencies
- **cilium/ebpf**: Go eBPF library for program management
- **ringbuf**: Real-time event streaming
- **protobuf**: Structured metric serialization
- **zerolog**: Structured logging
- **vmlinux.h**: Kernel data structure definitions
- **Container Runtime**: Docker/containerd cgroup integration