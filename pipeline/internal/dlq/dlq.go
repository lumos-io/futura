package dql

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/rs/zerolog/log"
)

const (
	InvalidEventsMessagesTopic  string = ""
	InvalidStatsMessagesTopic   string = ""
	InvalidObjectsMessagesTopic string = ""
)

// InvalidEvent represents the DLQ schema for Parquet
type InvalidEvent struct {
	RawMessage string `parquet:"name=raw_message, type=UTF8, encoding=PLAIN_DICTIONARY"`
	Error      string `parquet:"name=error_reason, type=UTF8, encoding=PLAIN_DICTIONARY"`
	Timestamp  int64  `parquet:"name=timestamp, type=INT64"`
}

type DLQHandler struct {
	kc stream.Stream

	s3Client   *s3.Client
	s3Bucket   string
	s3Prefix   string
	batchSize  int
	flushEvery time.Duration
	mu         sync.Mutex
	batch      []InvalidEvent
	wg         sync.WaitGroup
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

func (d *DLQHandler) StoreInvalidMessage(msg []byte, topic string) error {
	return d.kc.Publish(context.Background(), topic, msg)
}

func (d *DLQHandler) HandleInvalidMessages(ctx context.Context) error {
	d.wg.Add(3)

	go func() {
		defer d.wg.Done()

		if err := d.kc.Subscribe(ctx, InvalidEventsMessagesTopic, func(msg stream.Message, ack func() error) {

		}); err != nil {
			log.Error().Err(err).Msg("failed to validate the raw event message")
			return
		}
	}()

	return nil
}
