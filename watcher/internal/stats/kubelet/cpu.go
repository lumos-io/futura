package kubelet

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func addCPUMetrics(mb *metadata.NodeMetricsBuilder, cpuMetrics *metadata.NodeCPUMetricsBuilder, s *stats.CPUStats, currentTime time.Time, r resources, nodeCPULimit float64) {
	if s == nil {
		return
	}
	if s.UsageNanoCores != nil {
		usageCores := float64(*s.UsageNanoCores) / 1_000_000_000
		cpuMetrics.Usage = usageCores
		addCPUUtilizationMetrics(mb, cpuMetrics, usageCores, currentTime, r, nodeCPULimit)
	}
	// addCPUTimeMetric(mb,cpuMetrics.Time, s, currentTime)
}

func addCPUUtilizationMetrics(mb *metadata.NodeMetricsBuilder, cpuMetrics *metadata.NodeCPUMetricsBuilder, usageCores float64, currentTime time.Time, r resources, nodeCPULimit float64) {
	cpuMetrics.Utilization = usageCores

	if nodeCPULimit > 0 {
		cpuMetrics.NodeUtilization = usageCores / nodeCPULimit
	}
	if r.cpuLimit > 0 {
		cpuMetrics.LimitUtilization = usageCores / r.cpuLimit
	}
	if r.cpuRequest > 0 {
		cpuMetrics.RequestUtilization = usageCores / r.cpuRequest
	}
	mb.RecordCPUMetricsMetric("cpu.metrics", cpuMetrics, currentTime.UnixNano(), nil)
}

func addCPUTimeMetric(mb *metadata.NodeMetricsBuilder, cpuTime int64, s *stats.CPUStats, currentTime time.Time) {
	if s.UsageCoreNanoSeconds == nil {
		return
	}
	value := float64(*s.UsageCoreNanoSeconds) / 1_000_000_000

	mb.RecordUptime(currentTime, value)
	recordDataPoint(mb, currentTime, value)
}
