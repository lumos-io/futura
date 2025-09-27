package poller

import (
	"context"
	"fmt"
	"strings"
	"time"

	pbeg "github.com/opisvigilant/futura/proto/gen/engine"
	"io.lumos/futura/internal/cloudprovider"
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
	grpcClient            pbeg.RecommendationServiceClient
	cloudProviderManager  *cloudprovider.CloudProviderManager
}

func New(grpcConn *grpc.ClientConn, kubeClient client.Client) (*Poller, error) {
	cloudManager := cloudprovider.NewCloudProviderManager(kubeClient)

	// Register supported cloud providers
	cloudManager.RegisterProvider(cloudprovider.NewAWSProvider("us-east-1")) // TODO: Make region configurable
	cloudManager.RegisterProvider(cloudprovider.NewKindProvider("kind"))
	// TODO: Add GCP, Azure, Alibaba, DigitalOcean providers

	return &Poller{
		grpcClient:           pbeg.NewRecommendationServiceClient(grpcConn),
		cloudProviderManager: cloudManager,
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

// fetchAndApplyDecisions gets the recommendations from the server and initiate the action that will
// be applied to the target pod.
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

	// Call gRPC for app recommendations
	req := &pbeg.RecommendationAppRequest{App: &pbeg.AppRef{
		ApiKey:    apiKey,
		Namespace: "",
		AppName:   "",
		Kind:      pbeg.WorkloadKind_DEPLOYMENT,
	}}
	resp, err := p.grpcClient.GetAppRecommendation(ctx, req)
	if err != nil {
		logger.Error(err, "Failed to fetch optimization decision from backend")
		return
	}

	logger.Info("Received optimization decision", "decision_id", resp.DecisionId)

	// Apply app actions
	for _, action := range resp.Plan {
		switch action.Type {
		case "HPA_SCALE":
			if err := p.applyHPAScale(ctx, c, req.App, action.GetHpaScale().Replicas); err != nil {
				logger.Error(err, "Failed to apply HPA scale action")
			}
		case "VPA_RECOMMEND":
			if err := p.applyVPARecommendation(ctx, c, req.App, action.GetVpaRecommend()); err != nil {
				logger.Error(err, "Failed to apply VPA recommendation")
			}
		default:
			logger.Info("Unknown action type, skipping", "type", action.Type)
		}
	}

	// Also call cluster recommendations
	clusterReq := &pbeg.RecommendationClusterRequest{Cluster: &pbeg.ClusterRef{
		ApiKey: apiKey,
	}}
	clusterResp, err := p.grpcClient.GetClusterRecommendation(ctx, clusterReq)
	if err != nil {
		logger.Error(err, "Failed to fetch cluster optimization decision from backend")
	} else {
		logger.Info("Received cluster optimization decision",
			"decision_id", clusterResp.DecisionId,
			"action_type", clusterResp.Plan.ActionType,
			"confidence", clusterResp.Plan.Confidence)

		if err := p.applyClusterAction(ctx, c, clusterResp.Plan); err != nil {
			logger.Error(err, "Failed to apply cluster action", "action_type", clusterResp.Plan.ActionType)
		}
	}
}

func (p *Poller) applyHPAScale(ctx context.Context, c client.Client, app *pbeg.AppRef, replicas int32) error {
	dep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: app.Namespace,
			Name:      app.AppName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	patch := client.Apply
	return c.Patch(ctx, &dep, patch, client.ForceOwnership, client.FieldOwner("futura-optimizer"))
}

func (p *Poller) applyVPARecommendation(ctx context.Context, c client.Client, app *pbeg.AppRef, vpa *pbeg.VpaRecommendAction) error {
	cpuWithBuffer := int64(float64(vpa.CpuRequestMcpu) * 1.5) // multiply by 1.5
	cpuQty := resource.MustParse(fmt.Sprintf("%dm", vpa.CpuRequestMcpu))
	cpuQtyWithBuffer := resource.MustParse(fmt.Sprintf("%dm", cpuWithBuffer))

	memWithBuffer := int64(float64(vpa.MemoryMib) * 1.5) // multiply by 1.5
	memQty := resource.MustParse(fmt.Sprintf("%dMi", vpa.MemoryMib))
	memQtyWithBuffer := resource.MustParse(fmt.Sprintf("%dMi", memWithBuffer))

	dep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: app.Namespace,
			Name:      app.AppName,
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

// applyClusterAction handles all types of cluster scaling actions
func (p *Poller) applyClusterAction(ctx context.Context, c client.Client, plan *pbeg.ClusterActionPlan) error {
	logger := log.FromContext(ctx)

	switch plan.ActionType {
	case "provision_nodes":
		if provision := plan.GetProvision(); provision != nil {
			return p.applyClusterProvision(ctx, c, provision, plan)
		}
	case "deprovision_nodes":
		if deprovision := plan.GetDeprovision(); deprovision != nil {
			return p.applyClusterDeprovision(ctx, c, deprovision, plan)
		}
	case "no_action":
		if noAction := plan.GetNoAction(); noAction != nil {
			logger.Info("No cluster action needed", "reason", noAction.Reason, "reassess_in", noAction.ReassessInSeconds)
		}
		return nil
	default:
		return fmt.Errorf("unknown cluster action type: %s", plan.ActionType)
	}
	return nil
}

// applyClusterProvision provisions new nodes based on the recommendation
func (p *Poller) applyClusterProvision(ctx context.Context, c client.Client, provision *pbeg.ClusterProvisionAction, plan *pbeg.ClusterActionPlan) error {
	logger := log.FromContext(ctx)

	logger.Info("Applying cluster provisioning action",
		"strategy", provision.Strategy,
		"bin_packing_strategy", provision.BinPackingStrategy,
		"node_groups", len(provision.NodeGroups),
		"urgency", plan.Urgency,
		"cost_change_per_hour", plan.CostBenefit.CostChangePerHour)

	for i, nodeGroup := range provision.NodeGroups {
		logger.Info("Processing node group",
			"index", i,
			"name", nodeGroup.Name,
			"count", nodeGroup.Count,
			"capacity_type", nodeGroup.CapacityType,
			"instance_types", nodeGroup.InstanceTypes,
			"reason", nodeGroup.Reason)

		if err := p.provisionNodeGroup(ctx, c, nodeGroup); err != nil {
			logger.Error(err, "Failed to provision node group", "name", nodeGroup.Name)
			return fmt.Errorf("failed to provision node group %s: %w", nodeGroup.Name, err)
		}
	}

	return nil
}

// applyClusterDeprovision removes nodes based on the recommendation
func (p *Poller) applyClusterDeprovision(ctx context.Context, c client.Client, deprovision *pbeg.ClusterDeprovisionAction, plan *pbeg.ClusterActionPlan) error {
	logger := log.FromContext(ctx)

	logger.Info("Applying cluster deprovisioning action",
		"strategy", deprovision.Strategy,
		"nodes", len(deprovision.NodeNames),
		"max_parallel", deprovision.MaxParallel,
		"drain_timeout", deprovision.DrainTimeoutSeconds,
		"reason", deprovision.Reason)

	return p.deprovisionNodes(ctx, c, deprovision)
}

// provisionNodeGroup provisions a specific node group
func (p *Poller) provisionNodeGroup(ctx context.Context, c client.Client, nodeGroup *pbeg.NodeGroupProvision) error {
	logger := log.FromContext(ctx)

	// Determine cloud provider by detecting cluster environment
	cloudProvider, err := p.detectCloudProvider(ctx, c)
	if err != nil {
		logger.Error(err, "Failed to detect cloud provider")
		return fmt.Errorf("failed to detect cloud provider: %w", err)
	}

	logger.Info("Provisioning node group",
		"name", nodeGroup.Name,
		"instance_types", nodeGroup.InstanceTypes,
		"count", nodeGroup.Count,
		"capacity_type", nodeGroup.CapacityType,
		"availability_zone", nodeGroup.AvailabilityZone,
		"target_workloads", nodeGroup.TargetWorkloads,
		"cloud_provider", cloudProvider)

	// Use the cloud provider manager to provision the node group
	return p.cloudProviderManager.ProvisionNodeGroup(ctx, nodeGroup, cloudProvider)
}

// deprovisionNodes removes nodes from the cluster
func (p *Poller) deprovisionNodes(ctx context.Context, c client.Client, deprovision *pbeg.ClusterDeprovisionAction) error {
	logger := log.FromContext(ctx)

	// Determine cloud provider by detecting cluster environment
	cloudProvider, err := p.detectCloudProvider(ctx, c)
	if err != nil {
		logger.Error(err, "Failed to detect cloud provider")
		return fmt.Errorf("failed to detect cloud provider: %w", err)
	}

	logger.Info("Deprovisioning nodes",
		"nodes", deprovision.NodeNames,
		"strategy", deprovision.Strategy,
		"max_parallel", deprovision.MaxParallel,
		"cloud_provider", cloudProvider)

	// Use the cloud provider manager to deprovision the nodes
	return p.cloudProviderManager.DeprovisionNodes(ctx, deprovision.NodeNames, deprovision.Strategy, deprovision.MaxParallel, cloudProvider)
}

// detectCloudProvider detects the cloud provider by examining cluster nodes and their labels
func (p *Poller) detectCloudProvider(ctx context.Context, c client.Client) (string, error) {
	var nodes corev1.NodeList
	if err := c.List(ctx, &nodes); err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes.Items) == 0 {
		return "kind", nil // Default to kind for empty clusters or testing
	}

	// Check the first node for cloud provider indicators
	node := nodes.Items[0]

	// Check for cloud provider specific labels
	if _, ok := node.Labels["cloud.google.com/gke-nodepool"]; ok {
		return "gcp", nil
	}
	if _, ok := node.Labels["eks.amazonaws.com/nodegroup"]; ok {
		return "aws", nil
	}
	if _, ok := node.Labels["kubernetes.azure.com/agentpool"]; ok {
		return "azure", nil
	}
	if _, ok := node.Labels["alibabacloud.com/nodepool"]; ok {
		return "alibaba", nil
	}
	if _, ok := node.Labels["doks.digitalocean.com/node-pool"]; ok {
		return "digitalocean", nil
	}
	if _, ok := node.Labels["io.x-k8s.io/kind-node"]; ok {
		return "kind", nil
	}

	// Try to detect from provider ID
	if node.Spec.ProviderID != "" {
		if strings.HasPrefix(node.Spec.ProviderID, "aws://") {
			return "aws", nil
		}
		if strings.HasPrefix(node.Spec.ProviderID, "gce://") {
			return "gcp", nil
		}
		if strings.HasPrefix(node.Spec.ProviderID, "azure://") {
			return "azure", nil
		}
	}

	// Try to detect from instance type labels
	if instanceType, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
		// AWS instance types typically have format like m5.large
		if strings.Contains(instanceType, ".") && len(strings.Split(instanceType, ".")) == 2 {
			return "aws", nil
		}
		// GCP instance types typically have format like e2-standard-4
		if strings.Contains(instanceType, "-") && strings.Contains(instanceType, "standard") {
			return "gcp", nil
		}
	}

	// Default to kind for unknown environments (useful for local testing)
	return "kind", nil
}
