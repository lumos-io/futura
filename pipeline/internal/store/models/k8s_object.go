package models

import (
	"github.com/opisvigilant/futura/proto/gen/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type KubernetesClusterObject struct {
	Timestamp         *timestamppb.Timestamp `json:"timestamp,omitempty"`
	Type              string                 `json:"type,omitempty"`
	Kind              string                 `json:"kind,omitempty"`
	Namespace         string                 `json:"namespace,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Uid               string                 `json:"uid,omitempty"`
	Labels            map[string]string      `json:"labels,omitempty"`
	Annotations       map[string]string      `json:"annotations,omitempty"`
	NodeName          string                 `json:"node_name,omitempty"`
	Status            string                 `json:"status,omitempty"`
	Phase             string                 `json:"phase,omitempty"`
	RestartCount      int64                  `json:"restart_count,omitempty"`
	OwnerKind         string                 `json:"owner_kind,omitempty"`
	OwnerName         string                 `json:"owner_name,omitempty"`
	Replicas          int64                  `json:"replicas,omitempty"`
	ReadyReplicas     int64                  `json:"ready_replicas,omitempty"`
	AvailableReplicas int64                  `json:"available_replicas,omitempty"`
	UpdatedReplicas   int64                  `json:"updated_replicas,omitempty"`
	CurrentReplicas   int64                  `json:"current_replicas,omitempty"`
	Containers        []*ContainerSpec       `json:"containers,omitempty"`
	Volumes           []*VolumeSpec          `json:"volumes,omitempty"`
	Tolerations       []string               `json:"tolerations,omitempty"`
	Affinity          map[string]string      `json:"affinity,omitempty"`
	Extra             map[string]string      `json:"extra,omitempty"`
	ApiVersion        string                 `json:"api_version,omitempty"`
	// for HPA
	HpaMaxReplicas    int64  `json:"hpa_max_replicas,omitempty"`
	HpaMinReplicas    int64  `json:"hpa_min_replicas,omitempty"`
	HpaScaleTargetRef string `json:"hpa_scale_target_ref,omitempty"`
	// for Job
	JobActive      int64 `json:"job_active,omitempty"`
	JobFailed      int64 `json:"job_failed,omitempty"`
	JobSucceeded   int64 `json:"job_succeeded,omitempty"`
	JobParallelism int64 `json:"job_parallelism,omitempty"`
	JobCompletions int64 `json:"job_completions,omitempty"`
	// for Namespace
	NsPhaseValue int64 `json:"ns_phase_value,omitempty"`
	// for Node
	KubeletVersion          string                `json:"kubelet_version,omitempty"`
	OsType                  string                `json:"os_type,omitempty"`
	OsImage                 string                `json:"os_image,omitempty"`
	ContainerRuntime        string                `json:"container_runtime,omitempty"`
	ContainerRuntimeVersion string                `json:"container_runtime_version,omitempty"`
	Conditions              []*NodeCondition      `json:"conditions,omitempty"`
	Allocatable             *AllocatableResources `json:"allocatable,omitempty"`
	// for Pod
	PodReason string `json:"pod_reason,omitempty"`
	QosClass  string `json:"qos_class,omitempty"`
	// Cluster quota
	ClusterQuota *ClusterResourceQuotaMetadata `json:"cluster_quota,omitempty"`
	// for daemonset
	DaemonsetCurrentNumberScheduled int64               `json:"daemonset_current_number_scheduled,omitempty"`
	DaemonsetDesiredNumberScheduled int64               `json:"daemonset_desired_number_scheduled,omitempty"`
	DaemonsetNumberMisscheduled     int64               `json:"daemonset_number_misscheduled,omitempty"`
	DaemonsetNumberReady            int64               `json:"daemonset_number_ready,omitempty"`
	Metadata                        *common.Metadata    `json:"metadata,omitempty"`
	Enrichment                      *EnrichmentMetadata `json:"enrichment,omitempty"`
}

type ContainerSpec struct {
	Name                 string              `json:"name,omitempty"`
	Image                string              `json:"image,omitempty"`
	Resources            *ContainerResources `json:"resources,omitempty"`
	ContainerId          string              `json:"containerId,omitempty"`
	RestartsCount        int64               `json:"restarts_count,omitempty"`
	Ready                int64               `json:"ready,omitempty"`
	State                *ContainerState     `json:"state,omitempty"`
	LastTerminationState *ContainerState     `json:"last_termination_state,omitempty"`
	ImageTag             string              `json:"image_tag,omitempty"`
}

// CPU/Memory requests and limits for a container.
type ContainerResources struct {
	Limits   *ResourceQuantities `json:"limits,omitempty"`
	Requests *ResourceQuantities `json:"requests,omitempty"`
}

// Quantity strings, e.g. "500m", "128Mi"
type ResourceQuantities struct {
	Cpu              string `json:"cpu,omitempty"`
	Memory           string `json:"memory,omitempty"`
	Storage          string `json:"storage,omitempty"`
	EphemeralStorage string `json:"ephemeral_storage,omitempty"`
}

// Volumes attached to a pod
type VolumeSpec struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"` // EmptyDir, PVC, ConfigMap, etc.

}

type NodeCondition struct {
	Type    string `json:"type,omitempty"`   // e.g. "Ready"
	Status  string `json:"status,omitempty"` // "True", "False", "Unknown"
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type AllocatableResources struct {
	Cpu              string            `json:"cpu,omitempty"`
	Memory           string            `json:"memory,omitempty"`
	Pods             string            `json:"pods,omitempty"`
	EphemeralStorage string            `json:"ephemeral_storage,omitempty"`
	Others           map[string]string `json:"others,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

type ClusterResourceQuotaMetadata struct {
	Name        string            `json:"name,omitempty"`
	Uid         string            `json:"uid,omitempty"`
	TotalLimits []*QuotaResource  `json:"total_limits,omitempty"`
	TotalUsage  []*QuotaResource  `json:"total_usage,omitempty"`
	Quotas      []*NamespaceQuota `json:"quotas,omitempty"`
}

type NamespaceQuota struct {
	Namespace string           `json:"namespace,omitempty"`
	Limits    []*QuotaResource `json:"limits,omitempty"`
	Usage     []*QuotaResource `json:"usage,omitempty"`
}

type QuotaResource struct {
	Resource string `json:"resource,omitempty"` // e.g. "cpu", "memory", "count/pods"
	Value    int64  `json:"value,omitempty"`    // raw string, like "500m" or "2Gi"
}

type EnrichmentMetadata struct {
	OrganizationId uint32 `json:"organization_id,omitempty"`
	ClusterId      int64  `json:"cluster_id,omitempty"`
	K8SVersion     string `json:"k8s_version,omitempty"`
	ReceivedAtUnix int64  `json:"received_at_unix,omitempty"` // UNIX timestamp

}

type Metadata struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	WatcherVersion string `json:"watcher_version,omitempty"`
}
