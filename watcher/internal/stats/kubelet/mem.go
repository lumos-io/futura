package kubelet

import (
	"time"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
)

func addMemoryMetrics(mb *metadata.NodeMetricsBuilder, memoryMetrics metadata.MemoryMetrics, s *stats.MemoryStats, currentTime time.Time, r resources, nodeMemoryLimit float64) {
	if s == nil {
		return
	}

	recordIntDataPoint(mb, memoryMetrics.Available, s.AvailableBytes, currentTime)
	recordIntDataPoint(mb, memoryMetrics.Usage, s.UsageBytes, currentTime)
	recordIntDataPoint(mb, memoryMetrics.Rss, s.RSSBytes, currentTime)
	recordIntDataPoint(mb, memoryMetrics.WorkingSet, s.WorkingSetBytes, currentTime)
	recordIntDataPoint(mb, memoryMetrics.PageFaults, s.PageFaults, currentTime)
	recordIntDataPoint(mb, memoryMetrics.MajorPageFaults, s.MajorPageFaults, currentTime)

	if s.UsageBytes != nil {
		if r.memoryLimit > 0 {
			memoryMetrics.LimitUtilization(mb, currentTime, float64(*s.UsageBytes)/float64(r.memoryLimit))
		}
		if r.memoryRequest > 0 {
			memoryMetrics.RequestUtilization(mb, currentTime, float64(*s.UsageBytes)/float64(r.memoryRequest))
		}
		if nodeMemoryLimit > 0 {
			memoryMetrics.NodeUtilization(mb, currentTime, float64(*s.UsageBytes)/nodeMemoryLimit)
		}
	}
}
