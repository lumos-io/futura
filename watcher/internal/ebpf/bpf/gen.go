package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -target bpf -D__TARGET_ARCH_x86 -D__BPF_TRACING__"  L4 l4metrics.c -- -I.
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -target bpf -D__TARGET_ARCH_x86 -D__BPF_TRACING__" L7 l7probes.c -- -I.
