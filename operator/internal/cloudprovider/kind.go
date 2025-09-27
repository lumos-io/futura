package cloudprovider

import (
	"context"
	"fmt"

	pbeg "github.com/opisvigilant/futura/proto/gen/engine"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KindProvider implements CloudProvider for Kind (Kubernetes in Docker) local testing
type KindProvider struct {
	clusterName string
}

// NewKindProvider creates a new Kind cloud provider for local testing
func NewKindProvider(clusterName string) *KindProvider {
	return &KindProvider{
		clusterName: clusterName,
	}
}

func (p *KindProvider) GetName() string {
	return "kind"
}

func (p *KindProvider) ProvisionNodeGroup(ctx context.Context, nodeGroup *pbeg.NodeGroupProvision) error {
	logger := log.FromContext(ctx)

	logger.Info("Kind: Simulating node group provisioning",
		"cluster", p.clusterName,
		"name", nodeGroup.Name,
		"count", nodeGroup.Count,
		"instance_types", nodeGroup.InstanceTypes,
		"capacity_type", nodeGroup.CapacityType)

	// For Kind, we simulate node provisioning by logging the action
	// In a real scenario, this could:
	// 1. Add worker nodes to the Kind cluster
	// 2. Or simulate the provisioning for testing purposes

	logger.Info("Kind: Successfully simulated node provisioning",
		"node_group", nodeGroup.Name,
		"simulated_nodes", nodeGroup.Count)

	return nil
}

func (p *KindProvider) DeprovisionNodes(ctx context.Context, nodeNames []string, strategy string, maxParallel int32) error {
	logger := log.FromContext(ctx)

	logger.Info("Kind: Simulating node deprovisioning",
		"cluster", p.clusterName,
		"nodes", nodeNames,
		"strategy", strategy,
		"max_parallel", maxParallel)

	// For Kind, we simulate node deprovisioning by logging the action
	logger.Info("Kind: Successfully simulated node deprovisioning",
		"removed_nodes", nodeNames)

	return nil
}

func (p *KindProvider) GetNodeCapacity(ctx context.Context) ([]NodeCapacity, error) {
	// Kind uses the host machine resources, so we return simulated capacities
	// that represent typical development machine configurations

	capacities := []NodeCapacity{
		{
			InstanceType:           "kind-worker",
			InstanceFamily:         "kind",
			CPU:                    2,  // 2 CPU cores allocated to Kind worker
			Memory:                 4,  // 4 GB RAM
			Storage:                20, // 20 GB disk
			CostPerHour:            0,  // No cost for local testing
			AvailabilityZones:      []string{"kind-zone"},
			SupportedCapacityTypes: []string{"on-demand"}, // Kind doesn't support spot instances
		},
		{
			InstanceType:           "kind-worker-large",
			InstanceFamily:         "kind",
			CPU:                    4,  // 4 CPU cores for larger workloads
			Memory:                 8,  // 8 GB RAM
			Storage:                40, // 40 GB disk
			CostPerHour:            0,  // No cost for local testing
			AvailabilityZones:      []string{"kind-zone"},
			SupportedCapacityTypes: []string{"on-demand"},
		},
	}

	return capacities, nil
}

func (p *KindProvider) ValidateNodeGroup(nodeGroup *pbeg.NodeGroupProvision) error {
	if nodeGroup.Count <= 0 {
		return fmt.Errorf("node count must be positive")
	}

	if len(nodeGroup.InstanceTypes) == 0 {
		return fmt.Errorf("at least one instance type must be specified")
	}

	// Validate instance types are Kind instance types
	validKindTypes := map[string]bool{
		"kind-worker":       true,
		"kind-worker-large": true,
	}

	for _, instanceType := range nodeGroup.InstanceTypes {
		if !validKindTypes[instanceType] {
			return fmt.Errorf("invalid Kind instance type: %s (supported: kind-worker, kind-worker-large)", instanceType)
		}
	}

	// Kind only supports on-demand capacity type
	if nodeGroup.CapacityType != "on-demand" && nodeGroup.CapacityType != "" {
		return fmt.Errorf("Kind only supports on-demand capacity type, got: %s", nodeGroup.CapacityType)
	}

	// Validate availability zone
	if nodeGroup.AvailabilityZone != "" && nodeGroup.AvailabilityZone != "kind-zone" {
		return fmt.Errorf("invalid availability zone for Kind: %s (supported: kind-zone)", nodeGroup.AvailabilityZone)
	}

	return nil
}
