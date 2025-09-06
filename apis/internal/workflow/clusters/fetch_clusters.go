package workflowclusters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/providers"
	workflowsignals "github.com/opisvigilant/futura/apis/internal/workflow/signals"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/go-lib/kv"
	"github.com/opisvigilant/futura/go-lib/stream"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
	pbmt "github.com/opisvigilant/futura/proto/gen/common"
	"github.com/riverqueue/river"
)

type WorkflowFetchClustersInput struct {
	Config *config.Configuration

	OrganizationID     uint
	ProviderConnection *pb.ProviderConnection
	Credentials        map[string]string
	SecretID           string
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

	kvs, err := kv.NewRedisKVStore(job.Args.Config.Redis.Servers)
	if err != nil {
		return fmt.Errorf("failed to connect to KV Stora: %v", err)
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
		// generate api key
		ak, err := createAPIKeyEntry(ctx, kvs, job.Args.OrganizationID, metadata, job.Args.SecretID, job.Args.ProviderConnection)
		if err != nil {
			return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
				ProviderConnectionID: job.Args.ProviderConnection.Id,
				OrganizationID:       job.Args.OrganizationID,
				Status:               workflowsignals.StatusFailed,
				Error:                err.Error(),
			})
		}

		// enrich the object
		metadata.ProviderConnectionID = uint(job.Args.ProviderConnection.Id)
		metadata.OrganizationID = job.Args.OrganizationID
		metadata.APIKey = ak

		if err := storeClusterMetadata(job.Args.Config.Database, metadata); err != nil {
			return publishResult(ctx, js, workflowsignals.WorkflowFetchClustersStatusSignal{
				ProviderConnectionID: job.Args.ProviderConnection.Id,
				OrganizationID:       job.Args.OrganizationID,
				Status:               workflowsignals.StatusFailed,
				Error:                err.Error(),
			})
		}

		data[i] = workflowsignals.ClusterInfo{
			Name:   metadata.Name,
			APIKey: ak,
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
		Status:               workflowsignals.StatusSuccess,
		Data:                 data,
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

	if err := db.Model(&models.ProviderConnection{}).
		Where("id = ?", providerID).
		Updates(map[string]any{
			"status":            pb.ActivationStatus_ACTIVE,
			"imported_clusters": importedClusters,
		}).Error; err != nil {
		return err
	}
	return nil
}

func createAPIKeyEntry(ctx context.Context, store kv.KVStore, orgID uint, metadata *models.ClusterMetadata, secretID string, provider *pb.ProviderConnection) (string, error) {
	// generate the API Key value here
	apiKey, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(pbmt.ApiKeyInfo{
		OrganizationId:       uint32(orgID),
		Value:                apiKey,
		Status:               pbmt.ApiKeyStatus_ACTIVE,
		ClusterName:          metadata.Name,
		KubernetesVersion:    metadata.GetKubernetesVersion(),
		Region:               metadata.GetRegion(),
		ProviderConnectionId: uint32(metadata.ProviderConnectionID),
		ClusterId:            uint32(metadata.ID),
		SecretId:             secretID,
		CloudProviderEnum:    int32(provider.Provider.Number()),
		CreatedAt:            time.Now().String(),
	})
	if err != nil {
		return "", nil
	}

	// namespace: apikeys:<apikey_value> - key: providerConnectionID apikey_info
	ns := fmt.Sprintf("apikeys:%s", apiKey)
	if err := store.Put(ctx, ns, fmt.Sprintf("%d", metadata.ProviderConnectionID), b); err != nil {
		return "", nil
	}
	return apiKey, nil
}
