package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/rs/zerolog/log"

	pbsvc "github.com/opisvigilant/futura/proto/gen/analytics"
	pbtl "github.com/opisvigilant/futura/proto/gen/telemetry"
)

// AnalyticsServer provides analytics query functions
type AnalyticsServer struct {
	clickhouse driver.Conn
	config     *config.Configuration
}

// New creates a new Analytics instance
func New(cfg *config.Configuration) (*AnalyticsServer, error) {
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

// Close closes the ClickHouse connection
func (a *AnalyticsServer) Close() error {
	if a.clickhouse != nil {
		return a.clickhouse.Close()
	}
	return nil
}

// GetEvents retrieves events for a cluster
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

// GetNodes retrieves nodes for a cluster
func (a *AnalyticsServer) GetNodes(ctx context.Context, req *pbsvc.GetNodesByClusterIdRequest) (*pbsvc.GetNodesResponse, error) {
	log.Debug().
		Uint32("org_id", req.OrganizationId).
		Int64("cluster_id", req.ClusterId).
		Msg("GetNodes request")

	// Query to get latest node information with all related data
	// Simplified approach: get all data separately and join in application or use simpler subqueries
	query := `
		SELECT
			nodes.name,
			nodes.uid,
			nodes.status,
			nodes.kubelet_version,
			nodes.labels,
			nodes.cloud_provider,
			nodes.latest_timestamp,
			conditions.condition_types,
			conditions.condition_statuses,
			conditions.reasons,
			conditions.messages,
			resources.cpu,
			resources.memory,
			resources.pods,
			resources.ephemeral_storage,
			COALESCE(pod_counts.pod_count, 0) as current_pod_count
		FROM (
			SELECT
				name,
				argMax(uid, timestamp) as uid,
				argMax(status, timestamp) as status,
				argMax(kubelet_version, timestamp) as kubelet_version,
				argMax(labels, timestamp) as labels,
				argMax(cloud_provider, timestamp) as cloud_provider,
				max(timestamp) as latest_timestamp
			FROM kubernetes_objects
			WHERE organization_id = ?
			  AND cluster_id = ?
			  AND kind = 'Node'
			  AND timestamp >= now() - INTERVAL 5 MINUTE
			GROUP BY name
		) AS nodes
		LEFT JOIN (
			SELECT
				uid,
				groupArray(condition_type) as condition_types,
				groupArray(condition_status) as condition_statuses,
				groupArray(reason) as reasons,
				groupArray(message) as messages
			FROM kubernetes_node_conditions
			WHERE timestamp >= now() - INTERVAL 5 MINUTE
			GROUP BY uid
		) AS conditions ON nodes.uid = conditions.uid
		LEFT JOIN (
			SELECT
				uid,
				argMax(cpu, timestamp) as cpu,
				argMax(memory, timestamp) as memory,
				argMax(pods, timestamp) as pods,
				argMax(ephemeral_storage, timestamp) as ephemeral_storage
			FROM kubernetes_allocatable_resources
			WHERE timestamp >= now() - INTERVAL 5 MINUTE
			GROUP BY uid
		) AS resources ON nodes.uid = resources.uid
		LEFT JOIN (
			SELECT
				node_name,
				count(*) as pod_count
			FROM kubernetes_objects
			WHERE organization_id = ?
			  AND cluster_id = ?
			  AND kind = 'Pod'
			  AND status = 'Running'
			  AND timestamp >= now() - INTERVAL 5 MINUTE
			GROUP BY node_name
		) AS pod_counts ON nodes.name = pod_counts.node_name
		ORDER BY nodes.name
	`

	rows, err := a.clickhouse.Query(ctx, query,
		req.OrganizationId, req.ClusterId, // nodes
		req.OrganizationId, req.ClusterId, // pod_counts
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query nodes from ClickHouse")
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*pbsvc.NodeInfo

	for rows.Next() {
		var (
			name              string
			uid               string
			status            string
			kubeletVersion    string
			labels            map[string]string
			cloudProvider     string
			timestamp         time.Time
			conditionTypes    []string
			conditionStatuses []string
			reasons           []string
			messages          []string
			cpu               string
			memory            string
			pods              string
			ephemeralStorage  string
			currentPodCount   int32
		)

		if err := rows.Scan(
			&name,
			&uid,
			&status,
			&kubeletVersion,
			&labels,
			&cloudProvider,
			&timestamp,
			&conditionTypes,
			&conditionStatuses,
			&reasons,
			&messages,
			&cpu,
			&memory,
			&pods,
			&ephemeralStorage,
			&currentPodCount,
		); err != nil {
			log.Error().Err(err).Msg("Failed to scan node row")
			continue
		}

		// Build conditions
		var conditions []*pbtl.NodeCondition
		for i := range conditionTypes {
			var reason, message string
			if i < len(reasons) {
				reason = reasons[i]
			}
			if i < len(messages) {
				message = messages[i]
			}
			conditions = append(conditions, &pbtl.NodeCondition{
				Type:    conditionTypes[i],
				Status:  conditionStatuses[i],
				Reason:  reason,
				Message: message,
			})
		}

		// Extract instance type and zone from labels
		instanceType := labels["node.kubernetes.io/instance-type"]
		zone := labels["topology.kubernetes.io/zone"]
		if zone == "" {
			zone = labels["failure-domain.beta.kubernetes.io/zone"]
		}

		// Extract capacity type (spot vs on-demand)
		capacityType := "on-demand"
		if labels["eks.amazonaws.com/capacityType"] == "SPOT" ||
			labels["cloud.google.com/gke-preemptible"] == "true" ||
			labels["kubernetes.azure.com/scalesetpriority"] == "spot" {
			capacityType = "spot"
		}

		nodeInfo := &pbsvc.NodeInfo{
			Name:           name,
			Uid:            uid,
			Status:         status,
			KubeletVersion: kubeletVersion,
			InstanceType:   instanceType,
			Zone:           zone,
			CloudProvider:  cloudProvider,
			CapacityType:   capacityType,
			Timestamp:      timestamp.Unix(),
			Conditions:     conditions,
			Allocatable: &pbtl.AllocatableResources{
				Cpu:              cpu,
				Memory:           memory,
				Pods:             pods,
				EphemeralStorage: ephemeralStorage,
			},
			PodCount: currentPodCount,
		}

		nodes = append(nodes, nodeInfo)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Row iteration error")
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Debug().
		Int("node_count", len(nodes)).
		Msg("GetNodes response")

	return &pbsvc.GetNodesResponse{
		Nodes: nodes,
	}, nil
}

// GetClusterConfig retrieves cluster configuration
func (a *AnalyticsServer) GetClusterConfig(ctx context.Context, req *pbsvc.GetClusterConfigRequest) (*pbsvc.ClusterConfigResponse, error) {
	log.Debug().
		Uint32("org_id", req.OrganizationId).
		Int64("cluster_id", req.ClusterId).
		Msg("GetClusterConfig request")

	// For now, return mock data
	// TODO: Store cluster config in ClickHouse or fetch from operator
	return &pbsvc.ClusterConfigResponse{
		ClusterScalingMode:   "recommend",
		EnableHpa:            true,
		EnableVpa:            true,
		EnableNodeHandler:    true,
		CostSensitivity:      "balanced",
		MonthlyBudget:        "$5000",
		SpotInstancesAllowed: true,
		MaxSpotPercentage:    "30%",
	}, nil
}

// GetServices retrieves services for a cluster
func (a *AnalyticsServer) GetServices(ctx context.Context, req *pbsvc.GetServicesByClusterIdRequest) (*pbsvc.GetServicesResponse, error) {
	log.Debug().
		Uint32("org_id", req.OrganizationId).
		Int64("cluster_id", req.ClusterId).
		Msg("GetServices request")

	// Query to get latest eBPF metrics aggregated by service
	query := `
		SELECT
			service_name,
			namespace,
			argMax(pod_name, timestamp) as pod_name,
			argMax(container_id, timestamp) as container_name,
			-- HTTP metrics (sum across all containers for the service)
			sum(http_request_count) as http_request_count,
			sum(http_error_count) as http_error_count,
			avg(http_error_rate) as http_error_rate,
			avg(http_latency_p50) as http_latency_p50,
			avg(http_latency_p95) as http_latency_p95,
			avg(http_latency_p99) as http_latency_p99,
			-- Memory metrics
			sum(memory_alloc_count) as memory_alloc_count,
			sum(memory_bytes_allocated) as memory_alloc_bytes,
			sum(memory_free_count) as memory_free_count,
			sum(memory_net_allocated) as memory_net_allocated,
			sum(memory_gc_count) as memory_gc_count,
			-- CPU metrics
			avg(cpu_utilization) as cpu_utilization,
			sum(cpu_context_switches) as cpu_context_switches,
			avg(cpu_runqueue_latency_us) as cpu_runqueue_latency_us,
			-- Network metrics
			sum(network_bytes_sent) as network_bytes_sent,
			sum(network_bytes_received) as network_bytes_received,
			sum(network_connections_established) as network_connections_active,
			sum(network_connections_established + network_connections_closed) as network_connections_total,
			-- Filesystem metrics
			sum(fs_read_ops) as fs_read_ops,
			sum(fs_write_ops) as fs_write_ops,
			sum(fs_bytes_read) as fs_bytes_read,
			sum(fs_bytes_written) as fs_bytes_written,
			avg(fs_read_latency_p95_us) as fs_read_latency_avg_us,
			avg(fs_write_latency_p95_us) as fs_write_latency_avg_us,
			max(timestamp) as latest_timestamp
		FROM ebpf_metrics
		WHERE organization_id = ?
		  AND cluster_id = ?
		  AND service_name != ''
		  AND timestamp >= now() - INTERVAL 5 MINUTE
		GROUP BY service_name, namespace
		ORDER BY service_name, namespace
	`

	rows, err := a.clickhouse.Query(ctx, query, req.OrganizationId, req.ClusterId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query services from ClickHouse")
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []*pbsvc.ServiceMetrics

	for rows.Next() {
		var (
			serviceName              string
			namespace                string
			podName                  string
			containerName            string
			httpRequestCount         uint64
			httpErrorCount           uint64
			httpErrorRate            float64
			httpLatencyP50           float64
			httpLatencyP95           float64
			httpLatencyP99           float64
			memoryAllocCount         uint64
			memoryAllocBytes         uint64
			memoryFreeCount          uint64
			memoryNetAllocated       uint64
			memoryGcCount            uint64
			cpuUtilization           float64
			cpuContextSwitches       uint64
			cpuRunqueueLatencyUs     float64
			networkBytesSent         uint64
			networkBytesReceived     uint64
			networkConnectionsActive uint64
			networkConnectionsTotal  uint64
			fsReadOps                uint64
			fsWriteOps               uint64
			fsBytesRead              uint64
			fsBytesWritten           uint64
			fsReadLatencyAvgUs       float64
			fsWriteLatencyAvgUs      float64
			timestamp                time.Time
		)

		if err := rows.Scan(
			&serviceName,
			&namespace,
			&podName,
			&containerName,
			&httpRequestCount,
			&httpErrorCount,
			&httpErrorRate,
			&httpLatencyP50,
			&httpLatencyP95,
			&httpLatencyP99,
			&memoryAllocCount,
			&memoryAllocBytes,
			&memoryFreeCount,
			&memoryNetAllocated,
			&memoryGcCount,
			&cpuUtilization,
			&cpuContextSwitches,
			&cpuRunqueueLatencyUs,
			&networkBytesSent,
			&networkBytesReceived,
			&networkConnectionsActive,
			&networkConnectionsTotal,
			&fsReadOps,
			&fsWriteOps,
			&fsBytesRead,
			&fsBytesWritten,
			&fsReadLatencyAvgUs,
			&fsWriteLatencyAvgUs,
			&timestamp,
		); err != nil {
			log.Error().Err(err).Msg("Failed to scan service row")
			continue
		}

		service := &pbsvc.ServiceMetrics{
			ServiceName:              serviceName,
			Namespace:                namespace,
			PodName:                  podName,
			ContainerName:            containerName,
			HttpRequestCount:         httpRequestCount,
			HttpErrorCount:           httpErrorCount,
			HttpErrorRate:            httpErrorRate,
			HttpLatencyP50:           httpLatencyP50,
			HttpLatencyP95:           httpLatencyP95,
			HttpLatencyP99:           httpLatencyP99,
			MemoryAllocCount:         memoryAllocCount,
			MemoryAllocBytes:         memoryAllocBytes,
			MemoryFreeCount:          memoryFreeCount,
			MemoryNetAllocated:       memoryNetAllocated,
			MemoryGcCount:            memoryGcCount,
			CpuUtilization:           cpuUtilization,
			CpuContextSwitches:       cpuContextSwitches,
			CpuRunqueueLatencyUs:     cpuRunqueueLatencyUs,
			NetworkBytesSent:         networkBytesSent,
			NetworkBytesReceived:     networkBytesReceived,
			NetworkConnectionsActive: networkConnectionsActive,
			NetworkConnectionsTotal:  networkConnectionsTotal,
			FsReadOps:                fsReadOps,
			FsWriteOps:               fsWriteOps,
			FsBytesRead:              fsBytesRead,
			FsBytesWritten:           fsBytesWritten,
			FsReadLatencyAvgUs:       fsReadLatencyAvgUs,
			FsWriteLatencyAvgUs:      fsWriteLatencyAvgUs,
			Timestamp:                timestamp.Unix(),
		}

		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Row iteration error")
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Debug().
		Int("service_count", len(services)).
		Msg("GetServices response")

	return &pbsvc.GetServicesResponse{
		Services: services,
	}, nil
}

// GetOverviewMetrics retrieves overview metrics for a cluster
func (a *AnalyticsServer) GetOverviewMetrics(ctx context.Context, req *pbsvc.GetOverviewMetricsRequest) (*pbsvc.OverviewMetrics, error) {
	log.Debug().
		Uint32("org_id", req.OrganizationId).
		Int64("cluster_id", req.ClusterId).
		Msg("GetOverviewMetrics request")

	// Query to aggregate cluster overview metrics
	query := `
		WITH latest_states AS (
			SELECT
				kind,
				name,
				namespace,
				argMax(status, timestamp) as status,
				max(timestamp) as latest_timestamp
			FROM kubernetes_objects
			WHERE organization_id = ?
			  AND cluster_id = ?
			  AND timestamp >= now() - INTERVAL 5 MINUTE
			GROUP BY kind, name, namespace
		)
		SELECT
			-- Nodes
			countIf(kind = 'Node') as nodes_total,
			countIf(kind = 'Node' AND status = 'Ready') as nodes_ready,
			countIf(kind = 'Node' AND status != 'Ready') as nodes_not_ready,
			-- Workloads
			countIf(kind = 'Deployment') as deployments,
			countIf(kind = 'StatefulSet') as stateful_sets,
			countIf(kind = 'DaemonSet') as daemon_sets,
			countIf(kind = 'Job') as jobs,
			countIf(kind = 'CronJob') as cron_jobs,
			-- Pods
			countIf(kind = 'Pod') as pods_total,
			countIf(kind = 'Pod' AND status = 'Running') as pods_running,
			countIf(kind = 'Pod' AND status = 'Pending') as pods_pending,
			countIf(kind = 'Pod' AND status = 'Failed') as pods_failed,
			-- Other resources
			uniqExact(namespace) as namespaces,
			countIf(kind = 'Service') as services,
			countIf(kind = 'PersistentVolumeClaim') as pvcs
		FROM latest_states
	`

	var (
		nodesTotal    uint64
		nodesReady    uint64
		nodesNotReady uint64
		deployments   uint64
		statefulSets  uint64
		daemonSets    uint64
		jobs          uint64
		cronJobs      uint64
		podsTotal     uint64
		podsRunning   uint64
		podsPending   uint64
		podsFailed    uint64
		namespaces    uint64
		services      uint64
		pvcs          uint64
	)

	err := a.clickhouse.QueryRow(ctx, query, req.OrganizationId, req.ClusterId).Scan(
		&nodesTotal,
		&nodesReady,
		&nodesNotReady,
		&deployments,
		&statefulSets,
		&daemonSets,
		&jobs,
		&cronJobs,
		&podsTotal,
		&podsRunning,
		&podsPending,
		&podsFailed,
		&namespaces,
		&services,
		&pvcs,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query overview metrics from ClickHouse")
		return nil, fmt.Errorf("failed to query overview metrics: %w", err)
	}

	// Query for resource capacity and usage
	resourceQuery := `
		WITH node_resources AS (
			SELECT
				argMax(cpu, timestamp) as cpu,
				argMax(memory, timestamp) as memory
			FROM kubernetes_allocatable_resources
			WHERE uid IN (
				SELECT uid
				FROM kubernetes_objects
				WHERE organization_id = ?
				  AND cluster_id = ?
				  AND kind = 'Node'
				  AND timestamp >= now() - INTERVAL 5 MINUTE
			)
			AND timestamp >= now() - INTERVAL 5 MINUTE
		)
		SELECT
			sum(toFloat64OrZero(replaceRegexpAll(cpu, '[^0-9.]', ''))) as total_cpu,
			sum(toFloat64OrZero(replaceRegexpAll(memory, 'Mi|Gi|Ki|Ti', '')) *
				multiIf(
					position(memory, 'Ti') > 0, 1024*1024,
					position(memory, 'Gi') > 0, 1024,
					position(memory, 'Mi') > 0, 1,
					position(memory, 'Ki') > 0, 0.001,
					1
				)
			) as total_memory_mi
		FROM node_resources
	`

	var (
		totalCPU      float64
		totalMemoryMi float64
	)

	err = a.clickhouse.QueryRow(ctx, resourceQuery, req.OrganizationId, req.ClusterId).Scan(&totalCPU, &totalMemoryMi)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query resource capacity, using zeros")
		totalCPU = 0
		totalMemoryMi = 0
	}

	// Format capacity strings
	cpuCapacity := fmt.Sprintf("%.1f cores", totalCPU)
	memoryCapacity := fmt.Sprintf("%.1f GiB", totalMemoryMi/1024)

	// Query for events summary
	eventsQuery := `
		SELECT
			count(*) as total_events,
			countIf(event_severity_text = 'Warning') as warnings,
			countIf(event_severity_text = 'Error') as errors
		FROM kubernetes_events
		WHERE organization_id = ?
		  AND cluster_id = ?
		  AND object_timestamp >= now() - INTERVAL 1 HOUR
	`

	var (
		eventsTotal    uint64
		eventsWarnings uint64
		eventsErrors   uint64
	)

	err = a.clickhouse.QueryRow(ctx, eventsQuery, req.OrganizationId, req.ClusterId).Scan(
		&eventsTotal,
		&eventsWarnings,
		&eventsErrors,
	)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query events summary, using zeros")
		eventsTotal = 0
		eventsWarnings = 0
		eventsErrors = 0
	}

	log.Debug().
		Uint64("nodes", nodesTotal).
		Uint64("pods", podsTotal).
		Uint64("deployments", deployments).
		Msg("GetOverviewMetrics response")

	return &pbsvc.OverviewMetrics{
		NodesTotal:           int32(nodesTotal),
		NodesReady:           int32(nodesReady),
		NodesNotReady:        int32(nodesNotReady),
		Deployments:          int32(deployments),
		StatefulSets:         int32(statefulSets),
		DaemonSets:           int32(daemonSets),
		Jobs:                 int32(jobs),
		CronJobs:             int32(cronJobs),
		PodsTotal:            int32(podsTotal),
		PodsRunning:          int32(podsRunning),
		PodsPending:          int32(podsPending),
		PodsFailed:           int32(podsFailed),
		CpuCapacity:          cpuCapacity,
		CpuUsage:             "0 cores", // TODO: Calculate actual usage from metrics
		MemoryCapacity:       memoryCapacity,
		MemoryUsage:          "0 GiB", // TODO: Calculate actual usage from metrics
		Namespaces:           int32(namespaces),
		Services:             int32(services),
		EventsTotal:          int32(eventsTotal),
		EventsWarnings:       int32(eventsWarnings),
		EventsErrors:         int32(eventsErrors),
		Pvcs:                 int32(pvcs),
		StorageTotalCapacity: "0 GiB", // TODO: Calculate from PVCs
		EstimatedCost:        "",      // TODO: Add cost calculation
		Currency:             "USD",
	}, nil
}
