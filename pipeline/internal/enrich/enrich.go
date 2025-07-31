package enrich

import (
	"context"
	"sync"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
)

type Enricher struct {
	stream stream.Stream

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Enricher, error) {
	kc, err := stream.NewKafkaClient(config.Kafka.Brokers, "enrichment_group")
	if err != nil {
		return nil, err
	}
	return &Enricher{
		stream: kc,
		wg:     sync.WaitGroup{},
	}, nil
}

func (e *Enricher) Start(ctx context.Context) error {
	return nil
}
