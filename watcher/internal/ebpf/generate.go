package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package uprobe -output-dir ./bpf/uprobe uprobe ./bpf/uprobe/uprobe.c -- -I./bpf/headers
