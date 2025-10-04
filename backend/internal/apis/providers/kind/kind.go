package kindprovider

import (
	"context"
	"errors"
	"time"

	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"gorm.io/datatypes"
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
	return []string{"kind-cluster-1", "kind-cluster-2"}, nil
}

func (a *KindProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	return &models.ClusterMetadata{
		Name: clusterID,
		KindMetadata: &models.KindClusterMetadata{
			Status:           "Available",
			Version:          "v1.2.3",
			Region:           "localhost",
			Endpoint:         "localhost",
			ClusterCreatedAt: time.Now().UTC(),
			PlatformVersion:  "platform-v.3.2.1",
			Tags: datatypes.JSONMap{
				"test": "test-test",
				"key":  "value",
			},
		},
	}, nil
}
