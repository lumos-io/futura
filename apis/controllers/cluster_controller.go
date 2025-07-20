package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

func GetClusters(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	connectID, err := parseConnectID(c)
	if err != nil {
		return
	}

	var clusters []models.ClusterMetadata
	if err := models.GetDB().
		Preload("CloudProvider").
		Preload("Organization").
		Where("organization_id = ? AND cloud_provider_id = ?", orgID, connectID).
		Find(&clusters).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to fetch clusters metadata")
		return
	}
	utils.RespondOK(c, clusters)
}

func DeleteCluster(c *gin.Context) {
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

// helper to extract connect ID
func parseConnectID(c *gin.Context) (uint, error) {
	connectID, err := strconv.Atoi(c.Param("connect_id"))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Invalid connect id")
		return 0, err
	}
	return uint(connectID), nil
}
