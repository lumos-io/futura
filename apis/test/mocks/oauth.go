package mocks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/opisvigilant/futura/apis/test"
)

// MockOAuthServer represents a mock OAuth server for testing
type MockOAuthServer struct {
	TokenServer    *httptest.Server
	UserInfoServer *httptest.Server
	EmailServer    *httptest.Server // For GitHub emails
}

// NewMockGoogleOAuthServer creates a mock Google OAuth server
func NewMockGoogleOAuthServer() *MockOAuthServer {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(test.MockGoogleToken())
	}))

	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(test.MockGoogleUserInfo())
	}))

	return &MockOAuthServer{
		TokenServer:    tokenServer,
		UserInfoServer: userInfoServer,
	}
}

// NewMockGithubOAuthServer creates a mock GitHub OAuth server
func NewMockGithubOAuthServer() *MockOAuthServer {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(test.MockGithubToken())
	}))

	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(test.MockGithubUserInfo())
	}))

	emailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(test.MockGithubEmails())
	}))

	return &MockOAuthServer{
		TokenServer:    tokenServer,
		UserInfoServer: userInfoServer,
		EmailServer:    emailServer,
	}
}

// NewMockGoogleOAuthServerWithError creates a mock server that returns errors
func NewMockGoogleOAuthServerWithError(statusCode int, errorMsg string) *MockOAuthServer {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMsg))
	}))

	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMsg))
	}))

	return &MockOAuthServer{
		TokenServer:    tokenServer,
		UserInfoServer: userInfoServer,
	}
}

// Close shuts down all mock servers
func (m *MockOAuthServer) Close() {
	m.TokenServer.Close()
	m.UserInfoServer.Close()
	if m.EmailServer != nil {
		m.EmailServer.Close()
	}
}

// GetTokenURL returns the token server URL
func (m *MockOAuthServer) GetTokenURL() string {
	return m.TokenServer.URL
}

// GetUserInfoURL returns the user info server URL
func (m *MockOAuthServer) GetUserInfoURL() string {
	return m.UserInfoServer.URL
}

// GetEmailURL returns the email server URL (GitHub only)
func (m *MockOAuthServer) GetEmailURL() string {
	if m.EmailServer != nil {
		return m.EmailServer.URL
	}
	return ""
}
