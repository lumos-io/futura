package gcpprovider

import (
	"context"
	"errors"

	"github.com/opisvigilant/futura/apis/models"
)

const (
	GCP_PROJECT_ID       string = "ProjectId"
	GCP_CREDENTIALS_JSON string = "CredentialsJson"
)

type GCPProvider struct {
}

func New(credentials map[string]string) (*GCPProvider, error) {
	return &GCPProvider{}, nil
}

func (a *GCPProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type GCPCredentials struct {
}

func (a *GCPProvider) FetchClusters(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *GCPProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	m := &models.GKEClusterMetadata{
		Version: "v1.33.2",
		Region:  "us-east-2",
	}
	return &models.ClusterMetadata{
		Name:        clusterID,
		GKEMetadata: m,
	}, nil
}
