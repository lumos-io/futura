package models

import "time"

// OAuthState stores temporary state tokens for OAuth CSRF protection
type OAuthState struct {
	State     string    `gorm:"primaryKey;size:64" json:"state"`
	Provider  string    `gorm:"not null;size:20" json:"provider"` // "google" or "github"
	ExpiresAt time.Time `gorm:"not null;index" json:"expiresAt"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
