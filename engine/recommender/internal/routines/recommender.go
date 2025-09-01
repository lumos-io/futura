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
	targetReplicas, err := r.hpaRecommender.CalculateTargetReplicas()
	if err != nil {
		return nil, err
	}
	containersPatch, err := r.vpaRecommender.CalculateContainersPatch()
	if err != nil {
		return nil, err
	}
	return &pbeng.ActionPlan{
		Vertical:       containersPatch,
		TargetReplicas: targetReplicas,
	}, nil
}
