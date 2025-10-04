package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
)

type ClusterController struct {
	AnalyticsClient *analytics.Client
}

func NewClusterController(config *config.Configuration) (*ClusterController, error) {
	ac, err := analytics.New(config)
	if err != nil {
		return nil, err
	}
	return &ClusterController{
		AnalyticsClient: ac,
	}, nil
}

func (cc *ClusterController) GetClusters(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	connectID, err := parseConnectID(c)
	if err != nil {
		return
	}

	db := models.GetDB()

	var connect models.ProviderConnection
	if err := db.
		Where("organization_id = ? AND id = ?", orgID, connectID).
		Find(&connect).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to fetch connect")
		return
	}

	// Preload only the cluster for that particular provider and do not
	// do a massive JOIN operation
	switch connect.Provider {
	case models.AWS:
		db = db.Preload("EKSMetadata")
	case models.GoogleCloud:
		db = db.Preload("GKEMetadata")
	case models.Azure:
		db = db.Preload("AKSMetadata")
	case models.DigitalOcean:
		db = db.Preload("DOKSMetadata")
	case models.Alibaba:
		db = db.Preload("ACKMetadata")
	case models.Kind:
		db = db.Preload("KindMetadata")
	default:
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Invalid provider name")
		return
	}

	var clusters []models.ClusterMetadata
	if err := db.
		Where("organization_id = ? AND provider_connection_id = ?", orgID, connectID).
		Find(&clusters).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to fetch clusters metadata")
		return
	}
	utils.RespondOK(c, models.ConvertToProtoClusterMetadataList(clusters))
}

func (cc *ClusterController) DeleteCluster(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id")
		return
	}

	var cluster models.ClusterMetadata
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Cluster not found in this organization")
		return
	}

	if err := models.GetDB().Delete(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to delete cluster metadata")
		return
	}
	utils.RespondOK(c, nil)
}

func (cc *ClusterController) GetEvents(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id")
		return
	}
	events, err := cc.AnalyticsClient.GetEvents(&pban.GetEventsByClusterIdRequest{
		OrganizationId: uint32(orgID),
		ClusterId:      int64(clusterID),
		Limit:          100,
		Cursor:         0,
		Reverse:        false,
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ANALYTICS_OPERATION", "Failed to get cluster events")
		return
	}
	utils.RespondOK(c, events)
}

type ClusterMetricsResponse struct {
	Nodes struct {
		Total    int `json:"total"`
		Ready    int `json:"ready"`
		NotReady int `json:"notReady"`
	} `json:"nodes"`
	Workloads struct {
		Deployments  int `json:"deployments"`
		StatefulSets int `json:"statefulSets"`
		DaemonSets   int `json:"daemonSets"`
		Jobs         int `json:"jobs"`
		CronJobs     int `json:"cronJobs"`
	} `json:"workloads"`
	Pods struct {
		Total   int `json:"total"`
		Running int `json:"running"`
		Pending int `json:"pending"`
		Failed  int `json:"failed"`
	} `json:"pods"`
	Resources struct {
		CPUCapacity    string `json:"cpuCapacity"`
		CPUUsage       string `json:"cpuUsage"`
		MemoryCapacity string `json:"memoryCapacity"`
		MemoryUsage    string `json:"memoryUsage"`
	} `json:"resources"`
	Namespaces int `json:"namespaces"`
	Services   int `json:"services"`
	Events     struct {
		Total    int `json:"total"`
		Warnings int `json:"warnings"`
		Errors   int `json:"errors"`
	} `json:"events"`
	Storage struct {
		PVCs          int    `json:"pvcs"`
		TotalCapacity string `json:"totalCapacity"`
	} `json:"storage"`
	Cost struct {
		Estimated string `json:"estimated"`
		Currency  string `json:"currency"`
	} `json:"cost"`
}

func (cc *ClusterController) GetMetrics(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id")
		return
	}

	// Verify cluster belongs to organization
	var cluster models.ClusterMetadata
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Cluster not found in this organization")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Ticker to query analytics every 5 seconds for real-time updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	sendOverviewData := func() {
		// Query analytics service for overview metrics
		overviewResp, err := cc.AnalyticsClient.GetOverviewMetrics(&pban.GetOverviewMetricsRequest{
			OrganizationId: uint32(orgID),
			ClusterId:      int64(clusterID),
		})
		if err != nil {
			// Log error but continue streaming
			return
		}

		// Transform to match frontend's ClusterMetricsResponse structure
		metrics := ClusterMetricsResponse{}
		metrics.Nodes.Total = int(overviewResp.NodesTotal)
		metrics.Nodes.Ready = int(overviewResp.NodesReady)
		metrics.Nodes.NotReady = int(overviewResp.NodesNotReady)
		metrics.Workloads.Deployments = int(overviewResp.Deployments)
		metrics.Workloads.StatefulSets = int(overviewResp.StatefulSets)
		metrics.Workloads.DaemonSets = int(overviewResp.DaemonSets)
		metrics.Workloads.Jobs = int(overviewResp.Jobs)
		metrics.Workloads.CronJobs = int(overviewResp.CronJobs)
		metrics.Pods.Total = int(overviewResp.PodsTotal)
		metrics.Pods.Running = int(overviewResp.PodsRunning)
		metrics.Pods.Pending = int(overviewResp.PodsPending)
		metrics.Pods.Failed = int(overviewResp.PodsFailed)
		metrics.Resources.CPUCapacity = overviewResp.CpuCapacity
		metrics.Resources.CPUUsage = overviewResp.CpuUsage
		metrics.Resources.MemoryCapacity = overviewResp.MemoryCapacity
		metrics.Resources.MemoryUsage = overviewResp.MemoryUsage
		metrics.Namespaces = int(overviewResp.Namespaces)
		metrics.Services = int(overviewResp.Services)
		metrics.Events.Total = int(overviewResp.EventsTotal)
		metrics.Events.Warnings = int(overviewResp.EventsWarnings)
		metrics.Events.Errors = int(overviewResp.EventsErrors)
		metrics.Storage.PVCs = int(overviewResp.Pvcs)
		metrics.Storage.TotalCapacity = overviewResp.StorageTotalCapacity
		metrics.Cost.Estimated = overviewResp.EstimatedCost
		metrics.Cost.Currency = overviewResp.Currency

		b, err := json.Marshal(metrics)
		if err != nil {
			return
		}

		c.SSEvent("overview-metrics", string(b))
		flusher.Flush()
	}

	// Send initial data immediately
	sendOverviewData()

	// Loop and stream data every 5 seconds
	for {
		select {
		case <-ticker.C:
			sendOverviewData()
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

func (cc *ClusterController) GetSLOMetricsSSE(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id")
		return
	}

	// Verify cluster belongs to organization
	var cluster models.ClusterMetadata
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Cluster not found in this organization")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Ticker to query analytics every 5 seconds for real-time updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	sendServicesData := func() {
		// Query analytics service for services/SLO metrics
		servicesResp, err := cc.AnalyticsClient.GetServices(&pban.GetServicesByClusterIdRequest{
			OrganizationId: uint32(orgID),
			ClusterId:      int64(clusterID),
		})
		if err != nil {
			// Log error but continue streaming
			return
		}

		response := map[string]interface{}{
			"data": servicesResp.Services,
		}

		b, err := json.Marshal(response)
		if err != nil {
			return
		}

		c.SSEvent("slo-metrics", string(b))
		flusher.Flush()
	}

	// Send initial data immediately
	sendServicesData()

	// Loop and stream data every 5 seconds
	for {
		select {
		case <-ticker.C:
			sendServicesData()
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

func (cc *ClusterController) GetNodesSSE(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id")
		return
	}

	// Verify cluster belongs to organization
	var cluster models.ClusterMetadata
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Cluster not found in this organization")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Ticker to query analytics every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	sendNodesData := func() {
		// Query analytics service for nodes
		nodesResp, err := cc.AnalyticsClient.GetNodes(&pban.GetNodesByClusterIdRequest{
			OrganizationId: uint32(orgID),
			ClusterId:      int64(clusterID),
		})
		if err != nil {
			// Log error but continue streaming
			return
		}

		// Query analytics service for cluster config
		configResp, err := cc.AnalyticsClient.GetClusterConfig(&pban.GetClusterConfigRequest{
			OrganizationId: uint32(orgID),
			ClusterId:      int64(clusterID),
		})
		if err != nil {
			// Log error but continue streaming
			return
		}

		response := map[string]interface{}{
			"nodes":  nodesResp.Nodes,
			"config": configResp,
		}

		b, err := json.Marshal(response)
		if err != nil {
			return
		}

		c.SSEvent("nodes-data", string(b))
		flusher.Flush()
	}

	// Send initial data immediately
	sendNodesData()

	// Loop and stream data every minute
	for {
		select {
		case <-ticker.C:
			sendNodesData()
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

// helper to extract connect ID
func parseConnectID(c *gin.Context) (uint, error) {
	connectID, err := strconv.Atoi(c.Param("connect_id"))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Invalid connect id")
		return 0, err
	}
	return uint(connectID), nil
}
