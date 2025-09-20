package routines

import (
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	pbeng "github.com/opisvigilant/futura/proto/gen/engine"
)

type MultiDimensionRecommender struct {
	hpaRecommender *HPARecommender
	vpaRecommender *VPARecommender
}

func NewMultiDimensionRecommender(config *config.Configuration) (*MultiDimensionRecommender, error) {
	hpaRec, err := NewHPARecommender(config)
	if err != nil {
		return nil, err
	}
	vpaRec, err := NewVPARecommender(config)
	if err != nil {
		return nil, err
	}
	return &MultiDimensionRecommender{
		hpaRecommender: hpaRec,
		vpaRecommender: vpaRec,
	}, nil
}

func (r *MultiDimensionRecommender) GetActionPlan(app *pbeng.AppRef) (*pbeng.ActionPlan, error) {
	hpaResult, err := r.hpaRecommender.CalculateTargetReplicas(&HPAInput{
		OrganizationID: 1,
		ClusterID:      app.ClusterId,
		Namespace:      app.Namespace,
		DeploymentName: app.AppName,
		TargetCPUUtil:  90.0,
	})
	if err != nil {
		return nil, err
	}
	containersPatch, err := r.vpaRecommender.CalculateContainersPatch(&VPAInput{
		OrganizationID: 1,
		ClusterID:      app.ClusterId,
		Namespace:      app.Namespace,
		DeploymentName: app.AppName,
		MinCPUNano:     1,
		MinMemoryByte:  1,
	})
	if err != nil {
		return nil, err
	}

	vertical := make([]*pbeng.ContainerPatch, len(containersPatch.Containers))
	for i, c := range containersPatch.Containers {
		vertical[i] = &pbeng.ContainerPatch{
			ContainerName:             c.ContainerName,
			Cpu_95ThNano:              c.CPU95thNano,
			Memory_95ThBytes:          c.Memory95thBytes,
			RecommendedCpuNano:        c.RecommendedCPUNano,
			RecommendedMemoryBytes:    c.RecommendedMemoryB,
			CurrentCpuRequestNano:     c.CurrentCPURequestN,
			CurrentMemoryRequestBytes: c.CurrentMemoryReqB,
			RecommendationNotes:       c.RecommendationNotes,
		}
	}

	return &pbeng.ActionPlan{
		Vertical:       vertical,
		TargetReplicas: int32(hpaResult.DesiredReplicas),
	}, nil
}
