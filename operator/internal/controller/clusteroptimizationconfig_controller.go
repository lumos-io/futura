/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"google.golang.org/grpc"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	futurav1 "io.lumos/futura/api/v1"
	"io.lumos/futura/internal/watcher"

	pbop "github.com/opisvigilant/futura/proto/gen/operator"
)

// ClusterOptimizationConfigReconciler reconciles a ClusterOptimizationConfig object
type ClusterOptimizationConfigReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	grpcClient pbop.FuturaOptimizerClient
}

func NewClusterOptimizationConfigReconciler(c client.Client, scheme *runtime.Scheme, grpcConn *grpc.ClientConn) *ClusterOptimizationConfigReconciler {
	return &ClusterOptimizationConfigReconciler{
		Client:     c,
		Scheme:     scheme,
		grpcClient: pbop.NewFuturaOptimizerClient(grpcConn),
	}
}

// +kubebuilder:rbac:groups=futura.io.lumos,resources=clusteroptimizationconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=futura.io.lumos,resources=clusteroptimizationconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=futura.io.lumos,resources=clusteroptimizationconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterOptimizationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var config futurav1.ClusterOptimizationConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	grpcReq := &pbop.ClusterOptimizationConfigRequest{
		Config: &pbop.ClusterOptimizationConfig{
			ApiKey:                 config.Spec.ApiKey,
			CostSensitivity:        config.Spec.CostOptimization.CostSensitivity,
			MonthlyBudget:          config.Spec.CostOptimization.MaxMonthlyBudgetUSD,
			PreferredInstanceTypes: config.Spec.CostOptimization.PreferredInstanceTypes,
			AllowSpot:              config.Spec.CostOptimization.SpotInstanceAllowed,
			MaxSpotPercentage:      config.Spec.CostOptimization.MaxSpotPercentage,
		},
	}

	resp, err := r.grpcClient.SyncClusterOptimizationConfig(ctx, grpcReq)
	if err != nil || !resp.Success {
		logger.Error(err, "Failed to sync ClusterOptimizationConfig via gRPC", "message", resp.GetMessage())
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status to reflect sync success
	config.Status.LastSynced = metav1.Now()
	config.Status.Synced = true
	if err := r.Status().Update(ctx, &config); err != nil {
		return ctrl.Result{}, err
	}

	namespace := "default" // or make configurable

	// Build watcher ConfigMap (make sure you have BuildWatcherConfigMap func returning *corev1.ConfigMap)
	configMap := watcher.BuildWatcherConfigMap(namespace, config.Spec.ApiKey)
	if changed, err := EnsureConfigMap(ctx, r.Client, r.Scheme, &config, configMap); err != nil {
		logger.Error(err, "Failed to ensure watcher ConfigMap")
		return ctrl.Result{}, err
	} else if changed {
		logger.Info("Watcher ConfigMap created or updated")
	}

	// Build ServiceAccount
	sa := watcher.BuildWatcherServiceAccount(namespace)
	if changed, err := EnsureServiceAccount(ctx, r.Client, r.Scheme, &config, sa); err != nil {
		logger.Error(err, "Failed to ensure watcher ServiceAccount")
		return ctrl.Result{}, err
	} else if changed {
		logger.Info("Watcher ServiceAccount created or updated")
	}

	// Build ClusterRole
	cr := watcher.BuildWatcherClusterRole()
	if changed, err := EnsureClusterRole(ctx, r.Client, r.Scheme, &config, cr); err != nil {
		logger.Error(err, "Failed to ensure watcher ClusterRole")
		return ctrl.Result{}, err
	} else if changed {
		logger.Info("Watcher ClusterRole created or updated")
	}

	// Build ClusterRoleBinding
	crb := watcher.BuildWatcherClusterRoleBinding(namespace)
	if changed, err := EnsureClusterRoleBinding(ctx, r.Client, r.Scheme, &config, crb); err != nil {
		logger.Error(err, "Failed to ensure watcher ClusterRoleBinding")
		return ctrl.Result{}, err
	} else if changed {
		logger.Info("Watcher ClusterRoleBinding created or updated")
	}

	// Build DaemonSet
	ds := watcher.BuildWatcherDaemonSet(namespace)
	if changed, err := EnsureDaemonSet(ctx, r.Client, r.Scheme, &config, ds); err != nil {
		logger.Error(err, "Failed to ensure watcher DaemonSet")
		return ctrl.Result{}, err
	} else if changed {
		logger.Info("Watcher DaemonSet created or updated")
	}

	logger.Info("ClusterOptimizationConfig synced successfully and watcher installed", "name", config.Name)
	return ctrl.Result{}, nil
}

// EnsureDaemonSet ensures the watcher DaemonSet is created and up-to-date.
// Returns (changed, error).
func EnsureDaemonSet(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, desired *appsv1.DaemonSet) (bool, error) {
	logger := log.FromContext(ctx)

	// Set owner reference for garbage collection
	if err := ctrl.SetControllerReference(owner, desired, scheme); err != nil {
		return false, err
	}

	existing := &appsv1.DaemonSet{}
	err := c.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, desired); err != nil {
				return false, err
			}
			logger.Info("Created watcher DaemonSet", "name", desired.Name)
			return true, nil
		}
		return false, err
	}

	// Preserve metadata that must not be overwritten
	desired.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
	desired.ObjectMeta.UID = existing.ObjectMeta.UID
	desired.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp
	desired.ObjectMeta.ManagedFields = existing.ObjectMeta.ManagedFields
	desired.ObjectMeta.Generation = existing.ObjectMeta.Generation

	// Compare Spec and update if changed
	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) {
		existing.Spec = desired.Spec
		if err := c.Update(ctx, existing); err != nil {
			return false, err
		}
		logger.Info("Updated watcher DaemonSet", "name", desired.Name)
		return true, nil
	}

	logger.Info("Watcher DaemonSet up-to-date", "name", desired.Name)
	return false, nil
}

// EnsureConfigMap ensures the watcher ConfigMap exists and is up-to-date.
func EnsureConfigMap(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, cm *corev1.ConfigMap) (bool, error) {
	logger := log.FromContext(ctx)

	if err := ctrl.SetControllerReference(owner, cm, scheme); err != nil {
		return false, err
	}

	existing := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, cm); err != nil {
				return false, err
			}
			logger.Info("Created watcher ConfigMap", "name", cm.Name)
			return true, nil
		}
		return false, err
	}

	// Preserve metadata
	cm.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
	cm.ObjectMeta.UID = existing.ObjectMeta.UID
	cm.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp
	cm.ObjectMeta.ManagedFields = existing.ObjectMeta.ManagedFields
	cm.ObjectMeta.Generation = existing.ObjectMeta.Generation

	if !equality.Semantic.DeepEqual(existing.Data, cm.Data) {
		existing.Data = cm.Data
		if err := c.Update(ctx, existing); err != nil {
			return false, err
		}
		logger.Info("Updated watcher ConfigMap", "name", cm.Name)
		return true, nil
	}

	logger.Info("Watcher ConfigMap up-to-date", "name", cm.Name)
	return false, nil
}

// EnsureServiceAccount ensures the watcher ServiceAccount exists and is up-to-date.
func EnsureServiceAccount(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, sa *corev1.ServiceAccount) (bool, error) {
	logger := log.FromContext(ctx)

	if err := ctrl.SetControllerReference(owner, sa, scheme); err != nil {
		return false, err
	}

	existing := &corev1.ServiceAccount{}
	err := c.Get(ctx, types.NamespacedName{Name: sa.Name, Namespace: sa.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, sa); err != nil {
				return false, err
			}
			logger.Info("Created watcher ServiceAccount", "name", sa.Name)
			return true, nil
		}
		return false, err
	}

	// Preserve metadata
	sa.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
	sa.ObjectMeta.UID = existing.ObjectMeta.UID
	sa.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp
	sa.ObjectMeta.ManagedFields = existing.ObjectMeta.ManagedFields
	sa.ObjectMeta.Generation = existing.ObjectMeta.Generation

	// Compare fields that might change, e.g., AutomountServiceAccountToken
	if existing.AutomountServiceAccountToken == nil || (sa.AutomountServiceAccountToken != nil && *existing.AutomountServiceAccountToken != *sa.AutomountServiceAccountToken) {
		existing.AutomountServiceAccountToken = sa.AutomountServiceAccountToken
		if err := c.Update(ctx, existing); err != nil {
			return false, err
		}
		logger.Info("Updated watcher ServiceAccount", "name", sa.Name)
		return true, nil
	}

	logger.Info("Watcher ServiceAccount up-to-date", "name", sa.Name)
	return false, nil
}

// EnsureClusterRole ensures the watcher ClusterRole exists and is up-to-date.
func EnsureClusterRole(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, cr *rbacv1.ClusterRole) (bool, error) {
	logger := log.FromContext(ctx)

	if err := ctrl.SetControllerReference(owner, cr, scheme); err != nil {
		return false, err
	}

	existing := &rbacv1.ClusterRole{}
	err := c.Get(ctx, types.NamespacedName{Name: cr.Name}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, cr); err != nil {
				return false, err
			}
			logger.Info("Created watcher ClusterRole", "name", cr.Name)
			return true, nil
		}
		return false, err
	}

	// Preserve metadata
	cr.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
	cr.ObjectMeta.UID = existing.ObjectMeta.UID
	cr.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp
	cr.ObjectMeta.ManagedFields = existing.ObjectMeta.ManagedFields
	cr.ObjectMeta.Generation = existing.ObjectMeta.Generation

	if !equality.Semantic.DeepEqual(existing.Rules, cr.Rules) {
		existing.Rules = cr.Rules
		if err := c.Update(ctx, existing); err != nil {
			return false, err
		}
		logger.Info("Updated watcher ClusterRole", "name", cr.Name)
		return true, nil
	}

	logger.Info("Watcher ClusterRole up-to-date", "name", cr.Name)
	return false, nil
}

// EnsureClusterRoleBinding ensures the watcher ClusterRoleBinding exists and is up-to-date.
func EnsureClusterRoleBinding(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, crb *rbacv1.ClusterRoleBinding) (bool, error) {
	logger := log.FromContext(ctx)

	if err := ctrl.SetControllerReference(owner, crb, scheme); err != nil {
		return false, err
	}

	existing := &rbacv1.ClusterRoleBinding{}
	err := c.Get(ctx, types.NamespacedName{Name: crb.Name}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, crb); err != nil {
				return false, err
			}
			logger.Info("Created watcher ClusterRoleBinding", "name", crb.Name)
			return true, nil
		}
		return false, err
	}

	// Preserve metadata
	crb.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion
	crb.ObjectMeta.UID = existing.ObjectMeta.UID
	crb.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp
	crb.ObjectMeta.ManagedFields = existing.ObjectMeta.ManagedFields
	crb.ObjectMeta.Generation = existing.ObjectMeta.Generation

	if !equality.Semantic.DeepEqual(existing.Subjects, crb.Subjects) || !equality.Semantic.DeepEqual(existing.RoleRef, crb.RoleRef) {
		existing.Subjects = crb.Subjects
		existing.RoleRef = crb.RoleRef
		if err := c.Update(ctx, existing); err != nil {
			return false, err
		}
		logger.Info("Updated watcher ClusterRoleBinding", "name", crb.Name)
		return true, nil
	}

	logger.Info("Watcher ClusterRoleBinding up-to-date", "name", crb.Name)
	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterOptimizationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&futurav1.ClusterOptimizationConfig{}).
		Named("clusteroptimizationconfig").
		Complete(r)
}
