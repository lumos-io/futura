package stream

type Message interface {
	Data() []byte
	Ack() error
}

type HandlerFunc func(Message)

type Stream interface {
	Publish(subject string, data []byte) error
	Subscribe(subject string, handler HandlerFunc) error
	Close()
}
