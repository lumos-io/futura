//go:build ignore
// +build ignore
// file: cpu_tracker.bpf.c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#define MAX_ENTRIES 16384

// CPU metrics per container
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

// CPU hotspot tracking
struct cpu_hotspot {
    __u64 samples;
    char function_name[64];
    __u32 stack_id;
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct cpu_metrics);
    __uint(max_entries, MAX_ENTRIES);
} cpu_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);  // pid
    __type(value, __u64); // last_switch_time
    __uint(max_entries, MAX_ENTRIES);
} process_switch_times SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_STACK_TRACE);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, 16 * sizeof(__u64));
    __uint(max_entries, 1024);
} stack_traces SEC(".maps");

// Scheduler switch tracking
SEC("tracepoint/sched/sched_switch")
int trace_sched_switch(struct trace_event_raw_sched_switch *ctx) {
    __u32 prev_pid = ctx->prev_pid;
    __u32 next_pid = ctx->next_pid;
    __u64 current_time = bpf_ktime_get_ns();
    __u64 prev_cgroup = bpf_get_current_cgroup_id();

    // Track context switch for the previous process
    __u64 *last_time = bpf_map_lookup_elem(&process_switch_times, &prev_pid);
    if (last_time) {
        __u64 runtime = current_time - *last_time;

        // Update CPU metrics for previous process
        struct cpu_metrics *metrics = bpf_map_lookup_elem(&cpu_metrics_map, &prev_cgroup);
        if (!metrics) {
            struct cpu_metrics init_metrics = {};
            bpf_map_update_elem(&cpu_metrics_map, &prev_cgroup, &init_metrics, BPF_ANY);
            metrics = bpf_map_lookup_elem(&cpu_metrics_map, &prev_cgroup);
        }

        if (metrics) {
            __sync_fetch_and_add(&metrics->context_switches, 1);

            // Determine if this was voluntary or involuntary
            if (ctx->prev_state == 0) { // 0 = RUNNING state
                __sync_fetch_and_add(&metrics->involuntary_switches, 1);
            } else {
                __sync_fetch_and_add(&metrics->voluntary_switches, 1);
            }

            // Add runtime to appropriate category
            if (runtime > 0) {
                // Simple heuristic: system time vs user time
                // In real implementation, you'd need more sophisticated tracking
                __sync_fetch_and_add(&metrics->user_time_ns, runtime);
            }

            metrics->last_update = current_time;
        }
    }

    // Update time for next process
    bpf_map_update_elem(&process_switch_times, &next_pid, &current_time, BPF_ANY);

    return 0;
}

// Wakeup latency tracking
SEC("tracepoint/sched/sched_wakeup")
int trace_sched_wakeup(struct trace_event_raw_sched_wakeup_template *ctx) {
    __u32 pid = ctx->pid;
    __u64 current_time = bpf_ktime_get_ns();
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    // Track when process was woken up
    bpf_map_update_elem(&process_switch_times, &pid, &current_time, BPF_ANY);

    // Update thread count
    struct cpu_metrics *metrics = bpf_map_lookup_elem(&cpu_metrics_map, &cgroup_id);
    if (!metrics) {
        struct cpu_metrics init_metrics = {};
        bpf_map_update_elem(&cpu_metrics_map, &cgroup_id, &init_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&cpu_metrics_map, &cgroup_id);
    }

    if (metrics) {
        // Simple approximation - increment active threads on wakeup
        if (metrics->active_threads < 0xFFFFFFFF) {
            metrics->active_threads++;
        }
        metrics->last_update = current_time;
    }

    return 0;
}

// Process exit tracking
SEC("tracepoint/sched/sched_process_exit")
int trace_process_exit(struct trace_event_raw_sched_process_template *ctx) {
    __u32 pid = ctx->pid;
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    // Clean up process tracking
    bpf_map_delete_elem(&process_switch_times, &pid);

    // Update thread count
    struct cpu_metrics *metrics = bpf_map_lookup_elem(&cpu_metrics_map, &cgroup_id);
    if (metrics && metrics->active_threads > 0) {
        metrics->active_threads--;
        metrics->last_update = bpf_ktime_get_ns();
    }

    return 0;
}

// CPU profiling via perf events
SEC("perf_event")
int cpu_profile(struct bpf_perf_event_data *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    // Get stack trace
    __u32 stack_id = bpf_get_stackid(ctx, &stack_traces, BPF_F_USER_STACK);

    // Update CPU hotspot tracking
    // This is simplified - in reality you'd maintain a more sophisticated hotspot map
    struct cpu_metrics *metrics = bpf_map_lookup_elem(&cpu_metrics_map, &cgroup_id);
    if (!metrics) {
        struct cpu_metrics init_metrics = {};
        bpf_map_update_elem(&cpu_metrics_map, &cgroup_id, &init_metrics, BPF_ANY);
    }

    return 0;
}

char LICENSE[] SEC("license") = "GPL";