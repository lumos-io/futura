package controllers

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/utils"
)

type TeamMember struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type TeamResponse struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	MemberCount int          `json:"memberCount"`
	CreatedAt   time.Time    `json:"createdAt"`
	Members     []TeamMember `json:"members"`
}

func GetTeams(c *gin.Context) {
	orgID, err := parseOrgID(c)
	if err != nil {
		return
	}

	// TODO: Query actual teams from database
	// For now, generate mock teams
	teams := generateMockTeams(int(orgID))

	utils.RespondOK(c, teams)
}

func generateMockTeams(orgID int) []TeamResponse {
	teamDefinitions := []struct {
		Name        string
		Description string
	}{
		{"Engineering", "Software development and technical infrastructure team"},
		{"Product", "Product management and strategy team"},
		{"Design", "User experience and interface design team"},
		{"Marketing", "Marketing campaigns and brand management team"},
		{"Sales", "Sales operations and customer acquisition team"},
		{"Operations", "Business operations and process optimization team"},
		{"DevOps", "Development operations and infrastructure automation team"},
		{"Security", "Security engineering and compliance team"},
		{"Data", "Data analytics and business intelligence team"},
		{"Support", "Customer support and success team"},
		{"Finance", "Financial planning and accounting team"},
		{"HR", "Human resources and talent management team"},
	}

	rand.Seed(int64(orgID)) // Consistent results per org

	teams := make([]TeamResponse, len(teamDefinitions))
	now := time.Now()

	for i, def := range teamDefinitions {
		// Generate 3-15 members per team
		memberCount := 3 + rand.Intn(13)
		members := make([]TeamMember, memberCount)

		for j := 0; j < memberCount; j++ {
			members[j] = TeamMember{
				ID:    (i * 100) + j + 1,
				Name:  generateRandomName(i, j),
				Email: generateRandomEmail(i, j),
				Role:  generateRandomRole(),
			}
		}

		// Teams created over the last year
		createdAt := now.AddDate(0, 0, -rand.Intn(365))

		teams[i] = TeamResponse{
			ID:          i + 1,
			Name:        def.Name,
			Description: def.Description,
			MemberCount: memberCount,
			CreatedAt:   createdAt,
			Members:     members,
		}
	}

	return teams
}

func generateRandomName(teamIndex, memberIndex int) string {
	firstNames := []string{
		"John", "Jane", "Michael", "Sarah", "David", "Emily", "Robert", "Lisa",
		"William", "Jessica", "James", "Ashley", "Daniel", "Amanda", "Matthew",
	}
	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
		"Davis", "Rodriguez", "Martinez", "Anderson", "Taylor", "Moore",
	}

	firstName := firstNames[(teamIndex*7+memberIndex)%len(firstNames)]
	lastName := lastNames[(teamIndex*3+memberIndex)%len(lastNames)]
	return firstName + " " + lastName
}

func generateRandomEmail(teamIndex, memberIndex int) string {
	firstNames := []string{
		"john", "jane", "michael", "sarah", "david", "emily", "robert", "lisa",
		"william", "jessica", "james", "ashley", "daniel", "amanda", "matthew",
	}
	lastNames := []string{
		"smith", "johnson", "williams", "brown", "jones", "garcia", "miller",
		"davis", "rodriguez", "martinez", "anderson", "taylor", "moore",
	}

	firstName := firstNames[(teamIndex*7+memberIndex)%len(firstNames)]
	lastName := lastNames[(teamIndex*3+memberIndex)%len(lastNames)]
	return firstName + "." + lastName + "@company.com"
}

func generateRandomRole() string {
	roles := []string{"admin", "manager", "developer", "viewer"}
	roleWeights := []float64{0.1, 0.25, 0.8, 1.0} // Cumulative weights

	r := rand.Float64()
	for i, weight := range roleWeights {
		if r < weight {
			return roles[i]
		}
	}
	return "viewer"
}

type UpdateTeamRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Members     []TeamMember `json:"members"`
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

	// TODO: Update in database
	_ = orgID // Will be used when implementing real database operations

	updatedTeam := TeamResponse{
		ID:          teamID,
		Name:        input.Name,
		Description: input.Description,
		MemberCount: len(input.Members),
		CreatedAt:   time.Now().AddDate(0, 0, -100), // Mock data
		Members:     input.Members,
	}

	utils.RespondOK(c, updatedTeam)
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

	// TODO: Delete team from database
	// Mock validation
	if teamID < 1 || teamID > 12 {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Team not found in this organization")
		return
	}

	_ = orgID // Will be used when implementing real database operations

	utils.RespondOK(c, gin.H{"message": "Team deleted successfully"})
}
