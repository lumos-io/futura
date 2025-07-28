package workflow

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	workflowclusters "github.com/opisvigilant/futura/apis/internal/workflow/clusters"
)

const (
	WorkflowTaskQueueName = "workflows-task-queue"
)

type WorkflowManager struct {
	config *config.Configuration

	dbpool      *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
}

func New(config *config.Configuration) (*WorkflowManager, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Password, config.Database.Name, config.Database.SSLMode,
	)

	dbPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(dbPool), nil)
	if err != nil {
		return nil, err
	}
	_, err = migrator.Migrate(context.Background(), rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return nil, err
	}

	workers := river.NewWorkers()
	// add other workers here
	river.AddWorker(workers, &workflowclusters.WorkflowFetchClustersWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, err
	}

	return &WorkflowManager{
		config:      config,
		dbpool:      dbPool,
		riverClient: riverClient,
	}, nil
}

func (c *WorkflowManager) StartWorkers() error {
	return c.riverClient.Start(context.Background())
}

func (c *WorkflowManager) ExecuteFetchClustersWorkflow(input *workflowclusters.WorkflowFetchClustersInput) error {
	_, err := c.riverClient.Insert(context.Background(), workflowclusters.WorkflowFetchClustersInput{
		Config:             input.Config,
		OrganizationID:     input.OrganizationID,
		ProviderConnection: input.ProviderConnection,
		Credentials:        input.Credentials,
		SecretID:           input.SecretID,
	}, nil)
	return err
}

func (c *WorkflowManager) Stop() error {
	// Stop fetching new work and wait for active jobs to finish.
	if err := c.riverClient.Stop(context.Background()); err != nil {
		return err
	}
	// close db pool
	c.dbpool.Close()
	return nil
}
