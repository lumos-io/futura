package models

// Team represents a team within an organization
type Team struct {
	BaseModel
	Name           string `gorm:"not null" json:"name"`
	Description    string `json:"description"`
	OrganizationID uint   `gorm:"not null;index" json:"organizationId"`
	Organization   Organization
	Members        []User `gorm:"many2many:team_members;" json:"members"`
}

// TeamMember is the join table for Team and User many-to-many relationship
type TeamMember struct {
	TeamID uint `gorm:"primaryKey"`
	UserID uint `gorm:"primaryKey"`
	Team   Team
	User   User
}
