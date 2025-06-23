package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

func GetOrganizations(c *gin.Context) {
	var orgs []models.Organization
	if err := models.GetDB().Preload("Users").Find(&orgs).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to fetch organizations", nil)
		return
	}
	utils.RespondOK(c, orgs)
}

func CreateOrganization(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error(), nil)
		return
	}

	org := models.Organization{
		Name: input.Name,
	}
	if err := models.GetDB().Create(&org).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to create organization", nil)
		return
	}
	utils.RespondCreated(c, org)
}

func GetOrganization(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID", nil)
		return
	}

	var org models.Organization
	if err := models.GetDB().Preload("Users").First(&org, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Organization not found", nil)
		return
	}
	utils.RespondOK(c, org)
}

func UpdateOrganization(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID", nil)
		return
	}

	var org models.Organization
	if err := models.GetDB().First(&org, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Organization not found", nil)
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error(), nil)
		return
	}

	org.Name = input.Name
	if err := models.GetDB().Save(&org).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to update organization", nil)
		return
	}
	utils.RespondOK(c, org)
}

func DeleteOrganization(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID", nil)
		return
	}

	if err := models.GetDB().Delete(&models.Organization{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to delete organization", nil)
		return
	}
	utils.RespondOK(c, nil)
}

// helper
func parseID(param string) (uint, error) {
	i, err := strconv.Atoi(param)
	return uint(i), err
}
