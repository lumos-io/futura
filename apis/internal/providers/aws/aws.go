package awsprovider

import (
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
)

type AWSProvider struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*AWSProvider, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &AWSProvider{
		secretStore: ss,
	}, nil
}

