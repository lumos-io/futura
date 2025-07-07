package kubelet

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/opisvigilant/futura/watcher/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

type Accumulator struct {
	NodeStats *pbst.NodeStats
	PodStats  []*PodInfo
}

type PodInfo struct {
	PostStats      *pbst.PodStats
	ContainerStats []*pbst.ContainerStats
	VolumeStats    []*pbst.VolumeStats
}

func MetricsData(summary *stats.Summary, metadata Metadata) *Accumulator {
	pods := make([]*PodInfo, len(summary.Pods))
	for _, podStats := range summary.Pods {
		containers := make([]*pbst.ContainerStats, len(podStats.Containers))
		for _, containerStats := range podStats.Containers {
			// propagate the pod resource down to the container
			cs := ContainerStats(podStats, containerStats)
			containers = append(containers, cs)
		}
		var volumes []*pbst.VolumeStats
		if len(podStats.VolumeStats) > 0 {
			volumes = make([]*pbst.VolumeStats, len(podStats.VolumeStats))
			for _, volumeStats := range podStats.VolumeStats {
				// propagate the pod resource down to the container
				vs := VolumeStats(podStats, volumeStats)
				volumes = append(volumes, vs)
			}
		}
		pods = append(pods, &PodInfo{
			PostStats:      PodStats(podStats),
			ContainerStats: containers,
			VolumeStats:    volumes,
		})
	}

	return &Accumulator{
		NodeStats: NodeStats(summary.Node),
		PodStats:  pods,
	}
}

func NodeStats(s stats.NodeStats) *pbst.NodeStats {
	ns := &pbst.NodeStats{
		NodeName:  s.NodeName,
		StartTime: timestamppb.New(s.StartTime.Time),
		Cpu: &pbst.CPUStats{
			Time:                 timestamppb.New(s.CPU.Time.Time),
			UsageNanoCores:       utils.PointerToUint64(s.CPU.UsageNanoCores),
			UsageCoreNanoSeconds: utils.PointerToUint64(s.CPU.UsageCoreNanoSeconds),
			Psi: &pbst.PSIStats{
				Full: &pbst.PSIData{
					Total:  s.CPU.PSI.Full.Total,
					Avg10:  s.CPU.PSI.Full.Avg10,
					Avg60:  s.CPU.PSI.Full.Avg60,
					Avg300: s.CPU.PSI.Full.Avg300,
				},
				Some: &pbst.PSIData{
					Total:  s.CPU.PSI.Some.Total,
					Avg10:  s.CPU.PSI.Some.Avg10,
					Avg60:  s.CPU.PSI.Some.Avg60,
					Avg300: s.CPU.PSI.Some.Avg300,
				},
			},
		},
		Memory: &pbst.MemoryStats{
			Time:            timestamppb.New(s.Memory.Time.Time),
			AvailableBytes:  utils.PointerToUint64(s.Memory.AvailableBytes),
			UsageBytes:      utils.PointerToUint64(s.Memory.UsageBytes),
			WorkingSetBytes: utils.PointerToUint64(s.Memory.WorkingSetBytes),
			RssBytes:        utils.PointerToUint64(s.Memory.RSSBytes),
			PageFaults:      utils.PointerToUint64(s.Memory.PageFaults),
			MajorPageFaults: utils.PointerToUint64(s.Memory.MajorPageFaults),
			Psi: &pbst.PSIStats{
				Full: &pbst.PSIData{
					Total:  s.Memory.PSI.Full.Total,
					Avg10:  s.Memory.PSI.Full.Avg10,
					Avg60:  s.Memory.PSI.Full.Avg60,
					Avg300: s.Memory.PSI.Full.Avg300,
				},
				Some: &pbst.PSIData{
					Total:  s.Memory.PSI.Some.Total,
					Avg10:  s.Memory.PSI.Some.Avg10,
					Avg60:  s.Memory.PSI.Some.Avg60,
					Avg300: s.Memory.PSI.Some.Avg300,
				},
			},
		},
		Io: &pbst.IOStats{
			Time: timestamppb.New(s.IO.Time.Time),
			Psi: &pbst.PSIStats{
				Full: &pbst.PSIData{
					Total:  s.IO.PSI.Full.Total,
					Avg10:  s.IO.PSI.Full.Avg10,
					Avg60:  s.IO.PSI.Full.Avg60,
					Avg300: s.IO.PSI.Full.Avg300,
				},
				Some: &pbst.PSIData{
					Total:  s.IO.PSI.Some.Total,
					Avg10:  s.IO.PSI.Some.Avg10,
					Avg60:  s.IO.PSI.Some.Avg60,
					Avg300: s.IO.PSI.Some.Avg300,
				},
			},
		},
		Network: &pbst.NetworkStats{
			Time: timestamppb.New(s.IO.Time.Time),
			// the interface in use
			InterfaceStats: &pbst.InterfaceStats{
				Name:     s.Network.Name,
				RxBytes:  utils.PointerToUint64(s.Network.RxBytes),
				TxBytes:  utils.PointerToUint64(s.Network.TxBytes),
				RxErrors: utils.PointerToUint64(s.Network.RxErrors),
				TxErrors: utils.PointerToUint64(s.Network.TxErrors),
			},
		},
		Fs: &pbst.FsStats{
			Time:           timestamppb.New(s.IO.Time.Time),
			AvailableBytes: utils.PointerToUint64(s.Fs.AvailableBytes),
			CapacityBytes:  utils.PointerToUint64(s.Fs.CapacityBytes),
			UsedBytes:      utils.PointerToUint64(s.Fs.UsedBytes),
			InodesFree:     utils.PointerToUint64(s.Fs.InodesFree),
			Inodes:         utils.PointerToUint64(s.Fs.Inodes),
			InodesUsed:     utils.PointerToUint64(s.Fs.InodesUsed),
		},
		Runtime: &pbst.RuntimeStats{
			ImageFs: &pbst.FsStats{
				Time:           timestamppb.New(s.IO.Time.Time),
				AvailableBytes: utils.PointerToUint64(s.Runtime.ImageFs.AvailableBytes),
				CapacityBytes:  utils.PointerToUint64(s.Runtime.ImageFs.CapacityBytes),
				UsedBytes:      utils.PointerToUint64(s.Runtime.ImageFs.UsedBytes),
				InodesFree:     utils.PointerToUint64(s.Runtime.ImageFs.InodesFree),
				Inodes:         utils.PointerToUint64(s.Runtime.ImageFs.Inodes),
				InodesUsed:     utils.PointerToUint64(s.Runtime.ImageFs.InodesUsed),
			},
			ContainerFs: &pbst.FsStats{
				Time:           timestamppb.New(s.IO.Time.Time),
				AvailableBytes: utils.PointerToUint64(s.Runtime.ContainerFs.AvailableBytes),
				CapacityBytes:  utils.PointerToUint64(s.Runtime.ContainerFs.CapacityBytes),
				UsedBytes:      utils.PointerToUint64(s.Runtime.ContainerFs.UsedBytes),
				InodesFree:     utils.PointerToUint64(s.Runtime.ContainerFs.InodesFree),
				Inodes:         utils.PointerToUint64(s.Runtime.ContainerFs.Inodes),
				InodesUsed:     utils.PointerToUint64(s.Runtime.ContainerFs.InodesUsed),
			},
		},
		Rlimit: &pbst.RlimitStats{
			Time:                  timestamppb.New(s.IO.Time.Time),
			Maxpid:                utils.PointerToInt64(s.Rlimit.MaxPID),
			NumOfRunningProcesses: utils.PointerToInt64(s.Rlimit.NumOfRunningProcesses),
		},
		Swap: &pbst.SwapStats{
			Time:               timestamppb.New(s.IO.Time.Time),
			SwapAvailableBytes: utils.PointerToUint64(s.Swap.SwapAvailableBytes),
			SwapUsageBytes:     utils.PointerToUint64(s.Swap.SwapUsageBytes),
		},
	}

	if len(s.Network.Interfaces) > 0 {
		is := make([]*pbst.InterfaceStats, len(s.Network.Interfaces))
		for _, i := range s.Network.Interfaces {
			is = append(is, &pbst.InterfaceStats{
				Name:     i.Name,
				RxBytes:  utils.PointerToUint64(i.RxBytes),
				TxBytes:  utils.PointerToUint64(i.TxBytes),
				RxErrors: utils.PointerToUint64(i.RxErrors),
				TxErrors: utils.PointerToUint64(i.TxErrors),
			})
		}
		ns.Network.Interfaces = is
	}

	if len(s.SystemContainers) > 0 {
		sc := make([]*pbst.ContainerStats, len(s.SystemContainers))
		for _, cs := range s.SystemContainers {
			sc = append(sc, &pbst.ContainerStats{
				Name:      cs.Name,
				StartTime: timestamppb.New(cs.StartTime.Time),
				Cpu: &pbst.CPUStats{
					Time:                 timestamppb.New(s.CPU.Time.Time),
					UsageNanoCores:       *cs.CPU.UsageNanoCores,
					UsageCoreNanoSeconds: *cs.CPU.UsageCoreNanoSeconds,
					Psi: &pbst.PSIStats{
						Full: &pbst.PSIData{
							Total:  cs.CPU.PSI.Full.Total,
							Avg10:  cs.CPU.PSI.Full.Avg10,
							Avg60:  cs.CPU.PSI.Full.Avg60,
							Avg300: cs.CPU.PSI.Full.Avg300,
						},
						Some: &pbst.PSIData{
							Total:  cs.CPU.PSI.Some.Total,
							Avg10:  cs.CPU.PSI.Some.Avg10,
							Avg60:  cs.CPU.PSI.Some.Avg60,
							Avg300: cs.CPU.PSI.Some.Avg300,
						},
					},
				},
				Memory: &pbst.MemoryStats{},
				Io:     &pbst.IOStats{},
				// Accelerators: ,
				Rootfs: &pbst.FsStats{},
				Logs:   &pbst.FsStats{},
				// UserDefinedMetrics: ,
				Swap: &pbst.SwapStats{},
			})
		}
		ns.SystemContainers = sc
	}

	return ns
}

func PodStats(s stats.PodStats) *pbst.PodStats {
	return &pbst.PodStats{}
}

func ContainerStats(sPod stats.PodStats, s stats.ContainerStats) *pbst.ContainerStats {
	return &pbst.ContainerStats{}
}

func VolumeStats(sPod stats.PodStats, s stats.VolumeStats) *pbst.VolumeStats {
	return &pbst.VolumeStats{}
}
