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

	var clusters []models.Cluster
	if err := models.GetDB().Where("organization_id = ?", orgID).Find(&clusters).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to fetch clusters", nil)
		return
	}
	utils.RespondOK(c, clusters)
}

func CreateCluster(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Region      string `json:"region" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error(), nil)
		return
	}

	cluster := models.Cluster{
		Name:            input.Name,
		Description:     input.Description,
		Region:          input.Region,
		CloudProviderID: 1,
		OrganizationID:  *orgID,
	}
	if err := models.GetDB().Create(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to create cluster", nil)
		return
	}
	utils.RespondCreated(c, cluster)
}

func UpdateCluster(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id", nil)
		return
	}

	var cluster models.Cluster
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Cluster not found in this organization", nil)
		return
	}

	var input struct {
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error(), nil)
		return
	}

	cluster.Description = input.Description

	if err := models.GetDB().Save(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to update cluster", nil)
		return
	}
	utils.RespondOK(c, cluster)
}

func DeleteCluster(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid cluster_id", nil)
		return
	}

	var cluster models.Cluster
	if err := models.GetDB().Where("id = ? AND organization_id = ?", clusterID, orgID).First(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Cluster not found in this organization", nil)
		return
	}

	if err := models.GetDB().Delete(&cluster).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CLUSTER_OPERATION", "Failed to delete cluster", nil)
		return
	}
	utils.RespondOK(c, nil)
}
