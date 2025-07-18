package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
)

type SecretStore interface {
	GetCredentials(id uuid.UUID) (map[string]string, error)
	SetCredentials(organizationID uint, provider string, secretName pb.SecretName, creds map[string]string) (uuid.UUID, error)
	DeleteCredentials(id uuid.UUID) error
}

type InMemorySecretStore struct {
	secrets []*secret
}

type secret struct {
	ID             uuid.UUID         `json:"id"`
	OrganizationID uint              `json:"organizationId"`
	Provider       string            `json:"provider"`
	ConnectionName string            `json:"connectionName"`
	SecretName     pb.SecretName     `json:"secretName"`
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

func (m *InMemorySecretStore) GetCredentials(id uuid.UUID) (map[string]string, error) {
	for _, s := range m.secrets {
		if s.ID == id {
			return s.Credentials, nil
		}
	}
	return nil, errors.New("failed to fetch credentials for the organizationId and provider pair")
}

func (m *InMemorySecretStore) SetCredentials(organizationID uint, provider string, secretName pb.SecretName, creds map[string]string) (uuid.UUID, error) {
	id := uuid.New()
	m.secrets = append(m.secrets, &secret{
		ID:             id,
		OrganizationID: organizationID,
		Provider:       provider,
		SecretName:     secretName,
		Credentials:    creds,
	})
	if err := saveSecrets(m.secrets); err != nil {
		// id will be ignored in this case
		return id, err
	}
	return id, nil
}

func (m *InMemorySecretStore) DeleteCredentials(id uuid.UUID) error {
	newSecrets := []*secret{}
	for _, s := range m.secrets {
		if s.ID == id {
			continue
		}
		newSecrets = append(newSecrets, s)
	}
	m.secrets = newSecrets
	return saveSecrets(m.secrets)
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
