package metadata

import (
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewContainerStatsBuilder(s stats.ContainerStats) *pb.ContainerStats {
	csb := &pb.ContainerStats{
		Name:      s.Name,
		StartTime: toProtoTime(s.StartTime.Time),
		Cpu:       NewCPUStatsBuilder(s.CPU),
		Io:        NewIOStatsBuilder(s.IO),
		Rootfs:    NewFsStatsBuilder(s.Rootfs),
		Logs:      NewFsStatsBuilder(s.Logs),
		Swap:      NewSwapStatsBuilder(s.Swap),
	}
	if len(s.Accelerators) > 0 {
		csb.Accelerators = make([]*pb.AcceleratorStats, len(s.Accelerators))
		for i, a := range s.Accelerators {
			csb.Accelerators[i] = NewAcceleratorStatsBuilder(a)
		}
	}
	if len(s.UserDefinedMetrics) > 0 {
		csb.UserDefinedMetrics = make([]*pb.UserDefinedMetric, len(s.UserDefinedMetrics))
		for i, u := range s.UserDefinedMetrics {
			csb.UserDefinedMetrics[i] = NewUserDefinedMetricBuilder(u)
		}
	}
	return csb
}

func NewAcceleratorStatsBuilder(a stats.AcceleratorStats) *pb.AcceleratorStats {
	return &pb.AcceleratorStats{
		Make:        a.Make,
		Model:       a.Model,
		Id:          a.ID,
		MemoryTotal: a.MemoryTotal,
		MemoryUsed:  a.MemoryUsed,
		DutyCycle:   a.DutyCycle,
	}
}

func NewUserDefinedMetricBuilder(u stats.UserDefinedMetric) *pb.UserDefinedMetric {
	return &pb.UserDefinedMetric{
		Descriptor_: NewUserDefinedMetricDescriptorBuilder(u.UserDefinedMetricDescriptor),
		Time:        toProtoTime(u.Time.Time),
		Value:       u.Value,
	}
}

func NewUserDefinedMetricDescriptorBuilder(u stats.UserDefinedMetricDescriptor) *pb.UserDefinedMetricDescriptor {
	return &pb.UserDefinedMetricDescriptor{
		Name:   u.Name,
		Type:   string(u.Type),
		Units:  u.Units,
		Labels: u.Labels,
	}
}
