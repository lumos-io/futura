package dlq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
)

const (
	invalidEventsMessagesTopic  string = "invalid.k8s.events"
	invalidStatsMessagesTopic   string = "invalid.k8s.stats"
	invalidObjectsMessagesTopic string = "invalid.k8s.objects"
)

// invalidEvent represents the DLQ event
type invalidEvent struct {
	RawMessage string `json:"raw_message"`
	Error      string `json:"error"`
	Timestamp  int64  `json:"timestamp"`
}

type DLQHandler struct {
	kc stream.Stream
	wg sync.WaitGroup
}

func New(config *config.Configuration) (*DLQHandler, error) {
	kc, err := stream.NewKafkaClient(config.Kafka.Brokers, "deadletterqueue_group")
	if err != nil {
		return nil, err
	}
	return &DLQHandler{
		kc: kc,
		wg: sync.WaitGroup{},
	}, nil
}

func (d *DLQHandler) StoreInvalidEventMessage(msg []byte, err error) {
	b, _ := json.Marshal(invalidEvent{
		RawMessage: string(msg),
		Error:      err.Error(),
		Timestamp:  time.Now().Unix(),
	})
	d.kc.Publish(context.Background(), invalidEventsMessagesTopic, b)
}

func (d *DLQHandler) StoreInvalidStatMessage(msg []byte, err error) {
	b, _ := json.Marshal(invalidEvent{
		RawMessage: string(msg),
		Error:      err.Error(),
		Timestamp:  time.Now().Unix(),
	})
	d.kc.Publish(context.Background(), invalidStatsMessagesTopic, b)
}

func (d *DLQHandler) StoreInvalidObjectMessage(msg []byte, err error) {
	b, _ := json.Marshal(invalidEvent{
		RawMessage: string(msg),
		Error:      err.Error(),
		Timestamp:  time.Now().Unix(),
	})
	d.kc.Publish(context.Background(), invalidObjectsMessagesTopic, b)
}
