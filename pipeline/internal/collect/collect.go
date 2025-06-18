package collect

import (
	"context"
	"log"

	pb "github.com/opisvigilant/futura/pipeline/proto/gen/services"
	msg "github.com/opisvigilant/futura/proto/gen/messages"
)

type CollectServer struct {
	pb.UnimplementedCollectServiceServer
}

func NewCollectServer() *CollectServer {
	return &CollectServer{}
}

func (s *CollectServer) SendEvent(ctx context.Context, req *msg.KubernetesEvent) (*pb.CollectAck, error) {
	log.Printf("[EVENT] %s %s: %s", req.Metadata.ClusterId, req.EventType, req.Reason)

	// You could push this to JetStream here
	// nats.Publish("raw.k8s.events", jsonBody)

	return &pb.CollectAck{Status: "ok", Message: "event received"}, nil
}

func (s *CollectServer) SendMetric(ctx context.Context, req *msg.ContainerMetric) (*pb.CollectAck, error) {
	log.Printf("[METRIC] %s - CPU: %.2f cores, Mem: %d bytes",
		req.Metadata.ContainerName, req.CpuUsageCores, req.MemoryUsageBytes)

	// You could push this to JetStream here
	// nats.Publish("raw.k8s.metrics", jsonBody)

	return &pb.CollectAck{Status: "ok", Message: "metric received"}, nil
}
