package store

import (
	"context"
	"encoding/json"

	"github.com/opisvigilant/futura/go-lib/stream"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
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
	IdempotencyKey        string `json:"idempotency_key,omitempty"`
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

func (es *EventFlattener) Flatten(ctx context.Context, msg *pbev.KubernetesEvent) error {
	data := &flatK8SEvent{
		OrganizationId:        msg.Enrichment.OrganizationId,
		ClusterId:             msg.Enrichment.ClusterId,
		K8SVersion:            msg.Enrichment.K8SVersion,
		ReceivedAtUnix:        msg.Enrichment.ReceivedAtUnix,
		IdempotencyKey:        msg.Metadata.IdempotencyKey,
		WatcherVersion:        msg.Metadata.WatcherVersion,
		ObjectKind:            msg.ObjectKind,
		ObjectName:            msg.ObjectName,
		ObjectUid:             msg.ObjectUid,
		ObjectFieldpath:       msg.ObjectFieldpath,
		ObjectTimestamp:       msg.ObjectTimestamp,
		ObjectNamespace:       msg.ObjectNamespace,
		EventSeverityNumber:   msg.EventSeverityNumber,
		EventSeverityText:     msg.EventSeverityText,
		EventReason:           msg.EventReason,
		EventAction:           msg.EventAction,
		EventStarttime:        msg.EventStarttime,
		EventName:             msg.EventName,
		EventMessage:          msg.EventMessage,
		EventUid:              msg.EventUid,
		EventCount:            msg.EventCount,
		ObjectApiVersion:      msg.ObjectApiVersion,
		ObjectResourceVersion: msg.ObjectResourceVersion,
		NodeName:              msg.NodeName,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return es.kc.Publish(ctx, StoreKubernetesEventsTopic, b)
}
