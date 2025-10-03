package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/opisvigilant/futura/analytics/internal/config"
	"github.com/rs/zerolog/log"

	pbsvc "github.com/opisvigilant/futura/proto/gen/analytics"
	pbtl "github.com/opisvigilant/futura/proto/gen/telemetry"
)

type AnalyticsServer struct {
	pbsvc.UnimplementedAnalyticsServiceServer
	clickhouse driver.Conn
	config     *config.Configuration
}

func NewAnalyticsServer(cfg *config.Configuration) (*AnalyticsServer, error) {
	// Connect to ClickHouse
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Clickhouse.Servers,
		Auth: clickhouse.Auth{
			Database: cfg.Clickhouse.Database,
			Username: cfg.Clickhouse.Username,
			Password: cfg.Clickhouse.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Info().Msg("✅ Connected to ClickHouse")

	return &AnalyticsServer{
		clickhouse: conn,
		config:     cfg,
	}, nil
}

func (a *AnalyticsServer) Close() error {
	if a.clickhouse != nil {
		return a.clickhouse.Close()
	}
	return nil
}

func (a *AnalyticsServer) GetEvents(ctx context.Context, req *pbsvc.GetEventsByClusterIdRequest) (*pbsvc.GetEventsResponse, error) {
	log.Debug().
		Uint32("org_id", req.OrganizationId).
		Int64("cluster_id", req.ClusterId).
		Int32("limit", req.Limit).
		Int64("cursor", req.Cursor).
		Bool("reverse", req.Reverse).
		Msg("GetEvents request")

	// Default limit if not specified
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000 // Cap at 1000
	}

	// Build query based on pagination direction
	var query string
	var orderDirection string
	if req.Reverse {
		// Going backward in time (older events)
		orderDirection = "ASC"
		query = `
			SELECT
				organization_id,
				cluster_id,
				k8s_version,
				idempotency_key,
				watcher_version,
				received_at_unix,
				object_kind,
				object_name,
				object_uid,
				object_fieldpath,
				object_timestamp,
				object_namespace,
				event_severity_number,
				event_severity_text,
				event_reason,
				event_action,
				event_starttime,
				event_name,
				event_message,
				event_uid,
				event_count,
				object_api_version,
				object_resource_version,
				node_name
			FROM kubernetes_events
			WHERE organization_id = ? AND cluster_id = ?
			  AND object_timestamp < ?
			ORDER BY object_timestamp ` + orderDirection + `
			LIMIT ?
		`
	} else {
		// Going forward in time (newer events)
		orderDirection = "DESC"
		if req.Cursor == 0 {
			// First page - no cursor, get latest events
			query = `
				SELECT
					organization_id,
					cluster_id,
					k8s_version,
					idempotency_key,
					watcher_version,
					received_at_unix,
					object_kind,
					object_name,
					object_uid,
					object_fieldpath,
					object_timestamp,
					object_namespace,
					event_severity_number,
					event_severity_text,
					event_reason,
					event_action,
					event_starttime,
					event_name,
					event_message,
					event_uid,
					event_count,
					object_api_version,
					object_resource_version,
					node_name
				FROM kubernetes_events
				WHERE organization_id = ? AND cluster_id = ?
				ORDER BY object_timestamp ` + orderDirection + `
				LIMIT ?
			`
		} else {
			query = `
				SELECT
					organization_id,
					cluster_id,
					k8s_version,
					idempotency_key,
					watcher_version,
					received_at_unix,
					object_kind,
					object_name,
					object_uid,
					object_fieldpath,
					object_timestamp,
					object_namespace,
					event_severity_number,
					event_severity_text,
					event_reason,
					event_action,
					event_starttime,
					event_name,
					event_message,
					event_uid,
					event_count,
					object_api_version,
					object_resource_version,
					node_name
				FROM kubernetes_events
				WHERE organization_id = ? AND cluster_id = ?
				  AND object_timestamp < ?
				ORDER BY object_timestamp ` + orderDirection + `
				LIMIT ?
			`
		}
	}

	var rows driver.Rows
	var err error

	if req.Cursor == 0 && !req.Reverse {
		// First page query
		rows, err = a.clickhouse.Query(ctx, query, req.OrganizationId, req.ClusterId, limit)
	} else {
		// Subsequent pages with cursor
		rows, err = a.clickhouse.Query(ctx, query, req.OrganizationId, req.ClusterId, req.Cursor, limit)
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to query ClickHouse")
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*pbtl.KubernetesEvent
	var lastTimestamp int64
	var firstTimestamp int64

	for rows.Next() {
		var (
			orgID                 uint32
			clusterID             int64
			k8sVersion            string
			idempotencyKey        string
			watcherVersion        string
			receivedAtUnix        int64
			objectKind            string
			objectName            string
			objectUID             string
			objectFieldpath       string
			objectTimestamp       int64
			objectNamespace       string
			eventSeverityNumber   int64
			eventSeverityText     string
			eventReason           string
			eventAction           string
			eventStarttime        string
			eventName             string
			eventMessage          string
			eventUID              string
			eventCount            int64
			objectAPIVersion      string
			objectResourceVersion string
			nodeName              string
		)

		if err := rows.Scan(
			&orgID,
			&clusterID,
			&k8sVersion,
			&idempotencyKey,
			&watcherVersion,
			&receivedAtUnix,
			&objectKind,
			&objectName,
			&objectUID,
			&objectFieldpath,
			&objectTimestamp,
			&objectNamespace,
			&eventSeverityNumber,
			&eventSeverityText,
			&eventReason,
			&eventAction,
			&eventStarttime,
			&eventName,
			&eventMessage,
			&eventUID,
			&eventCount,
			&objectAPIVersion,
			&objectResourceVersion,
			&nodeName,
		); err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Track first and last timestamps for cursors
		if firstTimestamp == 0 {
			firstTimestamp = objectTimestamp
		}
		lastTimestamp = objectTimestamp

		event := &pbtl.KubernetesEvent{
			ObjectKind:            objectKind,
			ObjectName:            objectName,
			ObjectUid:             objectUID,
			ObjectFieldpath:       objectFieldpath,
			ObjectTimestamp:       objectTimestamp,
			ObjectNamespace:       objectNamespace,
			EventSeverityNumber:   eventSeverityNumber,
			EventSeverityText:     eventSeverityText,
			EventReason:           eventReason,
			EventAction:           eventAction,
			EventStarttime:        eventStarttime,
			EventName:             eventName,
			EventMessage:          eventMessage,
			EventUid:              eventUID,
			EventCount:            eventCount,
			ObjectApiVersion:      objectAPIVersion,
			ObjectResourceVersion: objectResourceVersion,
			NodeName:              nodeName,
			Metadata: &pbtl.Metadata{
				IdempotencyKey: idempotencyKey,
				WatcherVersion: watcherVersion,
			},
			Enrichment: &pbtl.EnrichmentMetadata{
				OrganizationId: orgID,
				ClusterId:      clusterID,
				K8SVersion:     k8sVersion,
				ReceivedAtUnix: receivedAtUnix,
			},
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Row iteration error")
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Set cursors for pagination
	var nextCursor, prevCursor int64
	if len(events) > 0 {
		if req.Reverse {
			// For reverse pagination, next cursor is the last event's timestamp
			nextCursor = lastTimestamp
			prevCursor = firstTimestamp
		} else {
			// For forward pagination, next cursor is the last event's timestamp
			nextCursor = lastTimestamp
			prevCursor = firstTimestamp
		}
	}

	log.Debug().
		Int("event_count", len(events)).
		Int64("next_cursor", nextCursor).
		Int64("prev_cursor", prevCursor).
		Msg("GetEvents response")

	return &pbsvc.GetEventsResponse{
		Events:     events,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
	}, nil
}
