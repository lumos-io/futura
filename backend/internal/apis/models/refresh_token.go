package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// RefreshToken stores refresh tokens with rotation support
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"uniqueIndex;not null;size:64" json:"-"` // SHA-256 hash of token
	JTI       string    `gorm:"uniqueIndex;not null;size:36" json:"jti"`
	UserID    uint      `gorm:"not null;index" json:"userId"`
	User      User      `json:"-"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expiresAt"`
	Revoked   bool      `gorm:"default:false;index" json:"revoked"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// HashToken creates SHA-256 hash of the token for storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
