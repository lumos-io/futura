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
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	futurav1 "io.lumos/futura/api/v1"

	pbop "github.com/opisvigilant/futura/proto/gen/operator"
)

// ServiceLevelObjectiveReconciler reconciles a ServiceLevelObjective object
type ServiceLevelObjectiveReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	grpcClient pbop.FuturaOptimizerClient
}

func NewSLOReconciler(k8sClient client.Client, grpcConn *grpc.ClientConn) *ServiceLevelObjectiveReconciler {
	return &ServiceLevelObjectiveReconciler{
		Client:     k8sClient,
		grpcClient: pbop.NewFuturaOptimizerClient(grpcConn),
	}
}

// +kubebuilder:rbac:groups=futura.io.lumos,resources=servicelevelobjectives,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=futura.io.lumos,resources=servicelevelobjectives/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=futura.io.lumos,resources=servicelevelobjectives/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceLevelObjectiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get ClusterOptimizationConfig (singleton)
	var config futurav1.ClusterOptimizationConfig
	err := r.Get(ctx, types.NamespacedName{
		Name:      "cluster-optimization-config", // fixed name
		Namespace: "futura-system",               // fixed namespace
	}, &config)
	if err != nil {
		// Optionally requeue if not found
		logger.Error(err, "ClusterOptimizationConfig not found")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	var slo futurav1.ServiceLevelObjective
	if err := r.Get(ctx, req.NamespacedName, &slo); err != nil {
		if errors.IsNotFound(err) {
			// Object deleted — handle cleanup if needed
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, err
	}

	grpcReq := &pbop.SyncSLORequest{
		ApiKey: config.Spec.ApiKey,
		Slo: &pbop.ServiceLevelObjective{
			ServiceName:      slo.Spec.ServiceName,
			TargetP95Latency: slo.Spec.TargetP95Latency,
			TargetErrorRate:  slo.Spec.TargetErrorRate,
			TargetThroughput: slo.Spec.TargetThroughput,
			Priority:         slo.Spec.Priority,
			LastUpdated:      timestamppb.New(slo.Status.LastUpdated.Time),
		},
	}

	// Call gRPC backend
	resp, err := r.grpcClient.SyncServiceLevelObjective(ctx, grpcReq)
	if err != nil || !resp.Success {
		logger.Error(err, "Failed to sync SLO via gRPC", "message", resp.GetMessage())
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status
	slo.Status.LastUpdated = metav1.Now()
	slo.Status.Synced = true
	if err := r.Status().Update(ctx, &slo); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("SLO synced successfully via gRPC", "service", slo.Spec.ServiceName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceLevelObjectiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&futurav1.ServiceLevelObjective{}).
		Named("servicelevelobjective").
		Complete(r)
}
