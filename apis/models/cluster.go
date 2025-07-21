package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClusterMetadata struct {
	gorm.Model
	APIKey          string        `gorm:"size:512" json:"-"` // don't expose in API responses by default
	CloudProviderID uint          `gorm:"index;not null"`
	CloudProvider   CloudProvider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	OrganizationID uint         `gorm:"index;not null"`
	Organization   Organization `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	EKSMetadata  *EKSClusterMetadata  `gorm:"foreignKey:ClusterMetadataID"`
	AKSMetadata  *AKSClusterMetadata  `gorm:"foreignKey:ClusterMetadataID"`
	GKEMetadata  *GKEClusterMetadata  `gorm:"foreignKey:ClusterMetadataID"`
	DOKSMetadata *DOKSClusterMetadata `gorm:"foreignKey:ClusterMetadataID"`
	ACKMetadata  *ACKClusterMetadata  `gorm:"foreignKey:ClusterMetadataID"`

	// KindMetadata is only used for testing - do not use it in production
	KindMetadata *KindClusterMetadata `gorm:"foreignKey:ClusterMetadataID"`
}

type EKSClusterMetadata struct {
	gorm.Model
	// Raw AWS values
	EKSClusterName   string            `gorm:"size:255;not null" json:"name"`
	Status           string            `gorm:"size:64" json:"status"`
	Version          string            `gorm:"size:32" json:"version"`
	Endpoint         string            `gorm:"size:512" json:"endpoint"`
	Arn              string            `gorm:"size:512;uniqueIndex" json:"arn"`
	EKSClusterID     string            `gorm:"size:512" json:"eksClusterId"`
	ClusterCreatedAt *time.Time        `json:"clusterCreatedAt"`
	PlatformVersion  string            `gorm:"size:64" json:"platformVersion"`
	Tags             datatypes.JSONMap `json:"tags"`

	// FK back to generic cluster record
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE"`
}

type GKEClusterMetadata struct {
	gorm.Model
	GKEClusterName string
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null"`
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE"`
}

type AKSClusterMetadata struct {
	gorm.Model
	AKSClusterName string
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null"`
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE"`
}

type DOKSClusterMetadata struct {
	gorm.Model
	AKSClusterName string
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null"`
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE"`
}

type ACKClusterMetadata struct {
	gorm.Model
	AKSClusterName string
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null"`
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE"`
}

/*
Attention: this below is only used in developemnt mode and behind a feature-flag
*/
type KindClusterMetadata struct {
	gorm.Model
	KindClusterName   string
	Status            string            `gorm:"size:64" json:"status"`
	Version           string            `gorm:"size:32" json:"version"`
	Endpoint          string            `gorm:"size:512" json:"endpoint"`
	EKSClusterID      string            `gorm:"size:512" json:"eksClusterId"`
	ClusterCreatedAt  *time.Time        `json:"clusterCreatedAt"`
	PlatformVersion   string            `gorm:"size:64" json:"platformVersion"`
	Tags              datatypes.JSONMap `json:"tags"`
	ClusterMetadataID uint              `gorm:"uniqueIndex;not null"`
	ClusterMetadata   *ClusterMetadata  `gorm:"constraint:OnDelete:CASCADE"`
}
