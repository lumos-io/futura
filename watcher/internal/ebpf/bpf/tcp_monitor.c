//go:build ignore

// #include "vmlinux.h"
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct stats_t { __u64 bytes_sent; __u64 send_calls; __u64 total_send_ns; __u64 bytes_recv; __u64 recv_calls; __u64 total_recv_ns; };

struct { __uint(type, BPF_MAP_TYPE_HASH); __uint(max_entries, 8192); __type(key, __u64); __type(value, struct stats_t); } stats_map SEC(".maps");

struct { __uint(type, BPF_MAP_TYPE_HASH); __uint(max_entries, 16384); __type(key, __u64); __type(value, __u64); } start_ts SEC(".maps");

SEC("kprobe/tcp_sendmsg")
int kprobe__tcp_sendmsg(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ts = bpf_ktime_get_ns();
    bpf_map_update_elem(&start_ts, &pid_tgid, &ts, BPF_ANY);
    __u64 size = 0;
#ifdef __x86_64__
    size = PT_REGS_PARM3(ctx);
#endif
    __u64 cgid = bpf_get_current_cgroup_id();
    struct stats_t zero = {};
    struct stats_t *s = bpf_map_lookup_elem(&stats_map, &cgid);
    if (!s) {
        bpf_map_update_elem(&stats_map, &cgid, &zero, BPF_NOEXIST);
        s = bpf_map_lookup_elem(&stats_map, &cgid);
        if (!s) return 0;
    }
    __sync_fetch_and_add(&s->bytes_sent, size);
    __sync_fetch_and_add(&s->send_calls, 1);
    return 0;
}

SEC("kretprobe/tcp_sendmsg")
int kretprobe__tcp_sendmsg(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 *tsp = bpf_map_lookup_elem(&start_ts, &pid_tgid);
    if (!tsp) return 0;
    __u64 delta = bpf_ktime_get_ns() - *tsp;
    bpf_map_delete_elem(&start_ts, &pid_tgid);
    __u64 cgid = bpf_get_current_cgroup_id();
    struct stats_t zero = {};
    struct stats_t *s = bpf_map_lookup_elem(&stats_map, &cgid);
    if (!s) {
        bpf_map_update_elem(&stats_map, &cgid, &zero, BPF_NOEXIST);
        s = bpf_map_lookup_elem(&stats_map, &cgid);
        if (!s) return 0;
    }
    __sync_fetch_and_add(&s->total_send_ns, delta);
    return 0;
}

SEC("kprobe/tcp_recvmsg")
int kprobe__tcp_recvmsg(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ts = bpf_ktime_get_ns();
    bpf_map_update_elem(&start_ts, &pid_tgid, &ts, BPF_ANY);
    __u64 size = 0;
#ifdef __x86_64__
    size = PT_REGS_PARM3(ctx);
#endif
    __u64 cgid = bpf_get_current_cgroup_id();
    struct stats_t zero = {};
    struct stats_t *s = bpf_map_lookup_elem(&stats_map, &cgid);
    if (!s) {
        bpf_map_update_elem(&stats_map, &cgid, &zero, BPF_NOEXIST);
        s = bpf_map_lookup_elem(&stats_map, &cgid);
        if (!s) return 0;
    }
    __sync_fetch_and_add(&s->bytes_recv, size);
    __sync_fetch_and_add(&s->recv_calls, 1);
    return 0;
}

SEC("kretprobe/tcp_recvmsg")
int kretprobe__tcp_recvmsg(struct pt_regs *ctx) {
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 *tsp = bpf_map_lookup_elem(&start_ts, &pid_tgid);
    if (!tsp) return 0;
    __u64 delta = bpf_ktime_get_ns() - *tsp;
    bpf_map_delete_elem(&start_ts, &pid_tgid);
    __u64 cgid = bpf_get_current_cgroup_id();
    struct stats_t zero = {};
    struct stats_t *s = bpf_map_lookup_elem(&stats_map, &cgid);
    if (!s) {
        bpf_map_update_elem(&stats_map, &cgid, &zero, BPF_NOEXIST);
        s = bpf_map_lookup_elem(&stats_map, &cgid);
        if (!s) return 0;
    }
    __sync_fetch_and_add(&s->total_recv_ns, delta);
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
