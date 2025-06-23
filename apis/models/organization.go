package models

import "gorm.io/gorm"

type Organization struct {
	gorm.Model
	Name    string
	Members []User `gorm:"many2many:organization_members;"`
}
