package collector

import (
	"context"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/events"
	"github.com/opisvigilant/futura/watcher/internal/logger"
)

type Collector struct {
	kubernetesEventsCollector *events.KubernetesEventsCollector
	// kubernete events
	// kubernetes objects
	// kubernetes kubelet
	// kubernetes logs??

}

func New(config *config.Configuration) (*Collector, error) {
	return &Collector{
		kubernetesEventsCollector: events.New(config),
	}, nil
}

func (c *Collector) Start(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Start(ctx); err != nil {
		logger.Logger().Fatal().Err(err).Msg("failed to start the kubernetes events collector...")
		return err
	}

	return nil
}

func (c *Collector) Shutdown(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
