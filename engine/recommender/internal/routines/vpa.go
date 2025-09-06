package routines

import (
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/go-lib/clickhouse"
	pbeng "github.com/opisvigilant/futura/proto/gen/engine"
)

type VPARecommender struct {
	client *clickhouse.Client
}

func NewVPARecommender(config *config.Configuration) (*VPARecommender, error) {
	client, err := clickhouse.New(config.Clickhouse.Servers, config.Clickhouse.Username,
		config.Clickhouse.Password, config.Clickhouse.Database, true)
	if err != nil {
		return nil, err
	}
	return &VPARecommender{
		client: client,
	}, nil
}

func (r *VPARecommender) CalculateContainersPatch() ([]*pbeng.ContainerPatch, error) {
	return nil, nil
}
