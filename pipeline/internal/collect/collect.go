package collect

import (
	"context"
	"errors"
	"fmt"

	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/opisvigilant/futura/pipeline/pkg/kv"
	"github.com/opisvigilant/futura/pipeline/pkg/stream"
	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbsvc "github.com/opisvigilant/futura/proto/gen/services"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/rs/zerolog/log"
)

type CollectServer struct {
	pbsvc.UnimplementedCollectServiceServer

	streamClient stream.Stream
	kvClient     kv.Store
}

func NewCollectServer(config *config.Configuration) (*CollectServer, error) {
	ctx := context.Background()
	js, err := stream.NewNATSJetstreamClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Stream: %v", err)
	}

	ns, err := kv.NewNATSStore(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KV: %v", err)
	}

	return &CollectServer{
		streamClient: js,
		kvClient:     ns,
	}, nil
}

func (s *CollectServer) SendEvent(ctx context.Context, req *pbev.KubernetesEventBatch) (*pbsvc.CollectAck, error) {
	if err := s.validateAPIKey(ctx, req.Apikey.Key); err != nil {
		return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
	}
	for _, event := range req.Events {
		log.Logger.Debug().Msg(event.String())
		// if err := s.streamClient.Publish("raw.k8s.events", []byte(event.String())); err != nil {
		// 	return nil, err
		// }
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "event received"}, nil
}

func (s *CollectServer) SendClusterObjects(ctx context.Context, req *pbcl.KubernetesClusterObjectBatch) (*pbsvc.CollectAck, error) {
	if err := s.validateAPIKey(ctx, req.Apikey.Key); err != nil {
		return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
	}
	for _, obj := range req.Objects {
		log.Logger.Debug().Msg(obj.String())
		// if err := s.streamClient.Publish("raw.k8s.metrics", []byte(metric.String())); err != nil {
		// 	return nil, err
		// }
	}
	return &pbsvc.CollectAck{Status: "ok", Message: "cluster objects received"}, nil
}

func (s *CollectServer) SendKubeletStats(ctx context.Context, req *pbst.KubernetesKubeletStats) (*pbsvc.CollectAck, error) {
	if err := s.validateAPIKey(ctx, req.Apikey.Key); err != nil {
		return &pbsvc.CollectAck{Status: "failed", Message: err.Error()}, nil
	}

	log.Logger.Debug().Msg(req.KubeletMetrics.String())
	// if err := s.streamClient.Publish("raw.k8s.kubelet", []byte(req.KubeletMetrics.String())); err != nil {
	// 	return nil, err
	// }

	return &pbsvc.CollectAck{Status: "ok", Message: "kubelet stats received"}, nil
}

func (s *CollectServer) validateAPIKey(ctx context.Context, apiKey string) error {
	b, err := s.kvClient.Get(ctx, apiKey)
	if err != nil {
		return err
	}
	if b == nil {
		return errors.New("could not find the apikey")
	}
	return nil
}
