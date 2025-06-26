package collect

import (
	"context"
	"fmt"

	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/opisvigilant/futura/pipeline/pkg/stream"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbsvc "github.com/opisvigilant/futura/proto/gen/services"
	pbwk "github.com/opisvigilant/futura/proto/gen/workload"
)

type CollectServer struct {
	pbsvc.UnimplementedCollectServiceServer

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

func (s *CollectServer) SendEvent(ctx context.Context, req *pbev.KubernetesEventBatch) (*pbsvc.CollectAck, error) {
	for _, event := range req.Events {
		fmt.Println(event)
		// if err := s.streamClient.Publish("raw.k8s.events", []byte(event.String())); err != nil {
		// 	return nil, err
		// }
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "event received"}, nil
}

func (s *CollectServer) SendMetric(ctx context.Context, req *pbwk.ContainerMetricBatch) (*pbsvc.CollectAck, error) {
	for _, metric := range req.Metrics {
		fmt.Println(metric)
		// if err := s.streamClient.Publish("raw.k8s.metrics", []byte(metric.String())); err != nil {
		// 	return nil, err
		// }
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "metric received"}, nil
}
