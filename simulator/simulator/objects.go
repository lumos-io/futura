package simulator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GenerateClusterObjectFromPod(nodeName string, pod *pb.PodStats, changeType string) *pb.KubernetesClusterObject {
	now := timestamppb.Now()
	return &pb.KubernetesClusterObject{
		Timestamp:    now,
		Type:         changeType,
		Kind:         "Pod",
		Namespace:    pod.PodRef.Namespace,
		Name:         pod.PodRef.Name,
		Uid:          pod.PodRef.Uid,
		Labels:       map[string]string{"app": "demo", "tier": "backend"},
		Annotations:  map[string]string{"description": "Simulated pod"},
		NodeName:     nodeName,
		Status:       "Running",
		Phase:        "Running",
		RestartCount: int64(rand.Intn(5)),
		OwnerKind:    "ReplicaSet",
		OwnerName:    fmt.Sprintf("%s-rs", pod.PodRef.Name),
		ApiVersion:   "v1",
		Containers: []*pb.ContainerSpec{
			{
				Name:  "main",
				Image: "nginx:1.21",
				Resources: &pb.ContainerResources{
					Limits:   &pb.ResourceQuantities{Cpu: "500m", Memory: "256Mi"},
					Requests: &pb.ResourceQuantities{Cpu: "250m", Memory: "128Mi"},
				},
				ContainerId:   fmt.Sprintf("docker://%s", uuid.NewString()),
				RestartsCount: int64(rand.Intn(3)),
				Ready:         1,
				State: &pb.ContainerState{
					State: &pb.ContainerState_Running{
						Running: &pb.ContainerStateRunning{StartedAt: now},
					},
				},
				ImageTag: "1.21",
			},
		},
		Volumes: []*pb.VolumeSpec{
			{Name: "default-token", Type: "Secret"},
		},
		Tolerations: []string{"node.kubernetes.io/not-ready:NoExecute"},
		Affinity:    map[string]string{"zone": "us-central1-a"},
		Enrichment: &pb.EnrichmentMetadata{
			OrganizationId: 42,
			ClusterId:      12345,
			K8SVersion:     "1.29.0",
			ReceivedAtUnix: time.Now().Unix(),
		},
	}
}
