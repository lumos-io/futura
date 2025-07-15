package gcpprovider

import (
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
)

type GCPProvider struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*GCPProvider, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &GCPProvider{
		secretStore: ss,
	}, nil
}
