package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name           string       `json:"name"`
	Email          string       `gorm:"uniqueIndex" json:"email"`
	Avatar         string       `json:"avatar"`
	Provider       string       `json:"provider"` // "google" or "github"
	ProviderID     string       `json:"providerId"`
	OrganizationID uint         `json:"organizationId"` // Nullable foreign key to the *default* org
	Organization   Organization `json:"organization"`   // The default org pointer (optional)
}
