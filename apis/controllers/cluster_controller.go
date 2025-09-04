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
		Where("organization_id = ? AND cloud_provider_id = ?", orgID, connectID).
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

// helper to extract connect ID
func parseConnectID(c *gin.Context) (uint, error) {
	connectID, err := strconv.Atoi(c.Param("connect_id"))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Invalid connect id")
		return 0, err
	}
	return uint(connectID), nil
}
