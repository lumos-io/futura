package validate_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opisvigilant/futura/backend/internal/pipeline/validate"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

func TestValidateKubernetesEvent(t *testing.T) {
	valid := &pb.KubernetesEvent{
		ObjectKind:          "Pod",
		ObjectName:          "nginx",
		ObjectUid:           "uid123",
		ObjectTimestamp:     time.Now().Unix(),
		EventUid:            "event123",
		EventSeverityNumber: 1,
		EventCount:          1,
		EventStarttime:      time.Now().Format(time.RFC3339),
	}

	v := &validate.Validator{}

	t.Run("valid event", func(t *testing.T) {
		require.NoError(t, v.ValidateKubernetesEvent(valid))
	})

	t.Run("nil event", func(t *testing.T) {
		require.ErrorContains(t, v.ValidateKubernetesEvent(nil), "event is nil")
	})

	t.Run("missing objectKind", func(t *testing.T) {
		ev := *valid
		ev.ObjectKind = ""
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "object_kind is required")
	})

	t.Run("timestamp too old", func(t *testing.T) {
		ev := *valid
		ev.ObjectTimestamp = time.Now().Add(-25 * time.Hour).Unix()
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "object_timestamp out of acceptable range")
	})

	t.Run("bad RFC3339 format", func(t *testing.T) {
		ev := *valid
		ev.EventStarttime = "not-a-date"
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "event_starttime not RFC3339")
	})

	t.Run("missing object_name", func(t *testing.T) {
		ev := *valid
		ev.ObjectName = ""
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "object_name is required")
	})

	t.Run("missing object_uid", func(t *testing.T) {
		ev := *valid
		ev.ObjectUid = ""
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "object_uid is required")
	})

	t.Run("zero object_timestamp", func(t *testing.T) {
		ev := *valid
		ev.ObjectTimestamp = 0
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "invalid object_timestamp")
	})

	t.Run("missing event_uid", func(t *testing.T) {
		ev := *valid
		ev.EventUid = ""
		require.ErrorContains(t, v.ValidateKubernetesEvent(&ev), "event_uid is required")
	})

}

func TestValidateKubernetesClusterObject(t *testing.T) {
	now := time.Now()
	valid := &pb.KubernetesClusterObject{
		Timestamp: timestamppb.New(now),
		Type:      "Deployment",
		Kind:      "Pod",
		Name:      "my-pod",
		Uid:       "uid456",
		Replicas:  3,
		Containers: []*pb.ContainerSpec{
			{Name: "c1", Image: "nginx", RestartsCount: 0},
		},
		Volumes: []*pb.VolumeSpec{
			{Name: "vol1", Type: "PersistentVolume"},
		},
		Conditions: []*pb.NodeCondition{
			{Type: "Ready", Status: "True"},
		},
		ClusterQuota: &pb.ClusterResourceQuotaMetadata{
			Name: "quota1", Uid: "quota-uid",
		},
	}

	v := &validate.Validator{}

	t.Run("valid object", func(t *testing.T) {
		require.NoError(t, v.ValidateKubernetesClusterObject(valid))
	})

	t.Run("missing name", func(t *testing.T) {
		obj := *valid
		obj.Name = ""
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "name is required")
	})

	t.Run("negative replica", func(t *testing.T) {
		obj := *valid
		obj.Replicas = -1
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "replicas must be non-negative")
	})

	t.Run("invalid container", func(t *testing.T) {
		obj := *valid
		obj.Containers = []*pb.ContainerSpec{
			{Name: "", Image: "nginx"},
		}
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "containers[0].name is required")
	})

	t.Run("invalid volume", func(t *testing.T) {
		obj := *valid
		obj.Volumes = []*pb.VolumeSpec{
			{Name: "", Type: ""},
		}
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "volumes[0].name is required")
	})

	t.Run("nil cluster object", func(t *testing.T) {
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(nil), "cluster object is nil")
	})

	t.Run("missing type", func(t *testing.T) {
		obj := *valid
		obj.Type = ""
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "type is required")
	})

	t.Run("missing kind", func(t *testing.T) {
		obj := *valid
		obj.Kind = ""
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "kind is required")
	})

	t.Run("missing uid", func(t *testing.T) {
		obj := *valid
		obj.Uid = ""
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "uid is required")
	})

	t.Run("missing cluster_quota.name", func(t *testing.T) {
		obj := *valid
		obj.ClusterQuota = &pb.ClusterResourceQuotaMetadata{
			Name: "",
			Uid:  "some-uid",
		}
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "cluster_quota.name is required")
	})

	t.Run("missing cluster_quota.uid", func(t *testing.T) {
		obj := *valid
		obj.ClusterQuota = &pb.ClusterResourceQuotaMetadata{
			Name: "quota-name",
			Uid:  "",
		}
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "cluster_quota.uid is required")
	})

	t.Run("timestamp is nil", func(t *testing.T) {
		obj := *valid
		obj.Timestamp = nil
		require.ErrorContains(t, v.ValidateKubernetesClusterObject(&obj), "timestamp is nil")
	})
}

func TestValidateKubeletMetrics(t *testing.T) {
	now := time.Now()
	valid := &pb.KubernetesKubeletStats{
		Node: &pb.NodeStats{
			NodeName:  "node1",
			StartTime: timestamppb.New(now),
			SystemContainers: []*pb.ContainerStats{
				{Name: "kubelet", StartTime: timestamppb.New(now)},
			},
		},
		Pods: []*pb.PodStats{
			{
				PodRef:    &pb.PodReference{Uid: "pod-uid"},
				StartTime: timestamppb.New(now),
				Containers: []*pb.ContainerStats{
					{Name: "app", StartTime: timestamppb.New(now)},
				},
			},
		},
	}

	v := &validate.Validator{}

	t.Run("valid metrics", func(t *testing.T) {
		require.NoError(t, v.ValidateKubeletMetrics(valid))
	})

	t.Run("missing node", func(t *testing.T) {
		m := *valid
		m.Node = nil
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "missing node stats")
	})

	t.Run("empty node name", func(t *testing.T) {
		m := *valid
		node := *valid.Node
		node.NodeName = ""
		m.Node = &node
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "node name is required")
	})

	t.Run("pod missing uid", func(t *testing.T) {
		m := *valid
		m.Pods = []*pb.PodStats{
			{PodRef: &pb.PodReference{Uid: ""}},
		}
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "pods[0]: missing podRef.uid")
	})

	t.Run("nil metrics", func(t *testing.T) {
		require.ErrorContains(t, v.ValidateKubeletMetrics(nil), "metrics message is nil")
	})

	t.Run("node is nil", func(t *testing.T) {
		m := *valid
		m.Node = nil
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "missing node stats")
	})

	t.Run("missing node name", func(t *testing.T) {
		m := *valid
		n := *m.Node
		n.NodeName = ""
		m.Node = &n
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "node name is required")
	})

	t.Run("container startTime is nil", func(t *testing.T) {
		m := *valid
		m.Node.SystemContainers[0].StartTime = nil
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "systemContainers[0].startTime is nil")
	})

	t.Run("pod container startTime is nil", func(t *testing.T) {
		m := *valid
		// Make sure system container StartTime is valid so we pass that check
		m.Node.SystemContainers[0].StartTime = timestamppb.Now()
		// Then trigger the nil on pod container
		m.Pods[0].Containers[0].StartTime = nil
		require.ErrorContains(t, v.ValidateKubeletMetrics(&m), "pods[0].containers[0].startTime is nil")
	})
}
