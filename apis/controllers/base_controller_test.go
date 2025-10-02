package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/version"
	"github.com/stretchr/testify/assert"
)

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "returns healthy status",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"success","data":{"result":"healthy"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)

			// Execute
			Healthz(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set test version values
	version.Version = "1.0.0"
	version.CommitSHA = "abc123"
	version.BuildTime = "2024-01-01"

	tests := []struct {
		name           string
		expectedStatus int
		checkVersion   bool
	}{
		{
			name:           "returns version information",
			expectedStatus: http.StatusOK,
			checkVersion:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/version", nil)

			// Execute
			Version(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkVersion {
				assert.Contains(t, w.Body.String(), "version")
				assert.Contains(t, w.Body.String(), "commit_sha")
				assert.Contains(t, w.Body.String(), "build_time")
				assert.Contains(t, w.Body.String(), "1.0.0")
				assert.Contains(t, w.Body.String(), "abc123")
				assert.Contains(t, w.Body.String(), "2024-01-01")
			}
		})
	}
}
