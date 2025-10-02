package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
	url := a.googleOAuthConfig.AuthCodeURL(a.oauthStateString)
	c.Redirect(http.StatusFound, url)
}

func (a *AuthController) GoogleCallback(c *gin.Context) {
	a.handleOAuthCallback(c, a.googleOAuthConfig, "google")
}

func (a *AuthController) GithubLogin(c *gin.Context) {
	url := a.githubOAuthConfig.AuthCodeURL(a.oauthStateString)
	c.Redirect(http.StatusFound, url)
}

func (a *AuthController) GithubCallback(c *gin.Context) {
	a.handleOAuthCallback(c, a.githubOAuthConfig, "github")
}

func (a *AuthController) handleOAuthCallback(c *gin.Context, config *oauth2.Config, provider string) {
	state := c.Query("state")
	if state != a.oauthStateString {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_STATE", "Invalid OAuth state")
		return
	}

	code := c.Query("code")
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "INVALID_CODE", "Failed to exchange token")
		return
	}

	client := config.Client(context.Background(), token)
	userInfo, err := fetchUserInfo(client, provider)
	if err != nil {
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

	refreshToken, _, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate refresh token")
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/refresh", // limit cookie to refresh endpoint
	})

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
	token, err := jwt.Parse(refreshTokenStr, func(t *jwt.Token) (any, error) {
		return utils.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_CLAIMS", "Invalid refresh claims")
		return
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_USER", "Invalid user in token")
		return
	}

	var user models.User
	if err := models.GetDB().First(&user, uint(userID)).Error; err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found")
		return
	}

	newAccessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate new access token")
		return
	}

	newRefreshToken, _, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate refresh token")
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/refresh", // limit cookie to refresh endpoint
	})

	c.JSON(http.StatusOK, gin.H{"refresh": "ok"})
}

func (a *AuthController) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   a.environment == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   a.environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/refresh", // limit cookie to refresh endpoint
	})

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
