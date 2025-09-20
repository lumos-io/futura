package routines

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/go-lib/clickhouse"
)

const (
	// percentile to use for VPA recommendations
	vpaPercentile = 0.95
	// safety margin multiplicative factor for VPA recommendations (e.g. 1.1 => +10%)
	vpaSafetyMargin = 1.10
)

type VPARecommender struct {
	client *clickhouse.Client
}

func NewVPARecommender(config *config.Configuration) (*VPARecommender, error) {
	client, err := clickhouse.New(config.Clickhouse.Servers, config.Clickhouse.Username,
		config.Clickhouse.Password, config.Clickhouse.Database, true)
	if err != nil {
		return nil, err
	}
	return &VPARecommender{
		client: client,
	}, nil
}

type VPAInput struct {
	OrganizationID uint32
	ClusterID      string
	Namespace      string
	DeploymentName string
	// optional policy min/max per container if needed
	MinCPUNano    uint64
	MinMemoryByte uint64
}

type ContainerVPA struct {
	ContainerName       string
	CPU95thNano         uint64
	Memory95thBytes     uint64
	RecommendedCPUNano  uint64
	RecommendedMemoryB  uint64
	CurrentCPURequestN  uint64
	CurrentMemoryReqB   uint64
	RecommendationNotes string
}

type VPAResult struct {
	Containers []ContainerVPA
	Reason     string
}

// ---------- VPA calculation ----------
//
// Assumptions & notes:
// - We assume container identity via pod_name prefix (deploymentName-) and kubernetes_containers.container_name
// - We compute quantile(0.95) for cpu & memory per container name across pods.
// - The recommended request = max(quantile95, MinCPU) * safety_margin (rounded).
func (r *VPARecommender) CalculateContainersPatch(input *VPAInput) (*VPAResult, error) {
	now := time.Now().UTC()
	start := now.Add(-metricsWindow)

	// 1) Build list of pod_uids for the deployment (again, using pod_name prefix assumption)
	qPods := `
SELECT DISTINCT pod_uid, pod_name
FROM kubelet_pod_metrics
WHERE timestamp BETWEEN ? AND ? AND pod_namespace = ? AND startsWith(pod_name, ?)
`
	rows, err := r.client.Query(context.Background(), qPods, start, now, input.Namespace, input.DeploymentName+"-")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var podUIDs []string
	for rows.Next() {
		var podUID, podName sql.NullString
		if err := rows.Scan(&podUID, &podName); err != nil {
			return nil, err
		}
		if podUID.Valid {
			podUIDs = append(podUIDs, podUID.String)
		}
	}
	if len(podUIDs) == 0 {
		return nil, fmt.Errorf("no pods found for deployment %s", input.DeploymentName)
	}

	// 2) For each container_name across these pods compute quantile 95 for cpu and memory
	// We'll use ClickHouse aggregation with WHERE pod_uid IN (...)
	// To protect SQL length, use temporary table technique would be better, but for simplicity we pass as array
	// ClickHouse supports IN with array of strings.
	// Build IN clause parameterization: (?, ?, ?)
	inParams := make([]interface{}, 0, len(podUIDs)+2)
	inParams = append(inParams, start, now)
	for _, pid := range podUIDs {
		inParams = append(inParams, pid)
	}

	// Query: group by container_name
	// Note: using quantileExact(0.95)(cpu_usage_nano_cores)
	placeholder := strings.Repeat("?,", len(podUIDs))
	placeholder = strings.TrimRight(placeholder, ",")
	qAgg := fmt.Sprintf(`
SELECT
    container_name,
    quantileExact(%f)(cpu_usage_nano_cores) AS cpu_95th,
    quantileExact(%f)(memory_usage_bytes) AS mem_95th
FROM kubelet_container_metrics
WHERE timestamp BETWEEN ? AND ? AND pod_uid IN (%s)
GROUP BY container_name
`, vpaPercentile, vpaPercentile, placeholder)

	// Build params: start, now, podUIDs...
	params := []interface{}{start, now}
	for _, p := range podUIDs {
		params = append(params, p)
	}

	rows2, err := r.client.Query(context.Background(), qAgg, params...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	containers := []ContainerVPA{}
	for rows2.Next() {
		var containerName sql.NullString
		var cpu95 sql.NullFloat64
		var mem95 sql.NullFloat64
		if err := rows2.Scan(&containerName, &cpu95, &mem95); err != nil {
			return nil, err
		}
		name := "<unknown>"
		if containerName.Valid {
			name = containerName.String
		}
		var cpu95n uint64
		var mem95b uint64
		if cpu95.Valid {
			cpu95n = uint64(math.Round(cpu95.Float64))
		}
		if mem95.Valid {
			mem95b = uint64(math.Round(mem95.Float64))
		}
		// 3) Get last seen current CPU/memory requests for this container (if available)
		qReq := `
SELECT cpu_requests, memory_requests
FROM kubernetes_containers
WHERE container_name = ? AND timestamp BETWEEN ? AND ?
ORDER BY timestamp DESC
LIMIT 1
`
		var cpuReqStr, memReqStr sql.NullString
		row := r.client.QueryRow(context.Background(), qReq, name, start, now)
		_ = row.Scan(&cpuReqStr, &memReqStr) // ignore error; can be nil

		var currCPUReqN uint64
		var currMemReqB uint64
		if cpuReqStr.Valid && cpuReqStr.String != "" {
			if v, err := parseK8sCPU(cpuReqStr.String); err == nil {
				currCPUReqN = v
			}
		}
		if memReqStr.Valid && memReqStr.String != "" {
			if v, err := parseK8sMemory(memReqStr.String); err == nil {
				currMemReqB = v
			}
		}

		// 4) Recommended = max(quantile, minAllowed) * safety_margin
		recoCPU := cpu95n
		if recoCPU < input.MinCPUNano {
			recoCPU = input.MinCPUNano
		}
		// apply safety margin
		recoCPUFloat := float64(recoCPU) * vpaSafetyMargin
		recoCPU = uint64(math.Ceil(recoCPUFloat))

		recoMem := mem95b
		if recoMem < input.MinMemoryByte {
			recoMem = input.MinMemoryByte
		}
		recoMemFloat := float64(recoMem) * vpaSafetyMargin
		recoMem = uint64(math.Ceil(recoMemFloat))

		note := fmt.Sprintf("cpu95=%d, mem95=%d, min_cpu=%d, min_mem=%d, safety_margin=%.2f",
			cpu95n, mem95b, input.MinCPUNano, input.MinMemoryByte, vpaSafetyMargin)

		c := ContainerVPA{
			ContainerName:       name,
			CPU95thNano:         cpu95n,
			Memory95thBytes:     mem95b,
			RecommendedCPUNano:  recoCPU,
			RecommendedMemoryB:  recoMem,
			CurrentCPURequestN:  currCPUReqN,
			CurrentMemoryReqB:   currMemReqB,
			RecommendationNotes: note,
		}
		containers = append(containers, c)
	}

	result := &VPAResult{
		Containers: containers,
		Reason:     fmt.Sprintf("computed VPA recommendations using %d pods window=%s", len(podUIDs), metricsWindow),
	}
	return result, nil
}
