package replicationcontroller

import (
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(rc *corev1.ReplicationController, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
	}

	if rc.Spec.Replicas != nil {
		mb.RecordK8sReplicationControllerDesiredDataPoint(ts, int64(*rc.Spec.Replicas))
		mb.RecordK8sReplicationControllerAvailableDataPoint(ts, int64(rc.Status.AvailableReplicas))
	}

	rb := mb.NewResourceBuilder()
	rb.SetK8sNamespaceName(rc.Namespace)
	rb.SetK8sReplicationcontrollerName(rc.Name)
	rb.SetK8sReplicationcontrollerUID(string(rc.UID))

	return obj
}

func GetMetadata(rc *corev1.ReplicationController) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(rc.UID): metadata.GetGenericMetadata(&rc.ObjectMeta, constants.K8sKindReplicationController),
	}
}
