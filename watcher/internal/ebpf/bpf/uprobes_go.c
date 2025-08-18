// +build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct req_stats_t { __u64 req_count; __u64 total_ns; };

struct { __uint(type, BPF_MAP_TYPE_HASH); __uint(max_entries, 8192); __type(key, __u64); __type(value, struct req_stats_t); } req_stats_map SEC(".maps");

struct { __uint(type, BPF_MAP_TYPE_HASH); __uint(max_entries, 16384); __type(key, __u64); __type(value, __u64); } req_start SEC(".maps");

SEC("uprobe/handler_entry")
int uprobe__handler_entry(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ts = bpf_ktime_get_ns();
    bpf_map_update_elem(&req_start, &pid_tgid, &ts, BPF_ANY);
    return 0;
}

SEC("uretprobe/handler_return")
int uretprobe__handler_return(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 *tsp = bpf_map_lookup_elem(&req_start, &pid_tgid);
    if (!tsp) return 0;
    __u64 delta = bpf_ktime_get_ns() - *tsp;
    bpf_map_delete_elem(&req_start, &pid_tgid);
    __u64 cgid = bpf_get_current_cgroup_id();
    struct req_stats_t zero = {};
    struct req_stats_t *rs = bpf_map_lookup_elem(&req_stats_map, &cgid);
    if (!rs) {
        bpf_map_update_elem(&req_stats_map, &cgid, &zero, BPF_NOEXIST);
        rs = bpf_map_lookup_elem(&req_stats_map, &cgid);
        if (!rs) return 0;
    }
    __sync_fetch_and_add(&rs->req_count, 1);
    __sync_fetch_and_add(&rs->total_ns, delta);
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
