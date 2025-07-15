package gcpprovider

import (
	"errors"
)

type GCPProvider struct {
}

func New() (*GCPProvider, error) {
	return &GCPProvider{}, nil
}

func (a *GCPProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type GCPCredentials struct {
}
