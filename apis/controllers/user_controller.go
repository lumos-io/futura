package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
)

type UserResponse struct {
	ID         int       `json:"id"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Teams      []string  `json:"teams"`
	CreatedAt  time.Time `json:"createdAt"`
	LastAccess time.Time `json:"lastAccess"`
	Status     string    `json:"status"`
}

func GetUsers(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	// TODO: Query actual users from database with roles, teams, etc.
	// For now, generate 50 mock users
	users := generateMockUsers(int(orgID), 50)

	utils.RespondOK(c, users)
}

func generateMockUsers(orgID, count int) []UserResponse {
	firstNames := []string{
		"John", "Jane", "Michael", "Sarah", "David", "Emily", "Robert", "Lisa",
		"William", "Jessica", "James", "Ashley", "Christopher", "Amanda", "Daniel",
		"Melissa", "Matthew", "Jennifer", "Joseph", "Stephanie", "Ryan", "Nicole",
		"Andrew", "Laura", "Brian", "Brittany", "Kevin", "Samantha", "Thomas",
		"Rachel", "Justin", "Elizabeth", "Brandon", "Megan", "Jacob", "Kayla",
		"Nicholas", "Lauren", "Tyler", "Anna", "Eric", "Olivia", "Aaron", "Emma",
		"Adam", "Sophia", "Jason", "Isabella", "Nathan", "Ava",
	}

	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
		"Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez",
		"Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
		"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark",
		"Ramirez", "Lewis", "Robinson", "Walker", "Young", "Allen", "King",
		"Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores", "Green",
		"Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
		"Carter", "Roberts",
	}

	teams := []string{
		"Engineering", "Product", "Design", "Marketing", "Sales", "Operations",
		"DevOps", "Security", "Data", "Support", "Finance", "HR",
	}

	users := make([]UserResponse, count)
	rand.Seed(int64(orgID)) // Use orgID as seed for consistent results per org

	for i := 0; i < count; i++ {
		firstName := firstNames[i%len(firstNames)]
		lastName := lastNames[(i/len(firstNames))%len(lastNames)]
		email := fmt.Sprintf("%s.%s@company.com",
			toLower(firstName),
			toLower(lastName))

		// Role distribution: more developers, fewer admins
		var role string
		roleRand := rand.Float64()
		if roleRand < 0.1 {
			role = "admin"
		} else if roleRand < 0.25 {
			role = "manager"
		} else if roleRand < 0.8 {
			role = "developer"
		} else {
			role = "viewer"
		}

		// Status distribution: mostly active
		var status string
		statusRand := rand.Float64()
		if statusRand < 0.75 {
			status = "active"
		} else if statusRand < 0.9 {
			status = "invited"
		} else {
			status = "inactive"
		}

		// Random 1-3 teams
		numTeams := rand.Intn(3) + 1
		userTeams := make([]string, numTeams)
		for j := 0; j < numTeams; j++ {
			userTeams[j] = teams[(i*3+j)%len(teams)]
		}

		// Random creation date in last 2 years
		now := time.Now()
		createdAt := now.AddDate(0, 0, -rand.Intn(730))

		// Last access: active users recently, inactive users long ago
		var lastAccess time.Time
		if status == "active" {
			// Last 7 days
			lastAccess = now.Add(-time.Minute * time.Duration(rand.Intn(10080)))
		} else if status == "invited" {
			// Never accessed (set to creation date)
			lastAccess = createdAt
		} else {
			// 30-90 days ago
			lastAccess = now.AddDate(0, 0, -(30 + rand.Intn(60)))
		}

		users[i] = UserResponse{
			ID:         i + 1,
			FirstName:  firstName,
			LastName:   lastName,
			Email:      email,
			Role:       role,
			Teams:      userTeams,
			CreatedAt:  createdAt,
			LastAccess: lastAccess,
			Status:     status,
		}
	}

	return users
}

func toLower(s string) string {
	if len(s) == 0 {
		return s
	}
	// Convert first character to lowercase
	first := s[0]
	if first >= 'A' && first <= 'Z' {
		first = first + 32
	}
	return string(first) + s[1:]
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

type UpdateUserRequest struct {
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Email     string   `json:"email" binding:"omitempty,email"`
	Role      string   `json:"role"`
	Teams     []string `json:"teams"`
	Status    string   `json:"status"`
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

	// Validate role
	validRoles := map[string]bool{"admin": true, "manager": true, "developer": true, "viewer": true}
	if input.Role != "" && !validRoles[input.Role] {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid role")
		return
	}

	// Validate status
	validStatuses := map[string]bool{"active": true, "inactive": true, "invited": true}
	if input.Status != "" && !validStatuses[input.Status] {
		utils.RespondError(c, http.StatusBadRequest, "BAD_INPUT", "Invalid status")
		return
	}

	// TODO: Update in database with new fields (using orgID for validation)
	// For now, return updated mock user
	_ = orgID // Will be used when implementing real database operations

	updatedUser := UserResponse{
		ID:         userID,
		FirstName:  input.FirstName,
		LastName:   input.LastName,
		Email:      input.Email,
		Role:       input.Role,
		Teams:      input.Teams,
		Status:     input.Status,
		CreatedAt:  time.Now().AddDate(0, 0, -100), // Mock data
		LastAccess: time.Now().Add(-time.Hour * 2),
	}

	utils.RespondOK(c, updatedUser)
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

	// TODO: Delete user from database
	// For now, just verify the user exists in this org (mock validation)
	if userID < 1 || userID > 50 {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "User not found in this organization")
		return
	}

	// In production, you would:
	// - Check if user exists and belongs to organization (using orgID)
	// - Remove user from organization (soft delete or hard delete)
	// - Optionally remove from all teams
	// - Log the deletion for audit
	_ = orgID // Will be used when implementing real database operations

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
