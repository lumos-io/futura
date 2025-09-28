package collector

import (
	"context"
	"os"

	"github.com/opisvigilant/futura/watcher/internal/cluster"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/events"
	"github.com/opisvigilant/futura/watcher/internal/stats"
	"github.com/opisvigilant/futura/watcher/internal/ebpf"
	"github.com/opisvigilant/futura/watcher/internal/sender"
	"github.com/opisvigilant/futura/watcher/pkg/kubernetes"

	"github.com/rs/zerolog/log"
)

type Collector struct {
	config                     *config.Configuration
	kubernetesEventsCollector  *events.KubernetesEventsCollector
	kubernetesClusterCollector *cluster.KubernetesClusterCollector
	kuberentesStatsCollector   *stats.KuberentesStatsCollector
	ebpfCollector              *ebpf.EbpfCollector
}

func New(config *config.Configuration) (*Collector, error) {
	kcc, err := cluster.New(config)
	if err != nil {
		return nil, err
	}

	return &Collector{
		config:                     config,
		kubernetesEventsCollector:  events.New(config),
		kubernetesClusterCollector: kcc,
		kuberentesStatsCollector:   stats.New(config),
		ebpfCollector:              nil, // Will be initialized in Start method
	}, nil
}

func (c *Collector) Start(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes events collector...")
		return err
	}
	if err := c.kubernetesClusterCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes cluster collector...")
		return err
	}
	if err := c.kuberentesStatsCollector.Start(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the kubernetes stats collector...")
		return err
	}

	// Initialize and start eBPF collector
	if err := c.startEBPFCollector(ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to start the eBPF collector...")
		return err
	}

	return nil
}

func (c *Collector) startEBPFCollector(ctx context.Context) error {
	// Create sender for eBPF metrics
	s, err := sender.New(ctx, c.config)
	if err != nil {
		return err
	}

	// Create Kubernetes client
	k8sClient, err := kubernetes.MakeClient(kubernetes.APIConfig{
		AuthType: kubernetes.AuthType(c.config.Kubernetes.Auth.AuthType),
		Context:  c.config.Kubernetes.Auth.KubeContextName,
	})
	if err != nil {
		return err
	}

	// Get node name from environment or default
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "localhost" // Default for local testing
	}

	// Get cluster ID from environment or default
	clusterID := os.Getenv("CLUSTER_NAME")
	if clusterID == "" {
		clusterID = "default-cluster"
	}

	// Create eBPF collector with sender integration
	ebpfCollector, err := ebpf.NewEbpfCollectorWithSender(k8sClient, nodeName, clusterID, s.EBPFMetricsChan)
	if err != nil {
		return err
	}
	c.ebpfCollector = ebpfCollector

	// Start the eBPF collector
	if err := c.ebpfCollector.Start(ctx); err != nil {
		return err
	}

	log.Info().Msg("eBPF collector started successfully")
	return nil
}

func (c *Collector) Shutdown(ctx context.Context) error {
	if err := c.kubernetesEventsCollector.Shutdown(ctx); err != nil {
		return err
	}
	if err := c.kubernetesClusterCollector.Shutdown(ctx); err != nil {
		return err
	}
	if err := c.kuberentesStatsCollector.Shutdown(ctx); err != nil {
		return err
	}

	// Shutdown eBPF collector if it was initialized
	if c.ebpfCollector != nil {
		if err := c.ebpfCollector.Close(); err != nil {
			log.Logger.Error().Err(err).Msg("Failed to close eBPF collector")
			return err
		}
	}

	return nil
}
