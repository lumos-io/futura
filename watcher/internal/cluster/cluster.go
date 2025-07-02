package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/internal/config"
	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

type KubernetesClusterCollector struct {
	resourceWatcher  *resourceWatcher
	k8sLeaderElector *k8s.K8sLeaderElection

	config *config.Configuration
	cancel context.CancelFunc
}

func New(config *config.Configuration, client kubernetes.Interface) (*KubernetesClusterCollector, error) {
	ms := metadata.NewStore()
	return &KubernetesClusterCollector{
		resourceWatcher:  newResourceWatcher(config, ms),
		k8sLeaderElector: k8s.NewK8sLeaderElection(config, client, config.Kubernetes.LeaseName),
		config:           config,
	}, nil
}

func (kr *KubernetesClusterCollector) startReceiver(ctx context.Context) error {
	if err := kr.resourceWatcher.initialize(); err != nil {
		return err
	}

	go func() {
		log.Logger.Info().Msg("Starting shared informers and wait for initial cache sync.")
		for _, informer := range kr.resourceWatcher.informerFactories {
			if informer == nil {
				continue
			}
			timedContextForInitialSync := kr.resourceWatcher.startWatchingResources(ctx, informer)

			// Wait till either the initial cache sync times out or until the cancel method
			// corresponding to this context is called.
			<-timedContextForInitialSync.Done()

			// If the context times out, set initialSyncTimedOut and report a fatal error. Currently
			// this timeout is 10 minutes, which appears to be long enough.
			if errors.Is(timedContextForInitialSync.Err(), context.DeadlineExceeded) {
				kr.resourceWatcher.initialSyncTimedOut.Store(true)
				log.Logger.Error().Msg("Timed out waiting for initial cache sync.")
				return
			}
		}

		log.Logger.Info().Msg("Completed syncing shared informer caches.")
		kr.resourceWatcher.initialSyncDone.Store(true)

		ticker := time.NewTicker(kr.config.Kubernetes.CollectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// TODO: Read the data here
				// ....
				log.Logger.Info().Msgf("%v", "bananaassssssss")

				// TODO: Send the data here
				// ....
				log.Logger.Info().Msgf("%v", "sendersssssssss")
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (kr *KubernetesClusterCollector) Start(ctx context.Context) error {
	ctx, kr.cancel = context.WithCancel(ctx)

	log.Logger.Info().Msg("Starting k8sClusterReceiver with leader election")
	kr.k8sLeaderElector.SetCallBackFuncs(
		func(ctx context.Context) {
			if err := kr.startReceiver(ctx); err != nil {
				log.Logger.Error().Err(err).Msg("Failed to start receiver")
			}
		}, func() {
			kr.stopReceiver()
		},
	)

	return nil
}

func (kr *KubernetesClusterCollector) stopReceiver() {
	log.Logger.Info().Msg("Stopping the receiver")
	if kr.cancel != nil {
		kr.cancel()
	}
}

func (kr *KubernetesClusterCollector) Shutdown(context.Context) error {
	kr.stopReceiver()
	return nil
}
