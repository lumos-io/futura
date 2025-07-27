package workflowclusters

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/providers"
	workflowsignals "github.com/opisvigilant/futura/apis/internal/workflow/signals"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/go-lib/stream"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
	"github.com/riverqueue/river"
)

type WorkflowFetchClustersInput struct {
	Config *config.Configuration

	OrganizationID     uint
	ProviderConnection *pb.ProviderConnection
	Credentials        map[string]string
}

func (WorkflowFetchClustersInput) Kind() string { return "workflow_fetch_clusters" }

type WorkflowFetchClustersWorker struct {
	river.WorkerDefaults[WorkflowFetchClustersInput]
}

func (w *WorkflowFetchClustersWorker) Work(ctx context.Context, job *river.Job[WorkflowFetchClustersInput]) error {
	js, err := stream.NewRedisStreamClient(job.Args.Config.Redis.Servers)
	if err != nil {
		return fmt.Errorf("failed to connect to Stream: %v", err)
	}

	client, err := providers.CreateProviderClient(ctx, providers.ProviderConfig{
		Provider:    job.Args.ProviderConnection.Provider,
		Credentials: job.Args.Credentials,
	})
	if err != nil {
		b, _ := json.Marshal(workflowsignals.WorkflowFetchClustersStatusSignal{
			ProviderConnectionID: job.Args.ProviderConnection.Id,
			OrganizationID:       job.Args.OrganizationID,
			Status:               workflowsignals.StatusFailed,
			Error:                "",
		})
		return js.Publish(ctx, workflowsignals.NatsWorkflowFetchClusterTopic, b)
	}

	allClusters, err := client.FetchClusters(ctx)
	if err != nil {
		return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
			ProviderConnectionID: job.Args.ProviderConnection.Id,
			OrganizationID:       job.Args.OrganizationID,
			Status:               workflowsignals.StatusFailed,
			Error:                err.Error(),
		})
	}

	data := make([]workflowsignals.ClusterInfo, len(allClusters))
	for i, clusterID := range allClusters {
		metadata, err := client.FetchClusterMetadata(ctx, clusterID)
		if err != nil {
			return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
				ProviderConnectionID: job.Args.ProviderConnection.Id,
				OrganizationID:       job.Args.OrganizationID,
				Status:               workflowsignals.StatusFailed,
				Error:                err.Error(),
			})
		}

		// enrich the object
		metadata.CloudProviderID = uint(job.Args.ProviderConnection.Id)
		metadata.OrganizationID = job.Args.OrganizationID

		if err := storeClusterMetadata(job.Args.Config.Database, metadata); err != nil {
			return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
				ProviderConnectionID: job.Args.ProviderConnection.Id,
				OrganizationID:       job.Args.OrganizationID,
				Status:               workflowsignals.StatusFailed,
				Error:                err.Error(),
			})
		}
		data[i] = workflowsignals.ClusterInfo{
			// Name: metadata.,
		}
	}

	// if we arrive here, we managed to get all the metadata correctly
	// we can now update the cloud_provider table to ACTIVE
	if err := updateCloudProviderStatus(job.Args.Config.Database, job.Args.ProviderConnection.Id, len(allClusters)); err != nil {
		return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
			ProviderConnectionID: job.Args.ProviderConnection.Id,
			OrganizationID:       job.Args.OrganizationID,
			Status:               workflowsignals.StatusFailed,
			Error:                err.Error(),
		})
	}

	return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
		ProviderConnectionID: job.Args.ProviderConnection.Id,
		OrganizationID:       job.Args.OrganizationID,
		Status:               workflowsignals.StatusFailed,
		Error:                "",
	})
}

func publishResult(ctx context.Context, js stream.Stream, result workflowsignals.WorkflowFetchClustersStatusSignal) error {
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return js.Publish(ctx, workflowsignals.NatsWorkflowFetchClusterTopic, b)

}

func storeClusterMetadata(dbConfig *config.Database, metadata *models.ClusterMetadata) error {
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
	// Save the ClusterMetadata
	if err := db.Create(metadata).Error; err != nil {
		return fmt.Errorf("failed to create ClusterMetadata: %w", err)
	}
	return nil
}

func updateCloudProviderStatus(dbConfig *config.Database, providerID int64, importedClusters int) error {
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

	if err := db.Model(&models.CloudProvider{}).
		Where("id = ?", providerID).
		Updates(map[string]any{
			"status":            pb.ActivationStatus_ACTIVE,
			"imported_clusters": importedClusters,
		}).Error; err != nil {
		return err
	}
	return nil
}
