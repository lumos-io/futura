package sender

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/google/uuid"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/utils"
	"github.com/rs/zerolog/log"

	pbsvc "github.com/opisvigilant/futura/proto/gen/services"
	pbtl "github.com/opisvigilant/futura/proto/gen/telemetry"
)

// Sender handler implements handler.Handler interface,
// Notify event to Sender
type Sender struct {
	ctx       context.Context
	pbc       pbsvc.CollectServiceClient
	batchSize int
	apiKey    string

	KubernetesEventChan         chan *pbtl.KubernetesEvent
	KubernetesClusterObjectChan chan *pbtl.KubernetesClusterObject
	KubernetesKubeletStats      chan *pbtl.KubernetesKubeletStats
	EBPFMetricsChan             chan *pbtl.EBPFMetrics
}

func New(ctx context.Context, config *config.Configuration) (*Sender, error) {
	// TODO: deal with TLS in gRPC and if in development environment switch to Insecure
	conn, err := grpc.NewClient(config.Collect.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := pbsvc.NewCollectServiceClient(conn)

	resourceChanSize := 200
	s := &Sender{
		ctx:                         ctx,
		batchSize:                   1000,
		apiKey:                      config.Collect.APIKey,
		pbc:                         client,
		KubernetesEventChan:         make(chan *pbtl.KubernetesEvent, 5*resourceChanSize),
		KubernetesClusterObjectChan: make(chan *pbtl.KubernetesClusterObject, 5*resourceChanSize),
		KubernetesKubeletStats:      make(chan *pbtl.KubernetesKubeletStats, 5*resourceChanSize),
		EBPFMetricsChan:             make(chan *pbtl.EBPFMetrics, 2*resourceChanSize),
	}

	eventsInterval := 5 * time.Second
	ebpfInterval := 60 * time.Second // Send eBPF metrics every minute
	go s.sendEventsInBatch(s.KubernetesEventChan, eventsInterval)
	go s.sendObjectsClusterInBatch(s.KubernetesClusterObjectChan, eventsInterval)
	go s.sendKubeletStats(s.KubernetesKubeletStats)
	go s.sendEBPFMetricsInBatch(s.EBPFMetricsChan, ebpfInterval)

	return s, nil
}

func (s *Sender) sendEventsInBatch(ch chan *pbtl.KubernetesEvent, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending events to backend")
			return
		case <-t.C:
			batch := make([]*pbtl.KubernetesEvent, 0, s.batchSize)
			loop := true

			for i := 0; (i < s.batchSize) && loop; i++ {
				select {
				case ev := <-ch:
					ev.Apikey = &pbtl.APIKey{
						Key: s.apiKey,
					}
					ev.Metadata = &pbtl.Metadata{
						IdempotencyKey: uuid.NewString(),
						WatcherVersion: utils.WatcherVersion,
					}
					batch = append(batch, ev)
				case <-time.After(100 * time.Millisecond):
					loop = false
				}
			}
			if len(batch) == 0 {
				continue
			}
			payload := &pbtl.KubernetesEventBatch{
				Events: batch,
			}
			// Send the batch to the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := s.pbc.CollectEvents(ctx, payload); err != nil {
				log.Logger.Error().Msgf("SendEvent failed: %v", err)
			}
		}
	}
}

func (s *Sender) sendObjectsClusterInBatch(ch chan *pbtl.KubernetesClusterObject, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending cluster objects to backend")
			return
		case <-t.C:
			batch := make([]*pbtl.KubernetesClusterObject, 0, s.batchSize)
			loop := true

			for i := 0; (i < s.batchSize) && loop; i++ {
				select {
				case ev := <-ch:
					ev.Apikey = &pbtl.APIKey{
						Key: s.apiKey,
					}
					ev.Metadata = &pbtl.Metadata{
						IdempotencyKey: uuid.NewString(),
						WatcherVersion: utils.WatcherVersion,
					}
					batch = append(batch, ev)
				case <-time.After(100 * time.Millisecond):
					loop = false
				}
			}

			if len(batch) == 0 {
				continue
			}

			payload := &pbtl.KubernetesClusterObjectBatch{
				Objects: batch,
			}

			// Send the batch to the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := s.pbc.CollectClusterObjects(ctx, payload); err != nil {
				log.Logger.Error().Msgf("SendEvent failed: %v", err)
			}
		}
	}
}

func (s *Sender) sendKubeletStats(ch chan *pbtl.KubernetesKubeletStats) {
	for {
		select {
		case ev := <-ch:
			ev.Apikey = &pbtl.APIKey{
				Key: s.apiKey,
			}
			ev.Metadata = &pbtl.Metadata{
				IdempotencyKey: uuid.NewString(),
				WatcherVersion: utils.WatcherVersion,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if _, err := s.pbc.CollectKubeletMetrics(ctx, ev); err != nil {
				log.Logger.Error().Msgf("SendKubeletMetrics failed: %v", err)
			}
			cancel()
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending kubelet stast objects to backend")
			return
		}
	}
}

func (s *Sender) sendEBPFMetricsInBatch(ch chan *pbtl.EBPFMetrics, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending eBPF metrics to backend")
			return
		case <-t.C:
			batch := make([]*pbtl.EBPFMetrics, 0, s.batchSize)
			loop := true

			for i := 0; (i < s.batchSize) && loop; i++ {
				select {
				case ev := <-ch:
					ev.Apikey = &pbtl.APIKey{
						Key: s.apiKey,
					}
					ev.Metadata = &pbtl.Metadata{
						IdempotencyKey: uuid.NewString(),
						WatcherVersion: utils.WatcherVersion,
					}
					batch = append(batch, ev)
				case <-time.After(100 * time.Millisecond):
					loop = false
				}
			}

			if len(batch) == 0 {
				continue
			}

			payload := &pbtl.EBPFMetricsBatch{
				Metrics: batch,
			}

			// Send the batch to the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := s.pbc.CollectEBPFMetrics(ctx, payload); err != nil {
				log.Logger.Error().Msgf("SendEBPFMetrics failed: %v", err)
			}
		}
	}
}
