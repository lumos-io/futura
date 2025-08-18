package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target bpf TcpMonitor tcp_monitor.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target bpf UProbes uprobes_go.c
