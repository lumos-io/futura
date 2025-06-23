package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name           string
	Email          string `gorm:"uniqueIndex"`
	Provider       string // "google" or "github"
	ProviderID     string
	OrganizationID *uint         // Nullable foreign key to the *default* org
	Organization   *Organization // The default org pointer (optional)
}
