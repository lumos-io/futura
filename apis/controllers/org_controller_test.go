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

func setupOrgTest(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)
	db := test.SetupTestDB(t)
	models.SetDB(db)

	router := gin.New()

	cleanup := func() {
		test.CleanupTestDB(t, db)
	}

	return router, cleanup
}

func TestGetOrganizations(t *testing.T) {
	router, cleanup := setupOrgTest(t)
	defer cleanup()

	db := models.GetDB()

	// Seed test data
	org1 := test.CreateTestOrganization(t, db, "Org 1")
	org2 := test.CreateTestOrganization(t, db, "Org 2")
	test.CreateTestUser(t, db, "user1@test.com", models.UserRoleAdmin, org1.ID)
	test.CreateTestUser(t, db, "user2@test.com", models.UserRoleDeveloper, org2.ID)

	router.GET("/organizations", GetOrganizations)

	tests := []struct {
		name               string
		expectedStatus     int
		expectedOrgCount   int
		checkPreloadedData bool
	}{
		{
			name:               "returns all organizations",
			expectedStatus:     http.StatusOK,
			expectedOrgCount:   2,
			checkPreloadedData: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/organizations", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "success", response["status"])

			data := response["data"].([]interface{})
			assert.Equal(t, tt.expectedOrgCount, len(data))

			if tt.checkPreloadedData {
				// Check that members are preloaded (JSON uses capitalized field names)
				org := data[0].(map[string]interface{})
				assert.Contains(t, org, "Members")
			}
		})
	}
}

func TestCreateOrganization(t *testing.T) {
	router, cleanup := setupOrgTest(t)
	defer cleanup()

	router.POST("/organizations", CreateOrganization)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkCreated   bool
	}{
		{
			name: "creates organization successfully",
			requestBody: map[string]interface{}{
				"name": "New Organization",
			},
			expectedStatus: http.StatusCreated,
			checkCreated:   true,
		},
		{
			name:           "fails with missing name",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name: "fails with empty name",
			requestBody: map[string]interface{}{
				"name": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
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
				assert.Equal(t, "New Organization", data["Name"])
				assert.NotZero(t, data["ID"])
			}
		})
	}
}

func TestGetOrganization(t *testing.T) {
	router, cleanup := setupOrgTest(t)
	defer cleanup()

	db := models.GetDB()
	org := test.CreateTestOrganization(t, db, "Test Org")
	test.CreateTestUser(t, db, "user@test.com", models.UserRoleAdmin, org.ID)

	router.GET("/organizations/:org_id", GetOrganization)

	tests := []struct {
		name           string
		orgID          string
		expectedStatus int
		expectedError  string
		checkData      bool
	}{
		{
			name:           "returns organization by ID",
			orgID:          fmt.Sprintf("%d", org.ID),
			expectedStatus: http.StatusOK,
			checkData:      true,
		},
		{
			name:           "returns 404 for non-existent org",
			orgID:          "99999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:           "returns 400 for invalid ID",
			orgID:          "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/organizations/"+tt.orgID, nil)
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

			if tt.checkData {
				data := response["data"].(map[string]interface{})
				assert.Equal(t, "Test Org", data["Name"])
				assert.Contains(t, data, "Members")
			}
		})
	}
}

func TestUpdateOrganization(t *testing.T) {
	router, cleanup := setupOrgTest(t)
	defer cleanup()

	db := models.GetDB()
	org := test.CreateTestOrganization(t, db, "Original Name")

	router.PUT("/organizations/:org_id", UpdateOrganization)

	tests := []struct {
		name           string
		orgID          string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkUpdated   bool
	}{
		{
			name:  "updates organization successfully",
			orgID: fmt.Sprintf("%d", org.ID),
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:  "fails with missing name",
			orgID: fmt.Sprintf("%d", org.ID),
			requestBody: map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:  "fails with empty name",
			orgID: fmt.Sprintf("%d", org.ID),
			requestBody: map[string]interface{}{
				"name": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:  "returns 404 for non-existent org",
			orgID: "99999",
			requestBody: map[string]interface{}{
				"name": "New Name",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:  "returns 400 for invalid ID",
			orgID: "invalid",
			requestBody: map[string]interface{}{
				"name": "New Name",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/organizations/"+tt.orgID, bytes.NewReader(body))
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
				assert.Equal(t, "Updated Name", data["Name"])

				// Verify in database
				var updatedOrg models.Organization
				db.First(&updatedOrg, org.ID)
				assert.Equal(t, "Updated Name", updatedOrg.Name)
			}
		})
	}
}

func TestDeleteOrganization(t *testing.T) {
	router, cleanup := setupOrgTest(t)
	defer cleanup()

	db := models.GetDB()

	router.DELETE("/organizations/:org_id", DeleteOrganization)

	tests := []struct {
		name           string
		setupOrg       bool
		orgID          string
		expectedStatus int
		expectedError  string
		checkDeleted   bool
	}{
		{
			name:           "deletes organization successfully",
			setupOrg:       true,
			expectedStatus: http.StatusOK,
			checkDeleted:   true,
		},
		{
			name:           "returns 400 for invalid ID",
			setupOrg:       false,
			orgID:          "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var orgID string
			var createdOrgID uint

			if tt.setupOrg {
				org := test.CreateTestOrganization(t, db, "To Delete")
				orgID = fmt.Sprintf("%d", org.ID)
				createdOrgID = org.ID
			} else {
				orgID = tt.orgID
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/organizations/"+orgID, nil)
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
				// Verify organization was deleted
				var org models.Organization
				err := db.First(&org, createdOrgID).Error
				assert.Error(t, err) // Should return error (record not found)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedID  uint
		expectError bool
	}{
		{
			name:        "parses valid ID",
			input:       "123",
			expectedID:  123,
			expectError: false,
		},
		{
			name:        "fails on invalid ID",
			input:       "invalid",
			expectedID:  0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parseID(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
