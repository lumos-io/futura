package workflowclusters

import (
	"context"
	"fmt"
	"time"

	"sync"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/opisvigilant/futura/apis/internal/config"
	pb "github.com/opisvigilant/futura/proto/gen/backend"
	"github.com/rs/zerolog/log"
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

	var client providerClient
	err := workflow.ExecuteActivity(ctx, CreateProviderClient, providerConfig{
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
		var metadata ClusterMetadata
		err = workflow.ExecuteActivity(ctx, FetchMetadata, client, clusterID).Get(ctx, &metadata)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed fetching metadata", "cluster", clusterID, "err", err)
			continue // or return err to fail fast
		}

		err = workflow.ExecuteActivity(ctx, StoreMetadata, metadata).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed storing metadata", "cluster", clusterID, "err", err)
			continue
		}
	}

	return nil
}

var (
	db     *gorm.DB
	dbOnce sync.Once
)

func NewDB(config *config.Database) *gorm.DB {
	dbOnce.Do(func() {
		// create a postgres client
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User,
			config.Password, config.Name, config.SSLMode,
		)
		var err error
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("couldn't open the connection to the db")
		}
	})
	return db
}

type providerClient interface{}

type providerConfig struct {
	Provider    pb.CloudProvider
	Credentials map[string]string
}

func CreateProviderClient(ctx context.Context, pc providerConfig) (providerClient, error) {
	return nil, nil
}

func FetchClusterIDs(ctx context.Context, client providerClient) ([]string, error) {
	// Some transformation or logic
	return nil, nil
}

type ClusterMetadata struct {
	OrganizationID uint
}

func FetchMetadata(ctx context.Context, input string) (ClusterMetadata, error) {
	return ClusterMetadata{}, nil
}

func StoreMetadata(ctx context.Context, metadata ClusterMetadata) error {
	// database := NewDB(nil)
	return nil
}
