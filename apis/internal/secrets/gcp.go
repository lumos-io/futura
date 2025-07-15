package secrets

import "github.com/opisvigilant/futura/apis/internal/config"

type GCPSecretStore struct {
}

func NewGCPSecretStore(config *config.Configuration) *GCPSecretStore {
	return &GCPSecretStore{}
}

func (m *GCPSecretStore) StoreCustomerCredentials(organizationID string, provider string, creds map[string]string) error {
	return nil
}

func (m *GCPSecretStore) GetCustomerCredentials(organizationID string, provider string) (map[string]string, error) {
	return nil, nil
}
