//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

// File system metrics per container
struct fs_metrics {
    __u64 read_ops;
    __u64 write_ops;
    __u64 open_ops;
    __u64 close_ops;
    __u64 sync_ops;
    __u64 bytes_read;
    __u64 bytes_written;
    __u64 read_latency_total;
    __u64 write_latency_total;
    __u64 open_latency_total;
    __u64 io_errors;
    __u64 permission_errors;
    __u64 last_update;
};

// File access pattern tracking
struct file_access {
    char file_path[256];
    __u64 access_count;
    __u64 bytes_accessed;
    __u64 latency_total;
    __u32 access_type; // 0=read, 1=write, 2=read_write
    __u64 cgroup_id;
};

// I/O latency bucket
struct fs_latency_bucket {
    __u64 count;
    __u64 upper_bound_us;
};

// I/O operation tracking
struct io_operation {
    __u64 start_time;
    __u64 cgroup_id;
    __u32 pid;
    __u32 op_type; // 0=read, 1=write, 2=open
    __s32 fd;
    __u64 size;
};

// File system event for real-time processing
struct fs_event {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 event_type; // 0=read, 1=write, 2=open, 3=close, 4=error
    __s32 fd;
    __u64 size;
    __u64 latency_us;
    char filename[256];
    __s32 error_code;
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // cgroup_id
    __type(value, struct fs_metrics);
} fs_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 1000);
    __type(key, __u64);     // file_path_hash
    __type(value, struct file_access);
} file_access_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // pid_fd combination
    __type(value, struct io_operation);
} io_operations_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} fs_events SEC(".maps");

// Helper to get cgroup ID
static __always_inline __u64 get_cgroup_id(void) {
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    return BPF_CORE_READ(task, cgroups, dfl_cgrp, kn, id);
}

// Helper to hash file path
static __always_inline __u64 hash_file_path(const char *path) {
    __u64 hash = 5381;
    for (int i = 0; i < 256 && path[i] != '\0'; i++) {
        hash = ((hash << 5) + hash) + path[i];
    }
    return hash;
}

// Tracepoint for sys_enter_openat
SEC("tracepoint/syscalls/sys_enter_openat")
int trace_openat_enter(struct trace_event_raw_sys_enter *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __u32 pid = pid_fd >> 32;
    __u64 cgroup_id = get_cgroup_id();

    struct io_operation op = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
        .op_type = 2, // open
        .fd = -1,
        .size = 0
    };

    bpf_map_update_elem(&io_operations_map, &pid_fd, &op, BPF_ANY);
    return 0;
}

// Tracepoint for sys_exit_openat
SEC("tracepoint/syscalls/sys_exit_openat")
int trace_openat_exit(struct trace_event_raw_sys_exit *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __s32 fd = (__s32)ctx->ret;

    struct io_operation *op = bpf_map_lookup_elem(&io_operations_map, &pid_fd);
    if (!op) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 latency_ns = now - op->start_time;
    __u64 latency_us = latency_ns / 1000;
    __u64 cgroup_id = op->cgroup_id;

    // Update metrics
    struct fs_metrics *metrics = bpf_map_lookup_elem(&fs_metrics_map, &cgroup_id);
    if (!metrics) {
        struct fs_metrics new_metrics = {0};
        new_metrics.open_ops = 1;
        new_metrics.open_latency_total = latency_us;
        new_metrics.last_update = now;
        bpf_map_update_elem(&fs_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->open_ops, 1);
        __sync_fetch_and_add(&metrics->open_latency_total, latency_us);
        metrics->last_update = now;
    }

    // Emit event if successful open
    if (fd >= 0) {
        struct fs_event *event = bpf_ringbuf_reserve(&fs_events, sizeof(*event), 0);
        if (event) {
            event->timestamp = now;
            event->cgroup_id = cgroup_id;
            event->pid = op->pid;
            event->event_type = 2; // open
            event->fd = fd;
            event->size = 0;
            event->latency_us = latency_us;
            event->error_code = 0;
            bpf_get_current_comm(&event->filename, sizeof(event->filename));
            bpf_ringbuf_submit(event, 0);
        }
    } else {
        // Error case
        __sync_fetch_and_add(&metrics->io_errors, 1);
    }

    bpf_map_delete_elem(&io_operations_map, &pid_fd);
    return 0;
}

// Tracepoint for sys_enter_read
SEC("tracepoint/syscalls/sys_enter_read")
int trace_read_enter(struct trace_event_raw_sys_enter *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __u32 pid = pid_fd >> 32;
    __u64 cgroup_id = get_cgroup_id();
    __s32 fd = (__s32)ctx->args[0];
    __u64 size = (__u64)ctx->args[2];

    struct io_operation op = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
        .op_type = 0, // read
        .fd = fd,
        .size = size
    };

    __u64 key = ((__u64)pid << 32) | (__u64)fd;
    bpf_map_update_elem(&io_operations_map, &key, &op, BPF_ANY);
    return 0;
}

// Tracepoint for sys_exit_read
SEC("tracepoint/syscalls/sys_exit_read")
int trace_read_exit(struct trace_event_raw_sys_exit *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __u32 pid = pid_fd >> 32;
    __s32 fd = -1; // We need to get this from the entry
    __s64 bytes_read = (__s64)ctx->ret;

    // Try to find the operation - we use a simplified key lookup
    struct io_operation *op = bpf_map_lookup_elem(&io_operations_map, &pid_fd);
    if (!op) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 latency_ns = now - op->start_time;
    __u64 latency_us = latency_ns / 1000;
    __u64 cgroup_id = op->cgroup_id;

    // Update metrics
    struct fs_metrics *metrics = bpf_map_lookup_elem(&fs_metrics_map, &cgroup_id);
    if (!metrics) {
        struct fs_metrics new_metrics = {0};
        new_metrics.read_ops = 1;
        new_metrics.read_latency_total = latency_us;
        if (bytes_read > 0) {
            new_metrics.bytes_read = bytes_read;
        }
        new_metrics.last_update = now;
        bpf_map_update_elem(&fs_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->read_ops, 1);
        __sync_fetch_and_add(&metrics->read_latency_total, latency_us);
        if (bytes_read > 0) {
            __sync_fetch_and_add(&metrics->bytes_read, bytes_read);
        } else {
            __sync_fetch_and_add(&metrics->io_errors, 1);
        }
        metrics->last_update = now;
    }

    // Emit event
    struct fs_event *event = bpf_ringbuf_reserve(&fs_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = now;
        event->cgroup_id = cgroup_id;
        event->pid = op->pid;
        event->event_type = 0; // read
        event->fd = op->fd;
        event->size = bytes_read > 0 ? bytes_read : 0;
        event->latency_us = latency_us;
        event->error_code = bytes_read < 0 ? -bytes_read : 0;
        bpf_ringbuf_submit(event, 0);
    }

    __u64 key = ((__u64)pid << 32) | (__u64)op->fd;
    bpf_map_delete_elem(&io_operations_map, &key);
    return 0;
}

// Tracepoint for sys_enter_write
SEC("tracepoint/syscalls/sys_enter_write")
int trace_write_enter(struct trace_event_raw_sys_enter *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __u32 pid = pid_fd >> 32;
    __u64 cgroup_id = get_cgroup_id();
    __s32 fd = (__s32)ctx->args[0];
    __u64 size = (__u64)ctx->args[2];

    struct io_operation op = {
        .start_time = bpf_ktime_get_ns(),
        .cgroup_id = cgroup_id,
        .pid = pid,
        .op_type = 1, // write
        .fd = fd,
        .size = size
    };

    __u64 key = ((__u64)pid << 32) | (__u64)fd;
    bpf_map_update_elem(&io_operations_map, &key, &op, BPF_ANY);
    return 0;
}

// Tracepoint for sys_exit_write
SEC("tracepoint/syscalls/sys_exit_write")
int trace_write_exit(struct trace_event_raw_sys_exit *ctx) {
    __u64 pid_fd = bpf_get_current_pid_tgid();
    __u32 pid = pid_fd >> 32;
    __s64 bytes_written = (__s64)ctx->ret;

    struct io_operation *op = bpf_map_lookup_elem(&io_operations_map, &pid_fd);
    if (!op) {
        return 0;
    }

    __u64 now = bpf_ktime_get_ns();
    __u64 latency_ns = now - op->start_time;
    __u64 latency_us = latency_ns / 1000;
    __u64 cgroup_id = op->cgroup_id;

    // Update metrics
    struct fs_metrics *metrics = bpf_map_lookup_elem(&fs_metrics_map, &cgroup_id);
    if (!metrics) {
        struct fs_metrics new_metrics = {0};
        new_metrics.write_ops = 1;
        new_metrics.write_latency_total = latency_us;
        if (bytes_written > 0) {
            new_metrics.bytes_written = bytes_written;
        }
        new_metrics.last_update = now;
        bpf_map_update_elem(&fs_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
    } else {
        __sync_fetch_and_add(&metrics->write_ops, 1);
        __sync_fetch_and_add(&metrics->write_latency_total, latency_us);
        if (bytes_written > 0) {
            __sync_fetch_and_add(&metrics->bytes_written, bytes_written);
        } else {
            __sync_fetch_and_add(&metrics->io_errors, 1);
        }
        metrics->last_update = now;
    }

    // Emit event
    struct fs_event *event = bpf_ringbuf_reserve(&fs_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = now;
        event->cgroup_id = cgroup_id;
        event->pid = op->pid;
        event->event_type = 1; // write
        event->fd = op->fd;
        event->size = bytes_written > 0 ? bytes_written : 0;
        event->latency_us = latency_us;
        event->error_code = bytes_written < 0 ? -bytes_written : 0;
        bpf_ringbuf_submit(event, 0);
    }

    __u64 key = ((__u64)pid << 32) | (__u64)op->fd;
    bpf_map_delete_elem(&io_operations_map, &key);
    return 0;
}

char _license[] SEC("license") = "GPL";