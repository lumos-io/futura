package secrets

import "github.com/opisvigilant/futura/apis/internal/config"

type AWSSecretStore struct {
}

func NewAWSSecretStore(config *config.Configuration) *AWSSecretStore {
	return &AWSSecretStore{}
}

func (m *AWSSecretStore) StoreCustomerCredentials(organizationID string, provider string, creds map[string]string) error {
	return nil
}

func (m *AWSSecretStore) GetCustomerCredentials(organizationID string, provider string) (map[string]string, error) {
	return nil, nil
}
