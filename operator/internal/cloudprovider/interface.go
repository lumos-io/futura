package cloudprovider

import (
	"context"

	pbeg "github.com/opisvigilant/futura/proto/gen/engine"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CloudProvider defines the interface for different cloud provider implementations
type CloudProvider interface {
	// GetName returns the name of the cloud provider (aws, gcp, azure, etc.)
	GetName() string

	// ProvisionNodeGroup provisions a new node group according to the specification
	ProvisionNodeGroup(ctx context.Context, nodeGroup *pbeg.NodeGroupProvision) error

	// DeprovisionNodes removes the specified nodes from the cluster
	DeprovisionNodes(ctx context.Context, nodeNames []string, strategy string, maxParallel int32) error

	// GetNodeCapacity returns the available instance types and their specifications
	GetNodeCapacity(ctx context.Context) ([]NodeCapacity, error)

	// ValidateNodeGroup validates that the node group specification is valid for this provider
	ValidateNodeGroup(nodeGroup *pbeg.NodeGroupProvision) error
}

// NodeCapacity represents the capacity information for a node instance type
type NodeCapacity struct {
	InstanceType           string
	InstanceFamily         string
	CPU                    float64 // CPU cores
	Memory                 float64 // Memory in GB
	Storage                float64 // Storage in GB (optional)
	NetworkBandwidth       float64 // Network bandwidth in Gbps (optional)
	CostPerHour            float64 // Cost per hour in USD (optional)
	AvailabilityZones      []string
	SupportedCapacityTypes []string // ["on-demand", "spot", "preemptible"]
}

// CloudProviderManager manages multiple cloud providers and routes requests
type CloudProviderManager struct {
	providers map[string]CloudProvider
	client    client.Client
}

// NewCloudProviderManager creates a new cloud provider manager
func NewCloudProviderManager(client client.Client) *CloudProviderManager {
	return &CloudProviderManager{
		providers: make(map[string]CloudProvider),
		client:    client,
	}
}

// RegisterProvider registers a cloud provider implementation
func (m *CloudProviderManager) RegisterProvider(provider CloudProvider) {
	m.providers[provider.GetName()] = provider
}

// GetProvider returns a cloud provider by name
func (m *CloudProviderManager) GetProvider(name string) (CloudProvider, bool) {
	provider, exists := m.providers[name]
	return provider, exists
}

// ProvisionNodeGroup provisions a node group using the appropriate cloud provider
func (m *CloudProviderManager) ProvisionNodeGroup(ctx context.Context, nodeGroup *pbeg.NodeGroupProvision, cloudProvider string) error {
	provider, exists := m.GetProvider(cloudProvider)
	if !exists {
		return &UnsupportedProviderError{Provider: cloudProvider}
	}

	if err := provider.ValidateNodeGroup(nodeGroup); err != nil {
		return &ValidationError{Provider: cloudProvider, Err: err}
	}

	return provider.ProvisionNodeGroup(ctx, nodeGroup)
}

// DeprovisionNodes removes nodes using the appropriate cloud provider
func (m *CloudProviderManager) DeprovisionNodes(ctx context.Context, nodeNames []string, strategy string, maxParallel int32, cloudProvider string) error {
	provider, exists := m.GetProvider(cloudProvider)
	if !exists {
		return &UnsupportedProviderError{Provider: cloudProvider}
	}

	return provider.DeprovisionNodes(ctx, nodeNames, strategy, maxParallel)
}

// GetSupportedProviders returns a list of registered cloud providers
func (m *CloudProviderManager) GetSupportedProviders() []string {
	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}
