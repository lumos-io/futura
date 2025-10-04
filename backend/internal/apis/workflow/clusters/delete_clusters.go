package workflowclusters

import (
	"context"

	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/apis/providers"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/opisvigilant/futura/backend/pkg/kv"
	"github.com/riverqueue/river"
)

type WorkflowDeleteClustersInput struct {
	Config *config.Configuration

	OrganizationID uint
	ConnectionID   uint
}

func (WorkflowDeleteClustersInput) Kind() string { return "workflow_delete_clusters" }

type WorkflowDeleteClustersWorker struct {
	river.WorkerDefaults[WorkflowDeleteClustersInput]
}

func (w *WorkflowDeleteClustersWorker) Work(ctx context.Context, job *river.Job[WorkflowDeleteClustersInput]) error {
	// fetch the current provider object
	var cp models.ProviderConnection
	if err := models.GetDB().Where("id = ? AND organization_id = ?", job.Args.ConnectionID, job.Args.OrganizationID).First(&cp).Error; err != nil {
		return err
	}
	// delete secrets
	providerAuth, err := providers.NewProviderAuth(job.Args.Config)
	if err != nil {
		return err
	}
	if err := providerAuth.DeleteCredentials(cp.SecretID); err != nil {
		return err
	}

	// delete clusters
	var clusters []models.ClusterMetadata
	if err := models.GetDB().
		Where("organization_id = ? AND provider_connection_id = ?", job.Args.OrganizationID, job.Args.ConnectionID).
		Find(&clusters).Error; err != nil {
		return err
	}

	kvstore, err := kv.NewRedisKVStore(job.Args.Config.Redis.Servers)
	if err != nil {
		return err
	}
	defer kvstore.Close()

	for _, cluster := range clusters {
		if err := kvstore.Delete(ctx, "apikeys", cluster.APIKey); err != nil {
			return err
		}
		// delete cluster metadata from DB
		// CHECK: does it a CASCADE delete?
		if err := models.GetDB().Delete(&cluster).Error; err != nil {
			return err
		}
	}

	// delete providerconnection object from DB
	if err := models.GetDB().Delete(&cp).Error; err != nil {
		return err
	}

	return nil
}
