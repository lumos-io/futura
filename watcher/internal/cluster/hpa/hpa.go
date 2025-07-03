package hpa

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	autoscalingv2 "k8s.io/api/autoscaling/v2"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
)

func RecordMetrics(hpa *autoscalingv2.HorizontalPodAutoscaler, ts time.Time) *pbcluster.KubernetesClusterObject {
	obj := &pbcluster.KubernetesClusterObject{
		Timestamp:         timestamppb.New(ts),
		ReadyReplicas:     int64(hpa.Status.CurrentReplicas),
		Replicas:          int64(hpa.Status.DesiredReplicas),
		Kind:              hpa.Kind,
		Uid:               string(hpa.UID),
		Name:              hpa.Name,
		Namespace:         hpa.Namespace,
		ApiVersion:        hpa.APIVersion,
		HpaMaxReplicas:    int64(hpa.Spec.MaxReplicas),
		HpaMinReplicas:    int64(*hpa.Spec.MinReplicas),
		HpaScaleTargetRef: hpa.Spec.ScaleTargetRef.Name,
	}

	return obj
}

func GetMetadata(hpa *autoscalingv2.HorizontalPodAutoscaler) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(hpa.UID): metadata.GetGenericMetadata(&hpa.ObjectMeta, "HPA"),
	}
}
