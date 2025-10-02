package models

import "time"

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventLogin            AuditEventType = "login"
	AuditEventLoginFailed      AuditEventType = "login_failed"
	AuditEventLogout           AuditEventType = "logout"
	AuditEventTokenRefresh     AuditEventType = "token_refresh"
	AuditEventTokenRefreshFail AuditEventType = "token_refresh_failed"
	AuditEventTokenRevoked     AuditEventType = "token_revoked"
	AuditEventOAuthStart       AuditEventType = "oauth_start"
	AuditEventOAuthCallback    AuditEventType = "oauth_callback"
	AuditEventOAuthFailed      AuditEventType = "oauth_failed"
)

// AuditLog records authentication and security events
type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EventType  AuditEventType `gorm:"not null;index" json:"eventType"`
	UserID     *uint          `gorm:"index" json:"userId,omitempty"` // Nullable for failed attempts
	Email      string         `gorm:"index" json:"email,omitempty"`
	IPAddress  string         `gorm:"index" json:"ipAddress"`
	UserAgent  string         `json:"userAgent,omitempty"`
	Provider   string         `json:"provider,omitempty"` // OAuth provider
	Success    bool           `gorm:"index" json:"success"`
	ErrorCode  string         `json:"errorCode,omitempty"`
	ErrorMsg   string         `json:"errorMsg,omitempty"`
	Metadata   string         `gorm:"type:jsonb" json:"metadata,omitempty"` // Additional context
	CreatedAt  time.Time      `gorm:"autoCreateTime;index" json:"createdAt"`
}
