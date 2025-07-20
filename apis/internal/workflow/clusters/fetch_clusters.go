package workflowclusters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/providers"
	"github.com/opisvigilant/futura/apis/models"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
)

func RegisterWorkflowFetchClusters(worker worker.Worker) {
	// register workflow and activities
	worker.RegisterWorkflow(WorkflowFetchClusters)
	worker.RegisterActivity(CreateProviderClient)
	worker.RegisterActivity(FetchClusterIDs)
	worker.RegisterActivity(FetchMetadata)
	worker.RegisterActivity(StoreMetadata)
}

type WorkflowFetchClustersInput struct {
	DBConfig *config.Database

	OrganizationID uint
	Provider       pb.CloudProvider
	Credentials    map[string]string
}

func WorkflowFetchClusters(ctx workflow.Context, input WorkflowFetchClustersInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second * 2,
			MaximumAttempts: 5,
		},
	})

	var client providers.ProviderClient
	err := workflow.ExecuteActivity(ctx, CreateProviderClient, providers.ProviderConfig{
		Provider:    input.Provider,
		Credentials: input.Credentials,
	}).Get(ctx, &client)
	if err != nil {
		return err
	}

	var clusterIDs []string
	err = workflow.ExecuteActivity(ctx, FetchClusterIDs, client).Get(ctx, &clusterIDs)
	if err != nil {
		return err
	}

	for _, clusterID := range clusterIDs {
		var metadata *models.ClusterMetadata
		err = workflow.ExecuteActivity(ctx, FetchMetadata, client, clusterID).Get(ctx, metadata)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed fetching metadata", "cluster", clusterID, "err", err)
			continue // or return err to fail fast
		}

		err = workflow.ExecuteActivity(ctx, StoreMetadata, input.DBConfig, input.Provider, metadata).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed storing metadata", "cluster", clusterID, "err", err)
			continue
		}
	}
	return nil
}

func CreateProviderClient(ctx context.Context, config providers.ProviderConfig) (providers.ProviderClient, error) {
	return providers.CreateProviderClient(ctx, config)
}

func FetchClusterIDs(ctx context.Context, client providers.ProviderClient) ([]string, error) {
	return client.FetchClusters(ctx)
}

func FetchMetadata(ctx context.Context, client providers.ProviderClient, clusterID string) (*models.ClusterMetadata, error) {
	return client.FetchClusterMetadata(ctx, clusterID)
}

func StoreMetadata(ctx context.Context, dbConfig *config.Database, provider pb.CloudProvider, metadata *models.ClusterMetadata) error {
	// create a postgres client
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User,
		dbConfig.Password, dbConfig.Name, dbConfig.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Save the ClusterMetadata
	if err := db.Create(metadata).Error; err != nil {
		return fmt.Errorf("failed to create ClusterMetadata: %w", err)
	}

	// Set the FK in cluster specific Metadata and save it
	switch provider {
	case pb.CloudProvider_ALIBABA:
		if metadata.ACKMetadata == nil {
			return errors.New("ACKMetadata is nil in the ClusterMetadata object for the Alibaba Provider")
		}
		metadata.ACKMetadata.ClusterMetadataID = metadata.ID
		if err := db.Create(metadata.ACKMetadata).Error; err != nil {
			return fmt.Errorf("failed to create ACKMetadata: %w", err)
		}
	case pb.CloudProvider_AWS:
		if metadata.EKSMetadata == nil {
			return errors.New("EKSMetadata is nil in the ClusterMetadata object for the AWS Provider")
		}
		metadata.EKSMetadata.ClusterMetadataID = metadata.ID
		if err := db.Create(metadata.EKSMetadata).Error; err != nil {
			return fmt.Errorf("failed to create EKSMetadata: %w", err)
		}
	case pb.CloudProvider_AZURE:
		if metadata.AKSMetadata == nil {
			return errors.New("AKSMetadata is nil in the ClusterMetadata object for the Azure Provider")
		}
		metadata.AKSMetadata.ClusterMetadataID = metadata.ID
		if err := db.Create(metadata.AKSMetadata).Error; err != nil {
			return fmt.Errorf("failed to create AKSMetadata: %w", err)
		}
	case pb.CloudProvider_DIGITALOCEAN:
		if metadata.DOKSMetadata == nil {
			return errors.New("DOKSMetadata is nil in the ClusterMetadata object for the DigitalOcean Provider")
		}
		metadata.DOKSMetadata.ClusterMetadataID = metadata.ID
		if err := db.Create(metadata.DOKSMetadata).Error; err != nil {
			return fmt.Errorf("failed to create DOKSMetadata: %w", err)
		}
	case pb.CloudProvider_GCP:
		if metadata.GKEMetadata == nil {
			return errors.New("GKEMetadata is nil in the ClusterMetadata object for the GCP Provider")
		}
		metadata.GKEMetadata.ClusterMetadataID = metadata.ID
		if err := db.Create(metadata.GKEMetadata).Error; err != nil {
			return fmt.Errorf("failed to create GKEMetadata: %w", err)
		}
	default:
		return errors.New("invalid pb.CloudProvider in StoreMetadata Activity")
	}

	return nil
}
