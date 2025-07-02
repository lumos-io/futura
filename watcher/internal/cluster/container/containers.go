package container

import (
	"fmt"
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/rs/zerolog/log"
	conventions "go.opentelemetry.io/otel/semconv/v1.6.1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/utils"
)

const (
	// Keys for container metadata used for entity attributes.
	containerKeyStatus         = "container.status"
	containerKeyStatusReason   = "container.status.reason"
	containerCreationTimestamp = "container.creation_timestamp"
	containerName              = "k8s.container.name"
	containerImageName         = "container.image.name"
	containerImageTag          = "container.image.tag"

	// Values for container metadata
	containerStatusRunning    = "running"
	containerStatusWaiting    = "waiting"
	containerStatusTerminated = "terminated"
)

// RecordSpecMetrics metricizes values from the container spec.
// This includes values like resource requests and limits.
func RecordSpecMetrics(c corev1.Container, pod *corev1.Pod, ts time.Time) *pbcluster.ContainerSpec {
	obj := &pbcluster.ContainerSpec{}
	for k, r := range c.Resources.Requests {
		//exhaustive:ignore
		switch k {
		case corev1.ResourceCPU:
			obj.Resources.Requests.Cpu = fmt.Sprintf("%v", float64(r.MilliValue())/1000.0)
		case corev1.ResourceMemory:
			obj.Resources.Requests.Memory = fmt.Sprintf("%v", r.Value())
		case corev1.ResourceStorage:
			obj.Resources.Requests.Storage = fmt.Sprintf("%v", r.Value())
		case corev1.ResourceEphemeralStorage:
			obj.Resources.Requests.EphemeralStorage = fmt.Sprintf("%v", r.Value())
		default:
			log.Logger.Debug().Msgf("unsupported request type `%v`", k)
		}
	}
	for k, l := range c.Resources.Limits {
		//exhaustive:ignore
		switch k {
		case corev1.ResourceCPU:
			obj.Resources.Limits.Cpu = fmt.Sprintf("%v", float64(l.MilliValue())/1000.0)
		case corev1.ResourceMemory:
			obj.Resources.Limits.Memory = fmt.Sprintf("%v", l.Value())
		case corev1.ResourceStorage:
			obj.Resources.Limits.Storage = fmt.Sprintf("%v", l.Value())
		case corev1.ResourceEphemeralStorage:
			obj.Resources.Limits.EphemeralStorage = fmt.Sprintf("%v", l.Value())
		default:
			log.Logger.Debug().Msgf("unsupported request type `%v`", k)
		}
	}

	var containerID string
	var imageStr string
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == c.Name {
			obj.ContainerID = cs.ContainerID
			obj.Image = cs.Image
			obj.RestartsCount = int64(cs.RestartCount)
			obj.Ready = boolToInt64(cs.Ready)

			if cs.LastTerminationState.Terminated != nil {
				obj.LastTerminationState.Terminated.Reason = cs.LastTerminationState.Terminated.Reason
			}
			break
		}
	}

	rb.SetK8sPodUID(string(pod.UID))
	rb.SetK8sPodName(pod.Name)
	rb.SetK8sNodeName(pod.Spec.NodeName)
	rb.SetK8sNamespaceName(pod.Namespace)
	rb.SetContainerID(utils.StripContainerID(containerID))
	rb.SetK8sContainerName(c.Name)
	image, err := docker.ParseImageName(imageStr)
	if err != nil {
		docker.LogParseError(err, imageStr, logger)
	} else {
		rb.SetContainerImageName(image.Repository)
		rb.SetContainerImageTag(image.Tag)
	}
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}

func GetMetadata(pod *corev1.Pod, cs corev1.ContainerStatus, logger *zap.Logger) *metadata.KubernetesMetadata {
	mdata := map[string]string{}

	imageStr := cs.Image
	image, err := docker.ParseImageName(cs.Image)
	if err != nil {
		docker.LogParseError(err, imageStr, logger)
	} else {
		mdata[containerImageName] = image.Repository
		mdata[containerImageTag] = image.Tag
	}
	mdata[containerName] = cs.Name
	mdata[constants.K8sKeyPodName] = pod.Name
	mdata[constants.K8sKeyPodUID] = string(pod.UID)
	mdata[constants.K8sKeyNamespaceName] = pod.Namespace
	mdata[constants.K8sKeyNodeName] = pod.Spec.NodeName

	if cs.State.Running != nil {
		mdata[containerKeyStatus] = containerStatusRunning
		if !cs.State.Running.StartedAt.IsZero() {
			mdata[containerCreationTimestamp] = cs.State.Running.StartedAt.Format(time.RFC3339)
		}
	}

	if cs.State.Terminated != nil {
		mdata[containerKeyStatus] = containerStatusTerminated
		mdata[containerKeyStatusReason] = cs.State.Terminated.Reason
		if !cs.State.Terminated.StartedAt.IsZero() {
			mdata[containerCreationTimestamp] = cs.State.Terminated.StartedAt.Format(time.RFC3339)
		}
	}

	if cs.State.Waiting != nil {
		mdata[containerKeyStatus] = containerStatusWaiting
		mdata[containerKeyStatusReason] = cs.State.Waiting.Reason
	}

	return &metadata.KubernetesMetadata{
		EntityType:    "container",
		ResourceIDKey: string(conventions.ContainerIDKey),
		ResourceID:    metadata.ResourceID(utils.StripContainerID(cs.ContainerID)),
		Metadata:      mdata,
	}
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
