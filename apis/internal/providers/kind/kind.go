package kindprovider

import (
	"context"
	"errors"

	"github.com/opisvigilant/futura/apis/models"
)

type KindProvider struct {
}

func New(credentials map[string]string) (*KindProvider, error) {
	return &KindProvider{}, nil
}

func (a *KindProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

func (a *KindProvider) FetchClusters(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *KindProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	return nil, nil
}
