package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func GetJWTSecret() []byte {
	return jwtSecret
}

// GenerateSecureRandomString generates a cryptographically secure random string
func GenerateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func VerifyRefreshToken(tokenStr string) (userID uint, jti string, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return 0, "", errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", errors.New("invalid refresh token claims")
	}

	uidFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", errors.New("missing user_id in token")
	}
	jtiClaim, ok := claims["jti"].(string)
	if !ok {
		return 0, "", errors.New("missing jti in token")
	}

	return uint(uidFloat), jtiClaim, nil
}

func GenerateAccessToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"jti":     uuid.NewString(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

func GenerateRefreshToken(userID uint) (string, string, error) {
	jti := uuid.NewString()

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"jti":     jti,
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(GetJWTSecret())
	return signed, jti, err
}
