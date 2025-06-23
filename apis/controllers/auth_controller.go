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
	err = db.Where("provider = ? AND provider_id = ?", provider, userInfo.ID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new user
			user = models.User{
				Name:       userInfo.Name,
				Email:      userInfo.Email,
				Provider:   provider,
				ProviderID: userInfo.ID,
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

	jwtToken, err := utils.GenerateJWT(user)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_OAUTH_OPERATION", "Failed to generate token", nil)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    jwtToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	c.Redirect(http.StatusTemporaryRedirect, os.Getenv("FRONTEND_URL"))
}

// Simplified user info response struct
type OAuthUserInfo struct {
	ID    string
	Email string
	Name  string
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
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return userInfo, err
		}
		userInfo = OAuthUserInfo{ID: body.ID, Email: body.Email, Name: body.Name}
	} else if provider == "github" {
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			return userInfo, err
		}
		defer resp.Body.Close()
		var body struct {
			ID    int    `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
			Login string `json:"login"`
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
			ID:    fmt.Sprintf("%d", body.ID),
			Email: email,
			Name:  body.Name,
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
