package models

import "gorm.io/gorm"

type CloudProvider struct {
	gorm.Model
	Name             CloudProviderName `json:"name"`
	Account          string            `gorm:"uniqueIndex" json:"account"`
	SecretID         string            `json:"secretId"`
	ActivationStatus ActivationStatus  `gorm:"default:PENDING" json:"activationStatus"`
	OrganizationID   uint              `gorm:"index;not null"`
	Organization     Organization
}

type CloudProviderName string

const (
	Alibaba      CloudProviderName = "ALIBABA"
	AWS          CloudProviderName = "AWS"
	Azure        CloudProviderName = "AZURE"
	DigitalOcean CloudProviderName = "DIGITALOCEAN"
	GoogleCloud  CloudProviderName = "GCP"
)

type ActivationStatus string

const (
	ActiveStatus     ActivationStatus = "ACTIVE"
	InProgressStatus ActivationStatus = "IN_PROGRESS"
	PendingStatus    ActivationStatus = "PENDING"
	SuspendedStatus  ActivationStatus = "SUSPENDED"
	FailedStatus     ActivationStatus = "FAILED"
)
