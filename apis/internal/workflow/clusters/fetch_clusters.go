package workflowclusters

import (
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func RegisterWorkflowFetchClusters(worker worker.Worker) {
	// register workflow and activities
	// ...
}

type WorkflowFetchClustersInput struct{}

func WorkflowFetchClusters(ctx workflow.Context, input WorkflowFetchClustersInput) error {
	// create a postgres client
	// ...

	// fetch the connections from DB
	// ...

	// for each connection
	//    create provider client
	//	  get Kubernetes clusters
	//    for each cluster
	//		 fetch metadata
	//       store metadata in DB
	// terminate workflow

	workflow.GetLogger(ctx).Info("MyWorkflow started", "input", input)

	return nil
}
