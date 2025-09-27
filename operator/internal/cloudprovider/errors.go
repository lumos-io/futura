package cloudprovider

import "fmt"

// UnsupportedProviderError is returned when a cloud provider is not supported
type UnsupportedProviderError struct {
	Provider string
}

func (e *UnsupportedProviderError) Error() string {
	return fmt.Sprintf("unsupported cloud provider: %s", e.Provider)
}

// ValidationError is returned when node group validation fails
type ValidationError struct {
	Provider string
	Err      error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for provider %s: %v", e.Provider, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ProvisioningError is returned when node provisioning fails
type ProvisioningError struct {
	Provider  string
	NodeGroup string
	Err       error
}

func (e *ProvisioningError) Error() string {
	return fmt.Sprintf("provisioning failed for provider %s, node group %s: %v", e.Provider, e.NodeGroup, e.Err)
}

func (e *ProvisioningError) Unwrap() error {
	return e.Err
}

// DeprovisioningError is returned when node deprovisioning fails
type DeprovisioningError struct {
	Provider string
	Nodes    []string
	Err      error
}

func (e *DeprovisioningError) Error() string {
	return fmt.Sprintf("deprovisioning failed for provider %s, nodes %v: %v", e.Provider, e.Nodes, e.Err)
}

func (e *DeprovisioningError) Unwrap() error {
	return e.Err
}
