package secrets

import (
	"errors"

	"github.com/opisvigilant/futura/apis/internal/config"
)

type AWSSecretStore struct {
}

func NewAWSSecretStore(config *config.Configuration) *AWSSecretStore {
	return &AWSSecretStore{}
}

func (m *AWSSecretStore) StoreCustomerCredentials(organizationID, secretID string, creds map[string]string) error {
	return errors.New("[AWSSecretStore] StoreCustomerCredentials not implemented")
}

func (m *AWSSecretStore) GetCustomerCredentials(organizationID, secretID string) (map[string]string, error) {
	return nil, errors.New("[AWSSecretStore] GetCustomerCredentials not implemented")
}
