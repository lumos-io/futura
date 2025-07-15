package alibabaprovider

import (
	"errors"
)

type AlibabaProvider struct {
}

func New() (*AlibabaProvider, error) {
	return &AlibabaProvider{}, nil
}

func (a *AlibabaProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type AlibabaCredentials struct {
}
