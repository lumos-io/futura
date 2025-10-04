package alibabaprovider

import (
	"context"
	"errors"

	"github.com/opisvigilant/futura/backend/internal/apis/models"
)

const (
	ALIBABA_ACCESS_KEY_ID string = "AccessKeyId"
	ALIBABA_ACCESS_SECRET string = "AccessSecret"
)

type AlibabaProvider struct {
}

func New(credentials map[string]string) (*AlibabaProvider, error) {
	return &AlibabaProvider{}, nil
}

func (a *AlibabaProvider) Test(creds map[string]string) error {
	return errors.New("not implemented")
}

func (a *AlibabaProvider) FetchClusters(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *AlibabaProvider) FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error) {
	return nil, nil
}

type AlibabaCredentials struct {
}
