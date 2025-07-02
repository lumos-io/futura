package kubernetes

import (
	"context"
	"sync"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type (
	StartCallback = func(context.Context)
	StopCallback  = func()
)

// SetCallBackFuncs set the functions that can be invoked when the leader wins or loss the election
func (lee *K8sLeaderElection) SetCallBackFuncs(onStartLeading StartCallback, onStopLeading StopCallback) {
	lee.onStartedLeading = append(lee.onStartedLeading, onStartLeading)
	lee.onStoppedLeading = append(lee.onStoppedLeading, onStopLeading)
}

// K8sLeaderElection is the main struct implementing the extension's behavior.
type K8sLeaderElection struct {
	config *config.Configuration
	client kubernetes.Interface

	leaseHolderID string
	cancel        context.CancelFunc
	waitGroup     sync.WaitGroup

	onStartedLeading []StartCallback
	onStoppedLeading []StopCallback
}

func NewK8sLeaderElection(config *config.Configuration, client kubernetes.Interface, leaseHolderID string) *K8sLeaderElection {
	return &K8sLeaderElection{
		config:        config,
		client:        client,
		leaseHolderID: leaseHolderID,
		waitGroup:     sync.WaitGroup{},
	}
}

// If the receiver sets a callback function then it would be invoked when the leader wins the election
func (lee *K8sLeaderElection) startedLeading(ctx context.Context) {
	for _, callback := range lee.onStartedLeading {
		callback(ctx)
	}
}

// If the receiver sets a callback function then it would be invoked when the leader loss the election
func (lee *K8sLeaderElection) stoppedLeading() {
	for _, callback := range lee.onStoppedLeading {
		callback()
	}
}

// Start begins the extension's processing.
func (lee *K8sLeaderElection) Start(_ context.Context) error {
	log.Logger.Info().Msgf("Starting k8s leader elector with UUID `%s`", lee.leaseHolderID)

	ctx := context.Background()
	ctx, lee.cancel = context.WithCancel(ctx)
	// Create the K8s leader elector
	leaderElector, err := newK8sLeaderElector(lee.config, lee.client, lee.startedLeading, lee.stoppedLeading, lee.leaseHolderID)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create k8s leader elector")
		return err
	}
	lee.waitGroup.Add(1)
	go func() {
		// Leader election loop stops if context is canceled or the leader elector loses the lease.
		// The loop allows continued participation in leader election, even if the lease is lost.
		defer lee.waitGroup.Done()
		for {
			leaderElector.Run(ctx)
			if ctx.Err() != nil {
				break
			}
			log.Logger.Info().Msg("Leader lease lost. Returning to standby mode...")
		}
	}()

	return nil
}

// Shutdown ends the extension's processing.
func (lee *K8sLeaderElection) Shutdown(context.Context) error {
	log.Logger.Info().Msgf("Stopping k8s leader elector with UUID `%s`", lee.leaseHolderID)
	if lee.cancel != nil {
		lee.cancel()
	}
	lee.waitGroup.Wait()
	return nil
}

func newK8sLeaderElector(config *config.Configuration, client kubernetes.Interface, onStartedLeading func(context.Context), onStoppedLeading func(), identity string) (*leaderelection.LeaderElector, error) {
	resourceLock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		config.Kubernetes.LeaseNamespace,
		config.Kubernetes.LeaseName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: identity,
		})
	if err != nil {
		return nil, err
	}

	leConfig := leaderelection.LeaderElectionConfig{
		Lock:          resourceLock,
		LeaseDuration: config.Kubernetes.LeaseDuration,
		RenewDeadline: config.Kubernetes.RenewDuration,
		RetryPeriod:   config.Kubernetes.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onStartedLeading,
			OnStoppedLeading: onStoppedLeading,
		},
	}

	return leaderelection.NewLeaderElector(leConfig)
}
