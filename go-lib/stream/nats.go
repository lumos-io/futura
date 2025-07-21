package stream

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

type jetstreamClient struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewNATSJetstreamClient(ctx context.Context, servers []string) (Stream, error) {
	nc, err := nats.Connect(strings.Join(servers, ","))
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream context error: %w", err)
	}

	return &jetstreamClient{nc: nc, js: js}, nil
}

func (j *jetstreamClient) Publish(subject string, data []byte) error {
	_, err := j.js.Publish(subject, data)
	return err
}

func (j *jetstreamClient) Subscribe(subject string, handler HandlerFunc) error {
	_, err := j.js.Subscribe(subject, func(m *nats.Msg) {
		handler(&jetstreamMsg{msg: m})
	}, nats.ManualAck())

	return err
}

func (j *jetstreamClient) Close() {
	j.nc.Drain()
	j.nc.Close()
}

type jetstreamMsg struct {
	msg *nats.Msg
}

func (j *jetstreamMsg) Data() []byte {
	return j.msg.Data
}

func (j *jetstreamMsg) Ack() error {
	return j.msg.Ack()
}
