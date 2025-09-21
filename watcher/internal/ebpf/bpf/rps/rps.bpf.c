//go:build ignore
// +build ignore
// file: rps.bpf.c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);   // cgroup_id
    __type(value, __u64); // request count
    __uint(max_entries, 16384);
} rps SEC(".maps");

SEC("tracepoint/sock/inet_sock_set_state")
int trace_sock_state(struct trace_event_raw_inet_sock_set_state *ctx)
{
    // We're interested in TCP connections: SYN_RECV (server side), ESTABLISHED (client side)
    int oldstate = ctx->oldstate;
    int newstate = ctx->newstate;

    if (newstate == TCP_ESTABLISHED && (oldstate == TCP_SYN_SENT || oldstate == TCP_SYN_RECV)) {
        __u64 cgid = bpf_get_current_cgroup_id();

        __u64 *count = bpf_map_lookup_elem(&rps, &cgid);
        if (!count) {
            __u64 init = 1;
            bpf_map_update_elem(&rps, &cgid, &init, BPF_ANY);
        } else {
            __sync_fetch_and_add(count, 1);
        }
    }
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
