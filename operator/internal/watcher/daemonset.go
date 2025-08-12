package watcher

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildWatcherDaemonSet(namespace string) *appsv1.DaemonSet {
	labels := map[string]string{
		"app.kubernetes.io/name":     "watcher",
		"app.kubernetes.io/instance": "watcher",
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "watcher",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyAlways,
					ServiceAccountName: "watcher",
					Containers: []corev1.Container{
						{
							Name:  "watcher",
							Image: "ko://github.com/opisvigilant/futura/watcher",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Requests: corev1.ResourceList{},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "configuration",
									MountPath: "/opt/watcher/config.toml",
									SubPath:   "config.toml",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "configuration",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "watcher-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
