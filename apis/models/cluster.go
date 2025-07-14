package models

import "gorm.io/gorm"

type Cluster struct {
	gorm.Model
	Name            string        `gorm:"uniqueIndex" json:"name"`
	Description     string        `json:"description"`
	Region          string        `json:"region"`
	CloudProviderID uint          `gorm:"index;not null"`
	CloudProvider   CloudProvider // No need for gorm tag unless custom
	OrganizationID  uint          `gorm:"index;not null"`
	Organization    Organization
}
