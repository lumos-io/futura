package metadata

import (
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/utils"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewCPUStatsBuilder(c *stats.CPUStats) *pb.CPUStats {
	return &pb.CPUStats{
		Time:                 toProtoTime(c.Time.Time),
		UsageNanoCores:       utils.PointerToUint64(c.UsageNanoCores),
		UsageCoreNanoSeconds: utils.PointerToUint64(c.UsageCoreNanoSeconds),
		Psi:                  NewPSIStatsBuilder(c.PSI),
	}
}

func NewMemoryStatsBuilder(m *stats.MemoryStats) *pb.MemoryStats {
	if m == nil {
		return &pb.MemoryStats{}
	}
	return &pb.MemoryStats{
		Time:            toProtoTime(m.Time.Time),
		AvailableBytes:  utils.PointerToUint64(m.AvailableBytes),
		UsageBytes:      utils.PointerToUint64(m.UsageBytes),
		WorkingSetBytes: utils.PointerToUint64(m.WorkingSetBytes),
		RssBytes:        utils.PointerToUint64(m.RSSBytes),
		PageFaults:      utils.PointerToUint64(m.PageFaults),
		MajorPageFaults: utils.PointerToUint64(m.MajorPageFaults),
		Psi:             NewPSIStatsBuilder(m.PSI),
	}
}

func NewIOStatsBuilder(m *stats.IOStats) *pb.IOStats {
	if m == nil {
		return &pb.IOStats{}
	}
	return &pb.IOStats{
		Time: toProtoTime(m.Time.Time),
		Psi:  NewPSIStatsBuilder(m.PSI),
	}
}

func NewNetworkStatsBuilder(m *stats.NetworkStats) *pb.NetworkStats {
	if m == nil {
		return &pb.NetworkStats{}
	}
	nb := &pb.NetworkStats{
		Time:           toProtoTime(m.Time.Time),
		InterfaceStats: NewInterfaceStatsBuilder(m.InterfaceStats),
	}
	if len(m.Interfaces) > 0 {
		nb.Interfaces = make([]*pb.InterfaceStats, len(m.Interfaces))
		for j, i := range m.Interfaces {
			nb.Interfaces[j] = NewInterfaceStatsBuilder(i)
		}
	}
	return nb
}

func NewInterfaceStatsBuilder(m stats.InterfaceStats) *pb.InterfaceStats {
	return &pb.InterfaceStats{
		Name:     m.Name,
		RxBytes:  utils.PointerToUint64(m.RxBytes),
		RxErrors: utils.PointerToUint64(m.RxErrors),
		TxBytes:  utils.PointerToUint64(m.TxBytes),
		TxErrors: utils.PointerToUint64(m.TxErrors),
	}
}

func NewFsStatsBuilder(m *stats.FsStats) *pb.FsStats {
	if m == nil {
		return &pb.FsStats{}
	}
	return &pb.FsStats{
		Time:           toProtoTime(m.Time.Time),
		AvailableBytes: utils.PointerToUint64(m.AvailableBytes),
		CapacityBytes:  utils.PointerToUint64(m.CapacityBytes),
		UsedBytes:      utils.PointerToUint64(m.UsedBytes),
		InodesFree:     utils.PointerToUint64(m.InodesFree),
		Inodes:         utils.PointerToUint64(m.Inodes),
		InodesUsed:     utils.PointerToUint64(m.InodesUsed),
	}
}

func NewRuntimeStatsBuilder(m *stats.RuntimeStats) *pb.RuntimeStats {
	if m == nil {
		return &pb.RuntimeStats{}
	}
	return &pb.RuntimeStats{
		ImageFs:     NewFsStatsBuilder(m.ImageFs),
		ContainerFs: NewFsStatsBuilder(m.ContainerFs),
	}
}

func NewRlimitStatsBuilder(m *stats.RlimitStats) *pb.RlimitStats {
	if m == nil {
		return &pb.RlimitStats{}
	}
	return &pb.RlimitStats{
		Time:                  toProtoTime(m.Time.Time),
		Maxpid:                utils.PointerToInt64(m.MaxPID),
		NumOfRunningProcesses: utils.PointerToInt64(m.NumOfRunningProcesses),
	}
}

func NewSwapStatsBuilder(m *stats.SwapStats) *pb.SwapStats {
	if m == nil {
		return &pb.SwapStats{}
	}
	return &pb.SwapStats{
		Time:               toProtoTime(m.Time.Time),
		SwapAvailableBytes: utils.PointerToUint64(m.SwapAvailableBytes),
		SwapUsageBytes:     utils.PointerToUint64(m.SwapUsageBytes),
	}
}

func NewPSIStatsBuilder(m *stats.PSIStats) *pb.PSIStats {
	if m == nil {
		return &pb.PSIStats{}
	}

	return &pb.PSIStats{
		Full: NewPSIDataBuilder(m.Full),
		Some: NewPSIDataBuilder(m.Some),
	}
}

func NewPSIDataBuilder(m stats.PSIData) *pb.PSIData {
	return &pb.PSIData{
		Total:  m.Total,
		Avg10:  m.Avg10,
		Avg60:  m.Avg60,
		Avg300: m.Avg300,
	}
}

func TransformVolumeStats(m *stats.VolumeStats) *pb.VolumeStats {
	if m == nil {
		return &pb.VolumeStats{}
	}
	vsb := &pb.VolumeStats{
		FsStats: NewFsStatsBuilder(&m.FsStats),
		Name:    m.Name,
	}
	if m.PVCRef != nil {
		vsb.PvcRef = &pb.PVCReference{
			Name:      m.PVCRef.Name,
			Namespace: m.PVCRef.Namespace,
		}
	}
	if m.VolumeHealthStats != nil {
		vsb.VolumeHealthStats = &pb.VolumeHealthStats{
			Abnormal: m.VolumeHealthStats.Abnormal,
		}
	}
	return vsb
}

func TransformProcessStats(m *stats.ProcessStats) *pb.ProcessStats {
	if m == nil {
		return &pb.ProcessStats{}
	}
	return &pb.ProcessStats{
		ProcessCount: utils.PointerToUint64(m.ProcessCount),
	}

}
