package sender

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/google/uuid"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/utils"
	"github.com/rs/zerolog/log"

	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbcm "github.com/opisvigilant/futura/proto/gen/common"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbsvc "github.com/opisvigilant/futura/proto/gen/services"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

// Sender handler implements handler.Handler interface,
// Notify event to Sender
type Sender struct {
	ctx       context.Context
	pbc       pbsvc.CollectServiceClient
	batchSize int
	apiKey    string

	KubernetesEventChan         chan *pbev.KubernetesEvent
	KubernetesClusterObjectChan chan *pbcl.KubernetesClusterObject
	KubernetesKubeletStats      chan *pbst.KubernetesKubeletStats
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
		KubernetesEventChan:         make(chan *pbev.KubernetesEvent, 5*resourceChanSize),
		KubernetesClusterObjectChan: make(chan *pbcl.KubernetesClusterObject, 5*resourceChanSize),
		KubernetesKubeletStats:      make(chan *pbst.KubernetesKubeletStats, 5*resourceChanSize),
	}

	eventsInterval := 5 * time.Second
	go s.sendEventsInBatch(s.KubernetesEventChan, eventsInterval)
	go s.sendObjectsClusterInBatch(s.KubernetesClusterObjectChan, eventsInterval)
	go s.sendKubeletStats(s.KubernetesKubeletStats)

	return s, nil
}

func (s *Sender) sendEventsInBatch(ch chan *pbev.KubernetesEvent, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending events to backend")
			return
		case <-t.C:
			randomDuration := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(randomDuration)

			batch := make([]*pbev.KubernetesEvent, 0, s.batchSize)
			loop := true

			for i := 0; (i < s.batchSize) && loop; i++ {
				select {
				case ev := <-ch:
					ev.Apikey = &pbcm.APIKey{
						Key: s.apiKey,
					}
					ev.Metadata = &pbcm.Metadata{
						IdempotencyKey: uuid.NewString(),
						WatcherVersion: utils.WatcherVersion,
					}
					batch = append(batch, ev)
				case <-time.After(100 * time.Millisecond):
					loop = false
				}
			}
			if len(batch) == 0 {
				return
			}
			payload := &pbev.KubernetesEventBatch{
				Events: batch,
			}
			// Send the batch to the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := s.pbc.SendEvents(ctx, payload); err != nil {
				log.Logger.Error().Msgf("SendEvent failed: %v", err)
			}
		}
	}
}

func (s *Sender) sendObjectsClusterInBatch(ch chan *pbcl.KubernetesClusterObject, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending cluster objects to backend")
			return
		case <-t.C:
			randomDuration := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(randomDuration)

			batch := make([]*pbcl.KubernetesClusterObject, 0, s.batchSize)
			loop := true

			for i := 0; (i < s.batchSize) && loop; i++ {
				select {
				case ev := <-ch:
					ev.Apikey = &pbcm.APIKey{
						Key: s.apiKey,
					}
					ev.Metadata = &pbcm.Metadata{
						IdempotencyKey: uuid.NewString(),
						WatcherVersion: utils.WatcherVersion,
					}
					batch = append(batch, ev)
				case <-time.After(100 * time.Millisecond):
					loop = false
				}
			}

			if len(batch) == 0 {
				return
			}

			payload := &pbcl.KubernetesClusterObjectBatch{
				Objects: batch,
			}

			// Send the batch to the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := s.pbc.SendClusterObjects(ctx, payload); err != nil {
				log.Logger.Error().Msgf("SendEvent failed: %v", err)
			}
		}
	}
}

func (s *Sender) sendKubeletStats(ch chan *pbst.KubernetesKubeletStats) {
	for {
		select {
		case ev := <-ch:
			ev.Apikey = &pbcm.APIKey{
				Key: s.apiKey,
			}
			ev.Metadata = &pbcm.Metadata{
				IdempotencyKey: uuid.NewString(),
				WatcherVersion: utils.WatcherVersion,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if _, err := s.pbc.SendKubeletMetrics(ctx, ev); err != nil {
				log.Logger.Error().Msgf("SendKubeletMetrics failed: %v", err)
			}
			cancel()
		case <-s.ctx.Done():
			log.Logger.Info().Msg("stopping sending kubelet stast objects to backend")
			return
		}
	}
}
