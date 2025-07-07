package kubelet

import (
	"fmt"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func getContainerResource(rb *metadata.ResourceBuilder, sPod stats.PodStats, sContainer stats.ContainerStats, k8sMetadata Metadata) error {
	rb.SetK8sPodUID(sPod.PodRef.UID)
	rb.SetK8sPodName(sPod.PodRef.Name)
	rb.SetK8sNamespaceName(sPod.PodRef.Namespace)
	rb.SetK8sContainerName(sContainer.Name)

	err := k8sMetadata.setExtraResources(rb, sPod.PodRef, MetadataLabelContainerID, sContainer.Name)
	if err != nil {
		return fmt.Errorf("failed to set extra labels from metadata: %w", err)
	}

	return nil
}

func getVolumeResourceOptions(rb *metadata.ResourceBuilder, sPod stats.PodStats, vs stats.VolumeStats, k8sMetadata Metadata) error {
	rb.SetK8sPodUID(sPod.PodRef.UID)
	rb.SetK8sPodName(sPod.PodRef.Name)
	rb.SetK8sNamespaceName(sPod.PodRef.Namespace)
	rb.SetK8sVolumeName(vs.Name)

	err := k8sMetadata.setExtraResources(rb, sPod.PodRef, MetadataLabelVolumeType, vs.Name)
	if err != nil {
		return fmt.Errorf("failed to set extra labels from metadata: %w", err)
	}

	return nil
}
