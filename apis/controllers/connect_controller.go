package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/providers"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"

	pb "github.com/opisvigilant/futura/proto/gen/backend"
)

type ConnectController struct {
	cloudProviderAuth *providers.CloudProviderAuth
}

func NewConnectController(config *config.Configuration) (*ConnectController, error) {
	p, err := providers.New(config)
	if err != nil {
		return nil, err
	}
	return &ConnectController{
		cloudProviderAuth: p,
	}, nil
}

func (cc *ConnectController) GetConnects(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}
	var connects []models.CloudProvider
	if err := models.GetDB().Where("organization_id = ?", orgID).Find(&connects).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to fetch connects")
		return
	}

	result := make([]pb.ProviderConnection, len(connects))
	for i, conn := range connects {
		s, err := models.ConvertToProtoFromActivationStatus(conn.Status)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert ActivationStatus to Proto")
			return
		}
		p, err := models.ConvertToProtoFromCloudProvider(conn.Provider)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert CloudProvider to Proto")
			return
		}
		result[i] = pb.ProviderConnection{
			Id:        int64(conn.ID),
			Provider:  p,
			Status:    s,
			CreatedAt: conn.CreatedAt.String(),
		}
	}

	utils.RespondOK(c, result)
}

func (cc *ConnectController) CreateConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	input := pb.CreateProviderConnectionRequest{}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	// create secret in provider
	if err := cc.cloudProviderAuth.SetCredentials(orgID, input.Provider.String(), input.SecretId, input.Credentials); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create secret storage")
		return
	}

	// Create an entry in the DB for the UI
	prov, err := models.ConvertToCloudProviderFromProto(input.Provider)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert pb.CloudProvider proto to CloudProviderName")
		return
	}
	cp := models.CloudProvider{
		Provider:       prov,
		SecretID:       input.SecretId,
		OrganizationID: orgID,
		Status:         models.ActiveStatus, // I assume the "Test Connection" was done before this operation is performed
	}
	if err := models.GetDB().Create(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create connection")
		return
	}

	s, err := models.ConvertToProtoFromActivationStatus(cp.Status)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert ActiveStatus to Proto")
		return
	}
	p, err := models.ConvertToProtoFromCloudProvider(cp.Provider)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert CloudProvider to Proto")
		return
	}
	utils.RespondCreated(c, pb.ProviderConnection{
		Id:        int64(cp.ID),
		Provider:  p,
		Status:    s,
		CreatedAt: cp.CreatedAt.String(),
	})
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
		Provider    string            `json:"provider" binding:"required"`
		SecretID    pb.SecretIdName   `json:"secretId" binding:"required"`
		Credentials map[string]string `json:"credentials" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	// update secret in the provider
	if err := cc.cloudProviderAuth.UpdateCredentials(orgID, input.Provider, input.SecretID, input.Credentials); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to update secret storage")
		return
	}

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
	var input struct {
		Provider    string            `json:"provider" binding:"required"`
		SecretID    string            `json:"secretId" binding:"required"`
		Credentials map[string]string `json:"credentials" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	// FIXME: ignore the error for now until I'm on an actual cloud provider
	// and we have all the sessions/clients/etc that I can test
	_ = cc.cloudProviderAuth.TestConnection(input.Provider, input.Credentials)

	utils.RespondOK(c, gin.H{"result": "ok"})
}
