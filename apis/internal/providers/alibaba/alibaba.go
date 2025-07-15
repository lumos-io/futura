package alibabaprovider

import (
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
)

type AlibabaProvider struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*AlibabaProvider, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &AlibabaProvider{
		secretStore: ss,
	}, nil
}
