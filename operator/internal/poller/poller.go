package poller

import (
	"context"
	"fmt"
	"time"

	pbeg "github.com/opisvigilant/futura/proto/gen/engine"
	futurav1 "io.lumos/futura/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/grpc"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Poller struct {
	grpcClient pbeg.FuturaOptimizerClient
}

func New(grpcConn *grpc.ClientConn) (*Poller, error) {
	return &Poller{
		grpcClient: pbeg.NewFuturaOptimizerClient(grpcConn),
	}, nil
}

func (p *Poller) PollOptimizer(c client.Client, scheme *runtime.Scheme, ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Poll interval
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.fetchAndApplyDecisions(ctx, c, scheme)
		}
	}
}

func (p *Poller) fetchAndApplyDecisions(ctx context.Context, c client.Client, scheme *runtime.Scheme) {
	logger := log.FromContext(ctx)

	// You might load ClusterOptimizationConfig to get the API key
	var configs futurav1.ClusterOptimizationConfigList
	if err := c.List(ctx, &configs); err != nil {
		logger.Error(err, "Failed to list ClusterOptimizationConfigs")
		return
	}

	if len(configs.Items) == 0 {
		return
	}

	apiKey := configs.Items[0].Spec.ApiKey

	// Call gRPC
	req := &pbeg.DecisionRequest{ClusterId: apiKey}
	resp, err := p.grpcClient.GetOptimizationDecision(ctx, req)
	if err != nil {
		logger.Error(err, "Failed to fetch optimization decision from backend")
		return
	}

	logger.Info("Received optimization decision", "decision_id", resp.DecisionId)

	// Apply actions
	for _, action := range resp.Actions {
		switch action.Type {
		case "HPA_SCALE":
			if err := p.applyHPAScale(ctx, c, resp.Target, action.GetHpaScale().Replicas); err != nil {
				logger.Error(err, "Failed to apply HPA scale action")
			}
		case "KARPENTER_PROVISION":
			if err := p.applyKarpenterProvision(ctx, c, action.GetKarpenterProvision()); err != nil {
				logger.Error(err, "Failed to apply Karpenter provision action")
			}
		case "VPA_RECOMMEND":
			if err := p.applyVPARecommendation(ctx, c, resp.Target, action.GetVpaRecommend()); err != nil {
				logger.Error(err, "Failed to apply VPA recommendation")
			}
		default:
			logger.Info("Unknown action type, skipping", "type", action.Type)
		}
	}
}

func (p *Poller) applyHPAScale(ctx context.Context, c client.Client, target *pbeg.TargetRef, replicas int32) error {
	dep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: target.Namespace,
			Name:      target.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	patch := client.Apply
	return c.Patch(ctx, &dep, patch, client.ForceOwnership, client.FieldOwner("futura-optimizer"))
}

func (p *Poller) applyKarpenterProvision(ctx context.Context, c client.Client, provision *pbeg.KarpenterProvisionAction) error {
	// This would create/update a Karpenter Provisioner CR
	// For simplicity, just log
	fmt.Printf("Would provision %d nodes of types %v (%s)\n", provision.Count, provision.InstanceTypes, provision.CapacityType)
	return nil
}

func (p *Poller) applyVPARecommendation(ctx context.Context, c client.Client, target *pbeg.TargetRef, vpa *pbeg.VpaRecommendAction) error {
	cpuWithBuffer := int64(float64(vpa.CpuRequestMcpu) * 1.5) // multiply by 1.5
	cpuQty := resource.MustParse(fmt.Sprintf("%dm", vpa.CpuRequestMcpu))
	cpuQtyWithBuffer := resource.MustParse(fmt.Sprintf("%dm", cpuWithBuffer))

	memWithBuffer := int64(float64(vpa.MemoryMib) * 1.5) // multiply by 1.5
	memQty := resource.MustParse(fmt.Sprintf("%dMi", vpa.MemoryMib))
	memQtyWithBuffer := resource.MustParse(fmt.Sprintf("%dMi", memWithBuffer))

	dep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: target.Namespace,
			Name:      target.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: vpa.Container,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    cpuQty,
								corev1.ResourceMemory: memQty,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    cpuQtyWithBuffer,
								corev1.ResourceMemory: memQtyWithBuffer,
							},
						},
					}},
				},
			},
		},
	}
	return c.Patch(ctx, &dep, client.Apply, client.ForceOwnership, client.FieldOwner("futura-optimizer"))
}
