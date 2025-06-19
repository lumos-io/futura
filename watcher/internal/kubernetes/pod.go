package kubernetes

import (
	"github.com/opisvigilant/futura/watcher/internal/models"
	corev1 "k8s.io/api/core/v1"
)

type Container struct {
	Name      string               `json:"name"`
	Namespace string               `json:"namespace"`
	PodUID    string               `json:"pod"` // Pod UID
	Image     string               `json:"image"`
	Ports     []models.AddressPort `json:"ports"`
}

func getContainers(pod *corev1.Pod) []*Container {
	containers := make([]*Container, 0)

	for _, container := range pod.Spec.Containers {
		ports := make([]models.AddressPort, 0)

		for _, port := range container.Ports {
			ports = append(ports, models.AddressPort{
				Port:     port.ContainerPort,
				Protocol: string(port.Protocol),
				Name:     port.Name,
			})
		}

		containers = append(containers, &Container{
			Name:      container.Name,
			Namespace: pod.Namespace,
			PodUID:    string(pod.UID),
			Image:     container.Image,
			Ports:     ports,
		})
	}
	return containers
}

func getOnAddPodFunc(ch chan any) func(any) {
	return func(obj any) {
		pod := obj.(*corev1.Pod)
		containers := getContainers(pod)

		ch <- ResourceMessage{
			ResourceType: POD,
			EventType:    ADD,
			Object:       obj,
		}

		for _, container := range containers {
			ch <- ResourceMessage{
				ResourceType: CONTAINER,
				EventType:    ADD,
				Object:       container,
			}
		}
	}
}

func getOnUpdatePodFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		pod := newObj.(*corev1.Pod)

		containers := getContainers(pod)
		ch <- ResourceMessage{
			ResourceType: POD,
			EventType:    UPDATE,
			Object:       newObj,
		}
		for _, container := range containers {
			ch <- ResourceMessage{
				ResourceType: CONTAINER,
				EventType:    UPDATE,
				Object:       container,
			}
		}
	}
}

func getOnDeletePodFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: POD,
			EventType:    DELETE,
			Object:       obj,
		}

		// no need to delete containers, they will be deleted automatically
	}
}
