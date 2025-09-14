package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/opisvigilant/futura/go-lib/stream"
	pbcl "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	StoreKubernetesObjectsTopic              = "store.k8s.objects"
	StoreKubernetesContainersTopic           = "store.k8s.containers"
	StoreKubernetesVolumesTopic              = "store.k8s.volumes"
	StoreKubernetesNodeConditionsTopic       = "store.k8s.node.conditions"
	StoreKubernetesAllocatableResourcesTopic = "store.k8s.allocatable.resources"
	StoreKubernetesClusterQuotasTopic        = "store.k8s.cluster.quotas"
	StoreKubernetesNamespaceQuotasTopic      = "store.k8s.namespace.quotas"
)

type ObjectFlattener struct {
	kc stream.Stream
}

func NewObjectFlattener(kc stream.Stream) *ObjectFlattener {
	return &ObjectFlattener{
		kc: kc,
	}
}

type flatKubernetesObject struct {
	OrganizationID                  uint32            `json:"organization_id"`
	ClusterID                       int64             `json:"cluster_id"`
	K8sVersion                      string            `json:"k8s_version"`
	ReceivedAtUnix                  int64             `json:"received_at_unix"`
	Timestamp                       int64             `json:"timestamp"`
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
	UID            string `json:"uid"`
	IdempotencyKey string `json:"idempotency_key"`
	Timestamp      int64  `json:"timestamp"`
	ContainerName  string `json:"container_name"`
	Image          string `json:"image"`
	ImageTag       string `json:"image_tag"`
	ContainerID    string `json:"container_id"`
	RestartsCount  int64  `json:"restarts_count"`
	Ready          int64  `json:"ready"`
	StateType      string `json:"state_type"` // Enum8 stored as String in JSON
	CPULimits      string `json:"cpu_limits"`
	MemoryLimits   string `json:"memory_limits"`
	CPURequests    string `json:"cpu_requests"`
	MemoryRequests string `json:"memory_requests"`
}

type flatKubernetesVolume struct {
	UID            string `json:"uid"`
	IdempotencyKey string `json:"idempotency_key"`
	Timestamp      int64  `json:"timestamp"`
	VolumeName     string `json:"volume_name"`
	VolumeType     string `json:"volume_type"`
}

type flatKubernetesNodeCondition struct {
	UID             string `json:"uid"`
	IdempotencyKey  string `json:"idempotency_key"`
	Timestamp       int64  `json:"timestamp"`
	ConditionType   string `json:"condition_type"`
	ConditionStatus string `json:"condition_status"`
	Reason          string `json:"reason"`
	Message         string `json:"message"`
}

type flatKubernetesAllocatableResource struct {
	UID              string            `json:"uid"`
	IdempotencyKey   string            `json:"idempotency_key"`
	Timestamp        int64             `json:"timestamp"`
	CPU              string            `json:"cpu"`
	Memory           string            `json:"memory"`
	Pods             string            `json:"pods"`
	EphemeralStorage string            `json:"ephemeral_storage"`
	Others           map[string]string `json:"others"`
}

type flatKubernetesClusterQuota struct {
	UID            string              `json:"uid"`
	IdempotencyKey string              `json:"idempotency_key"`
	Timestamp      int64               `json:"timestamp"`
	QuotaName      string              `json:"quota_name"`
	QuotaUID       string              `json:"quota_uid"`
	TotalLimits    []flatResourceTuple `json:"total_limits"`
	TotalUsage     []flatResourceTuple `json:"total_usage"`
}

type flatKubernetesNamespaceQuota struct {
	UID            string              `json:"uid"`
	IdempotencyKey string              `json:"idempotency_key"`
	Timestamp      int64               `json:"timestamp"`
	Namespace      string              `json:"namespace"`
	Limits         []flatResourceTuple `json:"limits"`
	Usage          []flatResourceTuple `json:"usage"`
}

type flatResourceTuple struct {
	IdempotencyKey string `json:"idempotency_key"`
	Resource       string `json:"resource"`
	Value          int64  `json:"value"`
}

func (os *ObjectFlattener) Flatten(ctx context.Context, msg *pbcl.KubernetesClusterObject) error {
	if msg == nil {
		return nil // nothing to do
	}

	timestamp := time.Now()

	// guard optional nested messages
	enrichment := msg.Enrichment
	metadata := msg.Metadata
	allocatable := msg.Allocatable
	clusterQuota := msg.ClusterQuota

	ko := &flatKubernetesObject{
		OrganizationID:                  enrichment.GetOrganizationId(),
		ClusterID:                       enrichment.GetClusterId(),
		K8sVersion:                      enrichment.GetK8SVersion(),
		ReceivedAtUnix:                  enrichment.GetReceivedAtUnix(),
		Timestamp:                       timestamp.Unix(),
		Type:                            msg.GetType(),
		Kind:                            msg.GetKind(),
		Namespace:                       msg.GetNamespace(),
		Name:                            msg.GetName(),
		UID:                             msg.GetUid(),
		Labels:                          safeMap(msg.Labels),
		Annotations:                     safeMap(msg.Annotations),
		NodeName:                        msg.GetNodeName(),
		Status:                          msg.GetStatus(),
		Phase:                           msg.GetPhase(),
		RestartCount:                    msg.GetRestartCount(),
		Replicas:                        msg.GetReplicas(),
		OwnerKind:                       msg.GetOwnerKind(),
		OwnerName:                       msg.GetOwnerName(),
		ReadyReplicas:                   msg.GetReadyReplicas(),
		AvailableReplicas:               msg.GetAvailableReplicas(),
		UpdatedReplicas:                 msg.GetUpdatedReplicas(),
		CurrentReplicas:                 msg.GetCurrentReplicas(),
		Tolerations:                     msg.Tolerations,
		Affinity:                        safeMap(msg.Affinity),
		Extra:                           safeMap(msg.Extra),
		APIVersion:                      msg.GetApiVersion(),
		HPAMaxReplicas:                  msg.GetHpaMaxReplicas(),
		HPAMinReplicas:                  msg.GetHpaMinReplicas(),
		HPAScaleTargetRef:               msg.GetHpaScaleTargetRef(),
		JobActive:                       msg.GetJobActive(),
		JobFailed:                       msg.GetJobFailed(),
		JobSucceeded:                    msg.GetJobSucceeded(),
		JobParallelism:                  msg.GetJobParallelism(),
		JobCompletions:                  msg.GetJobCompletions(),
		NsPhaseValue:                    msg.GetNsPhaseValue(),
		KubeletVersion:                  msg.GetKubeletVersion(),
		OSType:                          msg.GetOsType(),
		OSImage:                         msg.GetOsImage(),
		ContainerRuntime:                msg.GetContainerRuntime(),
		ContainerRuntimeVersion:         msg.GetContainerRuntimeVersion(),
		PodReason:                       msg.GetPodReason(),
		QOSClass:                        msg.GetQosClass(),
		DaemonsetCurrentNumberScheduled: msg.GetDaemonsetCurrentNumberScheduled(),
		DaemonsetDesiredNumberScheduled: msg.GetDaemonsetDesiredNumberScheduled(),
		DaemonsetNumberMisscheduled:     msg.GetDaemonsetNumberMisscheduled(),
		DaemonsetNumberReady:            msg.GetDaemonsetNumberReady(),
		IdempotencyKey:                  metadata.GetIdempotencyKey(),
		WatcherVersion:                  metadata.GetWatcherVersion(),
	}
	if b, err := json.Marshal(ko); err == nil {
		if err := os.kc.Publish(ctx, StoreKubernetesObjectsTopic, b); err != nil {
			return err
		}
	} else {
		return err
	}

	// containers
	for _, container := range msg.GetContainers() {
		if container.Name == "" {
			continue
		}

		var stateType string
		if container != nil && container.State != nil {
			stateType = container.State.String()
		}

		var cpuLimits, memLimits, cpuRequests, memRequests string
		if container != nil && container.Resources != nil {
			resources := container.GetResources()
			if resources.Limits != nil {
				cpuLimits = resources.Limits.Cpu
				memLimits = resources.Limits.Memory
			}
			if resources.Requests != nil {
				cpuRequests = resources.Requests.Cpu
				memRequests = resources.Requests.Memory
			}
		}

		kc := &flatKubernetesContainer{
			UID:            msg.GetUid(),
			IdempotencyKey: msg.Metadata.IdempotencyKey,
			Timestamp:      timestamp.Unix(),
			ContainerName:  container.GetName(),
			Image:          container.GetImage(),
			ImageTag:       container.GetImageTag(),
			ContainerID:    container.GetContainerId(),
			RestartsCount:  container.GetRestartsCount(),
			Ready:          container.GetReady(),
			StateType:      stateType,
			CPULimits:      cpuLimits,
			MemoryLimits:   memLimits,
			CPURequests:    cpuRequests,
			MemoryRequests: memRequests,
		}
		if b, err := json.Marshal(kc); err == nil {
			if err := os.kc.Publish(ctx, StoreKubernetesContainersTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// volumes
	for _, volume := range msg.GetVolumes() {
		kv := &flatKubernetesVolume{
			UID:            msg.GetUid(),
			IdempotencyKey: msg.Metadata.IdempotencyKey,
			Timestamp:      timestamp.Unix(),
			VolumeName:     volume.GetName(),
			VolumeType:     volume.GetType(),
		}
		if b, err := json.Marshal(kv); err == nil {
			if err := os.kc.Publish(ctx, StoreKubernetesVolumesTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// conditions
	for _, condition := range msg.GetConditions() {
		knc := &flatKubernetesNodeCondition{
			UID:             msg.GetUid(),
			IdempotencyKey:  msg.Metadata.IdempotencyKey,
			Timestamp:       timestamp.Unix(),
			ConditionType:   condition.GetType(),
			ConditionStatus: condition.GetStatus(),
			Reason:          condition.GetReason(),
			Message:         condition.GetMessage(),
		}
		if b, err := json.Marshal(knc); err == nil {
			if err := os.kc.Publish(ctx, StoreKubernetesNodeConditionsTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// allocatable
	if allocatable != nil {
		kar := &flatKubernetesAllocatableResource{
			UID:              msg.GetUid(),
			IdempotencyKey:   msg.Metadata.IdempotencyKey,
			Timestamp:        timestamp.Unix(),
			CPU:              allocatable.GetCpu(),
			Memory:           allocatable.GetMemory(),
			Pods:             allocatable.GetPods(),
			EphemeralStorage: allocatable.GetEphemeralStorage(),
			Others:           safeMap(allocatable.Others),
		}
		if b, err := json.Marshal(kar); err == nil {
			if err := os.kc.Publish(ctx, StoreKubernetesAllocatableResourcesTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// cluster quota
	if clusterQuota != nil {
		kcq := &flatKubernetesClusterQuota{
			UID:            msg.GetUid(),
			IdempotencyKey: msg.Metadata.IdempotencyKey,
			Timestamp:      timestamp.Unix(),
			QuotaName:      clusterQuota.GetName(),
			QuotaUID:       clusterQuota.GetUid(),
			TotalLimits:    make([]flatResourceTuple, len(clusterQuota.GetTotalLimits())),
			TotalUsage:     make([]flatResourceTuple, len(clusterQuota.GetTotalUsage())),
		}
		for i, limits := range clusterQuota.GetTotalLimits() {
			kcq.TotalLimits[i] = flatResourceTuple{
				IdempotencyKey: msg.Metadata.IdempotencyKey,
				Resource:       limits.GetResource(),
				Value:          limits.GetValue(),
			}
		}
		for i, usage := range clusterQuota.GetTotalUsage() {
			kcq.TotalUsage[i] = flatResourceTuple{
				IdempotencyKey: msg.Metadata.IdempotencyKey,
				Resource:       usage.GetResource(),
				Value:          usage.GetValue(),
			}
		}
		if b, err := json.Marshal(kcq); err == nil {
			if err := os.kc.Publish(ctx, StoreKubernetesClusterQuotasTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}

		// namespace quotas
		for _, quota := range clusterQuota.GetQuotas() {
			knq := &flatKubernetesNamespaceQuota{
				UID:            msg.GetUid(),
				IdempotencyKey: msg.Metadata.IdempotencyKey,
				Timestamp:      timestamp.Unix(),
				Namespace:      msg.GetNamespace(),
				Limits:         make([]flatResourceTuple, len(quota.GetLimits())),
				Usage:          make([]flatResourceTuple, len(quota.GetUsage())),
			}
			for i, limit := range quota.GetLimits() {
				knq.Limits[i] = flatResourceTuple{
					IdempotencyKey: msg.Metadata.IdempotencyKey,
					Resource:       limit.GetResource(),
					Value:          limit.GetValue(),
				}
			}
			for i, usage := range quota.GetUsage() {
				knq.Usage[i] = flatResourceTuple{
					IdempotencyKey: msg.Metadata.IdempotencyKey,
					Resource:       usage.GetResource(),
					Value:          usage.GetValue(),
				}
			}
			if b, err := json.Marshal(knq); err == nil {
				if err := os.kc.Publish(ctx, StoreKubernetesNamespaceQuotasTopic, b); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}
