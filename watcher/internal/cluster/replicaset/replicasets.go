package replicaset

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	appsv1 "k8s.io/api/apps/v1"
)

// Transform transforms the replica set to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new replicaset fields.
func Transform(rs *appsv1.ReplicaSet) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metadata.TransformObjectMeta(rs.ObjectMeta),
		Spec: appsv1.ReplicaSetSpec{
			Replicas: rs.Spec.Replicas,
		},
		Status: appsv1.ReplicaSetStatus{
			AvailableReplicas: rs.Status.AvailableReplicas,
		},
	}
}

func RecordMetrics(mb *metadata.MetricsBuilder, rs *appsv1.ReplicaSet, ts time.Time) {
	if rs.Spec.Replicas != nil {
		mb.RecordK8sReplicasetDesiredDataPoint(ts, int64(*rs.Spec.Replicas))
		mb.RecordK8sReplicasetAvailableDataPoint(ts, int64(rs.Status.AvailableReplicas))
	}

	rb := mb.NewResourceBuilder()
	rb.SetK8sNamespaceName(rs.Namespace)
	rb.SetK8sReplicasetName(rs.Name)
	rb.SetK8sReplicasetUID(string(rs.UID))
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}

func GetMetadata(rs *appsv1.ReplicaSet) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(rs.UID): metadata.GetGenericMetadata(&rs.ObjectMeta, constants.K8sKindReplicaSet),
	}
}
