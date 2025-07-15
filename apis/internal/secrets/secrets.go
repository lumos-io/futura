package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type SecretStore interface {
	StoreCustomerCredentials(organizationID string, provider string, creds map[string]string) error
	GetCustomerCredentials(organizationID string, provider string) (map[string]string, error)
}

type InMemorySecretStore struct {
	secrets []*secret
}

type secret struct {
	OrganizationID string            `json:"organizationId"`
	Provider       string            `json:"provider"`
	Credentials    map[string]string `json:"credentials"`
}

func NewInMemorySecretStore() (*InMemorySecretStore, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	secretsFileName := fmt.Sprintf("%s/secrets.json", wd)
	secrets, err := loadSecrets(secretsFileName)
	if err != nil {
		secrets = make([]*secret, 0)
		if err := createFileIfNotExists(secretsFileName, secrets); err != nil {
			return nil, err
		}
	}
	return &InMemorySecretStore{
		secrets: secrets,
	}, nil
}

func (m *InMemorySecretStore) StoreCustomerCredentials(organizationID string, provider string, creds map[string]string) error {
	m.secrets = append(m.secrets, &secret{
		OrganizationID: organizationID,
		Provider:       provider,
		Credentials:    creds,
	})
	return nil
}

func (m *InMemorySecretStore) GetCustomerCredentials(organizationID string, provider string) (map[string]string, error) {
	for _, s := range m.secrets {
		if s.OrganizationID == organizationID && s.Provider == provider {
			return s.Credentials, nil
		}
	}
	return nil, errors.New("failed to fetch credentials for the organizationId and provider pair")
}

// createFileIfNotExists ensures the file exists, and if not, creates it with an empty array
func createFileIfNotExists(path string, secrets []*secret) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		emptyData, err := json.Marshal(secrets) // empty JSON array
		if err != nil {
			return fmt.Errorf("failed to marshal empty array: %w", err)
		}

		if err := os.WriteFile(path, emptyData, 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	return nil
}

// loadSecrets reads the file and unmarshals its JSON content into a slice of secret
func loadSecrets(path string) ([]*secret, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var creds []*secret
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return creds, nil
}
