package secrets

import (
	"errors"

	"github.com/opisvigilant/futura/backend/internal/shared/config"
)

type GCPSecretStore struct {
}

func NewGCPSecretStore(config *config.Configuration) *GCPSecretStore {
	return &GCPSecretStore{}
}

func (m *GCPSecretStore) StoreCustomerCredentials(organizationID string, provider string, creds map[string]string) error {
	return errors.New("[GCPSecretStore] StoreCustomerCredentials not implemented")
}

func (m *GCPSecretStore) GetCustomerCredentials(organizationID string, provider string) (map[string]string, error) {
	return nil, errors.New("[GCPSecretStore] StoreCustomerCredentials not implemented")
}
