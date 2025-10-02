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

func setupUserTest(t *testing.T) (*gin.Engine, *test.TestFixtures, func()) {
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

func TestGetUsers(t *testing.T) {
	router, fixtures, cleanup := setupUserTest(t)
	defer cleanup()

	router.GET("/api/organizations/:org_id/users", GetUsers)

	tests := []struct {
		name              string
		orgID             string
		expectedStatus    int
		expectedUserCount int
		expectedError     string
	}{
		{
			name:              "returns users for organization",
			orgID:             fmt.Sprintf("%d", fixtures.Organization.ID),
			expectedStatus:    http.StatusOK,
			expectedUserCount: 2, // admin and dev user from fixtures
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
			req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+tt.orgID+"/users", nil)
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
				assert.Equal(t, tt.expectedUserCount, len(data))

				// Check response format
				if len(data) > 0 {
					user := data[0].(map[string]interface{})
					assert.Contains(t, user, "id")
					assert.Contains(t, user, "email")
					assert.Contains(t, user, "role")
					assert.Contains(t, user, "teams")
				}
			}
		})
	}
}

func TestInviteUser(t *testing.T) {
	router, fixtures, cleanup := setupUserTest(t)
	defer cleanup()

	db := models.GetDB()

	router.POST("/api/organizations/:org_id/users/invite", InviteUser)

	tests := []struct {
		name           string
		orgID          string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkCreated   bool
	}{
		{
			name:  "invites user successfully",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "john.doe@test.com",
				"role":       3, // Developer
			},
			expectedStatus: http.StatusCreated,
			checkCreated:   true,
		},
		{
			name:  "invites user with teams",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"first_name": "Jane",
				"last_name":  "Smith",
				"email":      "jane.smith@test.com",
				"role":       3,
				"team_ids":   []int{int(fixtures.Team.ID)},
			},
			expectedStatus: http.StatusCreated,
			checkCreated:   true,
		},
		{
			name:  "fails with existing user email",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"first_name": "Existing",
				"last_name":  "User",
				"email":      fixtures.AdminUser.Email,
				"role":       3,
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "USER_EXISTS",
		},
		{
			name:  "fails with missing first name",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"last_name": "Doe",
				"email":     "test@test.com",
				"role":      3,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:  "fails with invalid email",
			orgID: fmt.Sprintf("%d", fixtures.Organization.ID),
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "invalid-email",
				"role":       3,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:  "fails with invalid org_id",
			orgID: "invalid",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "john@test.com",
				"role":       3,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/organizations/"+tt.orgID+"/users/invite", bytes.NewReader(body))
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
				assert.Equal(t, tt.requestBody["email"], data["email"])
				assert.NotZero(t, data["id"])
				assert.Equal(t, float64(2), data["status"]) // UserStatusInvited

				// Verify in database
				var user models.User
				db.Where("email = ?", tt.requestBody["email"]).First(&user)
				assert.Equal(t, models.UserStatusInvited, user.Status)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	router, fixtures, cleanup := setupUserTest(t)
	defer cleanup()

	router.PUT("/api/organizations/:org_id/users/:user_id", UpdateUser)

	tests := []struct {
		name           string
		orgID          string
		userID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkUpdated   bool
	}{
		{
			name:   "updates user successfully",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: fmt.Sprintf("%d", fixtures.DevUser.ID),
			requestBody: map[string]interface{}{
				"first_name": "Updated",
				"last_name":  "Name",
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "updates user role",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: fmt.Sprintf("%d", fixtures.DevUser.ID),
			requestBody: map[string]interface{}{
				"role": 2, // Manager
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "updates user teams",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: fmt.Sprintf("%d", fixtures.DevUser.ID),
			requestBody: map[string]interface{}{
				"team_ids": []int{int(fixtures.Team.ID)},
			},
			expectedStatus: http.StatusOK,
			checkUpdated:   true,
		},
		{
			name:   "returns 404 for non-existent user",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: "99999",
			requestBody: map[string]interface{}{
				"first_name": "Test",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:   "returns 400 for invalid user_id",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: "invalid",
			requestBody: map[string]interface{}{
				"first_name": "Test",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
		{
			name:   "returns 400 for invalid email format",
			orgID:  fmt.Sprintf("%d", fixtures.Organization.ID),
			userID: fmt.Sprintf("%d", fixtures.DevUser.ID),
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/organizations/"+tt.orgID+"/users/"+tt.userID, bytes.NewReader(body))
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
				if firstName, ok := tt.requestBody["first_name"]; ok {
					assert.Equal(t, firstName, data["first_name"])
				}
				if role, ok := tt.requestBody["role"]; ok {
					assert.Equal(t, float64(role.(int)), data["role"])
				}
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	router, fixtures, cleanup := setupUserTest(t)
	defer cleanup()

	db := models.GetDB()

	router.DELETE("/api/organizations/:org_id/users/:user_id", DeleteUser)

	tests := []struct {
		name           string
		setupUser      bool
		orgID          string
		userID         string
		expectedStatus int
		expectedError  string
		checkDeleted   bool
	}{
		{
			name:           "deletes user successfully",
			setupUser:      true,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			expectedStatus: http.StatusOK,
			checkDeleted:   true,
		},
		{
			name:           "returns 404 for non-existent user",
			setupUser:      false,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			userID:         "99999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name:           "returns 400 for invalid user_id",
			setupUser:      false,
			orgID:          fmt.Sprintf("%d", fixtures.Organization.ID),
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "BAD_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var userID string
			var createdUserID uint

			if tt.setupUser {
				user := test.CreateTestUser(t, db, "todelete@test.com", models.UserRoleDeveloper, fixtures.Organization.ID)
				userID = fmt.Sprintf("%d", user.ID)
				createdUserID = user.ID
			} else {
				userID = tt.userID
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+tt.orgID+"/users/"+userID, nil)
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
				// Verify user was deleted
				var user models.User
				err := db.First(&user, createdUserID).Error
				assert.Error(t, err) // Should return error (record not found)
			}
		})
	}
}

func TestRoleConversion(t *testing.T) {
	tests := []struct {
		name       string
		role       models.UserRole
		protoEnum  int
		expectRole models.UserRole
	}{
		{"Admin", models.UserRoleAdmin, 1, models.UserRoleAdmin},
		{"Manager", models.UserRoleManager, 2, models.UserRoleManager},
		{"Developer", models.UserRoleDeveloper, 3, models.UserRoleDeveloper},
		{"Viewer", models.UserRoleViewer, 4, models.UserRoleViewer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test role to proto enum
			enum := roleToProtoEnum(tt.role)
			assert.Equal(t, tt.protoEnum, enum)

			// Test proto enum to role
			role := protoEnumToRole(tt.protoEnum)
			assert.Equal(t, tt.expectRole, role)
		})
	}
}

func TestStatusConversion(t *testing.T) {
	tests := []struct {
		name         string
		status       models.UserStatus
		protoEnum    int
		expectStatus models.UserStatus
	}{
		{"Active", models.UserStatusActive, 1, models.UserStatusActive},
		{"Invited", models.UserStatusInvited, 2, models.UserStatusInvited},
		{"Inactive", models.UserStatusInactive, 3, models.UserStatusInactive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test status to proto enum
			enum := statusToProtoEnum(tt.status)
			assert.Equal(t, tt.protoEnum, enum)

			// Test proto enum to status
			status := protoEnumToStatus(tt.protoEnum)
			assert.Equal(t, tt.expectStatus, status)
		})
	}
}

func TestParseOrgID(t *testing.T) {
	tests := []struct {
		name         string
		paramValue   string
		expectedID   uint
		expectError  bool
		expectedCode int
	}{
		{
			name:        "parses valid org_id",
			paramValue:  "123",
			expectedID:  123,
			expectError: false,
		},
		{
			name:         "fails on invalid org_id",
			paramValue:   "invalid",
			expectedID:   0,
			expectError:  true,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "org_id", Value: tt.paramValue}}

			id, err := parseOrgID(c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, w.Code)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
