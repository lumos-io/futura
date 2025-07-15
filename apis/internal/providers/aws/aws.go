package awsprovider

import (
	"errors"

	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/secrets"
	"github.com/opisvigilant/futura/apis/internal/secrets/secrettype"
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

type AWSCredentials struct {
	AccessKey       string
	SecretAccessKey string
	Region          string
	SessionToken    string // do I need this??
}

func (p *AWSProvider) GetCredentials(organizationID string, secretID string) (*AWSCredentials, error) {
	creds, err := p.secretStore.GetCredentials(organizationID, secretID)
	if err != nil {
		return nil, err
	}
	c := &AWSCredentials{}
	for k, v := range creds {
		switch k {
		case secrettype.AwsRegion:
			c.Region = v
		case secrettype.AwsAccessKeyId:
			c.AccessKey = v
		case secrettype.AwsSecretAccessKey:
			c.SecretAccessKey = v
		case secrettype.AwsSessionToken:
			c.SessionToken = v
		}
	}
	return c, nil
}

func (p *AWSProvider) SetCredentials(organizationID, secretID string, creds *AWSCredentials) error {
	return p.secretStore.SetCredentials(organizationID, secretID, map[string]string{
		secrettype.AwsRegion:          creds.Region,
		secrettype.AwsAccessKeyId:     creds.AccessKey,
		secrettype.AwsSecretAccessKey: creds.SecretAccessKey,
		secrettype.AwsSessionToken:    creds.SessionToken,
	})
}

func (p *AWSProvider) UpdateCredentials(organizationID, secretID string, creds *AWSCredentials) error {
	return errors.New("not implemented")
}

func (p *AWSProvider) DeleteCredentials(organizationID, secretID string) error {
	return errors.New("not implemented")
}
