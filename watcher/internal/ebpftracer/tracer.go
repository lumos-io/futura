package ebpftracer

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go bpf ./ebpf/ebpf.c -- -Iheaders

const mapKey uint32 = 0

func Tracer() {

	// Name of the kernel function to trace.
	fn := "sys_execve"

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %v", err)
	}
	defer objs.Close()

	// Open a Kprobe at the entry point of the kernel function and attach the
	// pre-compiled program. Each time the kernel function enters, the program
	// will increment the execution counter by 1. The read loop below polls this
	// map value once per second.
	kp, err := link.Kprobe(fn, objs.KprobeExecve, nil)
	if err != nil {
		log.Fatalf("opening kprobe: %s", err)
	}
	defer kp.Close()

	// Get the first-mounted cgroupv2 path.
	cgroupPath, err := detectCgroupPath()
	if err != nil {
		log.Fatal(err)
	}

	// Link the count_egress_packets program to the cgroup.
	l, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetEgress,
		Program: objs.CountEgressPackets,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// Read loop reporting the total amount of times the kernel
	// function was entered, once per second.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Waiting for events..")

	for range ticker.C {
		{
			var all_cpu_value []uint64
			if err := objs.KprobeMap.Lookup(mapKey, &all_cpu_value); err != nil {
				log.Fatalf("reading map: %v", err)
			}
			for cpuid, cpuvalue := range all_cpu_value {
				log.Printf("%s called %d times on CPU%v\n", fn, cpuvalue, cpuid)
			}
		}
		{
			var value uint64
			if err := objs.PktCount.Lookup(uint32(0), &value); err != nil {
				log.Fatalf("reading map: %v", err)
			}
			log.Printf("number of packets: %d\n", value)
		}
	}
}

// detectCgroupPath returns the first-found mount point of type cgroup2
// and stores it in the cgroupPath global variable.
func detectCgroupPath() (string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// example fields: cgroup2 /sys/fs/cgroup/unified cgroup2 rw,nosuid,nodev,noexec,relatime 0 0
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) >= 3 && fields[2] == "cgroup2" {
			return fields[1], nil
		}
	}

	return "", errors.New("cgroup2 not mounted")
}
