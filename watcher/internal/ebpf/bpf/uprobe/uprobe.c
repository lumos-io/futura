//go:build ignore
// +build ignore
// file: uprobe.c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, u32);   // PID
    __type(value, u64); // Request count
} rps_count SEC(".maps");

// Uprobe attached to HTTP handler
SEC("uprobe/handle_request")
int handle_request(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 zero = 0, *val;

    val = bpf_map_lookup_elem(&rps_count, &pid);
    if (val) {
        __sync_fetch_and_add(val, 1);
    } else {
        bpf_map_update_elem(&rps_count, &pid, &zero, BPF_NOEXIST);
        val = bpf_map_lookup_elem(&rps_count, &pid);
        if (val) __sync_fetch_and_add(val, 1);
    }
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
