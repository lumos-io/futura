//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

// Database metrics per container
struct db_metrics {
    __u64 query_count;
    __u64 query_time_total;
    __u64 slow_queries;
    __u64 active_connections;
    __u64 connection_timeouts;
    __u64 transactions;
    __u64 rollbacks;
    __u64 transaction_time_total;
    __u64 last_update;
};

// Cache metrics per container
struct cache_metrics {
    __u64 cache_hits;
    __u64 cache_misses;
    __u64 cache_sets;
    __u64 cache_gets;
    __u64 cache_deletes;
    __u64 cache_evictions;
    __u64 get_latency_total;
    __u64 set_latency_total;
    __u64 cache_size_bytes;
    __u64 last_update;
};

// Go-specific metrics per container
struct go_metrics {
    __u64 goroutines;
    __u64 goroutine_stack_size;
    __u64 gc_cycles;
    __u64 gc_pause_time_us;
    __u64 heap_size;
    __u64 heap_alloc;
    __u64 heap_idle;
    __u64 channel_sends;
    __u64 channel_receives;
    __u64 blocked_channels;
    __u64 last_update;
};

// Custom metric event
struct custom_metric {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    char metric_name[64];
    __u32 metric_type; // 0=counter, 1=gauge, 2=histogram, 3=timer
    double value;
    char labels[256]; // JSON-encoded labels
};

// Function timing for uprobes
struct function_timing {
    __u64 start_time;
    __u64 cgroup_id;
    __u32 pid;
    char function_name[128];
};

// Application event for real-time processing
struct app_event {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 event_type; // 0=db_query, 1=cache_op, 2=gc_event, 3=custom_metric
    __u64 duration_us;
    __u64 value;
    char details[256];
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // cgroup_id
    __type(value, struct db_metrics);
} db_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // cgroup_id
    __type(value, struct cache_metrics);
} cache_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // cgroup_id
    __type(value, struct go_metrics);
} go_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // pid_func_hash
    __type(value, struct function_timing);
} function_timings_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} app_events SEC(".maps");

// Helper to get cgroup ID
static __always_inline __u64 get_cgroup_id(void) {
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    return BPF_CORE_READ(task, cgroups, dfl_cgrp, kn, id);
}

// Helper to hash function name
static __always_inline __u64 hash_function_name(const char *name) {
    __u64 hash = 5381;
    for (int i = 0; i < 128 && name[i] != '\0'; i++) {
        hash = ((hash << 5) + hash) + name[i];
    }
    return hash;
}

// Generic uprobe entry for database functions
SEC("uprobe/db_query_start")
int trace_db_query_start(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;
    __u64 cgroup_id = get_cgroup_id();

    struct function_timing timing = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
    };

    // Copy function name
    bpf_probe_read_kernel_str(timing.function_name, sizeof(timing.function_name), "db_query");

    __u64 key = pid_tgid;
    bpf_map_update_elem(&function_timings_map, &key, &timing, BPF_ANY);
    return 0;
}

// Generic uprobe exit for database functions
SEC("uretprobe/db_query_end")
int trace_db_query_end(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();

    struct function_timing *timing = bpf_map_lookup_elem(&function_timings_map, &pid_tgid);
    if (!timing) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 duration_ns = now - timing->start_time;
    __u64 duration_us = duration_ns / 1000;
    __u64 cgroup_id = timing->cgroup_id;

    // Update database metrics
    struct db_metrics *metrics = bpf_map_lookup_elem(&db_metrics_map, &cgroup_id);
    if (!metrics) {
        struct db_metrics new_metrics = {0};
        new_metrics.query_count = 1;
        new_metrics.query_time_total = duration_us;
        if (duration_us > 1000000) { // 1 second threshold for slow queries
            new_metrics.slow_queries = 1;
        }
        new_metrics.last_update = now;
        bpf_map_update_elem(&db_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->query_count, 1);
        __sync_fetch_and_add(&metrics->query_time_total, duration_us);
        if (duration_us > 1000000) {
            __sync_fetch_and_add(&metrics->slow_queries, 1);
        }
        metrics->last_update = now;
    }

    // Emit event
    struct app_event *event = bpf_ringbuf_reserve(&app_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = now;
        event->cgroup_id = cgroup_id;
        event->pid = timing->pid;
        event->event_type = 0; // db_query
        event->duration_us = duration_us;
        event->value = 0;
        bpf_probe_read_kernel_str(event->details, sizeof(event->details), "database_query_completed");
        bpf_ringbuf_submit(event, 0);
    }

    bpf_map_delete_elem(&function_timings_map, &pid_tgid);
    return 0;
}

// Cache operation tracking - Redis/Memcached GET
SEC("uprobe/cache_get_start")
int trace_cache_get_start(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;
    __u64 cgroup_id = get_cgroup_id();

    struct function_timing timing = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
    };

    bpf_probe_read_kernel_str(timing.function_name, sizeof(timing.function_name), "cache_get");

    __u64 key = pid_tgid;
    bpf_map_update_elem(&function_timings_map, &key, &timing, BPF_ANY);
    return 0;
}

// Cache operation tracking - Redis/Memcached GET return
SEC("uretprobe/cache_get_end")
int trace_cache_get_end(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ret_val = ctx->ax; // x86_64 return value register

    struct function_timing *timing = bpf_map_lookup_elem(&function_timings_map, &pid_tgid);
    if (!timing) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 duration_ns = now - timing->start_time;
    __u64 duration_us = duration_ns / 1000;
    __u64 cgroup_id = timing->cgroup_id;

    // Update cache metrics
    struct cache_metrics *metrics = bpf_map_lookup_elem(&cache_metrics_map, &cgroup_id);
    if (!metrics) {
        struct cache_metrics new_metrics = {0};
        new_metrics.cache_gets = 1;
        new_metrics.get_latency_total = duration_us;
        if (ret_val != 0) { // Assume non-zero means hit
            new_metrics.cache_hits = 1;
        } else {
            new_metrics.cache_misses = 1;
        }
        new_metrics.last_update = now;
        bpf_map_update_elem(&cache_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->cache_gets, 1);
        __sync_fetch_and_add(&metrics->get_latency_total, duration_us);
        if (ret_val != 0) {
            __sync_fetch_and_add(&metrics->cache_hits, 1);
        } else {
            __sync_fetch_and_add(&metrics->cache_misses, 1);
        }
        metrics->last_update = now;
    }

    // Emit event
    struct app_event *event = bpf_ringbuf_reserve(&app_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = now;
        event->cgroup_id = cgroup_id;
        event->pid = timing->pid;
        event->event_type = 1; // cache_op
        event->duration_us = duration_us;
        event->value = ret_val != 0 ? 1 : 0; // hit=1, miss=0
        bpf_probe_read_kernel_str(event->details, sizeof(event->details), "cache_get_completed");
        bpf_ringbuf_submit(event, 0);
    }

    bpf_map_delete_elem(&function_timings_map, &pid_tgid);
    return 0;
}

// Go runtime tracking - GC start
SEC("uprobe/runtime_gc_start")
int trace_go_gc_start(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;
    __u64 cgroup_id = get_cgroup_id();

    struct function_timing timing = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
    };

    bpf_probe_read_kernel_str(timing.function_name, sizeof(timing.function_name), "runtime_gc");

    __u64 key = pid_tgid;
    bpf_map_update_elem(&function_timings_map, &key, &timing, BPF_ANY);
    return 0;
}

// Go runtime tracking - GC end
SEC("uretprobe/runtime_gc_end")
int trace_go_gc_end(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();

    struct function_timing *timing = bpf_map_lookup_elem(&function_timings_map, &pid_tgid);
    if (!timing) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 duration_ns = now - timing->start_time;
    __u64 duration_us = duration_ns / 1000;
    __u64 cgroup_id = timing->cgroup_id;

    // Update Go metrics
    struct go_metrics *metrics = bpf_map_lookup_elem(&go_metrics_map, &cgroup_id);
    if (!metrics) {
        struct go_metrics new_metrics = {0};
        new_metrics.gc_cycles = 1;
        new_metrics.gc_pause_time_us = duration_us;
        new_metrics.last_update = now;
        bpf_map_update_elem(&go_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->gc_cycles, 1);
        __sync_fetch_and_add(&metrics->gc_pause_time_us, duration_us);
        metrics->last_update = now;
    }

    // Emit event
    struct app_event *event = bpf_ringbuf_reserve(&app_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = now;
        event->cgroup_id = cgroup_id;
        event->pid = timing->pid;
        event->event_type = 2; // gc_event
        event->duration_us = duration_us;
        event->value = 0;
        bpf_probe_read_kernel_str(event->details, sizeof(event->details), "go_gc_completed");
        bpf_ringbuf_submit(event, 0);
    }

    bpf_map_delete_elem(&function_timings_map, &pid_tgid);
    return 0;
}

// Custom metric emission helper
SEC("uprobe/custom_metric_emit")
int trace_custom_metric(struct pt_regs *ctx) {
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    struct app_event *event = bpf_ringbuf_reserve(&app_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = bpf_ktime_get_ns();
        event->cgroup_id = cgroup_id;
        event->pid = pid;
        event->event_type = 3; // custom_metric
        event->duration_us = 0;

        // Try to read metric value from function argument
        // This would need to be customized per application
        event->value = ctx->di; // x86_64 first argument register

        bpf_probe_read_kernel_str(event->details, sizeof(event->details), "custom_metric_emitted");
        bpf_ringbuf_submit(event, 0);
    }

    return 0;
}

char _license[] SEC("license") = "GPL";