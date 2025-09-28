# Uprobe Program

## Overview
The uprobe eBPF program provides user-space function tracing capabilities by attaching to specific function entry points in user applications. This program specifically tracks HTTP request handler invocations to measure requests per second (RPS) metrics.

## Program Details

### File Structure
- **uprobe.c**: eBPF C source code for kernel-space uprobe attachment
- **uprobe.go**: Go wrapper providing user-space management and data collection
- **uprobe_bpf.go**: Auto-generated Go bindings (created by bpf2go)
- **uprobe_bpf.o**: Compiled eBPF bytecode object

### Attach Points
- **SEC("uprobe/handle_request")**: Attaches to the `handle_request` function in target user-space applications
- **Target**: HTTP request handlers in containerized applications

### Data Structures

#### Maps
```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, u32);   // Process ID (PID)
    __type(value, u64); // Request count
} rps_count;
```

## Functionality

### Core Logic
1. **Function Interception**: Intercepts calls to `handle_request` function
2. **PID Tracking**: Uses process ID as the primary key for request counting
3. **Atomic Counting**: Safely increments request counters using atomic operations
4. **Per-Process Metrics**: Maintains separate counters for each process

### Input Data
- **Process Context**: Current process ID from `bpf_get_current_pid_tgid()`
- **Function Entry**: Triggered when target function is called

### Output Metrics
- **Request Count**: Total number of function calls per process
- **RPS Data**: Requests per second calculated over time intervals

## Implementation Details

### eBPF Program (uprobe.c)
```c
SEC("uprobe/handle_request")
int handle_request(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    // Atomic increment of request counter for this PID
    // Thread-safe map operations
}
```

### Go Wrapper (uprobe.go)
- **Lifecycle Management**: Loads eBPF objects and manages resources
- **Polling Interface**: Implements `CollectorProgram` interface
- **Data Collection**: Reads map data every second
- **Memory Management**: Removes memory limits for eBPF operations

## Usage Patterns

### Deployment
1. **Target Identification**: Identify applications with `handle_request` functions
2. **Attachment**: Uprobe automatically attaches to matching function symbols
3. **Monitoring**: Continuous collection of function call metrics

### Data Flow
```
User Application → handle_request() → eBPF Uprobe → Hash Map → Go Collector → Metrics
```

## Performance Characteristics

### Overhead
- **Minimal Impact**: Uprobe adds ~100ns overhead per function call
- **Memory Efficient**: Hash map limited to 1024 entries
- **CPU Overhead**: Atomic operations ensure thread safety

### Scalability
- **Process Limit**: Supports up to 1024 concurrent processes
- **Update Frequency**: 1-second polling interval
- **Memory Usage**: ~8KB for map storage (1024 × 8 bytes)

## Security Considerations

### Privileges
- **CAP_BPF**: Requires BPF capability for loading programs
- **CAP_SYS_ADMIN**: Needed for uprobe attachment
- **Container Compatibility**: Works within privileged containers

### Data Safety
- **Read-Only Target**: Does not modify target application
- **Atomic Operations**: Prevents race conditions
- **Bounded Memory**: Map size prevents memory exhaustion

## Integration Points

### Container Context
- **Process Mapping**: PID-based tracking allows container attribution
- **Multi-tenant**: Isolates metrics by process boundaries
- **Kubernetes**: Compatible with pod-level monitoring

### Telemetry Pipeline
- **Collector Interface**: Implements `CollectorProgram` for unified management
- **Polling Model**: Regular data collection via `Poll()` method
- **Resource Cleanup**: Proper cleanup via `Close()` method

## Troubleshooting

### Common Issues
1. **Symbol Not Found**: Target application lacks `handle_request` function
2. **Permission Denied**: Insufficient privileges for uprobe attachment
3. **Map Full**: Exceeding 1024 process limit

### Debug Information
- **Log Output**: Request counts logged every second
- **Map Inspection**: Direct map access for debugging
- **Error Handling**: Graceful degradation on lookup failures

## Future Enhancements

### Potential Improvements
- **Dynamic Symbol Resolution**: Runtime discovery of target functions
- **Multi-Function Support**: Attach to multiple function types
- **Latency Metrics**: Measure function execution time
- **Stack Traces**: Capture call stack information

### Configuration Options
- **Configurable Symbols**: User-defined target functions
- **Sampling Rate**: Adjustable collection frequency
- **Map Size**: Dynamic sizing based on deployment scale

## Dependencies
- **cilium/ebpf**: Go eBPF library for object loading
- **vmlinux.h**: Kernel type definitions
- **bpf_helpers.h**: eBPF helper function definitions
- **bpf_core_read.h**: CO-RE (Compile Once, Run Everywhere) support