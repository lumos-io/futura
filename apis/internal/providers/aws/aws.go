package awsprovider

import (
	"errors"
)

type AWSProvider struct {
}

func New() (*AWSProvider, error) {
	return &AWSProvider{}, nil
}

type AWSCredentials struct {
	AccessKey       string
	SecretAccessKey string
	Region          string
	SessionToken    string // do I need this??
}

func (a *AWSProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}
