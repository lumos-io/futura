package collector

// aggregate data from different sources
// 1. k8s
// 2. metrics-server (TODO)

import (
	"context"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/kubernetes"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/metric"
	"github.com/opisvigilant/futura/watcher/internal/sender"
	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"
)

type Collector struct {
	ctx context.Context

	stopper  chan struct{} // stop signal for the informers
	doneChan chan struct{} // done signal for kubernetesCollector

	kubernetesCollector *kubernetes.Collector
	metricCollector     *metric.Collector

	// send data to datastore
	sender *sender.Sender
}

func New(cfg *config.Configuration, parentCtx context.Context, sender *sender.Sender) (*Collector, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	logger.Logger().Info().Msgf("in cluster value: %v", cfg.Kubernetes.InCluster)

	k8sClient, err := k8s.New(cfg.Kubernetes.InCluster)
	if err != nil {
		defer cancel()
		return nil, err
	}

	kubernetesCollector, err := kubernetes.New(k8sClient, cfg, parentCtx)
	if err != nil {
		defer cancel()
		return nil, err
	}

	metricCollector, err := metric.New(k8sClient, cfg, parentCtx)
	if err != nil {
		defer cancel()
		return nil, err
	}

	collector := &Collector{
		ctx:                 ctx,
		doneChan:            make(chan struct{}),
		kubernetesCollector: kubernetesCollector,
		metricCollector:     metricCollector,
		sender:              sender,
	}

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		defer cancel()
		c.close()
	}(collector)

	return collector, nil
}

func (c *Collector) Run(events chan any) {
	go c.kubernetesCollector.Start(events)
	go c.metricCollector.Start(60*time.Second, []string{})

	go c.processk8s(events)
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
