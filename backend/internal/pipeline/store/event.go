package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opisvigilant/futura/backend/pkg/stream"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	StoreKubernetesEventsTopic = "store.k8s.events"
)

type EventFlattener struct {
	kc stream.Stream
}

func NewEventFlattener(kc stream.Stream) *EventFlattener {
	return &EventFlattener{
		kc: kc,
	}
}

type flatK8SEvent struct {
	OrganizationId        uint32 `json:"organization_id"`
	ClusterId             int64  `json:"cluster_id"`
	K8SVersion            string `json:"k8s_version"`
	ReceivedAtUnix        int64  `json:"received_at_unix"`
	IdempotencyKey        string `json:"idempotency_key"`
	WatcherVersion        string `json:"watcher_version,omitempty"`
	ObjectKind            string `json:"object_kind,omitempty"`
	ObjectName            string `json:"object_name,omitempty"`
	ObjectUid             string `json:"object_uid,omitempty"`
	ObjectFieldpath       string `json:"object_fieldpath,omitempty"`
	ObjectTimestamp       int64  `json:"object_timestamp,omitempty"`
	ObjectNamespace       string `json:"object_namespace,omitempty"`
	EventSeverityNumber   int64  `json:"event_severity_number,omitempty"`
	EventSeverityText     string `json:"event_severity_text,omitempty"`
	EventReason           string `json:"event_reason,omitempty"`
	EventAction           string `json:"event_action,omitempty"`
	EventStarttime        string `json:"event_starttime,omitempty"`
	EventName             string `json:"event_name,omitempty"`
	EventMessage          string `json:"event_message,omitempty"`
	EventUid              string `json:"event_uid,omitempty"`
	EventCount            int64  `json:"event_count,omitempty"`
	ObjectApiVersion      string `json:"object_api_version,omitempty"`
	ObjectResourceVersion string `json:"object_resource_version,omitempty"`
	NodeName              string `json:"node_name,omitempty"`
}

func (es *EventFlattener) Flatten(ctx context.Context, msg *pb.KubernetesEvent) error {
	if msg == nil {
		return fmt.Errorf("nil KubernetesEvent")
	}

	data := &flatK8SEvent{
		OrganizationId:        safeUInt32Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) uint32 { return e.OrganizationId }),
		ClusterId:             safeInt64Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) int64 { return e.ClusterId }),
		K8SVersion:            safeStringPtr(msg.Enrichment, func(e *pb.EnrichmentMetadata) string { return e.K8SVersion }),
		ReceivedAtUnix:        safeInt64Ptr(msg.Enrichment, func(e *pb.EnrichmentMetadata) int64 { return e.ReceivedAtUnix }),
		IdempotencyKey:        safeStringPtr(msg.Metadata, func(m *pb.Metadata) string { return m.IdempotencyKey }),
		WatcherVersion:        safeStringPtr(msg.Metadata, func(m *pb.Metadata) string { return m.WatcherVersion }),
		ObjectKind:            msg.GetObjectKind(),
		ObjectName:            msg.GetObjectName(),
		ObjectUid:             msg.GetObjectUid(),
		ObjectFieldpath:       msg.GetObjectFieldpath(),
		ObjectTimestamp:       msg.GetObjectTimestamp(),
		ObjectNamespace:       msg.GetObjectNamespace(),
		EventSeverityNumber:   msg.GetEventSeverityNumber(),
		EventSeverityText:     msg.GetEventSeverityText(),
		EventReason:           msg.GetEventReason(),
		EventAction:           msg.GetEventAction(),
		EventStarttime:        msg.GetEventStarttime(),
		EventName:             msg.GetEventName(),
		EventMessage:          msg.GetEventMessage(),
		EventUid:              msg.GetEventUid(),
		EventCount:            msg.GetEventCount(),
		ObjectApiVersion:      msg.GetObjectApiVersion(),
		ObjectResourceVersion: msg.GetObjectResourceVersion(),
		NodeName:              msg.GetNodeName(),
	}

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return es.kc.Publish(ctx, StoreKubernetesEventsTopic, b)
}
