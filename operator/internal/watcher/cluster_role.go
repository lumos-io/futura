package watcher

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildWatcherClusterRole() *rbacv1.ClusterRole {
	labels := map[string]string{
		"app.kubernetes.io/name":     "watcher",
		"app.kubernetes.io/instance": "watcher",
	}

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "watcher",
			Labels: labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events", "namespaces", "namespaces/status", "nodes", "nodes/spec", "pods", "pods/status", "replicationcontrollers", "replicationcontrollers/status", "resourcequotas", "services"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"daemonsets", "deployments", "replicasets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes/stats"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes/proxy"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}
