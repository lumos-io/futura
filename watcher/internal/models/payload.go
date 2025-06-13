package models

type Metadata struct {
	IdempotencyKey string `json:"idempotency_key"`
	NodeID         string `json:"node_id"`
	WatcherVersion string `json:"watcher_version"`
}

type HealthCheckPayload struct {
	Metadata Metadata `json:"metadata"`
	Info     struct {
		MetricsEnabled bool `json:"metrics"`
	} `json:"watcher_info"`
	Telemetry struct {
		KernelVersion string `json:"kernel_version"`
		K8sVersion    string `json:"k8s_version"`
		CloudProvider string `json:"cloud_provider"`
	} `json:"telemetry"`
}

type EventPayload struct {
	Metadata Metadata `json:"metadata"`
	Events   []any    `json:"events"`
}
