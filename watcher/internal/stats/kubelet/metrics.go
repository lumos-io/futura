package kubelet

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

type Accumulator struct {
	NodeStats *pbst.NodeStats
	PodStats  []*pbst.PodStats
}

func MetricsData(summary *stats.Summary, md Metadata) *Accumulator {
	acc := &Accumulator{
		NodeStats: metadata.NewNodeStatsBuilder(summary.Node),
		PodStats:  make([]*pbst.PodStats, len(summary.Pods)),
	}
	for i, podStats := range summary.Pods {
		ps := metadata.NewPodStatsBuilder(podStats)
		acc.PodStats[i] = ps
	}
	return acc
}
