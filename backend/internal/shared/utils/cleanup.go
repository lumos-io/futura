package utils

import (
	"time"

	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/rs/zerolog/log"
)

// CleanupExpiredOAuthStates removes OAuth state tokens older than their expiration
func CleanupExpiredOAuthStates() error {
	db := models.GetDB()
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.OAuthState{})
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("Failed to cleanup expired OAuth states")
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up expired OAuth states")
	}
	return nil
}

// CleanupExpiredRefreshTokens removes refresh tokens that are expired
func CleanupExpiredRefreshTokens() error {
	db := models.GetDB()
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("Failed to cleanup expired refresh tokens")
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up expired refresh tokens")
	}
	return nil
}

// CleanupRevokedRefreshTokens removes revoked tokens older than 30 days (for audit trail)
func CleanupRevokedRefreshTokens() error {
	db := models.GetDB()
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	result := db.Where("revoked = ? AND revoked_at < ?", true, thirtyDaysAgo).Delete(&models.RefreshToken{})
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("Failed to cleanup old revoked tokens")
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up old revoked refresh tokens")
	}
	return nil
}

// RunSecurityCleanup runs all security-related cleanup tasks
func RunSecurityCleanup() {
	log.Info().Msg("Starting security cleanup tasks")

	if err := CleanupExpiredOAuthStates(); err != nil {
		log.Error().Err(err).Msg("OAuth state cleanup failed")
	}

	if err := CleanupExpiredRefreshTokens(); err != nil {
		log.Error().Err(err).Msg("Refresh token cleanup failed")
	}

	if err := CleanupRevokedRefreshTokens(); err != nil {
		log.Error().Err(err).Msg("Revoked token cleanup failed")
	}

	log.Info().Msg("Security cleanup tasks completed")
}
