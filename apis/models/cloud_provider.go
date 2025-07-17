package models

import (
	"fmt"

	"github.com/google/uuid"
	pb "github.com/opisvigilant/futura/proto/gen/backend"

	"gorm.io/gorm"
)

type CloudProvider struct {
	gorm.Model
	Provider       CloudProviderName `json:"name"`
	SecretName     pb.SecretName     `json:"secretName"`
	SecretID       uuid.UUID         `json:"secretId"`
	Status         ActivationStatus  `gorm:"default:PENDING" json:"activationStatus"`
	OrganizationID uint              `gorm:"index;not null"`
	Organization   Organization
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

func ConvertToProtoFromActivationStatus(val ActivationStatus) (pb.ActivationStatus, error) {
	switch val {
	case ActiveStatus:
		return pb.ActivationStatus_ACTIVE, nil
	case InProgressStatus:
		return pb.ActivationStatus_IN_PROGRESS, nil
	case PendingStatus:
		return pb.ActivationStatus_PENDING, nil
	case SuspendedStatus:
		return pb.ActivationStatus_SUSPENDED, nil
	case FailedStatus:
		return pb.ActivationStatus_FAILED, nil
	default:
		return -1, fmt.Errorf("invalid ActivationStatus - could not convert to Proto for the given value `%s`", val)
	}
}

func ConvertFromActivationStatusToProto(val pb.ActivationStatus) (ActivationStatus, error) {
	switch val {
	case pb.ActivationStatus_ACTIVE:
		return ActiveStatus, nil
	case pb.ActivationStatus_IN_PROGRESS:
		return InProgressStatus, nil
	case pb.ActivationStatus_PENDING:
		return PendingStatus, nil
	case pb.ActivationStatus_SUSPENDED:
		return SuspendedStatus, nil
	case pb.ActivationStatus_FAILED:
		return FailedStatus, nil
	default:
		return "", fmt.Errorf("invalid ActivationStatus - could not convert to Proto for the given value `%s`", val)
	}
}

func ConvertToProtoFromCloudProvider(val CloudProviderName) (pb.CloudProvider, error) {
	switch val {
	case Alibaba:
		return pb.CloudProvider_ALIBABA, nil
	case AWS:
		return pb.CloudProvider_AWS, nil
	case Azure:
		return pb.CloudProvider_AZURE, nil
	case DigitalOcean:
		return pb.CloudProvider_DIGITALOCEAN, nil
	case GoogleCloud:
		return pb.CloudProvider_GCP, nil
	default:
		return -1, fmt.Errorf("invalid Cloud Provider Name - could not convert to Proto for the given value `%s`", val)
	}
}

func ConvertToCloudProviderFromProto(val pb.CloudProvider) (CloudProviderName, error) {
	switch val {
	case pb.CloudProvider_ALIBABA:
		return Alibaba, nil
	case pb.CloudProvider_AWS:
		return AWS, nil
	case pb.CloudProvider_AZURE:
		return Azure, nil
	case pb.CloudProvider_DIGITALOCEAN:
		return DigitalOcean, nil
	case pb.CloudProvider_GCP:
		return GoogleCloud, nil
	default:
		return "", fmt.Errorf("invalid pb.CloudProvider - could not convert to CloudProviderName for the given value `%s`", val)
	}
}
