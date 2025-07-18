package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/rs/zerolog/log"

	workflowclusters "github.com/opisvigilant/futura/apis/internal/workflow/clusters"

	tpclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	WorkflowTaskQueueName = "workflows-task-queue"
)

type WorkflowManager struct {
	config         *config.Configuration
	temporalClient tpclient.Client
}

func New(config *config.Configuration) (*WorkflowManager, error) {
	c, err := tpclient.Dial(tpclient.Options{
		Namespace: config.Temporal.Namespace,
		HostPort:  fmt.Sprintf("%s:%s", config.Temporal.Address, config.Temporal.Port),
		Identity:  "apis-workflow-manager",
	})
	if err != nil {
		return nil, err
	}
	return &WorkflowManager{
		config:         config,
		temporalClient: c,
	}, nil
}

func (c *WorkflowManager) StartWorker() error {
	w := worker.New(c.temporalClient, WorkflowTaskQueueName, worker.Options{
		EnableLoggingInReplay: true,
	})

	// register the fetch all clusters for all the accounts
	workflowclusters.RegisterWorkflowFetchClusters(w)

	return w.Run(worker.InterruptCh())
}

func (c *WorkflowManager) ExecuteFetchClustersWorkflow(input *workflowclusters.WorkflowFetchClustersInput) error {
	opts := tpclient.StartWorkflowOptions{
		ID:        "fetch-clusters-workflow-" + time.Now().String(),
		TaskQueue: WorkflowTaskQueueName,
	}

	run, err := c.temporalClient.ExecuteWorkflow(
		context.Background(),
		opts,
		workflowclusters.WorkflowFetchClusters,
		input,
	)

	log.Logger.Info().Msgf("workflow with ID `%s` started", run.GetRunID())

	return err
}

func (c *WorkflowManager) Stop() error {
	c.temporalClient.Close()
	return nil
}
