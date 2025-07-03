package kubelet

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"go.uber.org/zap"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func MetricsData(logger *zap.Logger, summary *stats.Summary, metadata Metadata, metricGroupsToCollect map[MetricGroup]bool, allNetworkInterfaces map[MetricGroup]bool, mbs *metadata.MetricsBuilders) []pmetric.Metrics {
	acc := &metricDataAccumulator{
		metadata:              metadata,
		logger:                logger,
		metricGroupsToCollect: metricGroupsToCollect,
		allNetworkInterfaces:  allNetworkInterfaces,
		time:                  time.Now(),
		mbs:                   mbs,
	}
	acc.nodeStats(summary.Node)
	for _, podStats := range summary.Pods {
		acc.podStats(podStats)
		for _, containerStats := range podStats.Containers {
			// propagate the pod resource down to the container
			acc.containerStats(podStats, containerStats)
		}

		for _, volumeStats := range podStats.VolumeStats {
			// propagate the pod resource down to the container
			acc.volumeStats(podStats, volumeStats)
		}
	}
	return acc.metricssss
}
