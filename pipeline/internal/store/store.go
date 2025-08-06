package store

import (
	"context"
	"sync"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

const (
	EnrichedEventsTopic  = "enrich.k8s.events"
	EnrichedStatsTopic   = "enrich.k8s.stats"
	EnrichedObjectsTopic = "enrich.k8s.objects"

	// here all the kafka topics of the messages that
	// will be ingested into ClickHouse
	StoreKubernetesObjectsTopic              = "store.k8s.objects"
)

type Storer struct {
	stream stream.Stream

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Storer, error) {
	kc, err := stream.NewKafkaClient(config.Kafka.Brokers, "enrichment_group")
	if err != nil {
		return nil, err
	}
	return &Storer{
		stream: kc,
		wg:     sync.WaitGroup{},
	}, nil
}

func (e *Storer) Start(ctx context.Context) error {
	e.wg.Add(3)

	// Read event messages
	go func() {
		defer e.wg.Done()

		if err := e.stream.Subscribe(ctx, EnrichedEventsTopic, func(msg stream.Message, ack func() error) {
			var m pbev.KubernetesEvent
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the validated event message")
				return
			}

		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedEventsTopic)
		}
	}()

	// Read stats messages
	go func() {
		defer e.wg.Done()

		if err := e.stream.Subscribe(ctx, EnrichedStatsTopic, func(msg stream.Message, ack func() error) {
			var m pbst.KubernetesKubeletStats
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}

		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedStatsTopic)
		}
	}()

	// Read object messages
	go func() {
		defer e.wg.Done()

		if err := e.stream.Subscribe(ctx, EnrichedObjectsTopic, func(msg stream.Message, ack func() error) {
			var m pbcl.KubernetesClusterObject
			if err := protojson.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			if err := e.stream.Publish(ctx, StoreKubernetesObjectsTopic, nil); err != nil {
				log.Error().Err(err).Msgf("failed to publish message to stream `%s`", StoreKubernetesObjectsTopic)
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedObjectsTopic)
		}
	}()

	e.wg.Wait()

	return nil
}
