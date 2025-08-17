package ebpf

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type Stats struct {
	BytesSent   uint64
	SendCalls   uint64
	TotalSendNs uint64
	BytesRecv   uint64
	RecvCalls   uint64
	TotalRecvNs uint64
}

type EbpfCollector struct{}

func NewEbpfCollector() *EbpfCollector {
	return &EbpfCollector{}
}

func (e *EbpfCollector) Start() error {
	objPath := "bpf/tcp_monitor.o"

	spec, err := ebpf.LoadCollectionSpec(objPath)
	if err != nil {
		log.Fatalf("LoadCollectionSpec: %v", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Fatalf("NewCollection: %v", err)
	}
	defer coll.Close()

	progSend := coll.Programs["kprobe__tcp_sendmsg"]
	kp, err := link.Kprobe("tcp_sendmsg", progSend, nil)
	if err != nil {
		log.Fatalf("link.Kprobe send: %v", err)
	}
	defer kp.Close()

	progSendRet := coll.Programs["kretprobe__tcp_sendmsg"]
	krp, err := link.Kretprobe("tcp_sendmsg", progSendRet, nil)
	if err != nil {
		log.Fatalf("link.Kretprobe send: %v", err)
	}
	defer krp.Close()

	statsMap := coll.Maps["stats_map"]
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		iter := statsMap.Iterate()
		var key uint64
		var val Stats
		for iter.Next(&key, &val) {
			cgroupPath := lookupCgroupPathByInode(key)
			fmt.Printf("cgroup_inode=%d path=%s bytes_sent=%d calls=%d avg_send_ms=%.2f\n", key, cgroupPath, val.BytesSent, val.SendCalls, float64(val.TotalSendNs)/1e6/float64(max(1, int(val.SendCalls))))
		}
	}

	return nil
}

func lookupCgroupPathByInode(inode uint64) string {
	var res string
	filepath.WalkDir("/sys/fs/cgroup", func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		fi, err := os.Stat(path)
		if err != nil {
			return nil
		}
		st := fi.Sys().(*syscall.Stat_t)
		if uint64(st.Ino) == inode {
			res = path
			return filepath.SkipDir
		}
		return nil
	})
	return res
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
