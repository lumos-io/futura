package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	pban "github.com/opisvigilant/futura/proto/gen/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetNodes_Integration tests the GetNodes method with a real ClickHouse connection
// This test requires a running ClickHouse instance
func TestGetNodes_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup ClickHouse connection
	cfg := &config.Configuration{
		Clickhouse: &config.Clickhouse{
			Servers:  []string{"localhost:9000"},
			Database: "futura",
			Username: "user",
			Password: "password",
		},
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Clickhouse.Servers,
		Auth: clickhouse.Auth{
			Database: cfg.Clickhouse.Database,
			Username: cfg.Clickhouse.Username,
			Password: cfg.Clickhouse.Password,
		},
	})
	require.NoError(t, err, "Failed to connect to ClickHouse")
	defer conn.Close()

	// Test connection
	err = conn.Ping(context.Background())
	require.NoError(t, err, "Failed to ping ClickHouse")

	server := &AnalyticsServer{
		clickhouse: conn,
		config:     cfg,
	}

	// Test cases
	tests := []struct {
		name    string
		request *pban.GetNodesByClusterIdRequest
		wantErr bool
	}{
		{
			name: "Valid request with org_id=1, cluster_id=1",
			request: &pban.GetNodesByClusterIdRequest{
				OrganizationId: 1,
				ClusterId:      1,
			},
			wantErr: false,
		},
		{
			name: "Valid request with org_id=1, cluster_id=2",
			request: &pban.GetNodesByClusterIdRequest{
				OrganizationId: 1,
				ClusterId:      2,
			},
			wantErr: false,
		},
		{
			name: "Non-existent cluster",
			request: &pban.GetNodesByClusterIdRequest{
				OrganizationId: 999,
				ClusterId:      999,
			},
			wantErr: false, // Should return empty result, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			resp, err := server.GetNodes(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				// Nodes can be nil or empty if no data exists

				// Log the results for debugging
				nodeCount := 0
				if resp.Nodes != nil {
					nodeCount = len(resp.Nodes)
				}
				t.Logf("Found %d nodes for org_id=%d, cluster_id=%d",
					nodeCount, tt.request.OrganizationId, tt.request.ClusterId)

				// Validate node structure if any nodes returned
				for _, node := range resp.Nodes {
					assert.NotEmpty(t, node.Name, "Node name should not be empty")
					assert.NotEmpty(t, node.Uid, "Node UID should not be empty")
					// Other fields may be empty depending on data
					t.Logf("  Node: name=%s, uid=%s, status=%s, instance_type=%s, zone=%s",
						node.Name, node.Uid, node.Status, node.InstanceType, node.Zone)
				}
			}
		})
	}
}

// TestGetClusterConfig_Integration tests the GetClusterConfig method
func TestGetClusterConfig_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Configuration{
		Clickhouse: &config.Clickhouse{
			Servers:  []string{"localhost:9000"},
			Database: "futura",
			Username: "user",
			Password: "password",
		},
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Clickhouse.Servers,
		Auth: clickhouse.Auth{
			Database: cfg.Clickhouse.Database,
			Username: cfg.Clickhouse.Username,
			Password: cfg.Clickhouse.Password,
		},
	})
	require.NoError(t, err)
	defer conn.Close()

	server := &AnalyticsServer{
		clickhouse: conn,
		config:     cfg,
	}

	req := &pban.GetClusterConfigRequest{
		OrganizationId: 1,
		ClusterId:      1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.GetClusterConfig(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify default mock data
	assert.Equal(t, "recommend", resp.ClusterScalingMode)
	assert.True(t, resp.EnableHpa)
	assert.True(t, resp.EnableVpa)
	assert.True(t, resp.EnableNodeHandler)

	t.Logf("Cluster config: mode=%s, hpa=%v, vpa=%v, node_handler=%v, budget=%s",
		resp.ClusterScalingMode, resp.EnableHpa, resp.EnableVpa,
		resp.EnableNodeHandler, resp.MonthlyBudget)
}

// TestClickHouseQuery_Syntax validates that the query syntax is correct
// by attempting to run it with EXPLAIN
// TestGetServices_Integration tests the GetServices method with a real ClickHouse connection
func TestGetServices_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup ClickHouse connection
	cfg := &config.Configuration{
		Clickhouse: &config.Clickhouse{
			Servers:  []string{"localhost:9000"},
			Database: "futura",
			Username: "user",
			Password: "password",
		},
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Clickhouse.Servers,
		Auth: clickhouse.Auth{
			Database: cfg.Clickhouse.Database,
			Username: cfg.Clickhouse.Username,
			Password: cfg.Clickhouse.Password,
		},
	})
	require.NoError(t, err, "Failed to connect to ClickHouse")
	defer conn.Close()

	err = conn.Ping(context.Background())
	require.NoError(t, err, "Failed to ping ClickHouse")

	server := &AnalyticsServer{
		clickhouse: conn,
		config:     cfg,
	}

	tests := []struct {
		name    string
		request *pban.GetServicesByClusterIdRequest
		wantErr bool
	}{
		{
			name: "Valid request with org_id=1, cluster_id=1",
			request: &pban.GetServicesByClusterIdRequest{
				OrganizationId: 1,
				ClusterId:      1,
			},
			wantErr: false,
		},
		{
			name: "Valid request with org_id=1, cluster_id=2",
			request: &pban.GetServicesByClusterIdRequest{
				OrganizationId: 1,
				ClusterId:      2,
			},
			wantErr: false,
		},
		{
			name: "Non-existent cluster",
			request: &pban.GetServicesByClusterIdRequest{
				OrganizationId: 999,
				ClusterId:      999,
			},
			wantErr: false, // Should return empty result, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			resp, err := server.GetServices(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				serviceCount := 0
				if resp.Services != nil {
					serviceCount = len(resp.Services)
				}
				t.Logf("Found %d services for org_id=%d, cluster_id=%d",
					serviceCount, tt.request.OrganizationId, tt.request.ClusterId)

				// Validate service structure if any services returned
				for _, svc := range resp.Services {
					assert.NotEmpty(t, svc.ServiceName, "Service name should not be empty")
					assert.NotEmpty(t, svc.Namespace, "Namespace should not be empty")
					t.Logf("  Service: name=%s, namespace=%s, pod=%s, http_requests=%d, error_rate=%.2f%%",
						svc.ServiceName, svc.Namespace, svc.PodName, svc.HttpRequestCount, svc.HttpErrorRate)
				}
			}
		})
	}
}

func TestClickHouseQuery_Syntax(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Configuration{
		Clickhouse: &config.Clickhouse{
			Servers:  []string{"localhost:9000"},
			Database: "futura",
			Username: "user",
			Password: "password",
		},
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Clickhouse.Servers,
		Auth: clickhouse.Auth{
			Database: cfg.Clickhouse.Database,
			Username: cfg.Clickhouse.Username,
			Password: cfg.Clickhouse.Password,
		},
	})
	require.NoError(t, err)
	defer conn.Close()

	// The actual query from GetNodes
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test that query can be executed without syntax errors
	rows, err := conn.Query(ctx, query,
		uint32(1), int64(1), // nodes
		uint32(1), int64(1), // pod_counts
	)

	assert.NoError(t, err, "Query should execute without syntax errors")
	if err == nil && rows != nil {
		rows.Close()
	}
}
