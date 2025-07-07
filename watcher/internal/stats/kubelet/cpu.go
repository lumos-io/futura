package kubelet

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func addCPUMetrics(mb *metadata.MetricsBuilder, s *stats.CPUStats, currentTime time.Time, r resources, nodeCPULimit float64) {
	if s == nil {
		return
	}
	if s.UsageNanoCores != nil {
		usageCores := float64(*s.UsageNanoCores) / 1_000_000_000
		mb.CPUMetrics.SetUsage(usageCores)
		addCPUUtilizationMetrics(mb, usageCores, currentTime, r, nodeCPULimit)
	}
	addCPUTimeMetric(mb, s, currentTime)
}

func addCPUUtilizationMetrics(mb *metadata.MetricsBuilder, usageCores float64, currentTime time.Time, r resources, nodeCPULimit float64) {
	mb.CPUMetrics.SetUtilization(usageCores)

	if nodeCPULimit > 0 {
		mb.CPUMetrics.SetNodeUtilization(usageCores / nodeCPULimit)
	}
	if r.cpuLimit > 0 {
		mb.CPUMetrics.SetLimitUtilization(usageCores / r.cpuLimit)
	}
	if r.cpuRequest > 0 {
		mb.CPUMetrics.SetRequestUtilization(usageCores / r.cpuRequest)
	}
}

func addCPUTimeMetric(mb *metadata.MetricsBuilder, s *stats.CPUStats, currentTime time.Time) {
	if s.UsageCoreNanoSeconds == nil {
		return
	}
	value := float64(*s.UsageCoreNanoSeconds) / 1_000_000_000

	mb.CPUMetrics.SetTime(value)
	mb.CPUMetrics.SetCurrentTime(currentTime)
}
