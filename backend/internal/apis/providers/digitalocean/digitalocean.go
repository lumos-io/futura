package digitaloceanprovider

import (
	"context"
	"errors"

	"github.com/opisvigilant/futura/backend/internal/apis/models"
)

const (
	DIGITALOCEAN_ACCESS_TOKEN string = "AccessToken"
)

type DigitalOceanProvider struct {
}

func New(credentials map[string]string) (*DigitalOceanProvider, error) {
	return &DigitalOceanProvider{}, nil
}

func (a *DigitalOceanProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

type DigitalOceanCredentials struct {
}

func (a *DigitalOceanProvider) FetchClusters(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *DigitalOceanProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	m := &models.DOKSClusterMetadata{
		Version: "v1.33.2",
		Region:  "us-east-2",
	}
	return &models.ClusterMetadata{
		Name:         clusterID,
		DOKSMetadata: m,
	}, nil
}
