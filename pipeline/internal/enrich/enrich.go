package enrich

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/opisvigilant/futura/go-lib/kv"
	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/opisvigilant/futura/pipeline/internal/dlq"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbmt "github.com/opisvigilant/futura/proto/gen/common"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

const (
	ValidatedEventsTopic  = "validate.k8s.events"
	ValidatedStatsTopic   = "validate.k8s.stats"
	ValidatedObjectsTopic = "validate.k8s.objects"

	EnrichedEventsTopic  = "enrich.k8s.events"
	EnrichedStatsTopic   = "enrich.k8s.stats"
	EnrichedObjectsTopic = "enrich.k8s.objects"
)

type Enricher struct {
	config *config.Configuration
	rc     kv.KVStore
	dlq    *dlq.DLQHandler

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Enricher, error) {
	rc, err := kv.NewRedisKVStore(config.Redis.Servers)
	if err != nil {
		return nil, err
	}
	c, err := dlq.New(config)
	if err != nil {
		return nil, err
	}
	return &Enricher{
		config: config,
		rc:     rc,
		dlq:    c,
		wg:     sync.WaitGroup{},
	}, nil
}

func (e *Enricher) Start(ctx context.Context) error {
	e.wg.Add(3)

	// Read event messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "enrichment_group_events")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Validated Kubernete Events...")
		if err := kc.Subscribe(ctx, ValidatedEventsTopic, func(msg stream.Message, ack func() error) {
			var m pbev.KubernetesEvent
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the validated event message")
				return
			}
			enrichedEvent, err := e.EnrichEventMessage(&m)
			if err != nil {
				e.dlq.StoreInvalidEventMessage(msg.Data(), err)
				return
			}
			b, err := proto.Marshal(enrichedEvent)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched event message")
				return
			}
			if err := kc.Publish(ctx, EnrichedEventsTopic, b); err != nil {
				log.Error().Err(err).Msg("failed to publish the enriched event message to the store topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", ValidatedEventsTopic)
		}
	}()

	// Read stats messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "enrichment_group_stats")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Validated Kubernete Kubelet Stats...")
		if err := kc.Subscribe(ctx, ValidatedStatsTopic, func(msg stream.Message, ack func() error) {
			var m pbst.KubernetesKubeletStats
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}
			enrichedStats, err := e.EnrichStatsMessage(&m)
			if err != nil {
				e.dlq.StoreInvalidStatMessage(msg.Data(), err)
				return
			}

			b, err := proto.Marshal(enrichedStats)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched stats message")
				return
			}
			if err := kc.Publish(ctx, EnrichedStatsTopic, b); err != nil {
				log.Error().Err(err).Msg("failed to publish the enriched stats message to the store topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", ValidatedStatsTopic)
		}
	}()

	// Read object messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "enrichment_group_objects")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Validated Kubernete Cluster Object...")
		if err := kc.Subscribe(ctx, ValidatedObjectsTopic, func(msg stream.Message, ack func() error) {
			var m pbcl.KubernetesClusterObject
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			enrichedObject, err := e.EnrichObjectMessage(&m)
			if err != nil {
				e.dlq.StoreInvalidObjectMessage(msg.Data(), err)
				return
			}
			b, err := proto.Marshal(enrichedObject)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched object message")
				return
			}
			if err := kc.Publish(ctx, EnrichedObjectsTopic, b); err != nil {
				log.Error().Err(err).Msg("failed to publish the enriched object message to the store topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", ValidatedObjectsTopic)
		}
	}()

	e.wg.Wait()

	log.Info().Msg("Ready to say goodbye...")

	return nil
}

func (e *Enricher) EnrichEventMessage(m *pbev.KubernetesEvent) (*pbev.KubernetesEvent, error) {
	b, err := e.rc.Get(context.Background(), "apikeys", m.Apikey.Key)
	if err != nil {
		return nil, err
	}
	var apiKeyInfo pbmt.ApiKeyInfo
	if err := protojson.Unmarshal(b, &apiKeyInfo); err != nil {
		return nil, err
	}
	if m.Enrichment == nil {
		m.Enrichment = &pbev.EnrichmentMetadata{
			ClusterId:      int64(apiKeyInfo.ClusterId),
			K8SVersion:     apiKeyInfo.KubernetesVersion,
			OrganizationId: apiKeyInfo.OrganizationId,
			ReceivedAtUnix: time.Now().Unix(),
		}
	}
	return m, nil
}

func (e *Enricher) EnrichStatsMessage(m *pbst.KubernetesKubeletStats) (*pbst.KubernetesKubeletStats, error) {
	b, err := e.rc.Get(context.Background(), "apikeys", m.Apikey.Key)
	if err != nil {
		return nil, err
	}
	var apiKeyInfo pbmt.ApiKeyInfo
	if err := protojson.Unmarshal(b, &apiKeyInfo); err != nil {
		return nil, err
	}
	if m.Enrichment == nil {
		m.Enrichment = &pbst.EnrichmentMetadata{
			ClusterId:      int64(apiKeyInfo.ClusterId),
			OrganizationId: apiKeyInfo.OrganizationId,
			ReceivedAtUnix: time.Now().Unix(),
		}
	}
	return m, nil
}

func (e *Enricher) EnrichObjectMessage(m *pbcl.KubernetesClusterObject) (*pbcl.KubernetesClusterObject, error) {
	b, err := e.rc.Get(context.Background(), "apikeys", m.Apikey.Key)
	if err != nil {
		return nil, err
	}
	var apiKeyInfo pbmt.ApiKeyInfo
	if err := protojson.Unmarshal(b, &apiKeyInfo); err != nil {
		return nil, err
	}
	if m.Enrichment == nil {
		m.Enrichment = &pbcl.EnrichmentMetadata{
			ClusterId:      int64(apiKeyInfo.ClusterId),
			OrganizationId: apiKeyInfo.OrganizationId,
			K8SVersion:     apiKeyInfo.KubernetesVersion,
			ReceivedAtUnix: time.Now().Unix(),
		}
	}
	return m, nil
}
