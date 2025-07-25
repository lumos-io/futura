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

	CloudProviderID uint          `gorm:"index;not null" json:"cloudProviderId"`
	CloudProvider   CloudProvider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

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

type EKSClusterMetadata struct {
	BaseModel
	// Raw AWS values
	Name             string            `gorm:"size:255;not null" json:"name"`
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
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type GKEClusterMetadata struct {
	BaseModel
	Name string `gorm:"size:255;not null" json:"name"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type AKSClusterMetadata struct {
	BaseModel
	Name string `gorm:"size:255;not null" json:"name"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type DOKSClusterMetadata struct {
	BaseModel
	Name string `gorm:"size:255;not null" json:"name"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type ACKClusterMetadata struct {
	BaseModel
	Name string `gorm:"size:255;not null" json:"name"`
	// ... other fields ...
	ClusterMetadataID uint             `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

/*
Attention: this below is only used in development mode and behind a feature-flag
*/
type KindClusterMetadata struct {
	BaseModel
	Name              string            `gorm:"size:255;not null" json:"name"`
	Status            string            `gorm:"size:64" json:"status"`
	Version           string            `gorm:"size:32" json:"version"`
	Endpoint          string            `gorm:"size:512" json:"endpoint"`
	ClusterCreatedAt  time.Time         `json:"clusterCreatedAt"`
	PlatformVersion   string            `gorm:"size:64" json:"platformVersion"`
	Tags              datatypes.JSONMap `json:"tags"`
	ClusterMetadataID uint              `gorm:"uniqueIndex;not null" json:"clusterMetadataId"` // one-to-one enforced
	ClusterMetadata   *ClusterMetadata  `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

func ConvertToProtoClusterMetadataList(models []ClusterMetadata) []*pb.ClusterMetadata {
	var result []*pb.ClusterMetadata

	for _, model := range models {
		proto := &pb.ClusterMetadata{
			Id:              uint64(model.ID),
			CloudProviderId: uint64(model.CloudProviderID),
			OrganizationId:  uint64(model.OrganizationID),
		}

		if model.EKSMetadata != nil {
			proto.EksMetadata = &pb.EKSClusterMetadata{
				Id:              uint64(model.EKSMetadata.ID),
				Name:            model.EKSMetadata.Name,
				Status:          model.EKSMetadata.Status,
				Version:         model.EKSMetadata.Version,
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
				Id:   uint64(model.GKEMetadata.ID),
				Name: model.GKEMetadata.Name,
			}
		}

		if model.AKSMetadata != nil {
			proto.AksMetadata = &pb.AKSClusterMetadata{
				Id:   uint64(model.AKSMetadata.ID),
				Name: model.AKSMetadata.Name,
			}
		}

		if model.DOKSMetadata != nil {
			proto.DoksMetadata = &pb.DOKSClusterMetadata{
				Id:   uint64(model.DOKSMetadata.ID),
				Name: model.DOKSMetadata.Name,
			}
		}

		if model.ACKMetadata != nil {
			proto.AckMetadata = &pb.ACKClusterMetadata{
				Id:   uint64(model.ACKMetadata.ID),
				Name: model.ACKMetadata.Name,
			}
		}

		if model.KindMetadata != nil {
			proto.KindMetadata = &pb.KindClusterMetadata{
				Id:              uint64(model.KindMetadata.ID),
				Name:            model.KindMetadata.Name,
				Status:          model.KindMetadata.Status,
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
