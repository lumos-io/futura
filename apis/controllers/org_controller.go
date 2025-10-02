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
	if err := models.GetDB().Preload("Members").Find(&orgs).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to fetch organizations")
		return
	}
	utils.RespondOK(c, orgs)
}

func CreateOrganization(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	org := models.Organization{
		Name: input.Name,
	}
	if err := models.GetDB().Create(&org).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to create organization")
		return
	}
	utils.RespondCreated(c, org)
}

func GetOrganization(c *gin.Context) {
	id, err := parseID(c.Param("org_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID")
		return
	}

	var org models.Organization
	if err := models.GetDB().Preload("Members").First(&org, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Organization not found")
		return
	}
	utils.RespondOK(c, org)
}

func UpdateOrganization(c *gin.Context) {
	id, err := parseID(c.Param("org_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID")
		return
	}

	var org models.Organization
	if err := models.GetDB().First(&org, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Organization not found")
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	org.Name = input.Name
	if err := models.GetDB().Save(&org).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to update organization")
		return
	}
	utils.RespondOK(c, org)
}

func DeleteOrganization(c *gin.Context) {
	id, err := parseID(c.Param("org_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid organization ID")
		return
	}

	if err := models.GetDB().Delete(&models.Organization{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_OPERATION", "Failed to delete organization")
		return
	}
	utils.RespondOK(c, nil)
}

// helper
func parseID(param string) (uint, error) {
	i, err := strconv.Atoi(param)
	return uint(i), err
}
