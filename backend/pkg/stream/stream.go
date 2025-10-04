package stream

import "context"

type Message interface {
	Data() []byte
}

type HandlerFunc func(msg Message, ack func() error)

type Stream interface {
	Publish(ctx context.Context, stream string, data []byte) error
	Subscribe(ctx context.Context, stream string, handler HandlerFunc) error
	Close() error
}
