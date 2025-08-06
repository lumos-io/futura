package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opisvigilant/futura/go-lib/stream"
	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
)

const (
	StoreKubernetesContainersTopic           = "store.k8s.containers"
	StoreKubernetesVolumesTopic              = "store.k8s.volumes"
	StoreKubernetesNodeConditionsTopic       = "store.k8s.node.conditions"
	StoreKubernetesAllocatableResourcesTopic = "store.k8s.allocatable.resources"
	StoreKubernetesClusterQuotasTopic        = "store.k8s.cluster.quotas"
	StoreKubernetesNamespaceQuotasTopic      = "store.k8s.namespace.quotas"
)

type ObjectSplitter struct {
	kc stream.Stream
}

func NewObjectSplitter(kc stream.Stream) *ObjectSplitter {
	return &ObjectSplitter{
		kc: kc,
	}
}

type flatKubernetesObject struct {
	OrganizationID                  uint32            `json:"organization_id"`
	ClusterID                       int64             `json:"cluster_id"`
	K8sVersion                      string            `json:"k8s_version"`
	ReceivedAtUnix                  int64             `json:"received_at_unix"`
	Timestamp                       time.Time         `json:"timestamp"`
	Type                            string            `json:"type"`
	Kind                            string            `json:"kind"`
	Namespace                       string            `json:"namespace"`
	Name                            string            `json:"name"`
	UID                             string            `json:"uid"`
	Labels                          map[string]string `json:"labels"`
	Annotations                     map[string]string `json:"annotations"`
	NodeName                        string            `json:"node_name"`
	Status                          string            `json:"status"`
	Phase                           string            `json:"phase"`
	RestartCount                    int64             `json:"restart_count"`
	OwnerKind                       string            `json:"owner_kind"`
	OwnerName                       string            `json:"owner_name"`
	Replicas                        int64             `json:"replicas"`
	ReadyReplicas                   int64             `json:"ready_replicas"`
	AvailableReplicas               int64             `json:"available_replicas"`
	UpdatedReplicas                 int64             `json:"updated_replicas"`
	CurrentReplicas                 int64             `json:"current_replicas"`
	Tolerations                     []string          `json:"tolerations"`
	Affinity                        map[string]string `json:"affinity"`
	Extra                           map[string]string `json:"extra"`
	APIVersion                      string            `json:"api_version"`
	HPAMaxReplicas                  int64             `json:"hpa_max_replicas"`
	HPAMinReplicas                  int64             `json:"hpa_min_replicas"`
	HPAScaleTargetRef               string            `json:"hpa_scale_target_ref"`
	JobActive                       int64             `json:"job_active"`
	JobFailed                       int64             `json:"job_failed"`
	JobSucceeded                    int64             `json:"job_succeeded"`
	JobParallelism                  int64             `json:"job_parallelism"`
	JobCompletions                  int64             `json:"job_completions"`
	NsPhaseValue                    int64             `json:"ns_phase_value"`
	KubeletVersion                  string            `json:"kubelet_version"`
	OSType                          string            `json:"os_type"`
	OSImage                         string            `json:"os_image"`
	ContainerRuntime                string            `json:"container_runtime"`
	ContainerRuntimeVersion         string            `json:"container_runtime_version"`
	PodReason                       string            `json:"pod_reason"`
	QOSClass                        string            `json:"qos_class"`
	DaemonsetCurrentNumberScheduled int64             `json:"daemonset_current_number_scheduled"`
	DaemonsetDesiredNumberScheduled int64             `json:"daemonset_desired_number_scheduled"`
	DaemonsetNumberMisscheduled     int64             `json:"daemonset_number_misscheduled"`
	DaemonsetNumberReady            int64             `json:"daemonset_number_ready"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	WatcherVersion                  string            `json:"watcher_version"`
}

type flatKubernetesContainer struct {
	UID            string    `json:"uid"`
	Timestamp      time.Time `json:"timestamp"`
	ContainerName  string    `json:"container_name"`
	Image          string    `json:"image"`
	ImageTag       string    `json:"image_tag"`
	ContainerID    string    `json:"container_id"`
	RestartsCount  int64     `json:"restarts_count"`
	Ready          int64     `json:"ready"`
	StateType      string    `json:"state_type"` // Enum8 stored as String in JSON
	CPULimits      string    `json:"cpu_limits"`
	MemoryLimits   string    `json:"memory_limits"`
	CPURequests    string    `json:"cpu_requests"`
	MemoryRequests string    `json:"memory_requests"`
}

type flatKubernetesVolume struct {
	UID        string    `json:"uid"`
	Timestamp  time.Time `json:"timestamp"`
	VolumeName string    `json:"volume_name"`
	VolumeType string    `json:"volume_type"`
}

type flatKubernetesNodeCondition struct {
	UID             string    `json:"uid"`
	Timestamp       time.Time `json:"timestamp"`
	ConditionType   string    `json:"condition_type"`
	ConditionStatus string    `json:"condition_status"`
	Reason          string    `json:"reason"`
	Message         string    `json:"message"`
}

type flatKubernetesAllocatableResource struct {
	UID              string            `json:"uid"`
	Timestamp        time.Time         `json:"timestamp"`
	CPU              string            `json:"cpu"`
	Memory           string            `json:"memory"`
	Pods             string            `json:"pods"`
	EphemeralStorage string            `json:"ephemeral_storage"`
	Others           map[string]string `json:"others"`
}

type flatKubernetesClusterQuota struct {
	UID         string              `json:"uid"`
	Timestamp   time.Time           `json:"timestamp"`
	QuotaName   string              `json:"quota_name"`
	QuotaUID    string              `json:"quota_uid"`
	TotalLimits []flatResourceTuple `json:"total_limits"`
	TotalUsage  []flatResourceTuple `json:"total_usage"`
}

type flatKubernetesNamespaceQuota struct {
	UID       string              `json:"uid"`
	Timestamp time.Time           `json:"timestamp"`
	Namespace string              `json:"namespace"`
	Limits    []flatResourceTuple `json:"limits"`
	Usage     []flatResourceTuple `json:"usage"`
}

type flatResourceTuple struct {
	Resource string `json:"resource"`
	Value    int64  `json:"value"`
}

// Custom Unmarshal for ResourceTuple
func (r *flatResourceTuple) UnmarshalJSON(data []byte) error {
	var temp []any
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	if len(temp) != 2 {
		return fmt.Errorf("expected array of 2 elements, got %d", len(temp))
	}

	// First element should be string
	resource, ok := temp[0].(string)
	if !ok {
		return fmt.Errorf("expected string for resource, got %T", temp[0])
	}

	// Second element should be number
	var value int64
	switch v := temp[1].(type) {
	case float64:
		value = int64(v)
	case int64:
		value = v
	default:
		return fmt.Errorf("expected number for value, got %T", temp[1])
	}

	r.Resource = resource
	r.Value = value
	return nil
}

func (os *ObjectSplitter) Split(ctx context.Context, msg *pbcl.KubernetesClusterObject) error {
	timestamp := time.Now()
	ko := &flatKubernetesObject{
		OrganizationID:                  msg.Enrichment.OrganizationId,
		ClusterID:                       msg.Enrichment.ClusterId,
		K8sVersion:                      msg.Enrichment.K8SVersion,
		ReceivedAtUnix:                  msg.Enrichment.ReceivedAtUnix,
		Timestamp:                       timestamp,
		Type:                            msg.Type,
		Kind:                            msg.Kind,
		Namespace:                       msg.Namespace,
		Name:                            msg.Name,
		UID:                             msg.Uid,
		Labels:                          msg.Labels,
		Annotations:                     msg.Annotations,
		NodeName:                        msg.NodeName,
		Status:                          msg.Status,
		Phase:                           msg.Phase,
		RestartCount:                    msg.RestartCount,
		Replicas:                        msg.Replicas,
		OwnerKind:                       msg.OwnerKind,
		OwnerName:                       msg.OwnerName,
		ReadyReplicas:                   msg.ReadyReplicas,
		AvailableReplicas:               msg.AvailableReplicas,
		UpdatedReplicas:                 msg.UpdatedReplicas,
		CurrentReplicas:                 msg.CurrentReplicas,
		Tolerations:                     msg.Tolerations,
		Affinity:                        msg.Affinity,
		Extra:                           msg.Extra,
		APIVersion:                      msg.ApiVersion,
		HPAMaxReplicas:                  msg.HpaMaxReplicas,
		HPAMinReplicas:                  msg.HpaMinReplicas,
		HPAScaleTargetRef:               msg.HpaScaleTargetRef,
		JobActive:                       msg.JobActive,
		JobFailed:                       msg.JobFailed,
		JobSucceeded:                    msg.JobSucceeded,
		JobParallelism:                  msg.JobParallelism,
		JobCompletions:                  msg.JobCompletions,
		NsPhaseValue:                    msg.NsPhaseValue,
		KubeletVersion:                  msg.KubeletVersion,
		OSType:                          msg.OsType,
		OSImage:                         msg.OsImage,
		ContainerRuntime:                msg.ContainerRuntime,
		ContainerRuntimeVersion:         msg.ContainerRuntimeVersion,
		PodReason:                       msg.PodReason,
		QOSClass:                        msg.QosClass,
		DaemonsetCurrentNumberScheduled: msg.DaemonsetCurrentNumberScheduled,
		DaemonsetDesiredNumberScheduled: msg.DaemonsetDesiredNumberScheduled,
		DaemonsetNumberMisscheduled:     msg.DaemonsetNumberMisscheduled,
		DaemonsetNumberReady:            msg.DaemonsetNumberReady,
		IdempotencyKey:                  msg.Metadata.IdempotencyKey,
		WatcherVersion:                  msg.Metadata.IdempotencyKey,
	}
	b, err := json.Marshal(ko)
	if err != nil {
		return err
	}
	if err := os.kc.Publish(ctx, StoreKubernetesObjectsTopic, b); err != nil {
		return err
	}

	var kc *flatKubernetesContainer
	for _, container := range msg.Containers {
		kc = &flatKubernetesContainer{
			UID:            msg.Uid,
			Timestamp:      timestamp,
			ContainerName:  container.Name,
			Image:          container.Image,
			ImageTag:       container.ImageTag,
			ContainerID:    container.ContainerId,
			RestartsCount:  container.RestartsCount,
			Ready:          container.Ready,
			StateType:      container.State.String(),
			CPULimits:      container.Resources.Limits.Cpu,
			MemoryLimits:   container.Resources.Limits.Memory,
			CPURequests:    container.Resources.Requests.Cpu,
			MemoryRequests: container.Resources.Requests.Memory,
		}
		b, err := json.Marshal(kc)
		if err != nil {
			return err
		}
		if err := os.kc.Publish(ctx, StoreKubernetesContainersTopic, b); err != nil {
			return err
		}
	}

	return nil
}
