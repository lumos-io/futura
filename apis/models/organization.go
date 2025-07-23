package models

type Organization struct {
	BaseModel
	Name    string
	Members []User `gorm:"many2many:organization_members;"`
}
