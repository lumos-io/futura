package stream

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

type jetstreamClient struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	streamName string
}

func NewNATSJetstreamClient(servers []string, streamName string, streamSubjects []string) (Stream, error) {
	nc, err := nats.Connect(strings.Join(servers, ","))
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream context error: %w", err)
	}

	// Create or validate shared stream
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: streamSubjects,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream error: %w", err)
	}

	return &jetstreamClient{nc: nc, js: js, streamName: streamName}, nil
}

func (j *jetstreamClient) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := j.js.Publish(ctx, subject, data)
	return err
}

func (j *jetstreamClient) Subscribe(ctx context.Context, subject string, handler HandlerFunc) error {
	s, err := j.js.Stream(ctx, j.streamName)
	if err != nil {
		return fmt.Errorf("stream %s not found: %w", j.streamName, err)
	}

	// Unique durable name per subject
	durableName := strings.ReplaceAll(subject, ".", "_") + "_durable"

	// Filter to specific subject
	cons, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durableName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverNewPolicy, // only new messages
	})
	if err != nil {
		return fmt.Errorf("create consumer failed: %w", err)
	}

	cc, err := cons.Consume(func(m jetstream.Msg) {
		handler(&jetstreamMsg{msg: m})
	}, jetstream.ConsumeErrHandler(func(consumeCtx jetstream.ConsumeContext, err error) {
		log.Logger.Error().Err(err).Msg("consumer failed to get message")
	}))
	if err != nil {
		log.Logger.Error().Err(err).Msg("consumer error")
	}

	// Don't defer cc.Stop — you probably want to manage the lifecycle
	go func() {
		<-ctx.Done()
		cc.Stop()
	}()

	return nil
}

func (j *jetstreamClient) Close() {
	j.nc.Drain()
	j.nc.Close()
}

type jetstreamMsg struct {
	msg jetstream.Msg
}

func (j *jetstreamMsg) Data() []byte {
	return j.msg.Data()
}

func (j *jetstreamMsg) Ack() error {
	return j.msg.Ack()
}
