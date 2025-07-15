package digitaloceanprovider

import (
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
)

type DigitalOceanProvider struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*DigitalOceanProvider, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &DigitalOceanProvider{
		secretStore: ss,
	}, nil
}
