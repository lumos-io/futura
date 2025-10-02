package models

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInvited  UserStatus = "invited"
	UserStatusInactive UserStatus = "inactive"
)

type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleManager   UserRole = "manager"
	UserRoleDeveloper UserRole = "developer"
	UserRoleViewer    UserRole = "viewer"
)

type User struct {
	BaseModel
	FirstName      string       `json:"firstName"`
	LastName       string       `json:"lastName"`
	Name           string       `json:"name"` // Full name for backward compatibility
	Email          string       `gorm:"uniqueIndex" json:"email"`
	Avatar         string       `json:"avatar"`
	Provider       string       `json:"provider"`   // "google" or "github"
	ProviderID     string       `json:"providerId"` // OAuth provider user ID
	Role           UserRole     `gorm:"default:'developer'" json:"role"`
	Status         UserStatus   `gorm:"default:'invited'" json:"status"`
	LastAccess     *time.Time   `json:"lastAccess"` // Nullable
	OrganizationID uint         `json:"organizationId"`
	Organization   Organization `json:"organization"`
	Teams          []Team       `gorm:"many2many:team_members;" json:"teams"`
}
