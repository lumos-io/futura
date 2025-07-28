package workflowclusters

import (
	"context"

	"github.com/opisvigilant/futura/apis/internal/config"
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

	/*
		

	*/

	return nil
}
