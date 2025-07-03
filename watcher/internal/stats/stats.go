package stats

import (
	"context"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/sender"
	"github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	"github.com/rs/zerolog/log"
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

	s, err := sender.New(ctx, ksc.config)
	if err != nil {
		return err
	}

	log.Logger.Info().Msg("starting to watch namespaces for the events.")
	// if len(kec.config.Kubernetes.Namespaces) == 0 {
	// 	kec.startWatch(corev1.NamespaceAll, k8sClient, s)
	// } else {
	// 	for _, ns := range kec.config.Kubernetes.Namespaces {
	// 		kec.startWatch(ns, k8sClient, s)
	// 	}
	// }
	return nil
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
