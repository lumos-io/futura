package cluster

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// transformObject transforms the k8s object by removing the data that is not utilized by the receiver.
// Only highly utilized objects are transformed here while others are kept as is.
func transformObject(object any) (any, error) {
	switch o := object.(type) {
	case *corev1.Pod:
		return pod.Transform(o), nil
	case *corev1.Node:
		return node.Transform(o), nil
	case *appsv1.ReplicaSet:
		return replicaset.Transform(o), nil
	case *batchv1.Job:
		return jobs.Transform(o), nil
	case *appsv1.Deployment:
		return deployment.Transform(o), nil
	case *appsv1.DaemonSet:
		return daemonset.Transform(o), nil
	case *appsv1.StatefulSet:
		return statefulset.Transform(o), nil
	case *corev1.Service:
		return service.Transform(o), nil
	}
	return object, nil
}
