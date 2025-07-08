package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewNodeStatsBuilder(s stats.NodeStats) *pbst.NodeStats {
	nb := &pbst.NodeStats{
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
		nb.SystemContainers = make([]*pbst.ContainerStats, len(s.SystemContainers))
		for i, c := range s.SystemContainers {
			nb.SystemContainers[i] = NewContainerStatsBuilder(c)
		}
	}
	return nb
}
