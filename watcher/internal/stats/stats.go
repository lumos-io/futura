package stats

import (
	"context"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/sender"
	"github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	"github.com/rs/zerolog/log"

	k8s "k8s.io/client-go/kubernetes"
)

type KuberentesStatsCollector struct {
	config          *config.Configuration
	stopperChanList []chan struct{}
	startTime       time.Time
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(config *config.Configuration) *KuberentesStatsCollector {
	return &KuberentesStatsCollector{
		startTime: time.Now(),
		config:    config,
	}
}

func (ksc *KuberentesStatsCollector) Start(ctx context.Context) error {
	ksc.ctx, ksc.cancel = context.WithCancel(ctx)

	k8sClient, err := kubernetes.MakeClient(kubernetes.APIConfig{
		AuthType: kubernetes.AuthType(ksc.config.Kubernetes.Auth.AuthType),
		Context:  ksc.config.Kubernetes.Auth.KubeContextName,
	})
	if err != nil {
		return err
	}

	if err := ksc.startScrape(k8sClient, ksc.config.Kubernetes.StatsCollectionInterval); err != nil {
		return err
	}

	return nil
}

// scrapeKubeletStats scrapes kubelet /stats/summary periodically until the context is cancelled.
func (ksc *KuberentesStatsCollector) startScrape(client k8s.Interface, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sender, err := sender.New(ksc.ctx, ksc.config)
	if err != nil {
		return err
	}

	ks, err := NewKubeletScraper(ksc.config, client)
	if err != nil {
		return err
	}

	if err := ks.Init(); err != nil {
		return err
	}

	for {
		select {
		case <-ksc.ctx.Done():
			log.Logger.Info().Msg("Shutting down kubelet scraper...")
			return ks.Shutdown()
		case <-ticker.C:
			log.Logger.Info().Msg("Scraping kubelet stats...")

			data, err := ks.DoScrape()
			if err != nil {
				return err
			}
			sender.KubernetesKubeletMetrics <- data
		}
	}
}

func (ksc *KuberentesStatsCollector) Shutdown(context.Context) error {
	if ksc.cancel == nil {
		return nil
	}
	// Stop watching all the namespaces by closing all the stopper channels.
	for _, stopperChan := range ksc.stopperChanList {
		close(stopperChan)
	}
	ksc.cancel()
	return nil
}
