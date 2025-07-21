package workflowsignals

type Command string

const (
	CommandCancel Command = "cancel"
	CommandRetry  Command = "retry"
)

const SignalControlName = "workflow-signal-control"

type WorkflowControlSignal struct {
	Command Command `json:"command"` // e.g., "cancel", "retry", etc.
	Payload string  `json:"payload"` // optional context
}

type WorkflowStatus string

const (
	StatusSuccess WorkflowStatus = "SUCCESS"
	StatusFailed  WorkflowStatus = "FAILED"
)

const NatsWorkflowFetchClusterTopic = "organization.%s.fetchclustersworkflow.result"

type WorkflowFetchClustersStatusSignal struct {
	ProviderConnectionID int64          `json:"providerConnectionId"`
	OrganizationID       uint           `json:"organizationId"`
	Status               WorkflowStatus `json:"status"` // e.g., "FAILED", "SUCCESS"
	Error                string         `json:"error,omitempty"`
}
