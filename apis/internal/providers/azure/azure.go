package azureprovider

import (
	"errors"
)

type AzureProvider struct {
}

func New() (*AzureProvider, error) {
	return &AzureProvider{}, nil
}

func (a *AzureProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type AzureCredentials struct {
}
