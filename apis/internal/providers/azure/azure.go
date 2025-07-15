package azureprovider

import (
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
)

type AzureProvider struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*AzureProvider, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &AzureProvider{
		secretStore: ss,
	}, nil
}
