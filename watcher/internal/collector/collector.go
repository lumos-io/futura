package collector

// aggregate data from different sources
// 1. k8s
// 2. metrics-server (TODO)

import (
	"context"

	"github.com/opisvigilant/futura/watcher/internal/handlers"
	"github.com/opisvigilant/futura/watcher/internal/logger"
)

type Collector struct {
	ctx context.Context

	stopper  chan struct{} // stop signal for the informers
	doneChan chan struct{} // done signal for k8sCollector

	// send data to datastore
	eventsHandler handlers.Handler
}

func NewCollector(parentCtx context.Context, eventHandler handlers.Handler) *Collector {
	ctx, _ := context.WithCancel(parentCtx)

	collector := &Collector{
		ctx:           ctx,
		doneChan:      make(chan struct{}),
		eventsHandler: eventHandler,
	}

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		c.close()
	}(collector)

	return collector
}

func (c *Collector) Run(k8sChan <-chan any) {
	go c.processk8s(k8sChan)

	//TODO: progress metrics-server signal here
	// ...
}

func (c *Collector) processk8s(k8sChan <-chan any) {
	c.eventsHandler.HandleKubernetesEvent(k8sChan)
}

func (c *Collector) Done() <-chan struct{} {
	return c.doneChan
}

func (c *Collector) close() {
	logger.Logger().Info().Msg("Collector closing...")
}
