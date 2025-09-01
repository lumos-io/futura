package routines

import (
	"github.com/opisvigilant/futura/engine/recommender/internal/clickhouse"
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
)

type HPARecommender struct {
	client *clickhouse.Client
}

func NewHPARecommender(config *config.Configuration) (*HPARecommender, error) {
	client, err := clickhouse.New(config.Clickhouse)
	if err != nil {
		return nil, err
	}
	return &HPARecommender{
		client: client,
	}, nil
}

func (r *HPARecommender) CalculateTargetReplicas() (int32, error) {
	return 1, nil
}
