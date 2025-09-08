package deployment

import (
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	conventions "go.opentelemetry.io/otel/semconv/v1.6.1"
	"google.golang.org/protobuf/types/known/timestamppb"
	appsv1 "k8s.io/api/apps/v1"
)

// Transform transforms the pod to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new deployment fields.
func Transform(deployment *appsv1.Deployment) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metadata.TransformObjectMeta(deployment.ObjectMeta),
		Spec: appsv1.DeploymentSpec{
			Replicas: deployment.Spec.Replicas,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: deployment.Status.AvailableReplicas,
		},
	}
}

func RecordMetrics(dep *appsv1.Deployment, ts time.Time) *pb.KubernetesClusterObject {
	obj := &pb.KubernetesClusterObject{
		Timestamp:         timestamppb.New(ts),
		Namespace:         dep.Namespace,
		Name:              dep.Name,
		Uid:               string(dep.UID),
		Replicas:          int64(*dep.Spec.Replicas),
		AvailableReplicas: int64(dep.Status.AvailableReplicas),
	}
	return obj
}

func GetMetadata(dep *appsv1.Deployment) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	rm := metadata.GetGenericMetadata(&dep.ObjectMeta, constants.K8sKindDeployment)
	rm.Metadata[string(conventions.K8SDeploymentNameKey)] = dep.Name
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{metadata.ResourceID(dep.UID): rm}
}
