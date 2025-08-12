package watcher

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildWatcherClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	labels := map[string]string{
		"app.kubernetes.io/name":     "watcher",
		"app.kubernetes.io/instance": "watcher",
	}

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "watcher",
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "watcher",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "watcher",
				Namespace: namespace,
			},
		},
	}
}
