package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"gorm.io/gorm"
)

type AuthController struct {
	googleOAuthConfig *oauth2.Config
	githubOAuthConfig *oauth2.Config
	oauthStateString  string
	environment       string
	frontendURL       string
}

func NewAuthController(config *config.Configuration) *AuthController {
	return &AuthController{
		googleOAuthConfig: &oauth2.Config{
			ClientID:     config.OAuth.Google.ClientID,
			ClientSecret: config.OAuth.Google.ClientSecret,
			RedirectURL:  config.OAuth.Google.CallbackURL,
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		},
		githubOAuthConfig: &oauth2.Config{
			ClientID:     config.OAuth.GitHub.ClientID,
			ClientSecret: config.OAuth.GitHub.ClientSecret,
			RedirectURL:  config.OAuth.GitHub.CallbackURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
		oauthStateString: config.Secrets.CSFRSecret,
		environment:      config.Environment,
		frontendURL:      config.Frontend.URL,
	}
}

func (a *AuthController) GoogleLogin(c *gin.Context) {
	utils.LogOAuthStart(c, "google")

	state, err := a.generateOAuthState("google")
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "STATE_ERROR", "Failed to generate OAuth state")
		return
	}
	url := a.googleOAuthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusFound, url)
}

func (a *AuthController) GoogleCallback(c *gin.Context) {
	a.handleOAuthCallback(c, a.googleOAuthConfig, "google")
}

func (a *AuthController) GithubLogin(c *gin.Context) {
	utils.LogOAuthStart(c, "github")

	state, err := a.generateOAuthState("github")
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "STATE_ERROR", "Failed to generate OAuth state")
		return
	}
	url := a.githubOAuthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusFound, url)
}

func (a *AuthController) GithubCallback(c *gin.Context) {
	a.handleOAuthCallback(c, a.githubOAuthConfig, "github")
}

// generateOAuthState creates a unique, time-limited state token for CSRF protection
func (a *AuthController) generateOAuthState(provider string) (string, error) {
	// Generate cryptographically secure random state
	state, err := utils.GenerateSecureRandomString(32)
	if err != nil {
		return "", err
	}

	// Store state in database with 10 minute expiration
	db := models.GetDB()
	oauthState := models.OAuthState{
		State:     state,
		Provider:  provider,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := db.Create(&oauthState).Error; err != nil {
		return "", err
	}

	return state, nil
}

// validateOAuthState verifies and consumes the state token
func (a *AuthController) validateOAuthState(state, provider string) error {
	db := models.GetDB()

	var oauthState models.OAuthState
	if err := db.Where("state = ? AND provider = ?", state, provider).First(&oauthState).Error; err != nil {
		return fmt.Errorf("invalid state token")
	}

	// Check expiration
	if time.Now().After(oauthState.ExpiresAt) {
		// Clean up expired token
		db.Delete(&oauthState)
		return fmt.Errorf("state token expired")
	}

	// Delete state token after use (one-time use)
	db.Delete(&oauthState)

	return nil
}

func (a *AuthController) handleOAuthCallback(c *gin.Context, config *oauth2.Config, provider string) {
	state := c.Query("state")

	// Validate state token (CSRF protection)
	if err := a.validateOAuthState(state, provider); err != nil {
		utils.LogOAuthCallback(c, provider, false, nil, "", "INVALID_STATE", err.Error())
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_STATE", "Invalid OAuth state: "+err.Error())
		return
	}

	code := c.Query("code")

	// Create context with timeout for OAuth token exchange
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := config.Exchange(ctx, code)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			utils.LogOAuthCallback(c, provider, false, nil, "", "OAUTH_TIMEOUT", "Request timed out")
			utils.RespondError(c, http.StatusRequestTimeout, "OAUTH_TIMEOUT", "OAuth provider request timed out")
		} else {
			utils.LogOAuthCallback(c, provider, false, nil, "", "INVALID_CODE", err.Error())
			utils.RespondError(c, http.StatusInternalServerError, "INVALID_CODE", "Failed to exchange token")
		}
		return
	}

	// Create context with timeout for fetching user info
	userCtx, userCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer userCancel()

	client := config.Client(userCtx, token)
	userInfo, err := fetchUserInfo(client, provider)
	if err != nil {
		utils.LogOAuthCallback(c, provider, false, nil, "", "FAILED_USER_OPERATION", err.Error())
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to fetch user info")
		return
	}

	var user models.User
	db := models.GetDB()

	// Check if user exists by provider ID
	if err := db.Where("provider = ? AND provider_id = ?", provider, userInfo.ID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check if user was invited (exists with email but no provider ID)
			var invitedUser models.User
			if err := db.Where("email = ? AND status = ?", userInfo.Email, models.UserStatusInvited).First(&invitedUser).Error; err == nil {
				// User was invited! Link OAuth account to existing user record
				now := time.Now()
				invitedUser.Provider = provider
				invitedUser.ProviderID = userInfo.ID
				invitedUser.Avatar = userInfo.Avatar
				invitedUser.Status = models.UserStatusActive
				invitedUser.LastAccess = &now

				// Update name if not set
				if invitedUser.Name == "" || invitedUser.Name == invitedUser.FirstName+" "+invitedUser.LastName {
					invitedUser.Name = userInfo.Name
				}

				if err := db.Save(&invitedUser).Error; err != nil {
					utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to activate invited user")
					return
				}

				user = invitedUser
			} else {
				// Not invited - create new user with personal org
				user = models.User{
					Name:       userInfo.Name,
					Email:      userInfo.Email,
					Provider:   provider,
					ProviderID: userInfo.ID,
					Avatar:     userInfo.Avatar,
					Status:     models.UserStatusActive,
					Role:       models.UserRoleDeveloper,
				}

				// Create personal/default organization
				personalOrg := models.Organization{
					Name: fmt.Sprintf("%s's Personal Organization", userInfo.Name),
				}

				tx := db.Begin()
				if err := tx.Create(&personalOrg).Error; err != nil {
					tx.Rollback()
					utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_CREATION", "Failed to create personal organization")
					return
				}

				// Set user's default org ID
				user.OrganizationID = personalOrg.ID

				// Save user
				if err := tx.Create(&user).Error; err != nil {
					tx.Rollback()
					utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_CREATION", "Failed to create user")
					return
				}

				// Add user to organization members via many2many join
				if err := tx.Model(&personalOrg).Association("Members").Append(&user); err != nil {
					tx.Rollback()
					utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_MEMBERSHIP", "Failed to assign user to personal organization")
					return
				}

				tx.Commit()
			}
		} else {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_QUERY", "Failed to query user")
			return
		}
	} else {
		// Existing user - update last access
		now := time.Now()
		user.LastAccess = &now
		db.Save(&user)
	}

	accessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate access token")
		return
	}

	refreshToken, jti, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate refresh token")
		return
	}

	// Store refresh token in database
	refreshTokenRecord := models.RefreshToken{
		TokenHash: models.HashToken(refreshToken),
		JTI:       jti,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}
	if err := db.Create(&refreshTokenRecord).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TOKEN_STORAGE", "Failed to store refresh token")
		return
	}

	// Set cookies with enhanced security (SameSite=Strict)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
		Path:     "/auth/refresh",         // limit cookie to refresh endpoint
	})

	// Log successful OAuth login
	utils.LogOAuthCallback(c, provider, true, &user.ID, user.Email, "", "")
	utils.LogSuccessfulLogin(c, user.ID, user.Email, provider)

	c.Redirect(http.StatusTemporaryRedirect, a.frontendURL)
}

// Simplified user info response struct
type OAuthUserInfo struct {
	ID     string
	Email  string
	Name   string
	Avatar string
}

func fetchUserInfo(client *http.Client, provider string) (OAuthUserInfo, error) {
	var userInfo OAuthUserInfo

	if provider == "google" {
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			return userInfo, err
		}
		defer resp.Body.Close()
		var body struct {
			ID      string `json:"id"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return userInfo, err
		}
		userInfo = OAuthUserInfo{ID: body.ID, Email: body.Email, Name: body.Name, Avatar: body.Picture}
	} else if provider == "github" {
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			return userInfo, err
		}
		defer resp.Body.Close()
		var body struct {
			ID     int    `json:"id"`
			Email  string `json:"email"`
			Name   string `json:"name"`
			Login  string `json:"login"`
			Avatar string `json:"avatar_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return userInfo, err
		}
		email := body.Email
		if email == "" {
			// GitHub email fallback
			emailResp, err := client.Get("https://api.github.com/user/emails")
			if err != nil {
				return userInfo, err
			}
			defer emailResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
				for _, e := range emails {
					if e.Primary {
						email = e.Email
						break
					}
				}
			}
		}
		userInfo = OAuthUserInfo{
			ID:     fmt.Sprintf("%d", body.ID),
			Email:  email,
			Name:   body.Name,
			Avatar: body.Avatar,
		}
	}
	return userInfo, nil
}

func (a *AuthController) MeHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	utils.RespondOK(c, user)
}

func (a *AuthController) RefreshToken(c *gin.Context) {
	cookie, err := c.Request.Cookie("refresh_token")
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Missing refresh token")
		return
	}

	refreshTokenStr := cookie.Value

	// Verify JWT structure
	userID, jti, err := utils.VerifyRefreshToken(refreshTokenStr)
	if err != nil {
		utils.LogTokenRefresh(c, 0, "", false, "INVALID_TOKEN", err.Error())
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
		return
	}

	db := models.GetDB()

	// Check if token exists in database and is not revoked
	var storedToken models.RefreshToken
	tokenHash := models.HashToken(refreshTokenStr)
	if err := db.Where("token_hash = ? AND jti = ?", tokenHash, jti).First(&storedToken).Error; err != nil {
		utils.LogTokenRefresh(c, userID, "", false, "TOKEN_NOT_FOUND", "Token not found or already used")
		utils.RespondError(c, http.StatusUnauthorized, "TOKEN_NOT_FOUND", "Refresh token not found or already used")
		return
	}

	// Check if token is revoked
	if storedToken.Revoked {
		utils.LogTokenRefresh(c, userID, "", false, "TOKEN_REVOKED", "Token has been revoked")
		utils.RespondError(c, http.StatusUnauthorized, "TOKEN_REVOKED", "Refresh token has been revoked")
		return
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		db.Delete(&storedToken)
		utils.LogTokenRefresh(c, userID, "", false, "TOKEN_EXPIRED", "Token expired")
		utils.RespondError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token expired")
		return
	}

	// Verify user still exists
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.LogTokenRefresh(c, userID, "", false, "USER_NOT_FOUND", "User not found")
		utils.RespondError(c, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found")
		return
	}

	// REVOKE OLD TOKEN (token rotation)
	now := time.Now()
	storedToken.Revoked = true
	storedToken.RevokedAt = &now
	db.Save(&storedToken)

	// Generate new tokens
	newAccessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate new access token")
		return
	}

	newRefreshToken, newJTI, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate refresh token")
		return
	}

	// Store new refresh token in database
	newRefreshTokenRecord := models.RefreshToken{
		TokenHash: models.HashToken(newRefreshToken),
		JTI:       newJTI,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}
	if err := db.Create(&newRefreshTokenRecord).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_TOKEN_STORAGE", "Failed to store refresh token")
		return
	}

	// Set new cookies with enhanced security
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
		Path:     "/auth/refresh",         // limit cookie to refresh endpoint
	})

	// Log successful token refresh
	utils.LogTokenRefresh(c, user.ID, user.Email, true, "", "")

	c.JSON(http.StatusOK, gin.H{"refresh": "ok"})
}

func (a *AuthController) Logout(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if exists {
		if user, ok := userInterface.(models.User); ok {
			// Revoke all refresh tokens for this user
			db := models.GetDB()
			now := time.Now()
			db.Model(&models.RefreshToken{}).
				Where("user_id = ? AND revoked = ?", user.ID, false).
				Updates(map[string]interface{}{
					"revoked":    true,
					"revoked_at": now,
				})

			// Log successful logout
			utils.LogLogout(c, user.ID, user.Email)
		}
	}

	// Clear cookies with enhanced security settings
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteStrictMode, // Upgraded from Lax
		Path:     "/auth/refresh",         // limit cookie to refresh endpoint
	})

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
