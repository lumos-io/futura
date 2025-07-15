package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	awsprovider "github.com/opisvigilant/futura/apis/internal/providers/aws"
	azureprovider "github.com/opisvigilant/futura/apis/internal/providers/azure"
	digitaloceanprovider "github.com/opisvigilant/futura/apis/internal/providers/digitalocean"
	gcpprovider "github.com/opisvigilant/futura/apis/internal/providers/gcp"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
	"github.com/rs/zerolog/log"
)

const (
	AccessCredentialsSecretID string = "AccessCredentials"
)

type ConnectController struct {
	awsProvider          *awsprovider.AWSProvider
	gcpProvider          *gcpprovider.GCPProvider
	azureProvider        *azureprovider.AzureProvider
	digitaloceanProvider *digitaloceanprovider.DigitalOceanProvider
}

func NewConnectController(config *config.Configuration) (*ConnectController, error) {
	ap, err := awsprovider.New(config)
	if err != nil {
		return nil, err
	}
	gp, err := gcpprovider.New(config)
	if err != nil {
		return nil, err
	}
	azp, err := azureprovider.New(config)
	if err != nil {
		return nil, err
	}
	dop, err := digitaloceanprovider.New(config)
	if err != nil {
		return nil, err
	}
	return &ConnectController{
		awsProvider:          ap,
		gcpProvider:          gp,
		azureProvider:        azp,
		digitaloceanProvider: dop,
	}, nil
}

func (cc *ConnectController) GetConnects(c *gin.Context) {
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

func (cc *ConnectController) CreateConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input struct {
		Provider        string `json:"provider" binding:"required"`
		AccountID       string `json:"accountId" binding:"required"`
		AccessKey       string `json:"accessKey" binding:"required"`
		SecretAccessKey string `json:"secretAccessKey" binding:"required"`
		Region          string `json:"region" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	// Create a secret on the correct provider based on the input
	switch models.CloudProviderName(input.Provider) {
	case models.AWS:
		if err := cc.awsProvider.SetCredentials(fmt.Sprintf("%d", orgID), AccessCredentialsSecretID, &awsprovider.AWSCredentials{
			AccessKey:       input.AccessKey,
			SecretAccessKey: input.SecretAccessKey,
			Region:          input.Region,
		}); err != nil {
			log.Logger.Error().Err(err).Msg("Failed to create store secret for the aws provider")
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create store secret")
			return
		}
	}

	// Create an entry in the DB for the UI
	cp := models.CloudProvider{
		Name:             models.CloudProviderName(input.Provider),
		Account:          input.Provider,
		SecretID:         AccessCredentialsSecretID,
		OrganizationID:   *orgID,
		ActivationStatus: models.PendingStatus,
	}
	if err := models.GetDB().Create(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create connection")
		return
	}
	utils.RespondCreated(c, cp)
}

func (cc *ConnectController) UpdateConnect(c *gin.Context) {
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

	// update here
	// ...

	if err := models.GetDB().Save(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to update connection")
		return
	}
	utils.RespondOK(c, cp)
}

func (cc *ConnectController) DeleteConnect(c *gin.Context) {
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

func (cc *ConnectController) TestConnection(c *gin.Context) {
	utils.RespondOK(c, gin.H{"result": "ok"})
}
