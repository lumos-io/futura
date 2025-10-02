package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTeamTest(t *testing.T) (*gin.Engine, *test.TestFixtures, func()) {
	gin.SetMode(gin.TestMode)
	db := test.SetupTestDB(t)
	models.SetDB(db)

	fixtures := test.SeedTestData(t, db)
	router := gin.New()

	cleanup := func() {
		test.CleanupTestDB(t, db)
	}

	return router, fixtures, cleanup
}

func TestGetTeams(t *testing.T) {
	router, fixtures, cleanup := setupTeamTest(t)
	defer cleanup()

	router.GET("/api/organizations/:org_id/teams", GetTeams)

	tests := []struct {
		name              string
		orgID             string
		expectedStatus    int
		expectedTeamCount int
		expectedError     string
	}{
		{
			name:              "returns teams for organization",
			orgID:             fmt.Sprintf("%d", fixtures.Organization.ID),
			expectedStatus:    http.StatusOK,
			expectedTeamCount: 1, // One team from fixtures
		},
		{
			name:           "returns 404 for invalid org_id",
			orgID:          "invalid",
			expectedStatus: http.StatusNotFound,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+tt.orgID+"/teams", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedError != "" {
				assert.Equal(t, "error", response["status"])
				assert.Equal(t, tt.expectedError, response["errorCode"])
			} else {
				assert.Equal(t, "success", response["status"])
				data := response["data"].([]interface{})
				assert.Equal(t, tt.expectedTeamCount, len(data))

				// Check response format
				if len(data) > 0 {
					team := data[0].(map[string]interface{})
					assert.Contains(t, team, "id")
					assert.Contains(t, team, "name")
					assert.Contains(t, team, "members")
					assert.Contains(t, team, "member_count")
				}
			}
		})
	}
}

func TestCreateTeam(t *testing.T) {
	router, fixtures, cleanup := setupTeamTest(t)
	defer cleanup()

	router.POST("/api/organizations/:org_id/teams", CreateTeam)

	tests := []struct {
		name           string
		orgID          string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkCreated   bool
	}{
		{
			name:  "creates team successfully",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"name":        "Engineering Team",
				"description": "Core engineering team",
			},
			expectedStatus: http.StatusCreated,
			checkCreated:   true,
		},
		{
			name:  "creates team without description",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"name": "Product Team",
			},
			expectedStatus: http.StatusCreated,
			checkCreated:   true,
		},
		{
			name:  "fails with missing name",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"description": "No name team",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:  "fails with invalid org_id",
			orgID: "invalid",
			requestBody: map[string]interface{}{
				"name": "Test Team",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/organizations/"+tt.orgID+"/teams", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedError != "" {
				assert.Equal(t, "error", response["status"])
				assert.Equal(t, tt.expectedError, response["errorCode"])
			} else {
				assert.Equal(t, "success", response["status"])
			}

			if tt.checkCreated {
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.requestBody["name"], data["name"])
				assert.NotZero(t, data["id"])
				assert.Equal(t, float64(0), data["member_count"])
			}
		})
	}
}

func TestUpdateTeam(t *testing.T) {
	router, fixtures, cleanup := setupTeamTest(t)
	defer cleanup()

	router.PUT("/api/organizations/:org_id/teams/:team_id", UpdateTeam)

	tests := []struct {
		name           string
		orgID          string
		teamID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkUpdated   bool
	}{
		{
			name:   "updates team name successfully",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID: fmt.Sprintf("%d", fixtures.Team.ID),
			requestBody: map[string]interface{}{
				"name": "Updated Team Name",
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "updates team description",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID: fmt.Sprintf("%d", fixtures.Team.ID),
			requestBody: map[string]interface{}{
				"description": "Updated description",
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "updates team members",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID: fmt.Sprintf("%d", fixtures.Team.ID),
			requestBody: map[string]interface{}{
				"memberIds": []int{int(fixtures.AdminUser.ID), int(fixtures.DevUser.ID)},
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "returns 404 for non-existent team",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID: "99999",
			requestBody: map[string]interface{}{
				"name": "Test",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:   "returns 400 for invalid team_id",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID: "invalid",
			requestBody: map[string]interface{}{
				"name": "Test",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/organizations/"+tt.orgID+"/teams/"+tt.teamID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedError != "" {
				assert.Equal(t, "error", response["status"])
				assert.Equal(t, tt.expectedError, response["errorCode"])
			} else {
				assert.Equal(t, "success", response["status"])
			}

			if tt.checkUpdated {
				data := response["data"].(map[string]interface{})

				// Check updated fields
				if name, ok := tt.requestBody["name"]; ok {
					assert.Equal(t, name, data["name"])
				}
				if description, ok := tt.requestBody["description"]; ok {
					assert.Equal(t, description, data["description"])
				}
				if memberIds, ok := tt.requestBody["memberIds"]; ok {
					members := data["members"].([]interface{})
					assert.Equal(t, len(memberIds.([]int)), len(members))
				}
			}
		})
	}
}

func TestDeleteTeam(t *testing.T) {
	router, fixtures, cleanup := setupTeamTest(t)
	defer cleanup()

	db := models.GetDB()

	router.DELETE("/api/organizations/:org_id/teams/:team_id", DeleteTeam)

	tests := []struct {
		name           string
		setupTeam      bool
		orgID          string
		teamID         string
		expectedStatus int
		expectedError  string
		checkDeleted   bool
	}{
		{
			name:           "deletes team successfully",
			setupTeam:      true,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			expectedStatus: http.StatusOK,
			checkDeleted:   true,
		},
		{
			name:           "returns 404 for non-existent team",
			setupTeam:      false,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID:         "99999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:           "returns 400 for invalid team_id",
			setupTeam:      false,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			teamID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var teamID string
			var createdTeamID uint

			if tt.setupTeam {
				team := test.CreateTestTeam(t, db, "To Delete", fixtures.Organization.ID)
				teamID = fmt.Sprintf("%d", team.ID)
				createdTeamID = team.ID
			} else {
				teamID = tt.teamID
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+tt.orgID+"/teams/"+teamID, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedError != "" {
				assert.Equal(t, "error", response["status"])
				assert.Equal(t, tt.expectedError, response["errorCode"])
			} else {
				assert.Equal(t, "success", response["status"])
			}

			if tt.checkDeleted {
				// Verify team was deleted
				var team models.Team
				err := db.First(&team, createdTeamID).Error
				assert.Error(t, err) // Should return error (record not found)
			}
		})
	}
}
