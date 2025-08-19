// +build ignore
// l4metrics.c
//
// Collects TCP L4 metrics using:
// - sock:inet_sock_set_state tracepoint for TCP state transitions
// - kprobe tcp_sendmsg / tcp_cleanup_rbuf for bytes in/out
// Emits ringbuf events for ESTABLISHED and CLOSED.

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "GPL";

// Avoid pulling system headers; define what we need.
#ifndef AF_INET
#define AF_INET  2
#endif
#ifndef AF_INET6
#define AF_INET6 10
#endif
#ifndef IPPROTO_TCP
#define IPPROTO_TCP 6
#endif

#define TCP_ESTABLISHED 1
#define TCP_SYN_SENT    2
#define TCP_CLOSE       7

enum event_type {
    EVENT_ESTABLISHED = 1,
    EVENT_CLOSED = 2,
};

struct flow_key_v4 {
    __u32 saddr;  // host order
    __u32 daddr;  // host order
    __u16 sport;  // network order
    __u16 dport;  // network order
    __u32 netns;
    __u32 pid;
};

struct flow_state {
    __u64 ts_start_ns;
    __u64 ts_estab_ns;
    __u64 bytes_sent;
    __u64 bytes_recv;
    __u32 last_rtt_us;
    struct flow_key_v4 key;
};

struct flow_event_v4 {
    __u64 ts_ns;
    __u32 type;
    struct flow_key_v4 key;
    __u64 connect_latency_ns;
    __u64 bytes_sent;
    __u64 bytes_recv;
    __u64 ts_start_ns;
    __u64 ts_end_ns;
    __u64 sock_ptr;
    __u32 rtt_us;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, __u64);              // (u64)sk
    __type(value, struct flow_state);
} flows SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

static __always_inline bool is_tcp_family(__u16 family, __u16 proto) {
    return proto == IPPROTO_TCP && (family == AF_INET || family == AF_INET6);
}

SEC("tracepoint/sock/inet_sock_set_state")
int tp_inet_sock_set_state(struct trace_event_raw_inet_sock_set_state *ctx)
{
    __u16 family   = BPF_CORE_READ(ctx, family);
    __u16 newstate = BPF_CORE_READ(ctx, newstate);
    __u16 proto    = BPF_CORE_READ(ctx, protocol);

    if (!is_tcp_family(family, proto))
        return 0;

    __u64 now = bpf_ktime_get_ns();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // skaddr
    void *skp = NULL;
    bpf_core_read(&skp, sizeof(skp), &ctx->skaddr);
    __u64 skaddr = (__u64)skp;

    // IPv4 only in this minimal example
    if (family != AF_INET)
        return 0;

    // Ports & addresses (layout differs across kernels; read generically)
    __u16 sport = 0, dport = 0;
    __u32 saddr = 0, daddr = 0;
    bpf_core_read(&sport, sizeof(sport), &ctx->sport);
    bpf_core_read(&dport, sizeof(dport), &ctx->dport);
    // saddr/daddr can be __u32 or __u8[4] depending on kernel; generic read works
    bpf_core_read(&saddr, sizeof(saddr), &ctx->saddr);
    bpf_core_read(&daddr, sizeof(daddr), &ctx->daddr);

    __u32 netns = 0;
    struct sock *sk = (struct sock *)ctx->skaddr;
    if (sk) {
        struct net *net = BPF_CORE_READ(sk, __sk_common.skc_net.net);
        if (net)
            netns = BPF_CORE_READ(net, ns.inum);
    }

    if (newstate == TCP_SYN_SENT) {
        struct flow_state st = {};
        st.ts_start_ns = now;
        st.key.saddr   = saddr;
        st.key.daddr   = daddr;
        st.key.sport   = sport;
        st.key.dport   = dport;
        st.key.netns   = netns;
        st.key.pid     = pid;
        bpf_map_update_elem(&flows, &skaddr, &st, BPF_ANY);
        return 0;
    }

    if (newstate == TCP_ESTABLISHED) {
        struct flow_state *st = bpf_map_lookup_elem(&flows, &skaddr);
        if (!st) {
            struct flow_state nst = {};
            nst.ts_start_ns = now;
            nst.ts_estab_ns = now;
            nst.key.saddr = saddr; nst.key.daddr = daddr;
            nst.key.sport = sport; nst.key.dport = dport;
            nst.key.netns = netns; nst.key.pid   = pid;
            bpf_map_update_elem(&flows, &skaddr, &nst, BPF_ANY);

            struct flow_event_v4 *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
            if (e) {
                e->ts_ns = now; e->type = EVENT_ESTABLISHED;
                e->key = nst.key; e->connect_latency_ns = 0;
                e->bytes_sent = 0; e->bytes_recv = 0;
                e->ts_start_ns = nst.ts_start_ns; e->ts_end_ns = 0;
                e->sock_ptr = skaddr; e->rtt_us = 0;
                bpf_ringbuf_submit(e, 0);
            }
            return 0;
        }

        st->ts_estab_ns = now;
        struct flow_event_v4 *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
        if (e) {
            e->ts_ns = now; e->type = EVENT_ESTABLISHED;
            e->key = st->key;
            e->connect_latency_ns = (st->ts_start_ns ? now - st->ts_start_ns : 0);
            e->bytes_sent = st->bytes_sent;
            e->bytes_recv = st->bytes_recv;
            e->ts_start_ns = st->ts_start_ns;
            e->ts_end_ns = 0;
            e->sock_ptr = skaddr;
            e->rtt_us = st->last_rtt_us;
            bpf_ringbuf_submit(e, 0);
        }
        return 0;
    }

    if (newstate == TCP_CLOSE) {
        struct flow_state *st = bpf_map_lookup_elem(&flows, &skaddr);
        if (!st) return 0;
        struct flow_event_v4 *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
        if (e) {
            e->ts_ns = now; e->type = EVENT_CLOSED; e->key = st->key;
            e->connect_latency_ns =
                (st->ts_estab_ns > st->ts_start_ns && st->ts_start_ns) ?
                (st->ts_estab_ns - st->ts_start_ns) : 0;
            e->bytes_sent = st->bytes_sent; e->bytes_recv = st->bytes_recv;
            e->ts_start_ns = st->ts_start_ns; e->ts_end_ns = now;
            e->sock_ptr = skaddr; e->rtt_us = st->last_rtt_us;
            bpf_ringbuf_submit(e, 0);
        }
        bpf_map_delete_elem(&flows, &skaddr);
    }
    return 0;
}

// Bytes sent
SEC("kprobe/tcp_sendmsg")
int k_tcp_sendmsg(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    __u64 skaddr = (unsigned long)sk;
    size_t size = (size_t)PT_REGS_PARM3(ctx);

    struct flow_state *st = bpf_map_lookup_elem(&flows, &skaddr);
    if (st && size > 0) {
        __sync_fetch_and_add(&st->bytes_sent, size);
    }
    return 0;
}

// Bytes received
SEC("kprobe/tcp_cleanup_rbuf")
int k_tcp_cleanup_rbuf(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    __u64 skaddr = (unsigned long)sk;
    int copied = (int)PT_REGS_PARM2(ctx);
    if (copied <= 0) return 0;

    struct flow_state *st = bpf_map_lookup_elem(&flows, &skaddr);
    if (st) {
        __sync_fetch_and_add(&st->bytes_recv, (__u64)copied);
    }
    return 0;
}

// RTT updates (srtt_us >> 3)
SEC("kprobe/tcp_rcv_established")
int k_tcp_rcv_established(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    __u64 skaddr = (unsigned long)sk;
    struct flow_state *st = bpf_map_lookup_elem(&flows, &skaddr);
    if (!st) return 0;

    struct tcp_sock *tp = (struct tcp_sock *)sk;
    __u32 srtt = BPF_CORE_READ(tp, srtt_us); // scaled by 8
    st->last_rtt_us = srtt >> 3;
    return 0;
}
