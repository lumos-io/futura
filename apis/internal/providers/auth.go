package providers

import (
	"errors"

	"github.com/opisvigilant/futura/apis/internal/config"
	alibabaprovider "github.com/opisvigilant/futura/apis/internal/providers/alibaba"
	awsprovider "github.com/opisvigilant/futura/apis/internal/providers/aws"
	azureprovider "github.com/opisvigilant/futura/apis/internal/providers/azure"
	digitaloceanprovider "github.com/opisvigilant/futura/apis/internal/providers/digitalocean"
	gcpprovider "github.com/opisvigilant/futura/apis/internal/providers/gcp"
	"github.com/opisvigilant/futura/apis/internal/secrets"
	"github.com/opisvigilant/futura/apis/models"
)

type ProviderAuth interface {
	Test(creds map[string]string) error
}

type CloudProviderAuth struct {
	secretStore secrets.SecretStore
}

func New(config *config.Configuration) (*CloudProviderAuth, error) {
	ss, err := secrets.NewInMemorySecretStore()
	if err != nil {
		return nil, err
	}
	return &CloudProviderAuth{
		secretStore: ss,
	}, nil
}

func (cp *CloudProviderAuth) GetCredentials(organizationID uint, provider string, secretID string) (map[string]string, error) {
	return cp.secretStore.GetCredentials(organizationID, provider, secretID)
}

func (cp *CloudProviderAuth) SetCredentials(organizationID uint, provider string, secretID string, creds map[string]string) error {
	return cp.secretStore.SetCredentials(organizationID, provider, secretID, creds)
}

func (cp *CloudProviderAuth) UpdateCredentials(organizationID uint, provider string, secretID string, creds map[string]string) error {
	return cp.secretStore.UpdateCredentials(organizationID, provider, secretID, creds)
}

func (cp *CloudProviderAuth) DeleteCredentials(organizationID uint, provider string, secretID string) error {
	return cp.secretStore.DeleteCredentials(organizationID, provider, secretID)
}

func (cp *CloudProviderAuth) TestConnection(provider string, creds map[string]string) error {
	var ap ProviderAuth
	var err error
	switch models.CloudProviderName(provider) {
	case models.AWS:
		ap, err = awsprovider.New()
	case models.Alibaba:
		ap, err = alibabaprovider.New()
	case models.DigitalOcean:
		ap, err = digitaloceanprovider.New()
	case models.Azure:
		ap, err = azureprovider.New()
	case models.GoogleCloud:
		ap, err = gcpprovider.New()
	default:
		return errors.New("not a valid provider name")
	}
	if err != nil {
		return err
	}
	return ap.Test(creds)
}
