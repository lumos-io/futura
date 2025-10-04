package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := authenticateUser(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func authenticateUser(c *gin.Context) (models.User, error) {
	// Try normal access token first
	claims, err := parseAccessToken(c)
	if err == nil {
		return fetchUserFromClaims(claims)
	}

	// Try refresh flow if access failed
	claims, err = refreshTokensIfValid(c)
	if err != nil {
		return models.User{}, err
	}

	return fetchUserFromClaims(claims)
}

func parseAccessToken(c *gin.Context) (jwt.MapClaims, error) {
	tokenStr, err := c.Cookie("access_token")
	if err != nil || tokenStr == "" {
		return nil, errors.New("missing access token")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return utils.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid access token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid access token claims")
	}

	return claims, nil
}

func refreshTokensIfValid(c *gin.Context) (jwt.MapClaims, error) {
	refreshCookie, err := c.Cookie("refresh_token")
	if err != nil || refreshCookie == "" {
		return nil, errors.New("missing refresh token")
	}

	userID, _, err := utils.VerifyRefreshToken(refreshCookie)
	if err != nil {
		return nil, errors.New("refresh token expired or revoked")
	}

	// Rotate tokens
	accessToken, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return nil, errors.New("error in the access token")
	}
	refreshToken, _, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, errors.New("error in the refresh token")
	}

	setAuthCookies(c, accessToken, refreshToken)

	// Re-parse new access token to get claims
	token, _ := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return utils.GetJWTSecret(), nil
	})
	return token.Claims.(jwt.MapClaims), nil
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func fetchUserFromClaims(claims jwt.MapClaims) (models.User, error) {
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return models.User{}, errors.New("invalid user_id in token")
	}
	userID := uint(userIDFloat)

	db := models.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return models.User{}, errors.New("user not found")
	}

	return user, nil
}
