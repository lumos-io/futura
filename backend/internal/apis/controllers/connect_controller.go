package controllers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/apis/providers"
	"github.com/opisvigilant/futura/backend/internal/apis/workflow"
	workflowclusters "github.com/opisvigilant/futura/backend/internal/apis/workflow/clusters"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/opisvigilant/futura/proto/gen/backend"
)

type ConnectController struct {
	config            *config.Configuration
	cloudProviderAuth *providers.CloudProviderAuth
	workflowManager   *workflow.WorkflowManager
}

func NewConnectController(config *config.Configuration) (*ConnectController, error) {
	p, err := providers.NewProviderAuth(config)
	if err != nil {
		return nil, err
	}

	wfm, err := workflow.New(config)
	if err != nil {
		return nil, err
	}

	return &ConnectController{
		config:            config,
		cloudProviderAuth: p,
		workflowManager:   wfm,
	}, nil
}

func (cc *ConnectController) GetConnects(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}
	var connects []models.ProviderConnection
	if err := models.GetDB().Where("organization_id = ?", orgID).Find(&connects).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to fetch connects")
		return
	}
	result := make([]*pb.ProviderConnection, len(connects))
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
		result[i] = &pb.ProviderConnection{
			Id:               int64(conn.ID),
			Provider:         p,
			Status:           s,
			SecretId:         conn.SecretID.String(),
			CreatedAt:        conn.CreatedAt.String(),
			ConnectionName:   conn.ConnectionName,
			ImportedClusters: uint64(conn.ImportedClusters),
		}
	}
	utils.RespondOK(c, result)
}

func (cc *ConnectController) CreateConnect(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	var input pb.CreateProviderConnectionRequest
	if err := protojson.Unmarshal(body, &input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	// create secret in provider
	secretID, err := cc.cloudProviderAuth.SetCredentials(orgID, input.Provider.String(), input.SecretName, input.Credentials)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create secret storage")
		return
	}

	// Create an entry in the DB for the UI
	prov, err := models.ConvertToCloudProviderFromProto(input.Provider)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to convert pb.CloudProvider proto to CloudProviderName")
		return
	}
	cp := models.ProviderConnection{
		Provider:       prov,
		SecretID:       secretID,
		SecretName:     input.SecretName,
		ConnectionName: input.ConnectionName,
		OrganizationID: orgID,
		Status:         models.InProgressStatus, // I assume the "Test Connection" was done before this operation is performed
	}
	if err := models.GetDB().Create(&cp).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_CONNECT_OPERATION", "Failed to create connection")
		return
	}

	// prepare values to be passed to the proto message
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

	// create response object
	pc := &pb.ProviderConnection{
		Id:               int64(cp.ID),
		Provider:         p,
		Status:           s,
		SecretId:         secretID.String(),
		CreatedAt:        cp.CreatedAt.String(),
		ConnectionName:   cp.ConnectionName,
		ImportedClusters: 0, // it gets updated after the import
	}

	// trigger workflow to fetch all the clusters in a separate go routine
	go func() {
		if err := cc.workflowManager.ExecuteFetchClustersWorkflow(&workflowclusters.WorkflowFetchClustersInput{
			Config:             cc.config,
			OrganizationID:     orgID,
			ProviderConnection: pc,
			Credentials:        input.Credentials,
			SecretID:           secretID.String(),
		}); err != nil {
			log.Logger.Error().Err(err).Msg("FetchClustersWorkflow failed with error")
		}
	}()

	utils.RespondCreated(c, pc)
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

	// trigger workflow to delete all the clusters in a separate go routine
	go func() {
		if err := cc.workflowManager.ExecuteDeleteClustersWorkflow(&workflowclusters.WorkflowDeleteClustersInput{
			Config:         cc.config,
			OrganizationID: orgID,
			ConnectionID:   uint(connectionID),
		}); err != nil {
			log.Logger.Error().Err(err).Msg("DeleteClustersWorkflow failed with error")
		}
	}()

	utils.RespondOK(c, gin.H{"result": "deletion in progress"})
}

func (cc *ConnectController) TestConnection(c *gin.Context) {
	var input struct {
		Provider    string            `json:"provider" binding:"required"`
		SecretName  string            `json:"secretName" binding:"required"`
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
