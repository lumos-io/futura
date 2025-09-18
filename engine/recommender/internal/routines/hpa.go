package routines

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/go-lib/clickhouse"
)

const (
	// time window for metrics aggregation (change as needed)
	metricsWindow = 5 * time.Minute
)

type HPARecommender struct {
	client *clickhouse.Client
}

func NewHPARecommender(config *config.Configuration) (*HPARecommender, error) {
	client, err := clickhouse.New(config.Clickhouse.Servers, config.Clickhouse.Username,
		config.Clickhouse.Password, config.Clickhouse.Database, true)
	if err != nil {
		return nil, err
	}
	return &HPARecommender{
		client: client,
	}, nil
}

type HPAInput struct {
	OrganizationID uint32
	ClusterID      string
	Namespace      string
	DeploymentName string
	TargetCPUUtil  float64 // percent, e.g., 50.0
	MinReplicas    *int64  // optional
	MaxReplicas    *int64  // optional
}

type HPAResult struct {
	CurrentReplicas    int64
	CurrentCPUPercent  float64
	DesiredReplicas    int64
	MinReplicasBounded int64
	MaxReplicasBounded int64
	Reason             string
}

// ---------- HPA calculation ----------
//
// Assumptions & notes:
//   - We approximate "pods belonging to a deployment" by pod_name prefix matching "deploymentName-".
//     If your pod names don't match that pattern, replace with a label-based join using kubernetes_objects labels.
//   - cpu usage is taken from kubelet_pod_metrics.cpu_usage_nano_cores (sum over pods).
//   - cpu requests are taken from kubernetes_containers.cpu_requests for containers in the matching pods.
//     We multiply per-pod container requests by replica count when necessary.
//   - We use a time window (last 5 minutes) to average usage.
//
// Queries below are written to be reasonably efficient for ClickHouse but may need tuning.
func (r *HPARecommender) CalculateTargetReplicas(input *HPAInput) (*HPAResult, error) {
	// Compute time window
	now := time.Now().UTC()
	start := now.Add(-metricsWindow)

	// 1) Get current replica count from kubernetes_objects for Deployment
	var currentReplicas int64
	q1 := `
SELECT
    replicas
FROM kubernetes_objects
WHERE
    cluster_id = ? AND
    namespace = ? AND
    name = ? AND
    kind = 'Deployment'
ORDER BY timestamp DESC
LIMIT 1
`
	row := r.client.QueryRow(context.Background(), q1, input.ClusterID, input.Namespace, input.DeploymentName)
	if err := row.Scan(&currentReplicas); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deployment not found in kubernetes_objects")
		}
		return nil, err
	}

	// 2) Sum CPU usage across pods of the deployment in the window.
	// Using pod_name prefix assumption: pods typically named deploymentName-<hash>-<id>
	// If your naming differs, you must change this to label-based selection.
	var totalCPUUsageNano uint64
	var podCount int64
	q2 := `
SELECT
    sum(cpu_usage_nano_cores) AS total_cpu_nano,
    uniqExact(pod_name) AS pod_count
FROM kubelet_pod_metrics
WHERE
    timestamp BETWEEN ? AND ? AND
    pod_namespace = ? AND
    startsWith(pod_name, ?)
`
	row = r.client.QueryRow(context.Background(), q2, start, now, input.Namespace, input.DeploymentName+"-")
	if err := row.Scan(&totalCPUUsageNano, &podCount); err != nil {
		return nil, err
	}
	if podCount == 0 {
		return nil, fmt.Errorf("no pods found for deployment %s in namespace %s", input.DeploymentName, input.Namespace)
	}

	avgCPUUsagePerPodNano := float64(totalCPUUsageNano) / float64(podCount)

	// 3) Estimate avg CPU request per pod by reading kubernetes_containers.cpu_requests for sample of pods.
	// We'll take last known container requests for pods with the same prefix and average per pod.
	// This query returns sum of requests per pod; we'll average that across pods.
	// For parsing requests we retrieve the string and parse in Go.
	q3 := `
SELECT
    pod_uid,
    pod_name,
    container_name,
    cpu_requests
FROM kubernetes_containers
WHERE
    timestamp BETWEEN ? AND ? AND
    pod_uid IN (
        SELECT DISTINCT pod_uid FROM kubelet_pod_metrics
        WHERE timestamp BETWEEN ? AND ? AND pod_namespace = ? AND startsWith(pod_name, ?)
    )
ORDER BY timestamp DESC
`
	rows, err := r.client.Query(context.Background(), q3, start, now, start, now, input.Namespace, input.DeploymentName+"-")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map pod_uid -> total CPU request (nano) aggregated
	podCPUReqNano := map[string]uint64{}
	for rows.Next() {
		var podUID, podName, containerName, cpuReqStr sql.NullString
		if err := rows.Scan(&podUID, &podName, &containerName, &cpuReqStr); err != nil {
			return nil, err
		}
		if !cpuReqStr.Valid || strings.TrimSpace(cpuReqStr.String) == "" {
			continue
		}
		nano, err := parseK8sCPU(cpuReqStr.String)
		if err != nil {
			// skip invalid parse but log
			log.Printf("warning: couldn't parse cpu request %q for pod %s container %s: %v", cpuReqStr.String, podName.String, containerName.String, err)
			continue
		}
		podUIDStr := podUID.String
		podCPUReqNano[podUIDStr] += nano
	}
	// compute average CPU requests per pod
	var totalReqPerPodNanoSum float64
	for _, v := range podCPUReqNano {
		totalReqPerPodNanoSum += float64(v)
	}
	avgCPURequestPerPodNano := totalReqPerPodNanoSum / float64(len(podCPUReqNano))
	// fallback: if zero (no requests available), assume a small default
	if avgCPURequestPerPodNano == 0 {
		avgCPURequestPerPodNano = 100 * 1e6 // 100m = 0.1 core = 100_000_000 nano cores
	}

	// 4) Current utilization (%) = avgUsage / avgRequest * 100
	currentUtilPercent := (avgCPUUsagePerPodNano / avgCPURequestPerPodNano) * 100.0

	// 5) Desired replicas formula
	desiredReplicasF := float64(currentReplicas) * (currentUtilPercent / input.TargetCPUUtil)
	desiredReplicas := int64(math.Ceil(desiredReplicasF))
	if desiredReplicas < 1 {
		desiredReplicas = 1
	}
	// Apply bounds
	minBound := int64(1)
	maxBound := int64(1<<31 - 1)
	if input.MinReplicas != nil {
		minBound = *input.MinReplicas
	}
	if input.MaxReplicas != nil {
		maxBound = *input.MaxReplicas
	}
	bounded := desiredReplicas
	if bounded < minBound {
		bounded = minBound
	}
	if bounded > maxBound {
		bounded = maxBound
	}

	res := &HPAResult{
		CurrentReplicas:    currentReplicas,
		CurrentCPUPercent:  currentUtilPercent,
		DesiredReplicas:    bounded,
		MinReplicasBounded: minBound,
		MaxReplicasBounded: maxBound,
		Reason: fmt.Sprintf("avg_cpu_usage_per_pod_nano=%.0f, avg_cpu_request_per_pod_nano=%.0f -> current_util=%.2f%%, desired(before bound)=%d",
			avgCPUUsagePerPodNano, avgCPURequestPerPodNano, currentUtilPercent, desiredReplicas),
	}

	return res, nil
}
