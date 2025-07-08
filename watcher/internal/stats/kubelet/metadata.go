package kubelet

import (
	v1 "k8s.io/api/core/v1"
)

var (
	Type = "kubeletstats"
)

type Metadata struct {
	// Labels       map[MetadataLabel]bool
	PodsMetadata *v1.PodList
	// DetailedPVCResourceSetter func(rb *metadata.ResourceBuilder, volCacheID, volumeClaim, namespace string) error
	podResources       map[string]resources
	containerResources map[string]resources
	nodeInfo           NodeInfo
}

type resources struct {
	cpuRequest    float64
	cpuLimit      float64
	memoryRequest int64
	memoryLimit   int64
}

type NodeInfo struct {
	Name string
	// node's CPU capacity in cores
	CPUCapacity float64
	// node's Memory capacity in bytes
	MemoryCapacity float64
}

func getContainerResources(r *v1.ResourceRequirements) resources {
	if r == nil {
		return resources{}
	}

	return resources{
		cpuRequest:    r.Requests.Cpu().AsApproximateFloat64(),
		cpuLimit:      r.Limits.Cpu().AsApproximateFloat64(),
		memoryRequest: r.Requests.Memory().Value(),
		memoryLimit:   r.Limits.Memory().Value(),
	}
}

func NewMetadata(podsMetadata *v1.PodList, nodeInfo NodeInfo) Metadata {
	m := Metadata{
		// Labels:       getLabelsMap(labels),
		PodsMetadata: podsMetadata,
		// DetailedPVCResourceSetter: detailedPVCResourceSetter,
		podResources:       make(map[string]resources),
		containerResources: make(map[string]resources),
		nodeInfo:           nodeInfo,
	}

	if podsMetadata != nil {
		for _, pod := range podsMetadata.Items {
			var podResource resources
			allContainersCPULimitsDefined := true
			allContainersCPURequestsDefined := true
			allContainersMemoryLimitsDefined := true
			allContainersMemoryRequestsDefined := true
			for i := range pod.Spec.Containers {
				container := pod.Spec.Containers[i]
				containerResource := getContainerResources(&container.Resources)

				if allContainersCPULimitsDefined && containerResource.cpuLimit == 0 {
					allContainersCPULimitsDefined = false
					podResource.cpuLimit = 0
				}
				if allContainersCPURequestsDefined && containerResource.cpuRequest == 0 {
					allContainersCPURequestsDefined = false
					podResource.cpuRequest = 0
				}
				if allContainersMemoryLimitsDefined && containerResource.memoryLimit == 0 {
					allContainersMemoryLimitsDefined = false
					podResource.memoryLimit = 0
				}
				if allContainersMemoryRequestsDefined && containerResource.memoryRequest == 0 {
					allContainersMemoryRequestsDefined = false
					podResource.memoryRequest = 0
				}

				if allContainersCPULimitsDefined {
					podResource.cpuLimit += containerResource.cpuLimit
				}
				if allContainersCPURequestsDefined {
					podResource.cpuRequest += containerResource.cpuRequest
				}
				if allContainersMemoryLimitsDefined {
					podResource.memoryLimit += containerResource.memoryLimit
				}
				if allContainersMemoryRequestsDefined {
					podResource.memoryRequest += containerResource.memoryRequest
				}

				m.containerResources[string(pod.UID)+container.Name] = containerResource
			}
			m.podResources[string(pod.UID)] = podResource
		}
	}

	return m
}
