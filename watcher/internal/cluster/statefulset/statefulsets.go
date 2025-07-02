package statefulset

import (
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	// Keys for stateful set metadata.
	statefulSetCurrentVersion = "current_revision"
	statefulSetUpdateVersion  = "update_revision"
)

// Transform transforms the pod to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new statefulset fields.
func Transform(statefulset *appsv1.StatefulSet) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metadata.TransformObjectMeta(statefulset.ObjectMeta),
		Spec: appsv1.StatefulSetSpec{
			Replicas: statefulset.Spec.Replicas,
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   statefulset.Status.ReadyReplicas,
			CurrentReplicas: statefulset.Status.CurrentReplicas,
			UpdatedReplicas: statefulset.Status.UpdatedReplicas,
		},
	}
}

func RecordMetrics(ss *appsv1.StatefulSet, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	if ss.Spec.Replicas == nil {
		return nil
	}

	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp:       timestamppb.New(ts),
		Replicas:        int64(*ss.Spec.Replicas),
		ReadyReplicas:   int64(ss.Status.ReadyReplicas),
		UpdatedReplicas: int64(ss.Status.UpdatedReplicas),
		CurrentReplicas: int64(ss.Status.CurrentReplicas),
		Uid:             string(ss.UID),
		Name:            ss.Name,
		Namespace:       ss.Namespace,
	}

	return obj
}

func GetMetadata(ss *appsv1.StatefulSet) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	km := metadata.GetGenericMetadata(&ss.ObjectMeta, constants.K8sStatefulSet)
	km.Metadata[statefulSetCurrentVersion] = ss.Status.CurrentRevision
	km.Metadata[statefulSetUpdateVersion] = ss.Status.UpdateRevision

	return map[metadata.ResourceID]*metadata.KubernetesMetadata{metadata.ResourceID(ss.UID): km}
}
