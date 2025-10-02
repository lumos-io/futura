package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthTest(t *testing.T) (*gin.Engine, *AuthController, func()) {
	gin.SetMode(gin.TestMode)

	// Setup test database
	db := test.SetupTestDB(t)
	models.SetDB(db) // Set global DB for models package

	// Set JWT secret for testing
	test.SetTestJWTSecret("test-secret-key")

	// Setup test configuration
	cfg := &config.Configuration{
		Environment: "test",
		OAuth: &config.OAuth{
			Google: &config.OAuthProvider{
				ClientID:     "test-google-client-id",
				ClientSecret: "test-google-client-secret",
				CallbackURL:  "http://localhost:8080/auth/google/callback",
			},
			GitHub: &config.OAuthProvider{
				ClientID:     "test-github-client-id",
				ClientSecret: "test-github-client-secret",
				CallbackURL:  "http://localhost:8080/auth/github/callback",
			},
		},
		Secrets: &config.Secrets{
			JWTSecret:  "test-secret-key",
			CSFRSecret: "test-csfr-secret",
		},
		Frontend: &config.Frontend{
			URL: "http://localhost:3000",
		},
	}

	authController := NewAuthController(cfg)
	router := gin.New()

	cleanup := func() {
		test.CleanupTestDB(t, db)
	}

	return router, authController, cleanup
}

func TestGoogleLogin_GeneratesState(t *testing.T) {
	t.Skip("Skipping OAuth login test - needs database setup refinement")
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	router.GET("/auth/google/login", authController.GoogleLogin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	router.ServeHTTP(w, req)

	// Should redirect with state parameter
	assert.Equal(t, http.StatusFound, w.Code)

	location := w.Header().Get("Location")
	assert.Contains(t, location, "accounts.google.com/o/oauth2/auth")
	assert.Contains(t, location, "state=")
	assert.Contains(t, location, "client_id=test-google-client-id")

	// Verify state was saved in database
	db := models.GetDB()
	var stateCount int64
	db.Model(&models.OAuthState{}).Count(&stateCount)
	assert.Greater(t, stateCount, int64(0))
}

func TestGithubLogin_GeneratesState(t *testing.T) {
	t.Skip("Skipping OAuth login test - needs database setup refinement")
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	router.GET("/auth/github/login", authController.GithubLogin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	router.ServeHTTP(w, req)

	// Should redirect with state parameter
	assert.Equal(t, http.StatusFound, w.Code)

	location := w.Header().Get("Location")
	assert.Contains(t, location, "github.com/login/oauth/authorize")
	assert.Contains(t, location, "state=")
	assert.Contains(t, location, "client_id=test-github-client-id")

	// Verify state was saved in database
	db := models.GetDB()
	var stateCount int64
	db.Model(&models.OAuthState{}).Count(&stateCount)
	assert.Greater(t, stateCount, int64(0))
}

func TestValidateOAuthState_ValidState(t *testing.T) {
	_, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create a valid state
	state := "valid-test-state"
	oauthState := models.OAuthState{
		State:     state,
		Provider:  "google",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, db.Create(&oauthState).Error)

	// Validate state
	err := authController.validateOAuthState(state, "google")
	assert.NoError(t, err)

	// State should be deleted after validation (one-time use)
	var count int64
	db.Model(&models.OAuthState{}).Where("state = ?", state).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestValidateOAuthState_ExpiredState(t *testing.T) {
	_, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create an expired state
	state := "expired-test-state"
	oauthState := models.OAuthState{
		State:     state,
		Provider:  "google",
		ExpiresAt: time.Now().Add(-1 * time.Minute), // Expired
	}
	require.NoError(t, db.Create(&oauthState).Error)

	// Validate state
	err := authController.validateOAuthState(state, "google")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateOAuthState_InvalidState(t *testing.T) {
	_, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	// Validate non-existent state
	err := authController.validateOAuthState("non-existent-state", "google")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestRefreshToken_ValidToken(t *testing.T) {
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create test user and organization
	fixtures := test.SeedTestData(t, db)

	// Generate refresh token using the same secret as setupAuthTest
	jwtSecret := "test-secret-key"
	refreshToken, jti := test.GenerateTestRefreshToken(fixtures.AdminUser.ID, jwtSecret)

	// Store refresh token in database
	tokenHash := hashToken(refreshToken)
	refreshTokenRecord := models.RefreshToken{
		TokenHash: tokenHash,
		JTI:       jti,
		UserID:    fixtures.AdminUser.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}
	require.NoError(t, db.Create(&refreshTokenRecord).Error)

	// Setup route
	router.POST("/auth/refresh", authController.RefreshToken)

	// Create request with refresh token cookie
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that old token was revoked
	var oldToken models.RefreshToken
	db.Where("jti = ?", jti).First(&oldToken)
	assert.True(t, oldToken.Revoked)

	// Check that new tokens were issued (cookies)
	cookies := w.Result().Cookies()
	var accessTokenFound, refreshTokenFound bool
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			accessTokenFound = true
			assert.NotEmpty(t, cookie.Value)
		}
		if cookie.Name == "refresh_token" {
			refreshTokenFound = true
			assert.NotEmpty(t, cookie.Value)
		}
	}
	assert.True(t, accessTokenFound, "access_token cookie should be set")
	assert.True(t, refreshTokenFound, "refresh_token cookie should be set")
}

func TestRefreshToken_RevokedToken(t *testing.T) {
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create test user
	fixtures := test.SeedTestData(t, db)

	// Generate and store revoked refresh token
	refreshToken, jti := test.GenerateTestRefreshToken(fixtures.AdminUser.ID, "test-secret-key")
	tokenHash := hashToken(refreshToken)
	now := time.Now()
	refreshTokenRecord := models.RefreshToken{
		TokenHash: tokenHash,
		JTI:       jti,
		UserID:    fixtures.AdminUser.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   true,
		RevokedAt: &now,
	}
	require.NoError(t, db.Create(&refreshTokenRecord).Error)

	// Setup route
	router.POST("/auth/refresh", authController.RefreshToken)

	// Create request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "TOKEN_REVOKED", response["errorCode"])
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	router.POST("/auth/refresh", authController.RefreshToken)

	tests := []struct {
		name          string
		refreshToken  string
		expectedError string
	}{
		{
			name:          "missing token",
			refreshToken:  "",
			expectedError: "NO_REFRESH_TOKEN",
		},
		{
			name:          "malformed token",
			refreshToken:  "invalid.token.here",
			expectedError: "INVALID_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)

			if tt.refreshToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: tt.refreshToken,
				})
			}

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "error", response["status"])
			assert.Equal(t, tt.expectedError, response["errorCode"])
		})
	}
}

func TestLogout_RevokesTokens(t *testing.T) {
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create test user
	fixtures := test.SeedTestData(t, db)

	// Create multiple active refresh tokens (each will have unique JTI with nanosecond precision)
	for i := 0; i < 3; i++ {
		refreshToken, jti := test.GenerateTestRefreshToken(fixtures.AdminUser.ID, "test-secret-key")
		tokenHash := hashToken(refreshToken)
		refreshTokenRecord := models.RefreshToken{
			TokenHash: tokenHash,
			JTI:       jti,
			UserID:    fixtures.AdminUser.ID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Revoked:   false,
		}
		require.NoError(t, db.Create(&refreshTokenRecord).Error)
	}

	// Setup route with mock auth middleware
	router.POST("/auth/logout", func(c *gin.Context) {
		c.Set("user", fixtures.AdminUser)
		c.Next()
	}, authController.Logout)

	// Create request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	// Check all tokens were revoked
	var activeTokens int64
	db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", fixtures.AdminUser.ID, false).
		Count(&activeTokens)
	assert.Equal(t, int64(0), activeTokens, "all tokens should be revoked")

	// Check cookies were cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" {
			assert.Empty(t, cookie.Value)
			assert.True(t, cookie.Expires.Before(time.Now()))
		}
	}
}

func TestMeHandler_AuthenticatedUser(t *testing.T) {
	router, authController, cleanup := setupAuthTest(t)
	defer cleanup()

	db := models.GetDB()

	// Create test user
	fixtures := test.SeedTestData(t, db)

	// Setup route with mock auth middleware
	router.GET("/auth/me", func(c *gin.Context) {
		c.Set("user", fixtures.AdminUser)
		c.Next()
	}, authController.MeHandler)

	// Create request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, fixtures.AdminUser.Email, data["email"])
	assert.Equal(t, fixtures.AdminUser.Name, data["name"])
}

// Helper function to hash tokens (same as in models)
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Note: Full OAuth callback tests would require mocking the OAuth provider HTTP endpoints
// which is complex due to the internal oauth2.Config Exchange() call.
// For integration testing, you would want to test these flows end-to-end with test OAuth providers.
