package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/opisvigilant/futura/watcher/utils"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func NewCPUStatsBuilder(c *stats.CPUStats) *pbst.CPUStats {
	return &pbst.CPUStats{
		Time:                 toProtoTime(c.Time.Time),
		UsageNanoCores:       utils.PointerToUint64(c.UsageNanoCores),
		UsageCoreNanoSeconds: utils.PointerToUint64(c.UsageCoreNanoSeconds),
		Psi:                  NewPSIStatsBuilder(c.PSI),
	}
}

func NewMemoryStatsBuilder(m *stats.MemoryStats) *pbst.MemoryStats {
	if m == nil {
		return &pbst.MemoryStats{}
	}
	return &pbst.MemoryStats{
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

func NewIOStatsBuilder(m *stats.IOStats) *pbst.IOStats {
	if m == nil {
		return &pbst.IOStats{}
	}
	return &pbst.IOStats{
		Time: toProtoTime(m.Time.Time),
		Psi:  NewPSIStatsBuilder(m.PSI),
	}
}

func NewNetworkStatsBuilder(m *stats.NetworkStats) *pbst.NetworkStats {
	if m == nil {
		return &pbst.NetworkStats{}
	}
	nb := &pbst.NetworkStats{
		Time:           toProtoTime(m.Time.Time),
		InterfaceStats: NewInterfaceStatsBuilder(m.InterfaceStats),
	}
	if len(m.Interfaces) > 0 {
		nb.Interfaces = make([]*pbst.InterfaceStats, len(m.Interfaces))
		for j, i := range m.Interfaces {
			nb.Interfaces[j] = NewInterfaceStatsBuilder(i)
		}
	}
	return nb
}

func NewInterfaceStatsBuilder(m stats.InterfaceStats) *pbst.InterfaceStats {
	return &pbst.InterfaceStats{
		Name:     m.Name,
		RxBytes:  utils.PointerToUint64(m.RxBytes),
		RxErrors: utils.PointerToUint64(m.RxErrors),
		TxBytes:  utils.PointerToUint64(m.TxBytes),
		TxErrors: utils.PointerToUint64(m.TxErrors),
	}
}

func NewFsStatsBuilder(m *stats.FsStats) *pbst.FsStats {
	if m == nil {
		return &pbst.FsStats{}
	}
	return &pbst.FsStats{
		Time:           toProtoTime(m.Time.Time),
		AvailableBytes: utils.PointerToUint64(m.AvailableBytes),
		CapacityBytes:  utils.PointerToUint64(m.CapacityBytes),
		UsedBytes:      utils.PointerToUint64(m.UsedBytes),
		InodesFree:     utils.PointerToUint64(m.InodesFree),
		Inodes:         utils.PointerToUint64(m.Inodes),
		InodesUsed:     utils.PointerToUint64(m.InodesUsed),
	}
}

func NewRuntimeStatsBuilder(m *stats.RuntimeStats) *pbst.RuntimeStats {
	if m == nil {
		return &pbst.RuntimeStats{}
	}
	return &pbst.RuntimeStats{
		ImageFs:     NewFsStatsBuilder(m.ImageFs),
		ContainerFs: NewFsStatsBuilder(m.ContainerFs),
	}
}

func NewRlimitStatsBuilder(m *stats.RlimitStats) *pbst.RlimitStats {
	if m == nil {
		return &pbst.RlimitStats{}
	}
	return &pbst.RlimitStats{
		Time:                  toProtoTime(m.Time.Time),
		Maxpid:                utils.PointerToInt64(m.MaxPID),
		NumOfRunningProcesses: utils.PointerToInt64(m.NumOfRunningProcesses),
	}
}

func NewSwapStatsBuilder(m *stats.SwapStats) *pbst.SwapStats {
	if m == nil {
		return &pbst.SwapStats{}
	}
	return &pbst.SwapStats{
		Time:               toProtoTime(m.Time.Time),
		SwapAvailableBytes: utils.PointerToUint64(m.SwapAvailableBytes),
		SwapUsageBytes:     utils.PointerToUint64(m.SwapUsageBytes),
	}
}

func NewPSIStatsBuilder(m *stats.PSIStats) *pbst.PSIStats {
	if m == nil {
		return &pbst.PSIStats{}
	}

	return &pbst.PSIStats{
		Full: NewPSIDataBuilder(m.Full),
		Some: NewPSIDataBuilder(m.Some),
	}
}

func NewPSIDataBuilder(m stats.PSIData) *pbst.PSIData {
	return &pbst.PSIData{
		Total:  m.Total,
		Avg10:  m.Avg10,
		Avg60:  m.Avg60,
		Avg300: m.Avg300,
	}
}

func TransformVolumeStats(m *stats.VolumeStats) *pbst.VolumeStats {
	if m == nil {
		return &pbst.VolumeStats{}
	}
	vsb := &pbst.VolumeStats{
		FsStats: NewFsStatsBuilder(&m.FsStats),
		Name:    m.Name,
	}
	if m.PVCRef != nil {
		vsb.PvcRef = &pbst.PVCReference{
			Name:      m.PVCRef.Name,
			Namespace: m.PVCRef.Namespace,
		}
	}
	if m.VolumeHealthStats != nil {
		vsb.VolumeHealthStats = &pbst.VolumeHealthStats{
			Abnormal: m.VolumeHealthStats.Abnormal,
		}
	}
	return vsb
}

func TransformProcessStats(m *stats.ProcessStats) *pbst.ProcessStats {
	if m == nil {
		return &pbst.ProcessStats{}
	}
	return &pbst.ProcessStats{
		ProcessCount: utils.PointerToUint64(m.ProcessCount),
	}

}
