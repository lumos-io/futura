package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewPodStatsBuilder(p stats.PodStats) *pbst.PodStats {
	containers := make([]*pbst.ContainerStats, len(p.Containers))
	for i, containerStats := range p.Containers {
		// propagate the pod resource down to the container
		containers[i] = NewContainerStatsBuilder(containerStats)
	}
	var volumes []*pbst.VolumeStats
	if len(p.VolumeStats) > 0 {
		volumes = make([]*pbst.VolumeStats, len(p.VolumeStats))
		for i, volumeStats := range p.VolumeStats {
			// propagate the pod resource down to the container
			volumes[i] = TransformVolumeStats(&volumeStats)
		}
	}
	psb := &pbst.PodStats{
		PodRef: &pbst.PodReference{
			Name:      p.PodRef.Name,
			Namespace: p.PodRef.Namespace,
			Uid:       p.PodRef.UID,
		},
		StartTime:        toProtoTime(p.StartTime.Time),
		Cpu:              NewCPUStatsBuilder(p.CPU),
		Memory:           NewMemoryStatsBuilder(p.Memory),
		Io:               NewIOStatsBuilder(p.IO),
		Network:          NewNetworkStatsBuilder(p.Network),
		EphemeralStorage: NewFsStatsBuilder(p.EphemeralStorage),
		ProcessStats:     TransformProcessStats(p.ProcessStats),
		Swap:             NewSwapStatsBuilder(p.Swap),
		Containers:       containers,
		Volumes:          volumes,
	}

	return psb
}
