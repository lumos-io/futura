package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

func StreamKubeletStats(ctx context.Context, wg *sync.WaitGroup, nodeName string, pods []*pb.PodStats, out chan<- *pb.KubernetesKubeletStats) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Occasionally remove a pod
			if rand.Intn(10) == 0 && len(pods) > 3 {
				pods = append(pods[:rand.Intn(len(pods))], pods[rand.Intn(len(pods))+1:]...)
			}

			// Occasionally add a new pod
			if rand.Intn(10) == 0 {
				pods = append(pods, newRandomPod())
			}

			// Produce stats with the current pod set
			stats := generateKubeletStats(nodeName)
			stats.Pods = pods
			out <- stats

			time.Sleep(50 * time.Millisecond) // adjust as needed
		}
	}
}

func newRandomPod() *pb.PodStats {
	return &pb.PodStats{
		PodRef: &pb.PodReference{
			Name:      fmt.Sprintf("pod-%s", uuid.NewString()[:8]),
			Namespace: randomChoice([]string{"default", "kube-system", "prod", "staging"}),
			Uid:       uuid.NewString(),
		},
		StartTime:        timestamppb.New(time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second)),
		Containers:       []*pb.ContainerStats{randomContainer("app"), randomContainer("sidecar")},
		Cpu:              randomCPU(),
		Memory:           randomMemory(),
		Io:               randomIO(),
		Network:          randomNetwork(),
		Volumes:          []*pb.VolumeStats{{FsStats: randomFs(), Name: "data", PvcRef: &pb.PVCReference{Name: "pvc1", Namespace: "default"}, VolumeHealthStats: &pb.VolumeHealthStats{Abnormal: rand.Intn(2) == 0}}},
		EphemeralStorage: randomFs(),
		ProcessStats:     &pb.ProcessStats{ProcessCount: uint64(rand.Intn(100) + 1)},
		Swap:             randomSwap(),
	}
}

func generateKubeletStats(nodeName string) *pb.KubernetesKubeletStats {
	now := time.Now()
	return &pb.KubernetesKubeletStats{
		Node: &pb.NodeStats{
			NodeName:         nodeName,
			SystemContainers: []*pb.ContainerStats{randomContainer("kubelet"), randomContainer("runtime")},
			StartTime:        timestamppb.New(now.Add(-time.Hour)),
			Cpu:              randomCPU(),
			Memory:           randomMemory(),
			Io:               randomIO(),
			Network:          randomNetwork(),
			Fs:               randomFs(),
			Runtime:          &pb.RuntimeStats{ImageFs: randomFs(), ContainerFs: randomFs()},
			Rlimit:           &pb.RlimitStats{Time: timestamppb.New(now), Maxpid: 32768, NumOfRunningProcesses: int64(rand.Intn(300) + 50)},
			Swap:             randomSwap(),
		},
		Pods:       RandomPods(),
		Enrichment: &pb.EnrichmentMetadata{OrganizationId: 42, ClusterId: 12345, ReceivedAtUnix: now.Unix()},
	}
}

// ----------- Helpers ------------

func RandomPods() []*pb.PodStats {
	podCount := rand.Intn(10) + 5
	pods := make([]*pb.PodStats, podCount)
	for i := range podCount {
		ts := time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second)
		pods[i] = &pb.PodStats{
			PodRef: &pb.PodReference{
				Name:      fmt.Sprintf("pod-%s", uuid.NewString()[:8]),
				Namespace: randomChoice([]string{"default", "kube-system", "prod", "staging"}),
				Uid:       uuid.NewString(),
			},
			StartTime:        timestamppb.New(ts),
			Containers:       []*pb.ContainerStats{randomContainer("app"), randomContainer("sidecar")},
			Cpu:              randomCPU(),
			Memory:           randomMemory(),
			Io:               randomIO(),
			Network:          randomNetwork(),
			Volumes:          []*pb.VolumeStats{{FsStats: randomFs(), Name: "data", PvcRef: &pb.PVCReference{Name: "pvc1", Namespace: "default"}, VolumeHealthStats: &pb.VolumeHealthStats{Abnormal: rand.Intn(2) == 0}}},
			EphemeralStorage: randomFs(),
			ProcessStats:     &pb.ProcessStats{ProcessCount: uint64(rand.Intn(100) + 1)},
			Swap:             randomSwap(),
		}
	}
	return pods
}

func randomContainer(name string) *pb.ContainerStats {
	ts := time.Now().Add(-time.Duration(rand.Intn(7200)) * time.Second)
	return &pb.ContainerStats{
		Name:         name,
		StartTime:    timestamppb.New(ts),
		Cpu:          randomCPU(),
		Memory:       randomMemory(),
		Io:           randomIO(),
		Accelerators: []*pb.AcceleratorStats{{Make: "NVIDIA", Model: "A100", Id: uuid.NewString(), MemoryTotal: 40 * 1024 * 1024 * 1024, MemoryUsed: uint64(rand.Intn(40)) * 1024 * 1024 * 1024, DutyCycle: uint64(rand.Intn(100))}},
		Rootfs:       randomFs(),
		Logs:         randomFs(),
		Swap:         randomSwap(),
	}
}

func randomCPU() *pb.CPUStats {
	return &pb.CPUStats{
		Time:                 timestamppb.Now(),
		UsageNanoCores:       uint64(rand.Intn(2000) * 1e6),
		UsageCoreNanoSeconds: uint64(rand.Intn(1e12)),
	}
}

func randomMemory() *pb.MemoryStats {
	return &pb.MemoryStats{
		Time:            timestamppb.Now(),
		AvailableBytes:  uint64(rand.Intn(8*1024) * 1024 * 1024),
		UsageBytes:      uint64(rand.Intn(8*1024) * 1024 * 1024),
		WorkingSetBytes: uint64(rand.Intn(8*1024) * 1024 * 1024),
		RssBytes:        uint64(rand.Intn(8*1024) * 1024 * 1024),
		PageFaults:      uint64(rand.Intn(10000)),
		MajorPageFaults: uint64(rand.Intn(100)),
	}
}

func randomIO() *pb.IOStats {
	return &pb.IOStats{Time: timestamppb.Now()}
}

func randomNetwork() *pb.NetworkStats {
	return &pb.NetworkStats{
		Time: timestamppb.Now(),
		InterfaceStats: &pb.InterfaceStats{
			Name:     "eth0",
			RxBytes:  uint64(rand.Intn(1e9)),
			RxErrors: uint64(rand.Intn(100)),
			TxBytes:  uint64(rand.Intn(1e9)),
			TxErrors: uint64(rand.Intn(100)),
		},
		Interfaces: []*pb.InterfaceStats{
			{Name: "eth0", RxBytes: uint64(rand.Intn(1e9)), TxBytes: uint64(rand.Intn(1e9))},
			{Name: "eth1", RxBytes: uint64(rand.Intn(1e9)), TxBytes: uint64(rand.Intn(1e9))},
		},
	}
}

func randomFs() *pb.FsStats {
	return &pb.FsStats{
		Time:           timestamppb.Now(),
		AvailableBytes: uint64(rand.Intn(1000) * 1024 * 1024),
		CapacityBytes:  uint64(rand.Intn(2000) * 1024 * 1024),
		UsedBytes:      uint64(rand.Intn(2000) * 1024 * 1024),
		InodesFree:     uint64(rand.Intn(100000)),
		Inodes:         uint64(rand.Intn(200000)),
		InodesUsed:     uint64(rand.Intn(200000)),
	}
}

func randomSwap() *pb.SwapStats {
	return &pb.SwapStats{
		Time:               timestamppb.Now(),
		SwapAvailableBytes: uint64(rand.Intn(4*1024) * 1024 * 1024),
		SwapUsageBytes:     uint64(rand.Intn(4*1024) * 1024 * 1024),
	}
}
