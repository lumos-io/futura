package models

type Metadata struct {
	IdempotencyKey string `json:"idempotency_key"`
	NodeID         string `json:"node_id"`
	WatcherVersion string `json:"watcher_version"`
}

type HealthCheckPayload struct {
	Metadata Metadata `json:"metadata"`
	Info     struct {
		EbpfEnabled    bool `json:"ebpf"`
		MetricsEnabled bool `json:"metrics"`
	} `json:"watcher_info"`
	Telemetry struct {
		KernelVersion string `json:"kernel_version"`
		K8sVersion    string `json:"k8s_version"`
		CloudProvider string `json:"cloud_provider"`
	} `json:"telemetry"`
}

type EventPayload struct {
	Metadata Metadata      `json:"metadata"`
	Events   []interface{} `json:"events"`
}

// 0) StartTime
// 1) Latency
// 2) Source IP
// 3) Source Type
// 4) Source ID
// 5) Source Port
// 6) Destination IP
// 7) Destination Type
// 8) Destination ID
// 9) Destination Port
// 10) Protocol
// 11) Response Status Code
// 12) Fail Reason // TODO: not used yet
// 13) Method
// 14) Path
// 15) Encrypted (bool)
// 16) Seq
// 17) Tid

type ReqInfo [18]interface{}

type RequestsPayload struct {
	Metadata Metadata   `json:"metadata"`
	Requests []*ReqInfo `json:"requests"`
}

// 0) Timestamp
// 1) Tcp Seq Num
// 2) Tid
// 3) Ingress(true), Egress(false)
type TraceInfo [4]interface{}

type TracePayload struct {
	Metadata Metadata     `json:"metadata"`
	Traces   []*TraceInfo `json:"traffic"`
}

