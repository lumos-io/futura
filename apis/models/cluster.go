package models

import (
	"fmt"
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/backend"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gorm.io/datatypes"
)

type ClusterMetadata struct {
	BaseModel

	Name                 string             `gorm:"size:255;not null" json:"name"`
	APIKey               string             `gorm:"size:255;not null" json:"api_key"`
	ProviderConnectionID uint               `gorm:"index;not null" json:"providerConnectionId"`
	ProviderConnection   ProviderConnection `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	OrganizationID uint         `gorm:"index;not null" json:"organizationId"`
	Organization   Organization `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	EKSMetadata  *EKSClusterMetadata  `gorm:"foreignKey:ClusterMetadataID" json:"eksMetadata,omitempty"`
	AKSMetadata  *AKSClusterMetadata  `gorm:"foreignKey:ClusterMetadataID" json:"aksMetadata,omitempty"`
	GKEMetadata  *GKEClusterMetadata  `gorm:"foreignKey:ClusterMetadataID" json:"gkeMetadata,omitempty"`
	DOKSMetadata *DOKSClusterMetadata `gorm:"foreignKey:ClusterMetadataID" json:"doksMetadata,omitempty"`
	ACKMetadata  *ACKClusterMetadata  `gorm:"foreignKey:ClusterMetadataID" json:"ackMetadata,omitempty"`

	// KindMetadata is only used for testing - do not use it in production
	KindMetadata *KindClusterMetadata `gorm:"foreignKey:ClusterMetadataID" json:"kindMetadata,omitempty"`
}

func (cm *ClusterMetadata) GetKubernetesVersion() string {
	if cm.ACKMetadata != nil {
		return ""
	}

	return ""
}

func (cm *ClusterMetadata) GetRegion() string {
	return ""
}

type EKSClusterMetadata struct {
	BaseModel

	Region           string            `gorm:"size:32" json:"region"`
	Status           string            `gorm:"size:64" json:"status"`
	Version          string            `gorm:"size:32" json:"version"`
	Endpoint         string            `gorm:"size:512" json:"endpoint"`
	Arn              string            `gorm:"size:512;uniqueIndex" json:"arn"`
	EKSClusterID     string            `gorm:"size:512" json:"eksClusterId"`
	ClusterCreatedAt *time.Time        `json:"clusterCreatedAt"`
	PlatformVersion  string            `gorm:"size:64" json:"platformVersion"`
	Tags             datatypes.JSONMap `json:"tags"`

	// FK back to generic cluster record
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type GKEClusterMetadata struct {
	BaseModel

	Version string `gorm:"size:32" json:"version"`
	Region  string `gorm:"size:32" json:"region"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type AKSClusterMetadata struct {
	BaseModel

	Version string `gorm:"size:32" json:"version"`
	Region  string `gorm:"size:32" json:"region"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type DOKSClusterMetadata struct {
	BaseModel

	Version string `gorm:"size:32" json:"version"`
	Region  string `gorm:"size:32" json:"region"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type ACKClusterMetadata struct {
	BaseModel

	Version string `gorm:"size:32" json:"version"`
	Region  string `gorm:"size:32" json:"region"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

/*
Attention: this below is only used in development mode and behind a feature-flag
*/
type KindClusterMetadata struct {
	BaseModel

	Region            string            `gorm:"size:32" json:"region"`
	Status            string            `gorm:"size:64" json:"status"`
	Version           string            `gorm:"size:32" json:"version"`
	Endpoint          string            `gorm:"size:512" json:"endpoint"`
	ClusterCreatedAt  time.Time         `json:"clusterCreatedAt"`
	PlatformVersion   string            `gorm:"size:64" json:"platformVersion"`
	Tags              datatypes.JSONMap `json:"tags"`
	ClusterMetadataID uint              `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func ConvertToProtoClusterMetadataList(models []ClusterMetadata) []*pb.ClusterMetadata {
	var result []*pb.ClusterMetadata

	for _, model := range models {
		proto := &pb.ClusterMetadata{
			Id:              uint64(model.ID),
			Name:            model.Name,
			CloudProviderId: uint64(model.ProviderConnectionID),
			OrganizationId:  uint64(model.OrganizationID),
		}

		if model.EKSMetadata != nil {
			proto.EksMetadata = &pb.EKSClusterMetadata{
				Id:              uint64(model.EKSMetadata.ID),
				Region:          model.EKSMetadata.Region,
				Version:         model.EKSMetadata.Version,
				Status:          model.EKSMetadata.Status,
				Endpoint:        model.EKSMetadata.Endpoint,
				Arn:             model.EKSMetadata.Arn,
				EksClusterId:    model.EKSMetadata.EKSClusterID,
				PlatformVersion: model.EKSMetadata.PlatformVersion,
			}

			if model.EKSMetadata.ClusterCreatedAt != nil {
				proto.EksMetadata.ClusterCreatedAt = timestamppb.New(*model.EKSMetadata.ClusterCreatedAt)
			}

			if model.EKSMetadata.Tags != nil {
				tags := make(map[string]string)
				for k, v := range model.EKSMetadata.Tags {
					tags[k] = fmt.Sprintf("%v", v)
				}
				proto.EksMetadata.Tags = tags
			}
		}

		if model.GKEMetadata != nil {
			proto.GkeMetadata = &pb.GKEClusterMetadata{
				Id:      uint64(model.GKEMetadata.ID),
				Region:  model.GKEMetadata.Region,
				Version: model.GKEMetadata.Version,
			}
		}

		if model.AKSMetadata != nil {
			proto.AksMetadata = &pb.AKSClusterMetadata{
				Id:      uint64(model.AKSMetadata.ID),
				Region:  model.AKSMetadata.Region,
				Version: model.AKSMetadata.Version,
			}
		}

		if model.DOKSMetadata != nil {
			proto.DoksMetadata = &pb.DOKSClusterMetadata{
				Id:      uint64(model.DOKSMetadata.ID),
				Region:  model.DOKSMetadata.Region,
				Version: model.DOKSMetadata.Version,
			}
		}

		if model.ACKMetadata != nil {
			proto.AckMetadata = &pb.ACKClusterMetadata{
				Id:      uint64(model.ACKMetadata.ID),
				Region:  model.ACKMetadata.Region,
				Version: model.ACKMetadata.Version,
			}
		}

		if model.KindMetadata != nil {
			proto.KindMetadata = &pb.KindClusterMetadata{
				Id:              uint64(model.KindMetadata.ID),
				Status:          model.KindMetadata.Status,
				Region:          model.KindMetadata.Region,
				Version:         model.KindMetadata.Version,
				Endpoint:        model.KindMetadata.Endpoint,
				PlatformVersion: model.KindMetadata.PlatformVersion,
			}

			proto.KindMetadata.ClusterCreatedAt = timestamppb.New(model.KindMetadata.ClusterCreatedAt)

			if model.KindMetadata.Tags != nil {
				tags := make(map[string]string)
				for k, v := range model.KindMetadata.Tags {
					tags[k] = fmt.Sprintf("%v", v)
				}
				proto.KindMetadata.Tags = tags
			}
		}

		result = append(result, proto)
	}

	return result
}
