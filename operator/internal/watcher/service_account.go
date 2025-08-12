package watcher

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func BuildWatcherServiceAccount(namespace string) *corev1.ServiceAccount {
	labels := map[string]string{
		"app.kubernetes.io/name":     "watcher",
		"app.kubernetes.io/instance": "watcher",
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "watcher",
			Namespace: namespace,
			Labels:    labels,
		},
		AutomountServiceAccountToken: ptr.To(true),
	}
}
