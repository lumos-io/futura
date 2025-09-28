package collect

import (
	"context"
	"errors"
	"fmt"

	"github.com/opisvigilant/futura/go-lib/kv"
	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"google.golang.org/protobuf/proto"

	pbsvc "github.com/opisvigilant/futura/proto/gen/services"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/rs/zerolog/log"
)

const (
	RawEventsTopic  = "raw.k8s.events"
	RawStatsTopic   = "raw.k8s.stats"
	RawObjectsTopic = "raw.k8s.objects"
)

type CollectServer struct {
	pbsvc.UnimplementedCollectServiceServer

	streamClient stream.Stream
	kvClient     kv.KVStore
	namespace    string
}

func NewCollectServer(config *config.Configuration) (*CollectServer, error) {
	ks, err := stream.NewKafkaClient(config.Kafka.Brokers, "collect_events_group")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Stream: %v", err)
	}
	rss, err := kv.NewRedisKVStore(config.Redis.Servers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KV: %v", err)
	}
	return &CollectServer{
		streamClient: ks,
		kvClient:     rss,
		namespace:    config.Redis.Namespace,
	}, nil
}

func (s *CollectServer) Close() error {
	if err := s.kvClient.Close(); err != nil {
		return err
	}
	if err := s.streamClient.Close(); err != nil {
		return err
	}
	return nil
}

func (s *CollectServer) CollectEvents(ctx context.Context, req *pb.KubernetesEventBatch) (*pbsvc.CollectAck, error) {
	log.Info().Msg("received events...")

	for _, event := range req.Events {
		if err := s.validateAPIKey(ctx, event.Apikey.Key); err != nil {
			log.Error().Err(err)
			return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
		}
		bytes, err := proto.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kubelet stats: %w", err)
		}
		if err := s.streamClient.Publish(ctx, RawEventsTopic, bytes); err != nil {
			log.Error().Err(err)
			return nil, err
		}
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "event received"}, nil
}

func (s *CollectServer) CollectClusterObjects(ctx context.Context, req *pb.KubernetesClusterObjectBatch) (*pbsvc.CollectAck, error) {
	log.Info().Msg("received cluster objects...")

	for _, obj := range req.Objects {
		if err := s.validateAPIKey(ctx, obj.Apikey.Key); err != nil {
			log.Error().Err(err)
			return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
		}
		bytes, err := proto.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kubelet stats: %w", err)
		}
		if err := s.streamClient.Publish(ctx, RawObjectsTopic, bytes); err != nil {
			log.Error().Err(err)
			return nil, err
		}
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "cluster objects received"}, nil
}

func (s *CollectServer) CollectKubeletMetrics(ctx context.Context, req *pb.KubernetesKubeletStats) (*pbsvc.CollectAck, error) {
	log.Info().Msg("received kubelet metrics...")

	if err := s.validateAPIKey(ctx, req.Apikey.Key); err != nil {
		log.Error().Err(err)
		return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
	}
	bytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubelet stats: %w", err)
	}
	if err := s.streamClient.Publish(ctx, RawStatsTopic, bytes); err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "kubelet stats received"}, nil
}

func (s *CollectServer) CollectEBPFMetrics(ctx context.Context, req *pb.EBPFMetricsBatch) (*pbsvc.CollectAck, error) {
	return nil, nil
}

func (s *CollectServer) validateAPIKey(ctx context.Context, apiKey string) error {
	b, err := s.kvClient.Get(ctx, s.namespace, apiKey)
	if err != nil {
		return err
	}
	if b == nil {
		return errors.New("could not find the apikey")
	}
	return nil
}
