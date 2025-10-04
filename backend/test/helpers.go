package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Team{},
		&models.TeamMember{},
		&models.OAuthState{},
		&models.RefreshToken{},
		&models.AuditLog{},
		&models.ProviderConnection{},
		&models.ClusterMetadata{},
		&models.EKSClusterMetadata{},
		&models.GKEClusterMetadata{},
		&models.AKSClusterMetadata{},
		&models.DOKSClusterMetadata{},
		&models.ACKClusterMetadata{},
	)
	require.NoError(t, err)

	return db
}

// CleanupTestDB closes the database connection
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// SetupTestRouter creates a Gin router with test configuration
func SetupTestRouter(cfg *config.Configuration) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// NewTestContext creates a test Gin context with a response recorder
func NewTestContext(method, url string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

// NewAuthenticatedRequest creates an HTTP request with authentication cookies
func NewAuthenticatedRequest(method, url string, body io.Reader, accessToken, refreshToken string) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")

	if accessToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: accessToken,
		})
	}

	if refreshToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
	}

	return req
}

// ParseJSONResponse unmarshals JSON response into provided struct
func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), v)
	require.NoError(t, err)
}

// CreateTestUser creates a user in the test database
func CreateTestUser(t *testing.T, db *gorm.DB, email string, role models.UserRole, orgID uint) models.User {
	user := models.User{
		Name:           "Test User",
		Email:          email,
		Provider:       "google",
		ProviderID:     "test-provider-id",
		Status:         models.UserStatusActive,
		Role:           role,
		OrganizationID: orgID,
	}
	err := db.Create(&user).Error
	require.NoError(t, err)
	return user
}

// CreateTestOrganization creates an organization in the test database
func CreateTestOrganization(t *testing.T, db *gorm.DB, name string) models.Organization {
	org := models.Organization{
		Name: name,
	}
	err := db.Create(&org).Error
	require.NoError(t, err)
	return org
}

// CreateTestTeam creates a team in the test database
func CreateTestTeam(t *testing.T, db *gorm.DB, name string, orgID uint) models.Team {
	team := models.Team{
		Name:           name,
		OrganizationID: orgID,
	}
	err := db.Create(&team).Error
	require.NoError(t, err)
	return team
}

// GenerateTestAccessToken generates a valid access token for testing
func GenerateTestAccessToken(userID uint, secret string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// GenerateTestRefreshToken generates a valid refresh token for testing
func GenerateTestRefreshToken(userID uint, secret string) (string, string) {
	// Use nanosecond precision and add random suffix to ensure uniqueness
	jti := fmt.Sprintf("test-jti-%d-%d", time.Now().UnixNano(), userID)
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"jti":     jti,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString, jti
}

// MockAuthMiddleware bypasses authentication and sets user in context
func MockAuthMiddleware(user models.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}
}

// SetupMockRedis creates an in-memory Redis server for testing
func SetupMockRedis(t *testing.T) *miniredis.Miniredis {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	return mr
}

// SeedTestData seeds the database with common test data and returns fixtures
func SeedTestData(t *testing.T, db *gorm.DB) *TestFixtures {
	org := CreateTestOrganization(t, db, "Test Organization")
	admin := CreateTestUser(t, db, "admin@test.com", models.UserRoleAdmin, org.ID)
	developer := CreateTestUser(t, db, "dev@test.com", models.UserRoleDeveloper, org.ID)
	team := CreateTestTeam(t, db, "Test Team", org.ID)

	return &TestFixtures{
		Organization: org,
		AdminUser:    admin,
		DevUser:      developer,
		Team:         team,
	}
}

// TestFixtures holds commonly used test data
type TestFixtures struct {
	Organization models.Organization
	AdminUser    models.User
	DevUser      models.User
	Team         models.Team
}

// SetTestJWTSecret sets the JWT secret for testing
func SetTestJWTSecret(secret string) {
	utils.SetJWTSecret(secret)
}
