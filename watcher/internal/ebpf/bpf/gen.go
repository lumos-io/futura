package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go TcpMonitor ../internal/ebpf/bpf/tcp_monitor.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go UProbes ../internal/ebpf/bpf/uprobes_go.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go Latency ../internal/ebpf/bpf/latency.c
