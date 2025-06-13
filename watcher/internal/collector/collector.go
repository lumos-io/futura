package collector

// aggregate data from different sources
// 1. k8s
// 2. metrics-server (TODO)

import (
	"context"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/kubernetes"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/sender"
)

type Collector struct {
	ctx context.Context

	stopper  chan struct{} // stop signal for the informers
	doneChan chan struct{} // done signal for kubernetesCollector

	kubernetesCollector *kubernetes.Collector

	// send data to datastore
	sender *sender.Sender
}

func New(cfg *config.Configuration, parentCtx context.Context, sender *sender.Sender) *Collector {
	ctx, cancel := context.WithCancel(parentCtx)
	kubernetesCollector, err := kubernetes.New(cfg, parentCtx)
	if err != nil {
		panic(err)
	}

	collector := &Collector{
		ctx:                 ctx,
		doneChan:            make(chan struct{}),
		kubernetesCollector: kubernetesCollector,
		sender:              sender,
	}

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		defer cancel()
		c.close()
	}(collector)

	return collector
}

func (c *Collector) Run(events chan any) {
	go c.kubernetesCollector.Start(events)

	go c.processk8s(events)

	//TODO: progress metrics-server signal here
	// ...
}

func (c *Collector) processk8s(events <-chan any) {
	for data := range events {
		d := data.(kubernetes.ResourceMessage)
		switch d.ResourceType {
		case kubernetes.POD:
			c.processPod(d)
		case kubernetes.SERVICE:
			c.processSvc(d)
		case kubernetes.REPLICASET:
			c.processReplicaSet(d)
		case kubernetes.DEPLOYMENT:
			c.processDeployment(d)
		case kubernetes.ENDPOINTS:
			c.processEndpoints(d)
		case kubernetes.CONTAINER:
			c.processContainer(d)
		case kubernetes.DAEMONSET:
			c.processDaemonSet(d)
		case kubernetes.STATEFULSET:
			c.processStatefulSet(d)
		default:
			logger.Logger().Warn().Msgf("unknown resource type %s", d.ResourceType)
		}
	}
}

func (c *Collector) Done() <-chan struct{} {
	return c.doneChan
}

func (c *Collector) close() {
	logger.Logger().Info().Msg("Collector closing...")
}
