package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewContainerStatsBuilder(s stats.ContainerStats) *pbst.ContainerStats {
	csb := &pbst.ContainerStats{
		Name:      s.Name,
		StartTime: toProtoTime(s.StartTime.Time),
		Cpu:       NewCPUStatsBuilder(s.CPU),
		Io:        NewIOStatsBuilder(s.IO),
		Rootfs:    NewFsStatsBuilder(s.Rootfs),
		Logs:      NewFsStatsBuilder(s.Logs),
		Swap:      NewSwapStatsBuilder(s.Swap),
	}
	if len(s.Accelerators) > 0 {
		csb.Accelerators = make([]*pbst.AcceleratorStats, len(s.Accelerators))
		for i, a := range s.Accelerators {
			csb.Accelerators[i] = NewAcceleratorStatsBuilder(a)
		}
	}
	if len(s.UserDefinedMetrics) > 0 {
		csb.UserDefinedMetrics = make([]*pbst.UserDefinedMetric, len(s.UserDefinedMetrics))
		for i, u := range s.UserDefinedMetrics {
			csb.UserDefinedMetrics[i] = NewUserDefinedMetricBuilder(u)
		}
	}
	return csb
}

func NewAcceleratorStatsBuilder(a stats.AcceleratorStats) *pbst.AcceleratorStats {
	return &pbst.AcceleratorStats{
		Make:        a.Make,
		Model:       a.Model,
		Id:          a.ID,
		MemoryTotal: a.MemoryTotal,
		MemoryUsed:  a.MemoryUsed,
		DutyCycle:   a.DutyCycle,
	}
}

func NewUserDefinedMetricBuilder(u stats.UserDefinedMetric) *pbst.UserDefinedMetric {
	return &pbst.UserDefinedMetric{
		Descriptor_: NewUserDefinedMetricDescriptorBuilder(u.UserDefinedMetricDescriptor),
		Time:        toProtoTime(u.Time.Time),
		Value:       u.Value,
	}
}

func NewUserDefinedMetricDescriptorBuilder(u stats.UserDefinedMetricDescriptor) *pbst.UserDefinedMetricDescriptor {
	return &pbst.UserDefinedMetricDescriptor{
		Name:   u.Name,
		Type:   string(u.Type),
		Units:  u.Units,
		Labels: u.Labels,
	}
}
