//go:build ignore
// +build ignore
// file: http_metrics.bpf.c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#define MAX_ENTRIES 16384
#define MAX_ENDPOINTS 1024
#define HTTP_METHOD_MAX_LEN 16
#define HTTP_URL_MAX_LEN 256

// Request tracking structure
struct http_request {
    __u64 start_time;
    __u64 cgroup_id;
    __u32 pid;
    __u32 status_code;
    char method[HTTP_METHOD_MAX_LEN];
    char url[HTTP_URL_MAX_LEN];
};

// HTTP metrics aggregation
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

// Endpoint-specific metrics
struct endpoint_key {
    __u64 cgroup_id;
    char endpoint[128];
};

struct endpoint_metrics {
    __u64 request_count;
    __u64 error_count;
    __u64 latency_sum;
    __u64 p95_latency;
    __u64 last_update;
};

// Latency histogram bucket (renamed to avoid vmlinux conflict)
struct http_latency_bucket {
    __u64 count;
    __u64 upper_bound_us;
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // request_id (combination of pid, fd, timestamp)
    __type(value, struct http_request);
    __uint(max_entries, MAX_ENTRIES);
} active_requests SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct http_metrics);
    __uint(max_entries, MAX_ENTRIES);
} http_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct endpoint_key);
    __type(value, struct endpoint_metrics);
    __uint(max_entries, MAX_ENDPOINTS);
} endpoint_metrics_map SEC(".maps");

// Latency histogram (per cgroup)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);  // cgroup_id
    __type(value, struct http_latency_bucket[10]);  // 10 buckets
    __uint(max_entries, MAX_ENTRIES);
} latency_histogram SEC(".maps");

// Ring buffer for real-time events
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} http_events SEC(".maps");

// Event structure for ring buffer
struct http_event {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 status_code;
    __u64 latency_us;
    char method[HTTP_METHOD_MAX_LEN];
    char url[64];
};

// Helper functions
static __always_inline __u64 get_request_id(__u32 pid, __u32 fd) {
    return ((__u64)pid << 32) | fd;
}

static __always_inline void update_latency_histogram(__u64 cgroup_id, __u64 latency_us) {
    struct http_latency_bucket *buckets = bpf_map_lookup_elem(&latency_histogram, &cgroup_id);
    if (!buckets) {
        struct http_latency_bucket init_buckets[10] = {
            {0, 100},      // < 100us
            {0, 500},      // < 500us
            {0, 1000},     // < 1ms
            {0, 5000},     // < 5ms
            {0, 10000},    // < 10ms
            {0, 50000},    // < 50ms
            {0, 100000},   // < 100ms
            {0, 500000},   // < 500ms
            {0, 1000000},  // < 1s
            {0, 0xFFFFFFFF} // >= 1s
        };
        bpf_map_update_elem(&latency_histogram, &cgroup_id, init_buckets, BPF_ANY);
        buckets = bpf_map_lookup_elem(&latency_histogram, &cgroup_id);
    }

    if (buckets) {
        for (int i = 0; i < 10; i++) {
            if (latency_us <= buckets[i].upper_bound_us || i == 9) {
                __sync_fetch_and_add(&buckets[i].count, 1);
                break;
            }
        }
    }
}

static __always_inline void extract_http_info(struct http_request *req, void *buf, size_t size) {
    // Simple HTTP method and URL extraction
    // This is a simplified version - in production, you'd want more robust parsing
    char *data = (char *)buf;

    // Look for HTTP methods at the start of the buffer
    if (size >= 4) {
        if (data[0] == 'G' && data[1] == 'E' && data[2] == 'T' && data[3] == ' ') {
            __builtin_memcpy(req->method, "GET", 4);
            // Extract URL after "GET "
            for (int i = 4; i < size && i < (4 + HTTP_URL_MAX_LEN - 1); i++) {
                if (data[i] == ' ' || data[i] == '\r' || data[i] == '\n') break;
                req->url[i-4] = data[i];
            }
        } else if (data[0] == 'P' && data[1] == 'O' && data[2] == 'S' && data[3] == 'T') {
            __builtin_memcpy(req->method, "POST", 5);
        } else if (data[0] == 'P' && data[1] == 'U' && data[2] == 'T' && data[3] == ' ') {
            __builtin_memcpy(req->method, "PUT", 4);
        } else if (data[0] == 'D' && data[1] == 'E' && data[2] == 'L') {
            __builtin_memcpy(req->method, "DELETE", 7);
        }
    }
}

static __always_inline __u32 extract_status_code(void *buf, size_t size) {
    char *data = (char *)buf;

    // Look for "HTTP/1.1 " or "HTTP/2 " followed by status code
    for (int i = 0; i < size - 12; i++) {
        if (data[i] == 'H' && data[i+1] == 'T' && data[i+2] == 'T' && data[i+3] == 'P') {
            // Skip to status code (after "HTTP/x.x ")
            for (int j = i + 4; j < size - 3; j++) {
                if (data[j] == ' ' && j + 3 < size) {
                    // Parse 3-digit status code
                    if (data[j+1] >= '0' && data[j+1] <= '9' &&
                        data[j+2] >= '0' && data[j+2] <= '9' &&
                        data[j+3] >= '0' && data[j+3] <= '9') {
                        return (data[j+1] - '0') * 100 + (data[j+2] - '0') * 10 + (data[j+3] - '0');
                    }
                }
            }
        }
    }
    return 0;
}

// Tracepoint for socket write (HTTP request start)
SEC("tracepoint/syscalls/sys_enter_write")
int trace_http_request_start(struct trace_event_raw_sys_enter *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    // Get file descriptor
    int fd = (int)ctx->args[0];
    void *buf = (void *)ctx->args[1];
    size_t count = (size_t)ctx->args[2];

    // Only process potential HTTP traffic (heuristic: port-like fd and reasonable size)
    if (fd < 3 || count < 10 || count > 8192) {
        return 0;
    }

    __u64 request_id = get_request_id(pid, fd);

    struct http_request req = {};
    req.start_time = bpf_ktime_get_ns();
    req.cgroup_id = cgroup_id;
    req.pid = pid;

    // Try to extract HTTP information from the buffer
    extract_http_info(&req, buf, count < 512 ? count : 512);

    bpf_map_update_elem(&active_requests, &request_id, &req, BPF_ANY);

    return 0;
}

/*

struct trace_event_raw_sys_exit {
	struct trace_entry ent;
	long int id;
	long int ret;
	char __data[0];
};

*/

// Simple map to track fd for read operations
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);  // pid
    __type(value, __u32); // fd
    __uint(max_entries, MAX_ENTRIES);
} read_fd_tracker SEC(".maps");

// Tracepoint for socket read entry to track fd
SEC("tracepoint/syscalls/sys_enter_read")
int trace_http_read_start(struct trace_event_raw_sys_enter *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u32 fd = (__u32)ctx->args[0];

    // Store the fd for this pid
    bpf_map_update_elem(&read_fd_tracker, &pid, &fd, BPF_ANY);
    return 0;
}

// Tracepoint for socket read (HTTP response received)
SEC("tracepoint/syscalls/sys_exit_read")
int trace_http_request_end(struct trace_event_raw_sys_exit *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    long ret = ctx->ret;

    if (ret <= 0) {
        return 0;
    }

    // Get the fd from our tracker
    __u32 *fd_ptr = bpf_map_lookup_elem(&read_fd_tracker, &pid);
    if (!fd_ptr) {
        return 0;
    }
    __u32 fd = *fd_ptr;

    // Clean up the tracker entry
    bpf_map_delete_elem(&read_fd_tracker, &pid);

    __u64 request_id = get_request_id(pid, fd);

    struct http_request *req = bpf_map_lookup_elem(&active_requests, &request_id);
    if (!req) {
        return 0;
    }

    __u64 end_time = bpf_ktime_get_ns();
    __u64 latency_ns = end_time - req->start_time;
    __u64 latency_us = latency_ns / 1000;

    // Try to extract status code from response
    __u32 status_code = 200; // Default assumption
    // In a real implementation, you'd read the response buffer here

    // Update metrics
    struct http_metrics *metrics = bpf_map_lookup_elem(&http_metrics_map, &req->cgroup_id);
    if (!metrics) {
        struct http_metrics init_metrics = {};
        init_metrics.latency_min = 0xFFFFFFFFFFFFFFFF;
        bpf_map_update_elem(&http_metrics_map, &req->cgroup_id, &init_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&http_metrics_map, &req->cgroup_id);
    }

    if (metrics) {
        __sync_fetch_and_add(&metrics->request_count, 1);
        __sync_fetch_and_add(&metrics->latency_sum, latency_us);
        metrics->last_update = end_time;

        if (latency_us > metrics->latency_max) {
            metrics->latency_max = latency_us;
        }
        if (latency_us < metrics->latency_min) {
            metrics->latency_min = latency_us;
        }

        // Update status code counters
        if (status_code >= 200 && status_code < 300) {
            __sync_fetch_and_add(&metrics->status_2xx, 1);
        } else if (status_code >= 300 && status_code < 400) {
            __sync_fetch_and_add(&metrics->status_3xx, 1);
        } else if (status_code >= 400 && status_code < 500) {
            __sync_fetch_and_add(&metrics->status_4xx, 1);
            __sync_fetch_and_add(&metrics->error_count, 1);
        } else if (status_code >= 500) {
            __sync_fetch_and_add(&metrics->status_5xx, 1);
            __sync_fetch_and_add(&metrics->error_count, 1);
        }
    }

    // Update latency histogram
    update_latency_histogram(req->cgroup_id, latency_us);

    // Send event to ring buffer for real-time monitoring
    struct http_event *event = bpf_ringbuf_reserve(&http_events, sizeof(struct http_event), 0);
    if (event) {
        event->timestamp = end_time;
        event->cgroup_id = req->cgroup_id;
        event->pid = req->pid;
        event->status_code = status_code;
        event->latency_us = latency_us;
        __builtin_memcpy(event->method, req->method, HTTP_METHOD_MAX_LEN);
        __builtin_memcpy(event->url, req->url, 64);
        bpf_ringbuf_submit(event, 0);
    }

    // Clean up the request tracking
    bpf_map_delete_elem(&active_requests, &request_id);

    return 0;
}

// Alternative: Uprobe for Go HTTP handlers
SEC("uprobe/go_http_handler")
int trace_go_http_start(struct pt_regs *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    // For Go HTTP handlers, we can use the function parameters
    // This would need to be attached to specific Go HTTP handler functions

    __u64 request_id = get_request_id(pid, 0); // Use 0 as fd for uprobes

    struct http_request req = {};
    req.start_time = bpf_ktime_get_ns();
    req.cgroup_id = cgroup_id;
    req.pid = pid;
    __builtin_memcpy(req.method, "HTTP", 5);

    bpf_map_update_elem(&active_requests, &request_id, &req, BPF_ANY);

    return 0;
}

SEC("uretprobe/go_http_handler")
int trace_go_http_end(struct pt_regs *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 request_id = get_request_id(pid, 0);

    struct http_request *req = bpf_map_lookup_elem(&active_requests, &request_id);
    if (!req) {
        return 0;
    }

    __u64 end_time = bpf_ktime_get_ns();
    __u64 latency_ns = end_time - req->start_time;
    __u64 latency_us = latency_ns / 1000;

    // Update metrics (similar to syscall version)
    struct http_metrics *metrics = bpf_map_lookup_elem(&http_metrics_map, &req->cgroup_id);
    if (!metrics) {
        struct http_metrics init_metrics = {};
        init_metrics.latency_min = 0xFFFFFFFFFFFFFFFF;
        bpf_map_update_elem(&http_metrics_map, &req->cgroup_id, &init_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&http_metrics_map, &req->cgroup_id);
    }

    if (metrics) {
        __sync_fetch_and_add(&metrics->request_count, 1);
        __sync_fetch_and_add(&metrics->latency_sum, latency_us);
        __sync_fetch_and_add(&metrics->status_2xx, 1); // Assume success for uprobe
        metrics->last_update = end_time;

        if (latency_us > metrics->latency_max) {
            metrics->latency_max = latency_us;
        }
        if (latency_us < metrics->latency_min) {
            metrics->latency_min = latency_us;
        }
    }

    update_latency_histogram(req->cgroup_id, latency_us);

    // Clean up
    bpf_map_delete_elem(&active_requests, &request_id);

    return 0;
}

char LICENSE[] SEC("license") = "GPL";