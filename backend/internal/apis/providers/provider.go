package providers

import (
	"context"
	"errors"

	alibabaprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/alibaba"
	awsprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/aws"
	azureprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/azure"
	digitaloceanprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/digitalocean"
	gcpprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/gcp"
	kindprovider "github.com/opisvigilant/futura/backend/internal/apis/providers/kind"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
)

type ProviderClient interface {
	FetchClusters(ctx context.Context) ([]string, error)
	FetchClusterMetadata(ctx context.Context, clusterID string) (*models.ClusterMetadata, error)
}

type ProviderConfig struct {
	Provider    pb.CloudProvider
	Credentials map[string]string
}

func CreateProviderClient(ctx context.Context, config ProviderConfig) (ProviderClient, error) {
	var err error
	var pr ProviderClient
	switch config.Provider {
	case pb.CloudProvider_AWS:
		pr, err = awsprovider.New(config.Credentials)
	case pb.CloudProvider_ALIBABA:
		pr, err = alibabaprovider.New(config.Credentials)
	case pb.CloudProvider_DIGITALOCEAN:
		pr, err = digitaloceanprovider.New(config.Credentials)
	case pb.CloudProvider_AZURE:
		pr, err = azureprovider.New(config.Credentials)
	case pb.CloudProvider_GCP:
		pr, err = gcpprovider.New(config.Credentials)
	case pb.CloudProvider_KIND:
		pr, err = kindprovider.New(config.Credentials)
	default:
		return nil, errors.New("not a valid provider name")
	}
	if err != nil {
		return nil, err
	}
	return pr, nil
}
