package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

var (
	googleOAuthConfig *oauth2.Config
	githubOAuthConfig *oauth2.Config

	// FIXME: properly handle the CSFR token
	oauthStateString = "random-state-string" // You should generate a proper CSRF token per session
)

func GoogleLogin(c *gin.Context) {
	googleOAuthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("GOOGLE_CALLBACK_URL"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
	url := googleOAuthConfig.AuthCodeURL(oauthStateString)
	c.Redirect(http.StatusFound, url)
}

func GoogleCallback(c *gin.Context) {
	handleOAuthCallback(c, googleOAuthConfig, "google")
}

func GithubLogin(c *gin.Context) {
	githubOAuthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("GITHUB_CALLBACK_URL"),
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	url := githubOAuthConfig.AuthCodeURL(oauthStateString)
	c.Redirect(http.StatusFound, url)
}

func GithubCallback(c *gin.Context) {
	handleOAuthCallback(c, githubOAuthConfig, "github")
}

func handleOAuthCallback(c *gin.Context, config *oauth2.Config, provider string) {
	state := c.Query("state")
	if state != oauthStateString {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_STATE", "Invalid OAuth state", nil)
		return
	}

	code := c.Query("code")
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "INVALID_CODE", "Failed to exchange token", nil)
		return
	}

	client := config.Client(context.Background(), token)
	userInfo, err := fetchUserInfo(client, provider)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_OPERATION", "Failed to fetch user info", nil)
		return
	}

	var user models.User
	db := models.GetDB()

	// Check if user exists
	if err := db.Where("provider = ? AND provider_id = ?", provider, userInfo.ID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new user
			user = models.User{
				Name:       userInfo.Name,
				Email:      userInfo.Email,
				Provider:   provider,
				ProviderID: userInfo.ID,
				Avatar:     userInfo.Avatar,
			}

			// Create personal/default organization
			personalOrg := models.Organization{
				Name: fmt.Sprintf("%s's Personal Organization", userInfo.Name),
			}

			tx := db.Begin()
			if err := tx.Create(&personalOrg).Error; err != nil {
				tx.Rollback()
				utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_CREATION", "Failed to create personal organization", nil)
				return
			}

			// Set user's default org ID (optional, if you keep this pointer)
			user.OrganizationID = &personalOrg.ID

			// Save user
			if err := tx.Create(&user).Error; err != nil {
				tx.Rollback()
				utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_CREATION", "Failed to create user", nil)
				return
			}

			// Add user to organization members via many2many join
			if err := tx.Model(&personalOrg).Association("Members").Append(&user); err != nil {
				tx.Rollback()
				utils.RespondError(c, http.StatusInternalServerError, "FAILED_ORG_MEMBERSHIP", "Failed to assign user to personal organization", nil)
				return
			}

			tx.Commit()
		} else {
			utils.RespondError(c, http.StatusInternalServerError, "FAILED_USER_QUERY", "Failed to query user", nil)
			return
		}
	}

	accessToken, err := utils.GenerateAccessToken(user)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate access token", nil)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate refresh token", nil)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/refresh", // limit cookie to refresh endpoint
	})

	c.Redirect(http.StatusTemporaryRedirect, os.Getenv("FRONTEND_URL"))
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

func MeHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}
	utils.RespondOK(c, user)
}

func RefreshToken(c *gin.Context) {
	cookie, err := c.Request.Cookie("refresh_token")
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Missing refresh token", nil)
		return
	}

	tokenStr := cookie.Value
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return utils.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token", nil)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_CLAIMS", "Invalid refresh claims", nil)
		return
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_USER", "Invalid user in token", nil)
		return
	}

	var user models.User
	if err := models.GetDB().First(&user, uint(userID)).Error; err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	newAccessToken, err := utils.GenerateAccessToken(user)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate new access token", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
	})
}

func Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/refresh", // limit cookie to refresh endpoint
	})

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
