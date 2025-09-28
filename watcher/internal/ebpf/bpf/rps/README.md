# RPS (Requests Per Second) Program

## Overview
The RPS eBPF program monitors network connection establishment to calculate requests per second metrics at the container level. It tracks TCP connections transitioning to the ESTABLISHED state and attributes them to specific containers using cgroup IDs.

## Program Details

### File Structure
- **rps.bpf.c**: eBPF C source code for kernel-space TCP state monitoring
- **rps.go**: Go wrapper providing user-space management and metric collection
- **rps_bpf.go**: Auto-generated Go bindings (created by bpf2go)
- **rps_bpf.o**: Compiled eBPF bytecode object

### Attach Points
- **SEC("tracepoint/sock/inet_sock_set_state")**: Kernel tracepoint for TCP socket state changes
- **Target**: All TCP socket state transitions in the system

### Data Structures

#### Maps
```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);   // cgroup_id
    __type(value, __u64); // request count
    __uint(max_entries, 16384);
} rps;
```

## Functionality

### Core Logic
1. **State Monitoring**: Tracks TCP socket state transitions
2. **Connection Detection**: Identifies new connections (SYN_SENT/SYN_RECV → ESTABLISHED)
3. **Container Attribution**: Maps connections to containers via cgroup IDs
4. **Request Counting**: Accumulates connection counts per container

### TCP State Transitions Monitored
- **SYN_SENT → ESTABLISHED**: Client-side connection establishment
- **SYN_RECV → ESTABLISHED**: Server-side connection acceptance
- **Filtering**: Only counts transitions that represent new connections

### Input Data
- **TCP State Events**: Kernel tracepoint data for socket state changes
- **Cgroup Context**: Container identification via `bpf_get_current_cgroup_id()`
- **Socket Information**: Old and new TCP states

### Output Metrics
- **Connection Count**: Number of established connections per container
- **RPS Calculation**: Connections per second by container
- **Reset Behavior**: Counters reset after each collection cycle

## Implementation Details

### eBPF Program (rps.bpf.c)
```c
SEC("tracepoint/sock/inet_sock_set_state")
int trace_sock_state(struct trace_event_raw_inet_sock_set_state *ctx) {
    int oldstate = ctx->oldstate;
    int newstate = ctx->newstate;

    // Detect new connection establishment
    if (newstate == TCP_ESTABLISHED &&
        (oldstate == TCP_SYN_SENT || oldstate == TCP_SYN_RECV)) {
        // Atomic increment per cgroup
    }
}
```

### Go Wrapper (rps.go)
- **Polling Model**: 1-second intervals for metric collection
- **Map Iteration**: Reads all cgroup counters
- **Counter Reset**: Clears counters after reading for rate calculation
- **Container Context**: Provides per-container request rates

## Usage Patterns

### Network Monitoring
1. **Inbound Traffic**: Server applications accepting connections
2. **Outbound Traffic**: Client applications establishing connections
3. **Load Balancing**: Distribution of requests across container instances
4. **Service Mesh**: Inter-service communication patterns

### Data Flow
```
TCP Socket → State Change → eBPF Tracepoint → Cgroup Mapping → Hash Map → Go Collector → RPS Metrics
```

## Performance Characteristics

### Overhead
- **Kernel Level**: ~50ns per TCP state transition
- **Memory Efficient**: 128KB maximum map size (16384 × 8 bytes)
- **Collection Overhead**: Map iteration every second

### Scalability
- **Container Limit**: Supports up to 16,384 concurrent containers
- **Reset Pattern**: Prevents counter overflow
- **Atomic Operations**: Thread-safe increments

## Container Integration

### Cgroup Attribution
- **Automatic Detection**: Uses `bpf_get_current_cgroup_id()` for container mapping
- **Kubernetes Compatible**: Works with pod-level networking
- **Multi-tenant**: Isolates metrics by container boundaries

### Network Context
- **Service Discovery**: Maps network activity to specific services
- **Load Patterns**: Identifies high-traffic containers
- **Scaling Decisions**: Provides data for horizontal pod autoscaling

## Security Considerations

### Monitoring Scope
- **System-wide**: Observes all TCP connections
- **Read-only**: Does not modify network behavior
- **Privilege Requirements**: Needs CAP_BPF and CAP_SYS_ADMIN

### Data Privacy
- **Metadata Only**: Tracks connection counts, not content
- **Container Isolation**: Separate metrics per container
- **Rate Limiting**: Map size prevents memory exhaustion

## Integration Points

### Telemetry Pipeline
- **Collector Interface**: Implements `CollectorProgram` pattern
- **Real-time**: 1-second granularity for rapid response
- **Reset Model**: Provides rate-based metrics

### Kubernetes Integration
- **Pod Monitoring**: Container-level network activity
- **Service Metrics**: Aggregation by service labels
- **HPA Input**: Provides request rate for autoscaling

## Troubleshooting

### Common Issues
1. **High CPU Usage**: Too many TCP state transitions
2. **Map Overflow**: Exceeding 16,384 container limit
3. **Missing Data**: Containers not generating TCP traffic

### Debug Information
- **Console Output**: Prints cgroup ID and RPS every second
- **Map Inspection**: Direct access to current counters
- **Error Handling**: Graceful iteration error recovery

## Metrics Interpretation

### RPS Calculation
- **Formula**: Connections established per second
- **Granularity**: 1-second measurement windows
- **Accuracy**: Depends on TCP handshake timing

### Use Cases
- **HTTP Services**: Each request typically creates one TCP connection
- **Persistent Connections**: Lower RPS for keep-alive scenarios
- **Connection Pooling**: Multiple requests per connection

## Future Enhancements

### Protocol Support
- **HTTP/2**: Multiple streams per connection
- **gRPC**: Bidirectional streaming patterns
- **WebSocket**: Long-lived connection handling

### Advanced Metrics
- **Connection Duration**: Track connection lifecycle
- **Bandwidth Correlation**: Combine with data transfer metrics
- **Error Rates**: Monitor failed connection attempts

### Configuration Options
- **Sampling Rate**: Configurable collection intervals
- **Container Filtering**: Selective monitoring by namespace
- **Threshold Alerts**: Automatic anomaly detection

## Dependencies
- **cilium/ebpf**: Go eBPF library for object management
- **vmlinux.h**: Kernel type definitions for TCP states
- **bpf_helpers.h**: eBPF helper function access
- **bpf_tracing.h**: Tracepoint attachment support

## TCP States Reference
- **TCP_SYN_SENT**: Client initiated connection
- **TCP_SYN_RECV**: Server received connection request
- **TCP_ESTABLISHED**: Connection fully established
- **State Filtering**: Only tracks meaningful transitions