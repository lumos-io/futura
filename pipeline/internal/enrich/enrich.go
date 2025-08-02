package enrich

import (
	"context"
	"sync"
	"time"

	"github.com/opisvigilant/futura/go-lib/kv"
	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
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
	stream stream.Stream
	rc     kv.KVStore

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Enricher, error) {
	kc, err := stream.NewKafkaClient(config.Kafka.Brokers, "enrichment_group")
	if err != nil {
		return nil, err
	}

	rc, err := kv.NewRedisKVStore(config.Redis.Servers)
	if err != nil {
		return nil, err
	}
	return &Enricher{
		stream: kc,
		rc:     rc,
		wg:     sync.WaitGroup{},
	}, nil
}

func (e *Enricher) Start(ctx context.Context) error {
	e.wg.Add(3)

	// Read event messages
	go func() {
		defer e.wg.Done()

		if err := e.stream.Subscribe(ctx, ValidatedEventsTopic, func(msg stream.Message, ack func() error) {
			var m pbev.KubernetesEvent
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the validated event message")
				return
			}
			enrichedEvent, err := e.EnrichEventMessage(&m)
			if err != nil {
				log.Error().Err(err).Msg("failed to enrich the event message")
				return
			}

			b, err := protojson.Marshal(enrichedEvent)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched event message")
				return
			}
			if err := e.stream.Publish(ctx, EnrichedEventsTopic, b); err != nil {
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

		if err := e.stream.Subscribe(ctx, ValidatedStatsTopic, func(msg stream.Message, ack func() error) {
			var m pbst.KubernetesKubeletStats
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}
			enrichedStats, err := e.EnrichStatsMessage(&m)
			if err != nil {
				log.Error().Err(err).Msg("failed to enrich the stats message")
				return
			}

			b, err := protojson.Marshal(enrichedStats)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched stats message")
				return
			}
			if err := e.stream.Publish(ctx, EnrichedStatsTopic, b); err != nil {
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

		if err := e.stream.Subscribe(ctx, ValidatedObjectsTopic, func(msg stream.Message, ack func() error) {
			var m pbcl.KubernetesClusterObject
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			enrichedObject, err := e.EnrichObjectMessage(&m)
			if err != nil {
				log.Error().Err(err).Msg("failed to enrich the object message")
				return
			}

			b, err := protojson.Marshal(enrichedObject)
			if err != nil {
				log.Error().Err(err).Msg("failed to proto-marshal the enriched object message")
				return
			}
			if err := e.stream.Publish(ctx, EnrichedObjectsTopic, b); err != nil {
				log.Error().Err(err).Msg("failed to publish the enriched object message to the store topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", ValidatedObjectsTopic)
		}
	}()

	e.wg.Wait()

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
