package digitaloceanprovider

import (
	"errors"
)

type DigitalOceanProvider struct {
}

func New() (*DigitalOceanProvider, error) {
	return &DigitalOceanProvider{}, nil
}

func (a *DigitalOceanProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type DigitalOceanCredentials struct {
}
