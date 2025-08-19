// +build ignore
/*
 * l7probes.c
 *
 * Minimal, CO-RE-friendly L7 sampler (sendmsg + recvmsg):
 *  - stores scalar-only recv_ctx keyed by socket cookie (u64) at tcp_recvmsg entry
 *  - on kretprobe, recovers cookie via sk pointer from regs and looks up recv_ctx
 *  - extracts first iovec base (CO-RE safe) from msg pointer (from regs) and emits a ringbuf event
 *
 * Requirements:
 *  - vmlinux.h in same directory (bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h)
 *  - clang + modern libbpf headers (recommended). If BPF_CORE_FIELD_EXISTS missing, a fallback is used.
 *  - compile flags: -target bpf -D__TARGET_ARCH_x86 -D__BPF_TRACING__  (or arm64)
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>
#include <linux/limits.h>
#include <linux/const.h>

// Fallbacks in case not defined
#ifndef UINT32_MAX
#define UINT32_MAX 0xffffffffU
#endif

#ifndef INT32_MAX
#define INT32_MAX 2147483647
#endif

char LICENSE[] SEC("license") = "GPL";

/* compatibility: if libbpf header doesn't provide BPF_CORE_FIELD_EXISTS, define fallback (returns 0) */
#ifndef BPF_CORE_FIELD_EXISTS
#define BPF_CORE_FIELD_EXISTS(...) 0
#endif

#define MAX_SAMPLE 128

/* Event emitted to userspace */
struct l7_event {
    __u64 ts_ns;
    __u32 pid;
    __u32 netns;
    __u64 sock_ptr;
    __u32 len;     /* bytes copied into data */
    __u32 is_recv; /* 1 = recv (server->client), 0 = send (client->server) */
    char   data[MAX_SAMPLE];
};

/* ring buffer map for events */
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} l7_events SEC(".maps");

/* scalar-only recv context (safe for bpf2go)
 * keyed by socket cookie (u64) — avoids storing kernel pointers in map
 */
struct recv_ctx {
    __u64 sk_cookie; /* key */
    __u64 ts_ns;     /* timestamp at entry */
    __u32 len;       /* requested recv length */
    __u32 flags;     /* recv flags */
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 16384);
    __type(key, __u64);           /* socket cookie */
    __type(value, struct recv_ctx);
} recv_map SEC(".maps");

/* helper: get netns from sock (CO-RE) */
static __always_inline __u32 get_netns_from_sk(struct sock *sk)
{
    if (!sk) return 0;
    struct net *n = BPF_CORE_READ(sk, __sk_common.skc_net.net);
    if (!n) return 0;
    return BPF_CORE_READ(n, ns.inum);
}

/* helper: emit a sample given a user-space base pointer and a length */
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

/* Try to read the first iov/base from msg->msg_iter in a CO-RE friendly way.
 * Returns 0 on success and fills base_out/len_out. Non-zero on failure.
 *
 * Note: This uses compile-time #if BPF_CORE_FIELD_EXISTS checks. If your
 * libbpf headers are old such checks will evaluate to 0 and fallback will run.
 */
static __always_inline int iter_first_base_len_from_msg(const struct msghdr *msg, void **base_out, __u64 *len_out)
{
    if (!msg || !base_out || !len_out) return -1;

    struct iov_iter iter = {};
    bpf_core_read(&iter, sizeof(iter), &msg->msg_iter);

    /* Path A: iter.iov (most kernels) */
#if BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->iov)
    struct iovec iov0 = {};
    struct iovec *iovp = NULL;
    bpf_core_read(&iovp, sizeof(iovp), &iter.iov);
    if (!iovp) return -1;
    bpf_core_read(&iov0, sizeof(iov0), iovp);
    if (!iov0.iov_base || iov0.iov_len == 0) return -1;
    *base_out = iov0.iov_base;
    *len_out = iov0.iov_len;
    return 0;

    /* Path B: iter.ubuf (flat buffer) */
#elif BPF_CORE_FIELD_EXISTS(((struct iov_iter *)0)->ubuf)
    void *ubuf = NULL;
    bpf_core_read(&ubuf, sizeof(ubuf), &iter.ubuf);
    if (!ubuf) return -1;
    /* Use iter.count as available length */
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
    *len_out = kv0.iov_len;
    return 0;

#else
    /* feature detection disabled / unknown layout -> fail */
    (void)iter;
    return -1;
#endif
}

/* -----------------------
 * kprobe: tcp_sendmsg (outgoing)
 * -----------------------
 */
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

/* -----------------------
 * kprobe: tcp_recvmsg (entry) - stash scalar-only context keyed by socket cookie
 * -----------------------
 */
SEC("kprobe/tcp_recvmsg")
int BPF_KPROBE(kprobe_tcp_recvmsg_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags, int *addr_len)
{
    if (!sk) return 0;

    __u64 cookie = bpf_get_socket_cookie(sk);
    struct recv_ctx r = {};
    r.sk_cookie = cookie;
    r.ts_ns = bpf_ktime_get_ns();
    r.len = (len > UINT32_MAX) ? UINT32_MAX : (__u32)len;
    r.flags = (flags > INT32_MAX) ? INT32_MAX : (__u32)flags;

    bpf_map_update_elem(&recv_map, &cookie, &r, BPF_ANY);
    return 0;
}

/* -----------------------
 * kretprobe: tcp_recvmsg (exit) - get regs, recover sk/msg, lookup scalar ctx keyed by cookie
 * -----------------------
 */
SEC("kretprobe/tcp_recvmsg")
int BPF_KRETPROBE(kret_tcp_recvmsg)
{
    long ret = PT_REGS_RC(ctx);
    /* obtain first two args from regs: sk (arg1), msg (arg2) */
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct msghdr *msg = (struct msghdr *)PT_REGS_PARM2(ctx);

    if (!sk || !msg) {
        /* cleanup if possible: we could try to delete by cookie, but we need cookie */
        return 0;
    }

    __u64 cookie = bpf_get_socket_cookie(sk);
    struct recv_ctx *s = bpf_map_lookup_elem(&recv_map, &cookie);
    if (!s) return 0;

    if (ret > 0) {
        void *base = NULL;
        __u64 len = 0;
        if (iter_first_base_len_from_msg(msg, &base, &len) == 0 && base && len > 0) {
            __u64 want = len;
            if (want > (unsigned long)ret) want = ret;
            emit_sample_from_user(sk, base, want, /*is_recv=*/1);
        }
    }

    bpf_map_delete_elem(&recv_map, &cookie);
    return 0;
}
