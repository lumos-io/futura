package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

func GetConnects(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var connects []models.User
	if err := models.GetDB().Where("organization_id = ?", orgID).Find(&connects).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to fetch users")
		return
	}
	utils.RespondOK(c, connects)
}

func CreateConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input struct {
		Name     string `json:"name" binding:"required"`
		Account  string `json:"account" binding:"required"`
		RoleName string `json:"roleName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	cp := models.CloudProvider{
		Name:             models.CloudProviderName(input.Name),
		Account:          input.Account,
		RoleName:         input.RoleName,
		OrganizationID:   *orgID,
		ActivationStatus: models.PendingStatus,
	}

	if err := models.GetDB().Create(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create connection")
		return
	}
	utils.RespondCreated(c, cp)
}

func UpdateConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	connectID, err := strconv.Atoi(c.Param("connect_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid connect_id")
		return
	}

	var cp models.CloudProvider
	if err := models.GetDB().Where("id = ? AND organization_id = ?", connectID, orgID).First(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Connection not found in this organization")
		return
	}

	var input struct {
		Account  string `json:"account" binding:"required"`
		RoleName string `json:"roleName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	if input.Account != "" {
		cp.Account = input.Account
	}
	if input.RoleName != "" {
		cp.RoleName = input.RoleName
	}

	if err := models.GetDB().Save(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to update connection")
		return
	}
	utils.RespondOK(c, cp)
}

func DeleteConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	connectionID, err := strconv.Atoi(c.Param("connect_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid connect_id")
		return
	}

	var cp models.CloudProvider
	if err := models.GetDB().Where("id = ? AND organization_id = ?", connectionID, orgID).First(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Connection not found in this organization")
		return
	}

	if err := models.GetDB().Delete(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to delete connection")
		return
	}
	utils.RespondOK(c, nil)
}

func TestConnection(c *gin.Context) {

	utils.RespondOK(c, gin.H{"result": "ok"})
}
