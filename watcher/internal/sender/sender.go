package sender

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/google/uuid"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/utils"

	pb "github.com/opisvigilant/futura/proto/events/gen"
)

// Sender handler implements handler.Handler interface,
// Notify event to Sender
type Sender struct {
	ctx       context.Context
	pbc       pb.CollectServiceClient
	batchSize int

	PodEventChan         chan *pb.KubernetesEvent // *PodEvent
	ServiceEventChan     chan *pb.KubernetesEvent // *SvcEvent
	DeploymentEventChan  chan *pb.KubernetesEvent // *DepEvent
	ReplicaSetEventChan  chan *pb.KubernetesEvent // *RsEvent
	EndpointEventChan    chan *pb.KubernetesEvent // *EndpointsEvent
	ContainerEventChan   chan *pb.KubernetesEvent // *ContainerEvent
	DaemonSetEventChan   chan *pb.KubernetesEvent // *DaemonSetEvent
	StatefulSetEventChan chan *pb.KubernetesEvent // *StatefulSetEvent
	JobEventChan         chan *pb.KubernetesEvent // *JobEvent
	CronJobEventChan     chan *pb.KubernetesEvent // *CronJobEvent
}

// Init prepares Webhook configuration
func New(c *config.Configuration) (*Sender, error) {
	address := fmt.Sprintf("%s:%s", c.Collect.Host, c.Collect.Port)
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := pb.NewCollectServiceClient(conn)

	s := &Sender{
		batchSize: 1000,
		pbc:       client,
	}

	return s, nil
}

var resourceBatchSize int64 = 50

func (b *Sender) sendEventsInBatch(ch chan *pb.KubernetesEvent, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-b.ctx.Done():
			logger.Logger().Info().Msg("stopping sending events to backend")
			return
		case <-t.C:
			randomDuration := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(randomDuration)

			b.send(ch)
		}
	}
}

func (b *Sender) send(ch <-chan *pb.KubernetesEvent) {
	batch := make([]*pb.KubernetesEvent, 0, resourceBatchSize)
	loop := true

	for i := 0; (i < int(resourceBatchSize)) && loop; i++ {
		select {
		case ev := <-ch:
			batch = append(batch, ev)
		case <-time.After(100 * time.Millisecond):
			loop = false
		}
	}

	if len(batch) == 0 {
		return
	}

	payload := &pb.KubernetesEventBatch{
		Metadata: &pb.Metadata{
			IdempotencyKey: uuid.NewString(),
			WatcherVersion: utils.WatcherVersion,
			NodeName:       os.Getenv("NODE_NAME"),
		},
		Events: batch,
	}

	// Send the batch to the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := b.pbc.SendEvent(ctx, payload); err != nil {
		logger.Logger().Error().Msgf("SendEvent failed: %v", err)
	}
}
