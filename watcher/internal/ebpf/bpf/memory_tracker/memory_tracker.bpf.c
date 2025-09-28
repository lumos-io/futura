//go:build ignore
// +build ignore
// file: memory_tracker.bpf.c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#define MAX_ENTRIES 16384
#define MAX_STACK_DEPTH 16

// Memory allocation tracking structure
struct alloc_info {
    __u64 size;
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 stack_id;
};

// Memory metrics aggregation per container
struct memory_metrics {
    __u64 alloc_count;
    __u64 free_count;
    __u64 bytes_allocated;
    __u64 bytes_freed;
    __u64 net_allocated;
    __u64 small_allocs;     // < 1KB
    __u64 medium_allocs;    // 1KB - 64KB
    __u64 large_allocs;     // 64KB - 1MB
    __u64 huge_allocs;      // > 1MB
    __u64 page_faults;
    __u64 major_page_faults;
    __u64 last_update;
};

// Memory leak tracking
struct leak_candidate {
    __u64 address;
    __u64 size;
    __u64 alloc_time;
    __u32 stack_id;
    __u32 pid;
};

// GC metrics for managed languages
struct gc_metrics {
    __u64 gc_count;
    __u64 gc_time_ns;
    __u64 bytes_collected;
    __u64 last_gc_time;
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // address
    __type(value, struct alloc_info);
    __uint(max_entries, MAX_ENTRIES);
} active_allocs SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct memory_metrics);
    __uint(max_entries, MAX_ENTRIES);
} memory_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct leak_candidate[100]);  // Top 100 leak candidates
    __uint(max_entries, MAX_ENTRIES);
} leak_candidates SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct gc_metrics);
    __uint(max_entries, MAX_ENTRIES);
} gc_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_STACK_TRACE);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, MAX_STACK_DEPTH * sizeof(__u64));
    __uint(max_entries, 1024);
} stack_traces SEC(".maps");

// Ring buffer for memory events
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} memory_events SEC(".maps");

// Memory event structure
struct memory_event {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 event_type; // 0=alloc, 1=free, 2=leak_detected
    __u64 address;
    __u64 size;
    __u32 stack_id;
};

// Helper functions
static __always_inline void update_size_histogram(struct memory_metrics *metrics, __u64 size) {
    if (size < 1024) {
        __sync_fetch_and_add(&metrics->small_allocs, 1);
    } else if (size < 65536) {
        __sync_fetch_and_add(&metrics->medium_allocs, 1);
    } else if (size < 1048576) {
        __sync_fetch_and_add(&metrics->large_allocs, 1);
    } else {
        __sync_fetch_and_add(&metrics->huge_allocs, 1);
    }
}

static __always_inline void check_for_leaks(__u64 cgroup_id) {
    // Simple leak detection: allocations older than 5 minutes
    __u64 current_time = bpf_ktime_get_ns();
    __u64 leak_threshold = 5 * 60 * 1000000000ULL; // 5 minutes in nanoseconds

    // Iterate through active allocations (simplified, in real implementation
    // this would be more sophisticated)
    // For now, we'll just emit a warning event if we have too many allocations
}

// Kernel memory allocation tracking
SEC("tracepoint/kmem/kmalloc")
int trace_kmalloc(struct trace_event_raw_kmalloc *ctx) {
    __u64 address = (__u64)(uintptr_t)ctx->ptr;
    __u64 size = ctx->bytes_alloc;
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    if (address == 0) {
        return 0;
    }

    // Get stack trace
    __u32 stack_id = bpf_get_stackid(ctx, &stack_traces, BPF_F_USER_STACK);

    // Store allocation info
    struct alloc_info info = {
        .size = size,
        .timestamp = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
        .stack_id = stack_id,
    };
    bpf_map_update_elem(&active_allocs, &address, &info, BPF_ANY);

    // Update metrics
    struct memory_metrics *metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    if (!metrics) {
        struct memory_metrics init_metrics = {};
        bpf_map_update_elem(&memory_metrics_map, &cgroup_id, &init_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    }

    if (metrics) {
        __sync_fetch_and_add(&metrics->alloc_count, 1);
        __sync_fetch_and_add(&metrics->bytes_allocated, size);
        __sync_fetch_and_add(&metrics->net_allocated, size);
        update_size_histogram(metrics, size);
        metrics->last_update = bpf_ktime_get_ns();
    }

    // Emit allocation event
    struct memory_event *event = bpf_ringbuf_reserve(&memory_events, sizeof(struct memory_event), 0);
    if (event) {
        event->timestamp = bpf_ktime_get_ns();
        event->cgroup_id = cgroup_id;
        event->pid = pid;
        event->event_type = 0; // allocation
        event->address = address;
        event->size = size;
        event->stack_id = stack_id;
        bpf_ringbuf_submit(event, 0);
    }

    return 0;
}

SEC("tracepoint/kmem/kfree")
int trace_kfree(struct trace_event_raw_kfree *ctx) {
    __u64 address = (__u64)(uintptr_t)ctx->ptr;
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    if (address == 0) {
        return 0;
    }

    // Look up the allocation
    struct alloc_info *info = bpf_map_lookup_elem(&active_allocs, &address);
    if (!info) {
        // Free without corresponding alloc - might be from before we started monitoring
        return 0;
    }

    __u64 size = info->size;

    // Update metrics
    struct memory_metrics *metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    if (metrics) {
        __sync_fetch_and_add(&metrics->free_count, 1);
        __sync_fetch_and_add(&metrics->bytes_freed, size);
        __sync_fetch_and_sub(&metrics->net_allocated, size);
        metrics->last_update = bpf_ktime_get_ns();
    }

    // Emit free event
    struct memory_event *event = bpf_ringbuf_reserve(&memory_events, sizeof(struct memory_event), 0);
    if (event) {
        event->timestamp = bpf_ktime_get_ns();
        event->cgroup_id = cgroup_id;
        event->pid = pid;
        event->event_type = 1; // free
        event->address = address;
        event->size = size;
        event->stack_id = info->stack_id;
        bpf_ringbuf_submit(event, 0);
    }

    // Remove from active allocations
    bpf_map_delete_elem(&active_allocs, &address);

    return 0;
}

// User-space memory allocation tracking (malloc/free)
SEC("uprobe/malloc")
int trace_malloc(struct pt_regs *ctx) {
    size_t size = 0;
    bpf_probe_read_user(&size, sizeof(size), (void*)ctx->di); // x86_64 first argument
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // We'll get the return address in the uretprobe
    // For now, just update allocation metrics
    struct memory_metrics *metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    if (!metrics) {
        struct memory_metrics init_metrics = {};
        bpf_map_update_elem(&memory_metrics_map, &cgroup_id, &init_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    }

    if (metrics) {
        __sync_fetch_and_add(&metrics->alloc_count, 1);
        __sync_fetch_and_add(&metrics->bytes_allocated, size);
        __sync_fetch_and_add(&metrics->net_allocated, size);
        update_size_histogram(metrics, size);
        metrics->last_update = bpf_ktime_get_ns();
    }

    return 0;
}

SEC("uretprobe/malloc")
int trace_malloc_ret(struct pt_regs *ctx) {
    __u64 address = ctx->ax; // x86_64 return value in rax
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    if (address == 0) {
        return 0; // malloc failed
    }

    // We can't easily get the size here, so we'll use a placeholder
    // In a real implementation, we'd store the size from the entry probe
    __u32 stack_id = bpf_get_stackid(ctx, &stack_traces, BPF_F_USER_STACK);

    struct alloc_info info = {
        .size = 0, // Size would need to be stored from entry probe
        .timestamp = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
        .stack_id = stack_id,
    };
    bpf_map_update_elem(&active_allocs, &address, &info, BPF_ANY);

    return 0;
}

SEC("uprobe/free")
int trace_free(struct pt_regs *ctx) {
    __u64 address = ctx->di; // x86_64 first argument in rdi
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    if (address == 0) {
        return 0;
    }

    struct alloc_info *info = bpf_map_lookup_elem(&active_allocs, &address);
    if (!info) {
        return 0;
    }

    // Update metrics
    struct memory_metrics *metrics = bpf_map_lookup_elem(&memory_metrics_map, &cgroup_id);
    if (metrics) {
        __sync_fetch_and_add(&metrics->free_count, 1);
        __sync_fetch_and_add(&metrics->bytes_freed, info->size);
        __sync_fetch_and_sub(&metrics->net_allocated, info->size);
        metrics->last_update = bpf_ktime_get_ns();
    }

    bpf_map_delete_elem(&active_allocs, &address);

    return 0;
}

// Note: Page fault tracking removed due to missing tracepoint in current kernel
// Alternative: Could use software events or kprobes for page fault tracking

// Go GC tracking (example for Go runtime)
SEC("uprobe/runtime.GC")
int trace_go_gc_start(struct pt_regs *ctx) {
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    struct gc_metrics *gc = bpf_map_lookup_elem(&gc_metrics_map, &cgroup_id);
    if (!gc) {
        struct gc_metrics init_gc = {};
        bpf_map_update_elem(&gc_metrics_map, &cgroup_id, &init_gc, BPF_ANY);
        gc = bpf_map_lookup_elem(&gc_metrics_map, &cgroup_id);
    }

    if (gc) {
        __sync_fetch_and_add(&gc->gc_count, 1);
        gc->last_gc_time = bpf_ktime_get_ns();
    }

    return 0;
}

// Periodic leak detection
SEC("tracepoint/timer/hrtimer_expire_entry")
int periodic_leak_check(void *ctx) {
    // This would run periodically to check for memory leaks
    // Implementation would iterate through active_allocs and identify
    // allocations that are older than a threshold
    return 0;
}

char LICENSE[] SEC("license") = "GPL";