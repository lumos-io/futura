package utils

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/rs/zerolog/log"
)

// LogAuditEvent creates an audit log entry
func LogAuditEvent(c *gin.Context, eventType models.AuditEventType, userID *uint, email, provider string, success bool, errorCode, errorMsg string, metadata map[string]interface{}) {
	db := models.GetDB()

	// Serialize metadata to JSON
	var metadataJSON string
	if metadata != nil && len(metadata) > 0 {
		if bytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(bytes)
		} else {
			metadataJSON = "{}"
		}
	} else {
		metadataJSON = "{}"
	}

	auditLog := models.AuditLog{
		EventType: eventType,
		UserID:    userID,
		Email:     email,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Provider:  provider,
		Success:   success,
		ErrorCode: errorCode,
		ErrorMsg:  errorMsg,
		Metadata:  metadataJSON,
	}

	if err := db.Create(&auditLog).Error; err != nil {
		log.Error().Err(err).Str("event_type", string(eventType)).Msg("Failed to create audit log")
	}

	// Also log to structured logger
	logEvent := log.Info()
	if !success {
		logEvent = log.Warn()
	}

	logEvent.
		Str("event_type", string(eventType)).
		Str("ip", c.ClientIP()).
		Str("email", email).
		Bool("success", success).
		Str("error_code", errorCode).
		Msg("Audit event")
}

// LogSuccessfulLogin logs a successful login event
func LogSuccessfulLogin(c *gin.Context, userID uint, email, provider string) {
	LogAuditEvent(c, models.AuditEventLogin, &userID, email, provider, true, "", "", nil)
}

// LogFailedLogin logs a failed login attempt
func LogFailedLogin(c *gin.Context, email, provider, errorCode, errorMsg string) {
	LogAuditEvent(c, models.AuditEventLoginFailed, nil, email, provider, false, errorCode, errorMsg, nil)
}

// LogLogout logs a logout event
func LogLogout(c *gin.Context, userID uint, email string) {
	LogAuditEvent(c, models.AuditEventLogout, &userID, email, "", true, "", "", nil)
}

// LogTokenRefresh logs a token refresh event
func LogTokenRefresh(c *gin.Context, userID uint, email string, success bool, errorCode, errorMsg string) {
	eventType := models.AuditEventTokenRefresh
	if !success {
		eventType = models.AuditEventTokenRefreshFail
	}
	LogAuditEvent(c, eventType, &userID, email, "", success, errorCode, errorMsg, nil)
}

// LogTokenRevoked logs when tokens are revoked
func LogTokenRevoked(c *gin.Context, userID uint, email string, reason string) {
	metadata := map[string]interface{}{
		"reason": reason,
	}
	LogAuditEvent(c, models.AuditEventTokenRevoked, &userID, email, "", true, "", "", metadata)
}

// LogOAuthStart logs the start of OAuth flow
func LogOAuthStart(c *gin.Context, provider string) {
	LogAuditEvent(c, models.AuditEventOAuthStart, nil, "", provider, true, "", "", nil)
}

// LogOAuthCallback logs OAuth callback events
func LogOAuthCallback(c *gin.Context, provider string, success bool, userID *uint, email, errorCode, errorMsg string) {
	eventType := models.AuditEventOAuthCallback
	if !success {
		eventType = models.AuditEventOAuthFailed
	}
	LogAuditEvent(c, eventType, userID, email, provider, success, errorCode, errorMsg, nil)
}
