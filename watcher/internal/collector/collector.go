package collector

import (
	"context"

	"github.com/opisvigilant/futura/watcher/internal/cluster"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/events"
	"github.com/opisvigilant/futura/watcher/internal/stats"

	"github.com/rs/zerolog/log"
)

type Collector struct {
	kubernetesEventsCollector  *events.KubernetesEventsCollector
	kubernetesClusterCollector *cluster.KubernetesClusterCollector
	kuberentesStatsCollector   *stats.KuberentesStatsCollector
}

func New(config *config.Configuration) (*Collector, error) {
	kcc, err := cluster.New(config)
	if err != nil {
		return nil, err
	}
	return &Collector{
		kubernetesEventsCollector:  events.New(config),
		kubernetesClusterCollector: kcc,
		kuberentesStatsCollector:   stats.New(config),
	}, nil
}

func (c *Collector) Start(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes events collector...")
		return err
	}

	if err := c.kubernetesClusterCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes cluster collector...")
		return err
	}

	if err := c.kuberentesStatsCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes stats collector...")
		return err
	}

	return nil
}

func (c *Collector) Shutdown(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Shutdown(ctx); err != nil {
		return err
	}
	if err := c.kubernetesClusterCollector.Shutdown(ctx); err != nil {
		return err
	}
	if err := c.kuberentesStatsCollector.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
