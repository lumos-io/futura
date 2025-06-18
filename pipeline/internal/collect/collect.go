package collect

import (
	"context"
	"fmt"

	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/opisvigilant/futura/pipeline/pkg/stream"
	pb "github.com/opisvigilant/futura/proto/gen/events"
)

type CollectServer struct {
	pb.UnimplementedCollectServiceServer

	streamClient stream.Stream
}

func NewCollectServer(config *config.Configuration) (*CollectServer, error) {
	ctx := context.Background()
	js, err := stream.NewJetstreamClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Stream: %v", err)
	}
	return &CollectServer{
		streamClient: js,
	}, nil
}

func (s *CollectServer) SendEvent(ctx context.Context, req *pb.KubernetesEventBatch) (*pb.CollectAck, error) {
	for _, event := range req.Events {
		if err := s.streamClient.Publish("raw.k8s.events", []byte(event.String())); err != nil {
			return nil, err
		}
	}
	return &pb.CollectAck{Status: "ok", Message: "event received"}, nil
}

func (s *CollectServer) SendMetric(ctx context.Context, req *pb.ContainerMetricBatch) (*pb.CollectAck, error) {
	for _, metric := range req.Metrics {
		if err := s.streamClient.Publish("raw.k8s.metrics", []byte(metric.String())); err != nil {
			return nil, err
		}
	}
	return &pb.CollectAck{Status: "ok", Message: "metric received"}, nil
}
