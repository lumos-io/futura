package stream

import "context"

type Message interface {
	Data() []byte
	Ack() error
}

type HandlerFunc func(Message)

type Stream interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler HandlerFunc) error
	Close()
}
