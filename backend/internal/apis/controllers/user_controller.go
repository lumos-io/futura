package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
)

type UserResponse struct {
	ID         int      `json:"id"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Email      string   `json:"email"`
	Role       int      `json:"role"`
	Teams      []string `json:"teams"`
	CreatedAt  string   `json:"created_at"`
	LastAccess string   `json:"last_access"`
	Status     int      `json:"status"`
	Avatar     string   `json:"avatar"`
	Provider   string   `json:"provider"`
}

// Convert string role to proto enum
func roleToProtoEnum(role models.UserRole) int {
	switch role {
	case models.UserRoleAdmin:
		return 1 // USER_ADMIN
	case models.UserRoleManager:
		return 2 // USER_MANAGER
	case models.UserRoleDeveloper:
		return 3 // USER_DEVELOPER
	case models.UserRoleViewer:
		return 4 // USER_VIEWER
	default:
		return 0 // UNDEFINED_USER_ROLE
	}
}

// Convert proto enum to string role
func protoEnumToRole(roleEnum int) models.UserRole {
	switch roleEnum {
	case 1:
		return models.UserRoleAdmin
	case 2:
		return models.UserRoleManager
	case 3:
		return models.UserRoleDeveloper
	case 4:
		return models.UserRoleViewer
	default:
		return models.UserRoleDeveloper
	}
}

// Convert string status to proto enum
func statusToProtoEnum(status models.UserStatus) int {
	switch status {
	case models.UserStatusActive:
		return 1 // USER_ACTIVE
	case models.UserStatusInvited:
		return 2 // USER_INVITED
	case models.UserStatusInactive:
		return 3 // USER_INACTIVE
	default:
		return 0 // UNDEFINED_USER_STATUS
	}
}

// Convert proto enum to string status
func protoEnumToStatus(statusEnum int) models.UserStatus {
	switch statusEnum {
	case 1:
		return models.UserStatusActive
	case 2:
		return models.UserStatusInvited
	case 3:
		return models.UserStatusInactive
	default:
		return models.UserStatusInvited
	}
}

func GetUsers(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	db := models.GetDB()
	var users []models.User

	// Fetch users with preloaded teams
	if err := db.Preload("Teams").Where("organization_id = ?", orgID).Find(&users).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to fetch users")
		return
	}

	// Convert to response format
	response := make([]UserResponse, len(users))
	for i, user := range users {
		teams := make([]string, len(user.Teams))
		for j, team := range user.Teams {
			teams[j] = team.Name
		}

		lastAccess := ""
		if user.LastAccess != nil {
			lastAccess = user.LastAccess.Format(time.RFC3339)
		}

		response[i] = UserResponse{
			ID:         int(user.ID),
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			Email:      user.Email,
			Role:       roleToProtoEnum(user.Role),
			Teams:      teams,
			CreatedAt:  user.CreatedAt.Format(time.RFC3339),
			LastAccess: lastAccess,
			Status:     statusToProtoEnum(user.Status),
			Avatar:     user.Avatar,
			Provider:   user.Provider,
		}
	}

	utils.RespondOK(c, response)
}

type InviteUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Role      int    `json:"role"`
	TeamIDs   []int  `json:"team_ids"`
}

func InviteUser(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input InviteUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	db := models.GetDB()

	// Check if user already exists
	var existingUser models.User
	if err := db.Where("email = ? AND organization_id = ?", input.Email, orgID).First(&existingUser).Error; err == nil {
		utils.RespondError(c, http.StatusConflict, "USER_EXISTS", "User with this email already exists in the organization")
		return
	}

	// Convert proto enum to role
	role := protoEnumToRole(input.Role)

	// Create invited user
	user := models.User{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Name:           input.FirstName + " " + input.LastName,
		Email:          input.Email,
		Role:           role,
		Status:         models.UserStatusInvited,
		OrganizationID: orgID,
	}

	if err := db.Create(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to invite user")
		return
	}

	// Add user to teams if specified
	if len(input.TeamIDs) > 0 {
		var teams []models.Team
		teamIDs := make([]uint, len(input.TeamIDs))
		for i, id := range input.TeamIDs {
			teamIDs[i] = uint(id)
		}

		if err := db.Where("id IN ? AND organization_id = ?", teamIDs, orgID).Find(&teams).Error; err == nil {
			db.Model(&user).Association("Teams").Append(&teams)
		}
	}

	// Reload user with teams
	db.Preload("Teams").First(&user, user.ID)

	// Convert to response
	teams := make([]string, len(user.Teams))
	for i, team := range user.Teams {
		teams[i] = team.Name
	}

	response := UserResponse{
		ID:         int(user.ID),
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Email:      user.Email,
		Role:       roleToProtoEnum(user.Role),
		Teams:      teams,
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		LastAccess: "",
		Status:     statusToProtoEnum(user.Status),
		Avatar:     user.Avatar,
		Provider:   user.Provider,
	}

	utils.RespondCreated(c, response)
}

type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email" binding:"omitempty,email"`
	Role      int    `json:"role"`
	TeamIDs   []int  `json:"team_ids"`
	Status    int    `json:"status"`
}

func UpdateUser(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid user_id")
		return
	}

	var input UpdateUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	db := models.GetDB()

	// Verify user exists and belongs to organization
	var user models.User
	if err := db.Where("id = ? AND organization_id = ?", userID, orgID).First(&user).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "User not found in this organization")
		return
	}

	// Update fields
	if input.FirstName != "" {
		user.FirstName = input.FirstName
	}
	if input.LastName != "" {
		user.LastName = input.LastName
	}
	if user.FirstName != "" && user.LastName != "" {
		user.Name = user.FirstName + " " + user.LastName
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Role != 0 {
		user.Role = protoEnumToRole(input.Role)
	}
	if input.Status != 0 {
		user.Status = protoEnumToStatus(input.Status)
	}

	if err := db.Save(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to update user")
		return
	}

	// Update teams if specified
	if input.TeamIDs != nil {
		// Clear existing teams
		if err := db.Model(&user).Association("Teams").Clear(); err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to update user teams")
			return
		}

		// Add new teams
		if len(input.TeamIDs) > 0 {
			var teams []models.Team
			teamIDs := make([]uint, len(input.TeamIDs))
			for i, id := range input.TeamIDs {
				teamIDs[i] = uint(id)
			}

			if err := db.Where("id IN ? AND organization_id = ?", teamIDs, orgID).Find(&teams).Error; err == nil {
				db.Model(&user).Association("Teams").Append(&teams)
			}
		}
	}

	// Reload user with teams
	db.Preload("Teams").First(&user, userID)

	// Convert to response
	teams := make([]string, len(user.Teams))
	for i, team := range user.Teams {
		teams[i] = team.Name
	}

	lastAccess := ""
	if user.LastAccess != nil {
		lastAccess = user.LastAccess.Format(time.RFC3339)
	}

	response := UserResponse{
		ID:         int(user.ID),
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Email:      user.Email,
		Role:       roleToProtoEnum(user.Role),
		Teams:      teams,
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		LastAccess: lastAccess,
		Status:     statusToProtoEnum(user.Status),
		Avatar:     user.Avatar,
		Provider:   user.Provider,
	}

	utils.RespondOK(c, response)
}

func DeleteUser(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid user_id")
		return
	}

	db := models.GetDB()

	// Verify user exists and belongs to organization
	var user models.User
	if err := db.Where("id = ? AND organization_id = ?", userID, orgID).First(&user).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "User not found in this organization")
		return
	}

	// Clear team associations
	if err := db.Model(&user).Association("Teams").Clear(); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to remove user from teams")
		return
	}

	// Delete the user
	if err := db.Delete(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to delete user")
		return
	}

	utils.RespondOK(c, gin.H{"message": "User removed from organization successfully"})
}

// helper to extract org ID
func parseOrgID(c *gin.Context) (uint, error) {
	orgID, err := strconv.Atoi(c.Param("org_id"))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "Invalid organization_id")
		return 0, err
	}
	return uint(orgID), nil
}
