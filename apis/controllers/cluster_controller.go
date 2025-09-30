package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/pkg/analytics"
	"github.com/opisvigilant/futura/apis/utils"

	pban "github.com/opisvigilant/futura/proto/gen/analytics"
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

	// TODO: Query actual metrics from ClickHouse/Analytics service
	// For now, return mock data that the frontend expects
	metrics := ClusterMetricsResponse{}

	// Nodes metrics
	metrics.Nodes.Total = 3
	metrics.Nodes.Ready = 3
	metrics.Nodes.NotReady = 0

	// Workloads metrics
	metrics.Workloads.Deployments = 12
	metrics.Workloads.StatefulSets = 3
	metrics.Workloads.DaemonSets = 5
	metrics.Workloads.Jobs = 8
	metrics.Workloads.CronJobs = 2

	// Pods metrics
	metrics.Pods.Total = 45
	metrics.Pods.Running = 42
	metrics.Pods.Pending = 2
	metrics.Pods.Failed = 1

	// Resources metrics
	metrics.Resources.CPUCapacity = "12 cores"
	metrics.Resources.CPUUsage = "8.4 cores (70%)"
	metrics.Resources.MemoryCapacity = "48 GB"
	metrics.Resources.MemoryUsage = "32 GB (67%)"

	// Other metrics
	metrics.Namespaces = 8
	metrics.Services = 15

	// Events metrics
	metrics.Events.Total = 124
	metrics.Events.Warnings = 3
	metrics.Events.Errors = 1

	// Storage metrics
	metrics.Storage.PVCs = 10
	metrics.Storage.TotalCapacity = "500 GB"

	// Cost metrics
	metrics.Cost.Estimated = "1,245.00"
	metrics.Cost.Currency = "USD"

	utils.RespondOK(c, metrics)
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
