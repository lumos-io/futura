package replicationcontroller

import (
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(rc *corev1.ReplicationController, ts time.Time) *pb.KubernetesClusterObject {
	obj := &pb.KubernetesClusterObject{
		Timestamp: timestamppb.New(ts),
		Namespace: rc.Namespace,
		Name:      rc.Name,
		Uid:       string(rc.UID),
	}

	if rc.Spec.Replicas != nil {
		obj.Replicas = int64(*rc.Spec.Replicas)
		obj.AvailableReplicas = int64(rc.Status.AvailableReplicas)
	}

	return obj
}

func GetMetadata(rc *corev1.ReplicationController) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(rc.UID): metadata.GetGenericMetadata(&rc.ObjectMeta, constants.K8sKindReplicationController),
	}
}
