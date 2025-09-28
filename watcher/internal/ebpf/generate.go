package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package uprobe -target bpf -output-dir ./bpf/uprobe uprobe ./bpf/uprobe/uprobe.c -- -I./bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package rps -target bpf -output-dir ./bpf/rps rps ./bpf/rps/rps.bpf.c -- -I./bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package http_metrics -target bpf -output-dir ./bpf/http_metrics http_metrics ./bpf/http_metrics/http_metrics.bpf.c -- -I./bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package memory_tracker -target bpf -output-dir ./bpf/memory_tracker memory_tracker ./bpf/memory_tracker/memory_tracker.bpf.c -- -I./bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux -go-package cpu_tracker -target bpf -output-dir ./bpf/cpu_tracker cpu_tracker ./bpf/cpu_tracker/cpu_tracker.bpf.c -- -I./bpf/headers
