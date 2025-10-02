package test

// OAuth mock responses

// GoogleUserInfoResponse represents a mock Google OAuth user info response
type GoogleUserInfoResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// GoogleTokenResponse represents a mock Google OAuth token response
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

// GithubUserInfoResponse represents a mock GitHub OAuth user info response
type GithubUserInfoResponse struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GithubTokenResponse represents a mock GitHub OAuth token response
type GithubTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// GithubEmailResponse represents a mock GitHub email response
type GithubEmailResponse struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

// MockGoogleUserInfo returns a sample Google user info response
func MockGoogleUserInfo() GoogleUserInfoResponse {
	return GoogleUserInfoResponse{
		ID:            "google-user-123",
		Email:         "test@gmail.com",
		VerifiedEmail: true,
		Name:          "Test User",
		GivenName:     "Test",
		FamilyName:    "User",
		Picture:       "https://example.com/picture.jpg",
	}
}

// MockGoogleToken returns a sample Google token response
func MockGoogleToken() GoogleTokenResponse {
	return GoogleTokenResponse{
		AccessToken:  "mock-google-access-token",
		RefreshToken: "mock-google-refresh-token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IDToken:      "mock-id-token",
	}
}

// MockGithubUserInfo returns a sample GitHub user info response
func MockGithubUserInfo() GithubUserInfoResponse {
	return GithubUserInfoResponse{
		ID:        12345,
		Login:     "testuser",
		Email:     "test@github.com",
		Name:      "Test User",
		AvatarURL: "https://avatars.githubusercontent.com/u/12345",
	}
}

// MockGithubToken returns a sample GitHub token response
func MockGithubToken() GithubTokenResponse {
	return GithubTokenResponse{
		AccessToken: "mock-github-access-token",
		TokenType:   "bearer",
		Scope:       "read:user,user:email",
	}
}

// MockGithubEmails returns sample GitHub email responses
func MockGithubEmails() []GithubEmailResponse {
	return []GithubEmailResponse{
		{
			Email:      "test@github.com",
			Primary:    true,
			Verified:   true,
			Visibility: "public",
		},
	}
}
