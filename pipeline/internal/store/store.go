package store

import (
	"context"
	"sync"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	EnrichedEventsTopic  = "enrich.k8s.events"
	EnrichedStatsTopic   = "enrich.k8s.stats"
	EnrichedObjectsTopic = "enrich.k8s.objects"
	EnrichedEBPFTopic    = "enrich.ebpf.metrics"
)

type Storer struct {
	config *config.Configuration

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Storer, error) {
	return &Storer{
		config: config,
		wg:     sync.WaitGroup{},
	}, nil
}

func (e *Storer) Start(ctx context.Context) error {
	e.wg.Add(4)

	// Read event messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "store_group_events")
		if err != nil {
			panic(err)
		}
		splitter := NewEventFlattener(kc)
		log.Info().Msg("Start consuming Enriched Kubernete Events...")
		if err := kc.Subscribe(ctx, EnrichedEventsTopic, func(msg stream.Message, ack func() error) {
			var m pb.KubernetesEvent
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the validated event message")
				return
			}
			if err := splitter.Flatten(ctx, &m); err != nil {
				log.Error().Err(err).Msg("failed to split the event message")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedEventsTopic)
		}
	}()

	// Read stats messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "store_group_stats")
		if err != nil {
			panic(err)
		}
		splitter := NewStatsFlattener(kc)
		log.Info().Msg("Start consuming Enriched Kubernete Kubelet Stats...")
		if err := kc.Subscribe(ctx, EnrichedStatsTopic, func(msg stream.Message, ack func() error) {
			var m pb.KubernetesKubeletStats
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}
			if err := splitter.Flatten(ctx, &m); err != nil {
				log.Error().Err(err).Msg("failed to split the stats message")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedStatsTopic)
		}
	}()

	// Read object messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "store_group_objects")
		if err != nil {
			panic(err)
		}
		splitter := NewObjectFlattener(kc)
		log.Info().Msg("Start consuming Enriched Kubernete Cluster Object...")
		if err := kc.Subscribe(ctx, EnrichedObjectsTopic, func(msg stream.Message, ack func() error) {
			var m pb.KubernetesClusterObject
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			if err := splitter.Flatten(ctx, &m); err != nil {
				log.Error().Err(err).Msg("failed to split the object message")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedObjectsTopic)
		}
	}()

	// Read eBPF metrics messages
	go func() {
		defer e.wg.Done()

		kc, err := stream.NewKafkaClient(e.config.Kafka.Brokers, "store_group_ebpf")
		if err != nil {
			panic(err)
		}
		flattener := NewEBPFFlattener(kc)
		log.Info().Msg("Start consuming Enriched eBPF Metrics...")
		if err := kc.Subscribe(ctx, EnrichedEBPFTopic, func(msg stream.Message, ack func() error) {
			var m pb.EBPFMetrics
			if err := proto.Unmarshal(msg.Data(), &m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the enriched eBPF metrics message")
				return
			}
			if err := flattener.Flatten(ctx, &m); err != nil {
				log.Error().Err(err).Msg("failed to flatten the eBPF metrics message")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", EnrichedEBPFTopic)
		}
	}()

	e.wg.Wait()

	log.Info().Msg("Ready to say goodbye...")

	return nil
}
