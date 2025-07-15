package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

func GetUsers(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var users []models.User
	if err := models.GetDB().Where("organization_id = ?", orgID).Find(&users).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to fetch users")
		return
	}
	utils.RespondOK(c, users)
}

func CreateUser(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	var input struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	user := models.User{
		Name:           input.Name,
		Email:          input.Email,
		OrganizationID: orgID,
	}

	if err := models.GetDB().Create(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to create user")
		return
	}
	utils.RespondCreated(c, user)
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

	var user models.User
	if err := models.GetDB().Where("id = ? AND organization_id = ?", userID, orgID).First(&user).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "User not found in this organization")
		return
	}

	var input struct {
		Name  string `json:"name"`
		Email string `json:"email" binding:"omitempty,email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", err.Error())
		return
	}

	if input.Name != "" {
		user.Name = input.Name
	}
	if input.Email != "" {
		user.Email = input.Email
	}

	if err := models.GetDB().Save(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to update user")
		return
	}
	utils.RespondOK(c, user)
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

	var user models.User
	if err := models.GetDB().Where("id = ? AND organization_id = ?", userID, orgID).First(&user).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "BAD_INPUT", "User not found in this organization")
		return
	}

	if err := models.GetDB().Delete(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to delete user")
		return
	}
	utils.RespondOK(c, nil)
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
