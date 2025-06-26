package collector

import (
	"context"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/events"
	"github.com/opisvigilant/futura/watcher/internal/logger"
)

type Collector struct {
	// each collector here
	// kubernetes cluster
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
	go func() {
		if err := c.kubernetesEventsCollector.Start(ctx); err != nil {
			logger.Logger().Fatal().Err(err).Msg("failed to start the kubernetes events collector...")
			panic(err)
		}
	}()

	return nil
}

// type Collector struct {
// 	ctx context.Context

// 	stopper  chan struct{} // stop signal for the informers
// 	doneChan chan struct{} // done signal for kubernetesCollector

// 	kubernetesEventsCollector *events.Collector
// 	containerMetricsCollector     *metric.Collector

// 	// send data to datastore
// 	sender *sender.Sender
// }

// func New(cfg *config.Configuration, parentCtx context.Context, sender *sender.Sender) (*Collector, error) {
// 	ctx, cancel := context.WithCancel(parentCtx)

// 	logger.Logger().Info().Msgf("in cluster value: %v", cfg.Kubernetes.InCluster)

// 	k8sClient, err := k8s.New(cfg.Kubernetes.InCluster)
// 	if err != nil {
// 		defer cancel()
// 		return nil, err
// 	}

// 	kubernetesEventsCollector, err := events.New(k8sClient, cfg, parentCtx)
// 	if err != nil {
// 		defer cancel()
// 		return nil, err
// 	}

// 	metricCollector, err := metric.New(k8sClient, cfg, parentCtx)
// 	if err != nil {
// 		defer cancel()
// 		return nil, err
// 	}

// 	collector := &Collector{
// 		ctx:                 ctx,
// 		doneChan:            make(chan struct{}),
// 		kubernetesEventsCollector: kubernetesEventsCollector,
// 		containerMetricsCollector:     metricCollector,
// 		sender:              sender,
// 	}

// 	go func(c *Collector) {
// 		<-c.ctx.Done() // wait for context to be cancelled
// 		defer cancel()
// 		c.close()
// 	}(collector)

// 	return collector, nil
// }

// func (c *Collector) Run(events chan any) {
// 	go c.kubernetesEventsCollector.Start(events)
// 	go c.containerMetricsCollector.Start(60*time.Second, []string{})

// 	go c.processk8s(events)
// }

// func (c *Collector) processk8s(evs <-chan any) {
// 	for data := range evs {
// 		d := data.(events.ResourceMessage)
// 		switch d.ResourceType {
// 		case events.POD:
// 			c.processPod(d)
// 		case events.SERVICE:
// 			c.processSvc(d)
// 		case events.REPLICASET:
// 			c.processReplicaSet(d)
// 		case events.DEPLOYMENT:
// 			c.processDeployment(d)
// 		case events.ENDPOINTS:
// 			c.processEndpoints(d)
// 		case events.CONTAINER:
// 			c.processContainer(d)
// 		case events.DAEMONSET:
// 			c.processDaemonSet(d)
// 		case events.STATEFULSET:
// 			c.processStatefulSet(d)
// 		default:
// 			logger.Logger().Warn().Msgf("unknown resource type %s", d.ResourceType)
// 		}
// 	}
// }

// func (c *Collector) Done() <-chan struct{} {
// 	return c.doneChan
// }

// func (c *Collector) close() {
// 	logger.Logger().Info().Msg("Collector closing...")
// }
