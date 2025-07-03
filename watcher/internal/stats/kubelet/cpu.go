package kubelet

import (
	"time"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
)

func addCPUMetrics(mb *metadata.MetricsBuilder, cpuMetrics metadata.CPUMetrics, s *stats.CPUStats, currentTime time.Time, r resources, nodeCPULimit float64) {
	if s == nil {
		return
	}
	if s.UsageNanoCores != nil {
		usageCores := float64(*s.UsageNanoCores) / 1_000_000_000
		cpuMetrics.Usage(mb, currentTime, usageCores)
		addCPUUtilizationMetrics(mb, cpuMetrics, usageCores, currentTime, r, nodeCPULimit)
	}
	addCPUTimeMetric(mb, cpuMetrics.Time, s, currentTime)
}

func addCPUUtilizationMetrics(mb *metadata.MetricsBuilder, cpuMetrics metadata.CPUMetrics, usageCores float64, currentTime time.Time, r resources, nodeCPULimit float64) {
	cpuMetrics.Utilization(mb, currentTime, usageCores)

	if nodeCPULimit > 0 {
		cpuMetrics.NodeUtilization(mb, currentTime, usageCores/nodeCPULimit)
	}
	if r.cpuLimit > 0 {
		cpuMetrics.LimitUtilization(mb, currentTime, usageCores/r.cpuLimit)
	}
	if r.cpuRequest > 0 {
		cpuMetrics.RequestUtilization(mb, currentTime, usageCores/r.cpuRequest)
	}
}

func addCPUTimeMetric(mb *metadata.MetricsBuilder, recordDataPoint metadata.RecordDoubleDataPointFunc, s *stats.CPUStats, currentTime time.Time) {
	if s.UsageCoreNanoSeconds == nil {
		return
	}
	value := float64(*s.UsageCoreNanoSeconds) / 1_000_000_000
	recordDataPoint(mb, currentTime, value)
}
