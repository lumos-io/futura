# CPU Tracker Program

## Overview
The CPU Tracker eBPF program provides comprehensive CPU usage monitoring and performance analysis at the container level. It tracks CPU utilization, context switches, scheduler latency, thread states, and CPU hotspots by intercepting scheduler events and performance monitoring counters.

## Program Details

### File Structure
- **cpu_tracker.bpf.c**: eBPF C source code for kernel-space CPU monitoring
- **cpu_tracker.go**: Go wrapper providing user-space management and metric aggregation
- **cpu_tracker_bpf.go**: Auto-generated Go bindings (created by bpf2go)
- **cpu_tracker_bpf.o**: Compiled eBPF bytecode object

### Attach Points
- **Tracepoints**: `sched:sched_switch`, `sched:sched_wakeup`, `sched:sched_migrate_task`
- **Kprobes**: `finish_task_switch`, `schedule`, `try_to_wake_up`
- **PMU Events**: CPU performance counters (cycles, instructions, cache misses)
- **Software Events**: Context switches, page faults, CPU migrations

### Data Structures

#### CPU Metrics Aggregation
```c
struct cpu_metrics {
    __u64 user_time_ns;
    __u64 system_time_ns;
    __u64 context_switches;
    __u64 voluntary_switches;
    __u64 involuntary_switches;
    __u64 runqueue_latency_sum;
    __u64 runqueue_samples;
    __u32 active_threads;
    __u32 blocked_threads;
    __u64 last_update;
};
```

#### CPU Hotspot Analysis
```c
struct cpu_hotspot {
    __u64 samples;
    char function_name[64];
    __u32 stack_id;
};
```

#### Process Context Tracking
```c
struct process_context {
    __u32 pid;
    __u64 last_switch_time;
    __u64 accumulated_runtime;
    __u32 cpu_affinity;
    __u8 current_state;
};
```

## Maps and Storage

### CPU Metrics Aggregation
- **cpu_metrics_map**: Per-container CPU statistics
- **Key**: cgroup_id
- **Value**: cpu_metrics with timing and switching data
- **Size**: 16,384 entries
- **Update**: Real-time atomic operations during scheduler events

### Process Switch Tracking
- **process_switch_times**: Last context switch time per process
- **Key**: Process ID (PID)
- **Value**: Timestamp of last context switch
- **Purpose**: Calculate CPU runtime and context switch frequency

### Performance Profiling
- **stack_traces**: Call stack information for CPU hotspots
- **Type**: BPF_MAP_TYPE_STACK_TRACE
- **Depth**: 16 stack frames maximum
- **Usage**: Identify CPU-intensive code paths

### Thread State Monitoring
- **thread_states**: Current state of all threads
- **Key**: Thread ID
- **Value**: Current thread state (running, blocked, waiting)
- **Update**: Real-time state transitions

## Functionality

### CPU Usage Calculation
1. **Context Switch Interception**: Monitor scheduler `sched_switch` events
2. **Runtime Measurement**: Calculate time slices between context switches
3. **User vs System Time**: Differentiate between user and kernel CPU usage
4. **Container Attribution**: Map CPU usage to containers via cgroups

### Scheduler Analysis
- **Voluntary Switches**: Process yields CPU voluntarily (I/O wait, sleep)
- **Involuntary Switches**: Process preempted by scheduler (time slice expired)
- **Runqueue Latency**: Time spent waiting in scheduler runqueue
- **CPU Affinity**: Track process migration between CPU cores

### Performance Profiling
- **Sampling**: Periodic stack trace collection during CPU usage
- **Hotspot Identification**: Functions consuming most CPU time
- **Call Graph Analysis**: CPU usage distribution across function calls
- **Bottleneck Detection**: Identify performance-critical code paths

## Implementation Details

### Scheduler Switch Monitoring
```c
SEC("tracepoint/sched/sched_switch")
int trace_sched_switch(struct trace_event_raw_sched_switch *ctx) {
    __u32 prev_pid = ctx->prev_pid;
    __u32 next_pid = ctx->next_pid;
    __u64 current_time = bpf_ktime_get_ns();

    // Calculate runtime for previous process
    __u64 runtime = current_time - last_switch_time;

    // Update metrics based on switch type
    if (ctx->prev_state == 0) { // RUNNING state
        increment_involuntary_switches(prev_cgroup);
    } else {
        increment_voluntary_switches(prev_cgroup);
    }
}
```

### CPU Time Attribution
```c
// Differentiate user vs system time based on context
if (is_kernel_context(ctx)) {
    __sync_fetch_and_add(&metrics->system_time_ns, runtime);
} else {
    __sync_fetch_and_add(&metrics->user_time_ns, runtime);
}
```

### Runqueue Latency Measurement
```c
SEC("tracepoint/sched/sched_wakeup")
int trace_sched_wakeup(struct trace_event_raw_sched_wakeup *ctx) {
    __u64 wakeup_time = bpf_ktime_get_ns();
    record_wakeup_time(ctx->pid, wakeup_time);
}

// In sched_switch, calculate latency
__u64 latency = switch_time - wakeup_time;
update_runqueue_latency(cgroup_id, latency);
```

## Performance Characteristics

### Overhead Analysis
- **Context Switch Overhead**: ~100ns per context switch event
- **Memory Usage**: ~2MB for all maps combined
- **CPU Impact**: <0.5% additional CPU load
- **Accuracy**: Nanosecond precision for timing measurements

### Scalability Metrics
- **Process Tracking**: 16,384 concurrent processes
- **Containers**: 16,384 containers maximum
- **Stack Traces**: 1,024 unique call stacks
- **Event Rate**: ~1M context switches/second capacity

## Container Integration

### Cgroup Attribution
- **Automatic Mapping**: Uses `bpf_get_current_cgroup_id()` for container identification
- **Hierarchical Metrics**: Supports nested cgroup hierarchies
- **Resource Isolation**: Separate CPU metrics per container
- **Kubernetes Integration**: Pod-level and namespace-level aggregation

### CPU Resource Management
- **CPU Quotas**: Monitor compliance with CPU limits
- **Throttling Detection**: Identify when containers are CPU-throttled
- **Fair Share Analysis**: Verify CPU resource distribution
- **Performance Impact**: Measure impact of resource constraints

## Data Output and Integration

### Protobuf Schema
```protobuf
message CPUPatterns {
    ContainerInfo container = 1;
    CPUUsageMetrics usage = 2;
    ContextSwitchMetrics context_switches = 3;
    SchedulerMetrics scheduler = 4;
    PerformanceMetrics performance = 5;
    ThreadMetrics threads = 6;
    google.protobuf.Timestamp timestamp = 7;
}
```

### Real-time Metrics
- **CPU Utilization**: Real-time CPU usage percentage
- **Context Switch Rate**: Switches per second
- **Scheduler Latency**: Average and P95 runqueue latency
- **Thread States**: Active, blocked, and waiting thread counts

### Performance Profiling Data
- **CPU Hotspots**: Top CPU-consuming functions
- **Call Graphs**: Function call hierarchy with CPU attribution
- **Performance Bottlenecks**: Slowest code paths identified
- **Optimization Opportunities**: High-impact optimization targets

## Use Cases

### Application Performance Monitoring
- **CPU Profiling**: Identify CPU-intensive application components
- **Performance Regression**: Detect CPU performance degradation
- **Optimization Guidance**: Pinpoint code requiring optimization
- **Scalability Analysis**: CPU scaling characteristics under load

### Container Resource Management
- **Resource Planning**: Right-size CPU allocations
- **Performance Isolation**: Verify container CPU isolation
- **Efficiency Monitoring**: CPU utilization optimization
- **Cost Management**: Optimize CPU resource costs

### DevOps and SRE
- **Performance Troubleshooting**: Root cause analysis for CPU issues
- **Capacity Planning**: Predict CPU resource requirements
- **SLA Monitoring**: CPU performance SLA compliance
- **Auto-scaling**: CPU-based horizontal pod autoscaling

## Scheduler Analysis

### Context Switch Patterns
- **High Switch Rate**: Possible CPU contention or inefficient threading
- **Low Switch Rate**: CPU-bound workloads or efficient scheduling
- **Voluntary vs Involuntary**: Application behavior and scheduling efficiency
- **Migration Frequency**: CPU affinity and NUMA considerations

### Thread State Analysis
- **Runnable Threads**: Threads ready to execute
- **Blocked Threads**: Threads waiting for resources
- **I/O Wait**: Threads blocked on I/O operations
- **Sleeping Threads**: Threads in timed wait states

### Scheduler Latency
- **Runqueue Depth**: Number of processes waiting to run
- **Priority Inversion**: High-priority tasks delayed by low-priority tasks
- **CPU Affinity**: Impact of process migration on performance
- **Load Balancing**: Effectiveness of scheduler load distribution

## Troubleshooting

### Common Issues
1. **High Context Switch Rate**: Inefficient threading or excessive parallelism
2. **High Scheduler Latency**: CPU oversubscription or priority issues
3. **CPU Hotspots**: Inefficient algorithms or tight loops
4. **Thread Pool Exhaustion**: Insufficient thread resources

### Debug Features
- **Stack Trace Analysis**: Identify CPU-intensive code paths
- **Timeline Analysis**: CPU usage patterns over time
- **Process Correlation**: Map CPU usage to specific processes
- **Container Attribution**: Verify cgroup-based resource attribution

### Performance Tuning
- **CPU Affinity**: Optimize process-to-CPU mapping
- **Scheduling Policy**: Adjust process priorities and scheduling classes
- **Thread Pool Sizing**: Optimize application thread pool configurations
- **Algorithm Optimization**: Replace CPU-intensive algorithms

## Security Considerations

### Privilege Requirements
- **CAP_BPF**: Required for eBPF program loading
- **CAP_SYS_ADMIN**: Needed for scheduler tracepoint access
- **CAP_PERFMON**: Required for performance counter access

### Data Privacy
- **Function Names**: Optional anonymization of function symbols
- **Process Information**: Limited to CPU timing, no sensitive data
- **Container Isolation**: Strict separation of metrics between containers

### Security Monitoring
- **CPU Exhaustion**: Detection of CPU denial-of-service attacks
- **Cryptomining**: Unusual CPU usage patterns indicating cryptocurrency mining
- **Process Behavior**: Abnormal CPU consumption patterns
- **Resource Abuse**: Detection of resource limit violations

## Advanced Features

### CPU Frequency Scaling
- **Dynamic Frequency**: Monitor CPU frequency changes
- **Power Management**: Track power-saving mode impacts
- **Performance States**: P-state and C-state monitoring
- **Thermal Throttling**: CPU throttling due to temperature

### NUMA Awareness
- **NUMA Node Affinity**: Track process NUMA node placement
- **Memory Locality**: CPU and memory locality correlation
- **Cross-node Traffic**: Inter-NUMA node communication costs
- **Optimization Opportunities**: NUMA-aware placement recommendations

### Real-time Analysis
- **Streaming Metrics**: Real-time CPU metric streaming
- **Threshold Alerts**: Automatic alerts for CPU anomalies
- **Predictive Analysis**: CPU usage trend prediction
- **Anomaly Detection**: Machine learning-based anomaly identification

## Future Enhancements

### Extended Metrics
- **CPU Cache Performance**: L1/L2/L3 cache miss rates
- **Branch Prediction**: Branch misprediction rates
- **Pipeline Stalls**: CPU pipeline efficiency metrics
- **Vector Instructions**: SIMD instruction usage analysis

### Machine Learning Integration
- **Performance Prediction**: Predict CPU performance issues
- **Workload Classification**: Automatic workload type identification
- **Optimization Recommendations**: AI-driven performance tuning
- **Capacity Forecasting**: Predictive capacity planning

### Integration Improvements
- **APM Integration**: Application performance monitoring correlation
- **Distributed Tracing**: CPU metrics in distributed traces
- **CI/CD Integration**: Performance regression detection in pipelines
- **Cloud Provider APIs**: Integration with cloud CPU metrics

## Dependencies
- **cilium/ebpf**: Go eBPF library for program management
- **vmlinux.h**: Kernel scheduler data structures
- **bpf_tracing.h**: Tracepoint and kprobe support
- **bpf_core_read.h**: Portable kernel data access
- **perf_event**: Linux perf event subsystem
- **cgroup**: Container runtime cgroup integration