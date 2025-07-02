package daemonset

import (
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	appsv1 "k8s.io/api/apps/v1"
)

// Transform transforms the pod to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new daemonset fields.
func Transform(ds *appsv1.DaemonSet) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metadata.TransformObjectMeta(ds.ObjectMeta),
		Status: appsv1.DaemonSetStatus{
			CurrentNumberScheduled: ds.Status.CurrentNumberScheduled,
			DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
			NumberMisscheduled:     ds.Status.NumberMisscheduled,
			NumberReady:            ds.Status.NumberReady,
		},
	}
}

func RecordMetrics(ds *appsv1.DaemonSet, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
		Namespace: ds.Namespace,
		Name:      ds.Name,
		Uid:       string(ds.UID),
	}

	// TODO: how do I store the below data???
	// mb.RecordK8sDaemonsetCurrentScheduledNodesDataPoint(ts, int64(ds.Status.CurrentNumberScheduled))
	// mb.RecordK8sDaemonsetDesiredScheduledNodesDataPoint(ts, int64(ds.Status.DesiredNumberScheduled))
	// mb.RecordK8sDaemonsetMisscheduledNodesDataPoint(ts, int64(ds.Status.NumberMisscheduled))
	// mb.RecordK8sDaemonsetReadyNodesDataPoint(ts, int64(ds.Status.NumberReady))

	return obj
}

func GetMetadata(ds *appsv1.DaemonSet) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(ds.UID): metadata.GetGenericMetadata(&ds.ObjectMeta, constants.K8sKindDaemonSet),
	}
}
