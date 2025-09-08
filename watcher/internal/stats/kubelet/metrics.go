package kubelet

import (
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

type Accumulator struct {
	NodeStats *pb.NodeStats
	PodStats  []*pb.PodStats
}

func MetricsData(summary *stats.Summary, md Metadata) *Accumulator {
	acc := &Accumulator{
		NodeStats: metadata.NewNodeStatsBuilder(summary.Node),
		PodStats:  make([]*pb.PodStats, len(summary.Pods)),
	}
	for i, podStats := range summary.Pods {
		ps := metadata.NewPodStatsBuilder(podStats)
		acc.PodStats[i] = ps
	}
	return acc
}

func (a *Accumulator) Emit() *pb.KubernetesKubeletStats {
	return &pb.KubernetesKubeletStats{
		Node: a.NodeStats,
		Pods: a.PodStats,
	}
}
