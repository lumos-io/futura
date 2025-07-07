package kubelet

import (
	"time"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"github.com/opisvigilant/futura/watcher/utils"
)

func addMemoryMetrics(mb *metadata.MetricsBuilder, s *stats.MemoryStats, currentTime time.Time, r resources, nodeMemoryLimit float64) {
	if s == nil {
		return
	}

	mb.MemoryMetrics.SetCurrentTime(currentTime)
	mb.MemoryMetrics.SetAvailable(utils.PointerToUint64(s.AvailableBytes))
	mb.MemoryMetrics.SetUsage(utils.PointerToUint64(s.UsageBytes))
	mb.MemoryMetrics.SetRss(utils.PointerToUint64(s.RSSBytes))
	mb.MemoryMetrics.SetWorkingSet(utils.PointerToUint64(s.WorkingSetBytes))
	mb.MemoryMetrics.SetPageFaults(utils.PointerToUint64(s.PageFaults))
	mb.MemoryMetrics.SetMajorPageFaults(utils.PointerToUint64(s.MajorPageFaults))

	if s.UsageBytes != nil {
		if r.memoryLimit > 0 {
			mb.MemoryMetrics.SetLimitUtilization(float64(*s.UsageBytes) / float64(r.memoryLimit))
		}
		if r.memoryRequest > 0 {
			mb.MemoryMetrics.SetRequestUtilization(float64(*s.UsageBytes) / float64(r.memoryRequest))
		}
		if nodeMemoryLimit > 0 {
			mb.MemoryMetrics.SetNodeUtilization(float64(*s.UsageBytes) / nodeMemoryLimit)
		}
	}
}
