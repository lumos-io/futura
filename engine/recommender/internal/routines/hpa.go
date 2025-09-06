package routines

import (
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/go-lib/clickhouse"
)

type HPARecommender struct {
	client *clickhouse.Client
}

func NewHPARecommender(config *config.Configuration) (*HPARecommender, error) {
	client, err := clickhouse.New(config.Clickhouse.Servers, config.Clickhouse.Username,
		config.Clickhouse.Password, config.Clickhouse.Database, true)
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
