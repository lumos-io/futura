package replicationcontroller

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(mb *metadata.MetricsBuilder, rc *corev1.ReplicationController, ts time.Time) {
	if rc.Spec.Replicas != nil {
		mb.RecordK8sReplicationControllerDesiredDataPoint(ts, int64(*rc.Spec.Replicas))
		mb.RecordK8sReplicationControllerAvailableDataPoint(ts, int64(rc.Status.AvailableReplicas))
	}

	rb := mb.NewResourceBuilder()
	rb.SetK8sNamespaceName(rc.Namespace)
	rb.SetK8sReplicationcontrollerName(rc.Name)
	rb.SetK8sReplicationcontrollerUID(string(rc.UID))
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}

func GetMetadata(rc *corev1.ReplicationController) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(rc.UID): metadata.GetGenericMetadata(&rc.ObjectMeta, constants.K8sKindReplicationController),
	}
}
