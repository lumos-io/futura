package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package uprobe -target bpf -output-dir ./bpf/uprobe uprobe ./bpf/uprobe/uprobe.c -- -I./bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package rps -target bpf -output-dir ./bpf/rps rps ./bpf/rps/rps.bpf.c -- -I./bpf/headers
