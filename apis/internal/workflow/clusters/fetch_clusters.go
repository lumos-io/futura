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
	js, err := stream.NewNATSJetstreamClient(job.Args.Config.Nats.Servers, "workflow_stream", []string{"fetchclustersworkflow.*"})
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
		return js.Publish(context.Background(), workflowsignals.NatsWorkflowFetchClusterTopic, b)
	}

	allClusters, err := client.FetchClusters(ctx)
	if err != nil {
		return publishResult(ctx, js, job.Args.ProviderConnection.Id, job.Args.OrganizationID, workflowsignals.StatusFailed, err.Error())
	}

	for _, clusterID := range allClusters {
		metadata, err := client.FetchClusterMetadata(ctx, clusterID)
		// enrich the object
		metadata.CloudProviderID = uint(job.Args.ProviderConnection.Id)
		metadata.OrganizationID = job.Args.OrganizationID

		if err != nil {
			return publishResult(ctx, js, job.Args.ProviderConnection.Id, job.Args.OrganizationID, workflowsignals.StatusFailed, err.Error())
		}
		if err := storeClusterMetadata(job.Args.Config.Database, metadata); err != nil {
			return publishResult(ctx, js, job.Args.ProviderConnection.Id, job.Args.OrganizationID, workflowsignals.StatusFailed, err.Error())
		}
	}

	// if we arrive here, we managed to get all the metadata correctly
	// we can now update the cloud_provider table to ACTIVE
	if err := updateCloudProviderStatus(job.Args.Config.Database, job.Args.ProviderConnection.Id); err != nil {
		return publishResult(ctx, js, job.Args.ProviderConnection.Id, job.Args.OrganizationID, workflowsignals.StatusFailed, err.Error())
	}

	return publishResult(ctx, js, job.Args.ProviderConnection.Id, job.Args.OrganizationID, workflowsignals.StatusSuccess, "")
}

func publishResult(ctx context.Context, js stream.Stream, providerConnectionID int64, organizationID uint, success workflowsignals.WorkflowStatus, errMsg string) error {
	b, err := json.Marshal(workflowsignals.WorkflowFetchClustersStatusSignal{
		ProviderConnectionID: providerConnectionID,
		OrganizationID:       organizationID,
		Status:               success,
		Error:                errMsg,
	})
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

func updateCloudProviderStatus(dbConfig *config.Database, providerID int64) error {
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
	if err := db.Model(&models.CloudProvider{}).Where("id = ?", providerID).Update("status", pb.ActivationStatus_ACTIVE).Error; err != nil {
		return err
	}
	return nil
}
