package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

type TeamMemberResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type TeamResponse struct {
	ID          int                  `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	MemberCount int                  `json:"member_count"`
	CreatedAt   string               `json:"created_at"`
	Members     []TeamMemberResponse `json:"members"`
}

func GetTeams(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	db := models.GetDB()
	var teams []models.Team

	// Fetch teams with preloaded members
	if err := db.Preload("Members").Where("organization_id = ?", orgID).Find(&teams).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to fetch teams")
		return
	}

	// Convert to response format
	response := make([]TeamResponse, len(teams))
	for i, team := range teams {
		members := make([]TeamMemberResponse, len(team.Members))
		for j, member := range team.Members {
			members[j] = TeamMemberResponse{
				ID:    int(member.ID),
				Name:  member.Name,
				Email: member.Email,
				Role:  "developer", // TODO: Add role field to User model
			}
		}

		response[i] = TeamResponse{
			ID:          int(team.ID),
			Name:        team.Name,
			Description: team.Description,
			MemberCount: len(team.Members),
			CreatedAt:   team.CreatedAt.Format(time.RFC3339),
			Members:     members,
		}
	}

	utils.RespondOK(c, response)
}

type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberIDs   []int  `json:"memberIds"`
}

func CreateTeam(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input CreateTeamRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	db := models.GetDB()

	team := models.Team{
		Name:           input.Name,
		Description:    input.Description,
		OrganizationID: orgID,
	}

	if err := db.Create(&team).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to create team")
		return
	}

	response := TeamResponse{
		ID:          int(team.ID),
		Name:        team.Name,
		Description: team.Description,
		MemberCount: 0,
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		Members:     []TeamMemberResponse{},
	}

	utils.RespondCreated(c, response)
}

func UpdateTeam(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	teamID, err := strconv.Atoi(c.Param("team_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid team_id")
		return
	}

	var input UpdateTeamRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	db := models.GetDB()

	// Verify team exists and belongs to organization
	var team models.Team
	if err := db.Where("id = ? AND organization_id = ?", teamID, orgID).First(&team).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Team not found in this organization")
		return
	}

	// Update team fields
	if input.Name != "" {
		team.Name = input.Name
	}
	team.Description = input.Description

	if err := db.Save(&team).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to update team")
		return
	}

	// Update team members if memberIds provided
	if input.MemberIDs != nil {
		// Clear existing associations
		if err := db.Model(&team).Association("Members").Clear(); err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to update team members")
			return
		}

		// Add new members
		if len(input.MemberIDs) > 0 {
			var users []models.User
			userIDs := make([]uint, len(input.MemberIDs))
			for i, id := range input.MemberIDs {
				userIDs[i] = uint(id)
			}

			if err := db.Where("id IN ? AND organization_id = ?", userIDs, orgID).Find(&users).Error; err != nil {
				utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to fetch users")
				return
			}

			if err := db.Model(&team).Association("Members").Append(&users); err != nil {
				utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to add team members")
				return
			}
		}
	}

	// Reload team with members
	if err := db.Preload("Members").First(&team, teamID).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to reload team")
		return
	}

	// Convert to response
	members := make([]TeamMemberResponse, len(team.Members))
	for i, member := range team.Members {
		members[i] = TeamMemberResponse{
			ID:    int(member.ID),
			Name:  member.Name,
			Email: member.Email,
			Role:  "developer", // TODO: Add role field to User model
		}
	}

	response := TeamResponse{
		ID:          int(team.ID),
		Name:        team.Name,
		Description: team.Description,
		MemberCount: len(team.Members),
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		Members:     members,
	}

	utils.RespondOK(c, response)
}

func DeleteTeam(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	teamID, err := strconv.Atoi(c.Param("team_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid team_id")
		return
	}

	db := models.GetDB()

	// Verify team exists and belongs to organization
	var team models.Team
	if err := db.Where("id = ? AND organization_id = ?", teamID, orgID).First(&team).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Team not found in this organization")
		return
	}

	// Clear associations first (many2many relationship)
	if err := db.Model(&team).Association("Members").Clear(); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to remove team members")
		return
	}

	// Delete the team
	if err := db.Delete(&team).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TEAM_OPERATION", "Failed to delete team")
		return
	}

	utils.RespondOK(c, gin.H{"message": "Team deleted successfully"})
}
