package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type SecretStore interface {
	GetCredentials(organizationID string, secretID string) (map[string]string, error)
	SetCredentials(organizationID, secretID string, creds map[string]string) error
	UpdateCredentials(organizationID, secretID string, creds map[string]string) error
	DeleteCredentials(organizationID, secretID string) error
}

type InMemorySecretStore struct {
	secrets []*secret
}

type secret struct {
	OrganizationID string            `json:"organizationId"`
	SecretID       string            `json:"secretId"`
	Credentials    map[string]string `json:"credentials"`
}

var secretsFileName string = ""

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	secretsFileName = fmt.Sprintf("%s/secrets.json", wd)
}

func NewInMemorySecretStore() (*InMemorySecretStore, error) {
	secrets, err := loadSecrets()
	if err != nil {
		secrets = make([]*secret, 0)
		if err := createFileIfNotExists(secrets); err != nil {
			return nil, err
		}
	}
	return &InMemorySecretStore{
		secrets: secrets,
	}, nil
}

func (m *InMemorySecretStore) GetCredentials(organizationID string, secretID string) (map[string]string, error) {
	for _, s := range m.secrets {
		if s.OrganizationID == organizationID && s.SecretID == secretID {
			return s.Credentials, nil
		}
	}
	return nil, errors.New("failed to fetch credentials for the organizationId and provider pair")
}

func (m *InMemorySecretStore) SetCredentials(organizationID, secretID string, creds map[string]string) error {
	m.secrets = append(m.secrets, &secret{
		OrganizationID: organizationID,
		SecretID:       secretID,
		Credentials:    creds,
	})
	return saveSecrets(m.secrets)
}

func (m *InMemorySecretStore) UpdateCredentials(organizationID, secretID string, creds map[string]string) error {
	return errors.New("not implemented")
}

func (m *InMemorySecretStore) DeleteCredentials(organizationID, secretID string) error {
	return errors.New("not implemented")
}

// createFileIfNotExists ensures the file exists, and if not, creates it with an empty array
func createFileIfNotExists(secrets []*secret) error {
	if _, err := os.Stat(secretsFileName); errors.Is(err, os.ErrNotExist) {
		emptyData, err := json.Marshal(secrets) // empty JSON array
		if err != nil {
			return fmt.Errorf("failed to marshal empty array: %w", err)
		}

		if err := os.WriteFile(secretsFileName, emptyData, 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	return nil
}

// loadSecrets reads the file and unmarshals its JSON content into a slice of secret
func loadSecrets() ([]*secret, error) {
	data, err := os.ReadFile(secretsFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var creds []*secret
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return creds, nil
}

func saveSecrets(secrets []*secret) error {
	b, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	if err := os.WriteFile(secretsFileName, b, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
