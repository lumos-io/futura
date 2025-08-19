// +build ignore
/*
 * l7probes.c
 *
 * Minimal CO-RE L7 sampler (HTTP/gRPC starter):
 *  - kprobe tcp_sendmsg: sample outgoing payload (first iov)
 *  - kprobe tcp_recvmsg (entry): stash (msg,sk) -> map
 *  - kretprobe tcp_recvmsg (exit): read user buffer (first iov) after copy, emit
 *
 * Requirements:
 *  - vmlinux.h next to this file (bpftool btf dump ... -> vmlinux.h)
 *  - clang + libbpf headers; compile with: -target bpf -D__TARGET_ARCH_x86 -D__BPF_TRACING__
 *
 * Notes:
 *  - If your libbpf headers are old (no BPF_CORE_FIELD_EXISTS), the
 *    compile-time detection will be disabled (no iov extraction). Upgrade libbpf
 *    for best results.
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "GPL";

/* Compatibility: if BPF_CORE_FIELD_EXISTS not provided by libbpf headers,
 * define it to 0 so compilation succeeds (feature detection disabled).
 * Prefer upgrading libbpf - modern headers provide this macro.
 */
#ifndef BPF_CORE_FIELD_EXISTS
#define BPF_CORE_FIELD_EXISTS(...) 0
#endif

#define MAX_SAMPLE 128

/* Event delivered to user space via ringbuf */
struct l7_event {
    __u64 ts_ns;
    __u32 pid;
    __u32 netns;
    __u64 sock_ptr;
    __u32 len;     /* bytes copied into data (<= MAX_SAMPLE) */
    __u32 is_recv; /* 1=recv (server->client), 0=send (client->server) */
    char   data[MAX_SAMPLE];
};

/* Ring buffer map */
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} l7_events SEC(".maps");

/* Per-task context stored on tcp_recvmsg entry so we can read buffer on return */
struct recv_ctx {
    const struct msghdr *msg;
    struct sock *sk;
    size_t len;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 16384);
    __type(key, __u64);          /* pid_tgid */
    __type(value, struct recv_ctx);
} recv_map SEC(".maps");

/* Helper: get netns inode from sock using CO-RE */
static __always_inline __u32 get_netns_from_sk(struct sock *sk)
{
    if (!sk) return 0;
    struct net *net = BPF_CORE_READ(sk, __sk_common.skc_net.net);
    if (!net) return 0;
    return BPF_CORE_READ(net, ns.inum);
}

/* Emit sample from user-space buffer pointer `base` (best-effort) */
static __always_inline void emit_sample_from_user(struct sock *sk, const void *base, __u64 want, __u32 is_recv)
{
    if (!sk || !base || want == 0) return;

    __u64 cap = want;
    if (cap > MAX_SAMPLE) cap = MAX_SAMPLE;

    struct l7_event *e = bpf_ringbuf_reserve(&l7_events, sizeof(*e), 0);
    if (!e) return;

    e->ts_ns = bpf_ktime_get_ns();
    e->pid   = bpf_get_current_pid_tgid() >> 32;
    e->netns = get_netns_from_sk(sk);
    e->sock_ptr = (unsigned long)sk;
    e->is_recv = is_recv;
    e->len = (__u32)cap;

    if (bpf_probe_read_user(e->data, cap, base) == 0) {
        bpf_ringbuf_submit(e, 0);
    } else {
        bpf_ringbuf_discard(e, 0);
    }
}

/* Try to extract first base+len from msg->msg_iter in a CO-RE friendly way.
 * Uses compile-time selected field names. If no supported layout is found,
 * returns -1.
 *
 * Note: when BPF_CORE_FIELD_EXISTS == 0 (compat fallback), none of these
 * blocks will be compiled and function returns -1.
 */
static __always_inline int iter_first_base_len_from_msg(const struct msghdr *msg, void **base_out, __u64 *len_out)
{
    if (!msg || !base_out || !len_out) return -1;

    struct iov_iter iter = {};
    bpf_core_read(&iter, sizeof(iter), &msg->msg_iter);

    /* Path A: iter.iov (common kernel layout) */
#if BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->iov)
    struct iovec iov0 = {};
    struct iovec *iovp = NULL;
    bpf_core_read(&iovp, sizeof(iovp), &iter.iov);
    if (!iovp) return -1;
    bpf_core_read(&iov0, sizeof(iov0), iovp);
    if (!iov0.iov_base || iov0.iov_len == 0) return -1;
    *base_out = iov0.iov_base;
    *len_out  = iov0.iov_len;
    return 0;

    /* Path B: iter.ubuf (flat user buffer) */
#elif BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->ubuf)
    void *ubuf = NULL;
    bpf_core_read(&ubuf, sizeof(ubuf), &iter.ubuf);
    if (!ubuf) return -1;
    /* try iter.count as length */
#if BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->count)
    __u64 cnt = 0;
    bpf_core_read(&cnt, sizeof(cnt), &iter.count);
    if (cnt == 0) return -1;
    *base_out = ubuf;
    *len_out = cnt;
    return 0;
#else
    return -1;
#endif

    /* Path C: iter.kvec (kernel vec) */
#elif BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->kvec)
    struct kvec kv0 = {};
    struct kvec *kvp = NULL;
    bpf_core_read(&kvp, sizeof(kvp), &iter.kvec);
    if (!kvp) return -1;
    bpf_core_read(&kv0, sizeof(kv0), kvp);
    if (!kv0.iov_base || kv0.iov_len == 0) return -1;
    *base_out = kv0.iov_base;
    *len_out  = kv0.iov_len;
    return 0;

#else
    /* Unknown layout or feature detection disabled; cannot extract buffer */
    (void)iter; /* silence unused */
    return -1;
#endif
}

/* kprobe: outgoing data - tcp_sendmsg */
SEC("kprobe/tcp_sendmsg")
int BPF_KPROBE(kprobe_tcp_sendmsg, struct sock *sk, struct msghdr *msg, size_t size)
{
    if (!msg || (long)size <= 0) return 0;

    void *base = NULL;
    __u64 len = 0;
    if (iter_first_base_len_from_msg(msg, &base, &len) == 0 && base && len > 0) {
        __u64 want = len;
        if (want > (unsigned long)size) want = size;
        emit_sample_from_user(sk, base, want, /*is_recv=*/0);
    }
    return 0;
}

/* kprobe: recvmsg entry - stash context keyed by pid_tgid */
SEC("kprobe/tcp_recvmsg")
int BPF_KPROBE(kprobe_tcp_recvmsg_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags, int *addr_len)
{
    if (!msg) return 0;
    __u64 id = bpf_get_current_pid_tgid();
    struct recv_ctx r = {};
    r.msg = msg;
    r.sk  = sk;
    r.len = len;
    bpf_map_update_elem(&recv_map, &id, &r, BPF_ANY);
    return 0;
}

/* kretprobe: recvmsg exit - read user buffer AFTER kernel copied bytes */
SEC("kretprobe/tcp_recvmsg")
int BPF_KRETPROBE(kret_tcp_recvmsg)
{
    long ret = PT_REGS_RC(ctx);
    __u64 id = bpf_get_current_pid_tgid();
    struct recv_ctx *s = bpf_map_lookup_elem(&recv_map, &id);
    if (!s) return 0;

    if (ret > 0) {
        void *base = NULL;
        __u64 len = 0;
        if (iter_first_base_len_from_msg(s->msg, &base, &len) == 0 && base && len > 0) {
            __u64 want = len;
            if (want > (unsigned long)ret) want = ret;
            emit_sample_from_user(s->sk, base, want, /*is_recv=*/1);
        }
    }

    bpf_map_delete_elem(&recv_map, &id);
    return 0;
}
