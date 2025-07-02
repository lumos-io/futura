package pod

import (
	"strings"
	"time"

	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/container"
	"github.com/opisvigilant/futura/watcher/internal/cluster/gvk"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/internal/cluster/service"
	"github.com/opisvigilant/futura/watcher/utils"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"

	conventions "go.opentelemetry.io/otel/semconv/v1.6.1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

const (
	// Keys for pod metadata and entity attributes. These are NOT used by resource attributes.
	podCreationTime = "pod.creation_timestamp"
	podPhase        = "k8s.pod.phase"
	podStatusReason = "k8s.pod.status_reason"
)

// Transform transforms the pod to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new pod fields.
func Transform(pod *corev1.Pod) *corev1.Pod {
	newPod := &corev1.Pod{
		ObjectMeta: metadata.TransformObjectMeta(pod.ObjectMeta),
		Spec: corev1.PodSpec{
			NodeName: pod.Spec.NodeName,
		},
		Status: corev1.PodStatus{
			Phase:    pod.Status.Phase,
			QOSClass: pod.Status.QOSClass,
			Reason:   pod.Status.Reason,
		},
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.ContainerID == "" {
			continue
		}
		newPod.Status.ContainerStatuses = append(newPod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:                 cs.Name,
			Image:                cs.Image,
			ContainerID:          cs.ContainerID,
			RestartCount:         cs.RestartCount,
			Ready:                cs.Ready,
			State:                cs.State,
			LastTerminationState: cs.LastTerminationState,
		})
	}
	for _, c := range pod.Spec.Containers {
		newPod.Spec.Containers = append(newPod.Spec.Containers, corev1.Container{
			Name: c.Name,
			Resources: corev1.ResourceRequirements{
				Requests: c.Resources.Requests,
				Limits:   c.Resources.Limits,
			},
		})
	}
	return newPod
}

func RecordMetrics(pod *corev1.Pod, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp:  timestamppb.New(ts),
		Status:     string(pod.Status.Phase),
		Reason:     string(pod.Status.Reason),
		Namespace:  pod.Namespace,
		NodeName:   pod.Spec.NodeName,
		Name:       pod.Name,
		Uid:        string(pod.UID),
		QosClass:   string(pod.Status.QOSClass),
		Containers: make([]*pbcluster.ContainerSpec, 1),
	}

	for _, c := range pod.Spec.Containers {
		obj.Containers = append(obj.Containers, container.RecordSpecMetrics(c, pod, ts))
	}
	return obj
}

func reasonToInt(reason string) int32 {
	switch reason {
	case "Evicted":
		return 1
	case "NodeAffinity":
		return 2
	case "NodeLost":
		return 3
	case "Shutdown":
		return 4
	case "UnexpectedAdmissionError":
		return 5
	default:
		return 6
	}
}

func phaseToInt(phase corev1.PodPhase) int32 {
	switch phase {
	case corev1.PodPending:
		return 1
	case corev1.PodRunning:
		return 2
	case corev1.PodSucceeded:
		return 3
	case corev1.PodFailed:
		return 4
	case corev1.PodUnknown:
		return 5
	default:
		return 5
	}
}

// GetMetadata returns all metadata associated with the pod.
func GetMetadata(pod *corev1.Pod, mc *metadata.Store) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	meta := utils.MergeStringMaps(map[string]string{}, pod.Labels)

	meta[podCreationTime] = pod.CreationTimestamp.Format(time.RFC3339)
	phase := pod.Status.Phase
	if phase == "" {
		phase = corev1.PodUnknown
	}
	meta[podPhase] = string(phase)
	reason := pod.Status.Reason
	if reason != "" {
		meta[podStatusReason] = reason
	}

	meta[string(conventions.K8SNodeNameKey)] = pod.Spec.NodeName

	for _, or := range pod.OwnerReferences {
		kind := strings.ToLower(or.Kind)
		meta[metadata.GetOTelNameFromKind(kind)] = or.Name
		meta[metadata.GetOTelUIDFromKind(kind)] = string(or.UID)

		// defer syncing replicaset and job workload metadata.
		if or.Kind == constants.K8sKindReplicaSet || or.Kind == constants.K8sKindJob {
			continue
		}
		meta[constants.K8sKeyWorkLoadKind] = or.Kind
		meta[constants.K8sKeyWorkLoadName] = or.Name
	}

	if store := mc.Get(gvk.Service); store != nil {
		meta = utils.MergeStringMaps(meta, service.GetPodServiceTags(pod, store))
	}

	if store := mc.Get(gvk.Job); store != nil {
		meta = utils.MergeStringMaps(meta, collectPodJobProperties(pod, store))
	}

	if store := mc.Get(gvk.ReplicaSet); store != nil {
		meta = utils.MergeStringMaps(meta, collectPodReplicaSetProperties(pod, store))
	}

	meta[constants.K8sKeyNamespaceName] = pod.Namespace
	meta[constants.K8sKeyPodName] = pod.Name

	podID := metadata.ResourceID(pod.UID)
	return metadata.MergeKubernetesMetadataMaps(map[metadata.ResourceID]*metadata.KubernetesMetadata{
		podID: {
			EntityType:    "k8s.pod",
			ResourceIDKey: string(conventions.K8SPodUIDKey),
			ResourceID:    podID,
			Metadata:      meta,
		},
	}, getPodContainerProperties(pod))
}

// collectPodJobProperties checks if pod owner of type Job is cached. Check owners reference
// on Job to see if it was created by a CronJob. Sync metadata accordingly.
func collectPodJobProperties(pod *corev1.Pod, jobStore cache.Store) map[string]string {
	jobRef := utils.FindOwnerWithKind(pod.OwnerReferences, constants.K8sKindJob)
	if jobRef != nil {
		job, exists, err := jobStore.GetByKey(utils.GetIDForCache(pod.Namespace, jobRef.Name))
		if err != nil {
			logError(err, jobRef, pod.UID)
			return nil
		} else if !exists {
			logDebug(jobRef, pod.UID)
			return nil
		}

		jobObj := job.(*batchv1.Job)
		if cronJobRef := utils.FindOwnerWithKind(jobObj.OwnerReferences, constants.K8sKindCronJob); cronJobRef != nil {
			return getWorkloadProperties(cronJobRef, string(conventions.K8SCronJobNameKey))
		}
		return getWorkloadProperties(jobRef, string(conventions.K8SJobNameKey))
	}
	return nil
}

// collectPodReplicaSetProperties checks if pod owner of type ReplicaSet is cached. Check owners reference
// on ReplicaSet to see if it was created by a Deployment. Sync metadata accordingly.
func collectPodReplicaSetProperties(pod *corev1.Pod, replicaSetstore cache.Store) map[string]string {
	rsRef := utils.FindOwnerWithKind(pod.OwnerReferences, constants.K8sKindReplicaSet)
	if rsRef != nil {
		replicaSet, exists, err := replicaSetstore.GetByKey(utils.GetIDForCache(pod.Namespace, rsRef.Name))
		if err != nil {
			logError(err, rsRef, pod.UID)
			return nil
		} else if !exists {
			logDebug(rsRef, pod.UID)
			return nil
		}

		replicaSetObj := replicaSet.(*appsv1.ReplicaSet)
		if deployRef := utils.FindOwnerWithKind(replicaSetObj.OwnerReferences, constants.K8sKindDeployment); deployRef != nil {
			return getWorkloadProperties(deployRef, string(conventions.K8SDeploymentNameKey))
		}
		return getWorkloadProperties(rsRef, string(conventions.K8SReplicaSetNameKey))
	}
	return nil
}

func logDebug(ref *v1.OwnerReference, podUID types.UID) {
	log.Logger.Debug().
		Str(string(conventions.K8SPodUIDKey), string(podUID)).
		Str(string(conventions.K8SJobUIDKey), string(ref.UID)).
		Msg("Resource does not exist in store, properties from it will not be synced.")
}

func logError(err error, ref *v1.OwnerReference, podUID types.UID) {
	log.Logger.Error().
		Msg("Failed to get resource from store, properties from it will not be synced.").
		Str(string(conventions.K8SPodUIDKey), string(podUID)).
		Str(string(conventions.K8SJobUIDKey), string(ref.UID)).
		Err(err)
}

// getWorkloadProperties returns workload metadata for provided owner reference.
func getWorkloadProperties(ref *v1.OwnerReference, labelKey string) map[string]string {
	uidKey := metadata.GetOTelUIDFromKind(strings.ToLower(ref.Kind))
	return map[string]string{
		constants.K8sKeyWorkLoadKind: ref.Kind,
		constants.K8sKeyWorkLoadName: ref.Name,
		labelKey:                     ref.Name,
		uidKey:                       string(ref.UID),
	}
}

func getPodContainerProperties(pod *corev1.Pod) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	km := map[metadata.ResourceID]*metadata.KubernetesMetadata{}
	for _, cs := range pod.Status.ContainerStatuses {
		md := container.GetMetadata(pod, cs)
		km[md.ResourceID] = md
	}
	return km
}
