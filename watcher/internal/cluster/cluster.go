package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/collection"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/internal/config"
	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	"github.com/rs/zerolog/log"
)

type KubernetesClusterCollector struct {
	dataCollector    *collection.DataCollector
	resourceWatcher  *resourceWatcher
	k8sLeaderElector *k8s.K8sLeaderElection

	config *config.Configuration
	cancel context.CancelFunc
}

var eventMap map[string]*metadata.KubernetesResourceEvent

func init() {
	eventMap = make(map[string]*metadata.KubernetesResourceEvent, 100_000)
}

func New(config *config.Configuration) (*KubernetesClusterCollector, error) {
	client, err := k8s.MakeClient(k8s.APIConfig{
		AuthType: k8s.AuthType(config.Kubernetes.AuthType),
		Context:  config.Kubernetes.KubeContextName,
	})
	if err != nil {
		return nil, err
	}
	ms := metadata.NewStore()
	return &KubernetesClusterCollector{
		dataCollector:    collection.NewDataCollector(ms),
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
		for e := range kr.Events() {
			eventMap[e.UID] = e
			log.Debug().Interface("resource_event", e).Msg("Received ResourceEvent")
		}
	}()

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
				data := kr.dataCollector.CollectMetricData(time.Now())
				for _, m := range data {
					if m != nil {
						// Decorate with event type if available
						if event, ok := eventMap[m.Uid]; ok {
							m.Type = string(event.Type)
							m.Name = event.Name
							m.Namespace = event.Namespace
							m.Extra = event.Metadata
						}
					}
				}
				// TODO: Send the data here
				// ...
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (kr *KubernetesClusterCollector) Events() <-chan *metadata.KubernetesResourceEvent {
	return kr.resourceWatcher.events
}

func (kr *KubernetesClusterCollector) Start(ctx context.Context) error {
	ctx, kr.cancel = context.WithCancel(ctx)

	log.Logger.Info().Msg("Starting kubernetesClusterReceiver with leader election...")

	if err := kr.k8sLeaderElector.Start(ctx); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to start kubernetesClusterReceiver...")
		return err
	}

	kr.k8sLeaderElector.SetCallBackFuncs(
		func(ctx context.Context) {
			log.Logger.Info().Msg("Starting resource watcher...")
			if err := kr.startReceiver(ctx); err != nil {
				log.Logger.Error().Err(err).Msg("Failed to start receiver...")
			}
		}, func() {
			kr.stopReceiver()
		},
	)
	return nil
}

func (kr *KubernetesClusterCollector) stopReceiver() {
	log.Logger.Info().Msg("Stopping the receiver...")
	if kr.cancel != nil {
		kr.cancel()
	}
}

func (kr *KubernetesClusterCollector) Shutdown(context.Context) error {
	kr.stopReceiver()
	return nil
}
