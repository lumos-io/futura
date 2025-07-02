package hpa

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	autoscalingv2 "k8s.io/api/autoscaling/v2"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
)

func RecordMetrics(hpa *autoscalingv2.HorizontalPodAutoscaler, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp:      timestamppb.New(ts),
		ReadyReplicas:  int32(hpa.Status.CurrentReplicas),
		Replicas:       int32(hpa.Status.DesiredReplicas),
		Kind:           hpa.Kind,
		Uid:            string(hpa.UID),
		Name:           hpa.Name,
		Namespace:      hpa.Namespace,
		ApiVersion:     hpa.APIVersion,
		MaxReplicas:    int32(hpa.Spec.MaxReplicas),
		MinReplicas:    int32(*hpa.Spec.MinReplicas),
		ScaleTargetRef: hpa.Spec.ScaleTargetRef.Name,
	}

	return obj
}

func GetMetadata(hpa *autoscalingv2.HorizontalPodAutoscaler) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(hpa.UID): metadata.GetGenericMetadata(&hpa.ObjectMeta, "HPA"),
	}
}
