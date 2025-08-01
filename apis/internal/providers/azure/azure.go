package azureprovider

import (
	"context"
	"errors"

	"github.com/opisvigilant/futura/apis/models"
)

const (
	AZURE_TENANT_ID     string = "TenantId"
	AZURE_CLIENT_ID     string = "ClientId"
	AZURE_CLIENT_SECRET string = "ClientSecret"
)

type AzureProvider struct {
}

func New(credentials map[string]string) (*AzureProvider, error) {
	return &AzureProvider{}, nil
}

func (a *AzureProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type AzureCredentials struct {
}

func (a *AzureProvider) FetchClusters(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *AzureProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	m := &models.AKSClusterMetadata{
		Version: "v1.33.2",
		Region:  "us-east-2",
	}
	return &models.ClusterMetadata{
		Name:        clusterID,
		AKSMetadata: m,
	}, nil
}
