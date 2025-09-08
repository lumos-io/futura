package metadata

import (
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewNodeStatsBuilder(s stats.NodeStats) *pb.NodeStats {
	nb := &pb.NodeStats{
		NodeName:  s.NodeName,
		StartTime: toProtoTime(s.StartTime.Time),
		Cpu:       NewCPUStatsBuilder(s.CPU),
		Memory:    NewMemoryStatsBuilder(s.Memory),
		Io:        NewIOStatsBuilder(s.IO),
		Network:   NewNetworkStatsBuilder(s.Network),
		Fs:        NewFsStatsBuilder(s.Fs),
		Runtime:   NewRuntimeStatsBuilder(s.Runtime),
		Rlimit:    NewRlimitStatsBuilder(s.Rlimit),
		Swap:      NewSwapStatsBuilder(s.Swap),
	}
	if len(s.SystemContainers) > 0 {
		nb.SystemContainers = make([]*pb.ContainerStats, len(s.SystemContainers))
		for i, c := range s.SystemContainers {
			nb.SystemContainers[i] = NewContainerStatsBuilder(c)
		}
	}
	return nb
}
