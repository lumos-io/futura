package stream

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
)

type kafkaClient struct {
	producer sarama.SyncProducer
	brokers  []string
	groupID  string
}

type kafkaMessage struct {
	data []byte
}

func (m *kafkaMessage) Data() []byte {
	return m.data
}

func NewKafkaClient(brokers []string, groupID string) (Stream, error) {
	return &kafkaClient{
		brokers: brokers,
		groupID: groupID,
	}, nil
}

func (kc *kafkaClient) initProducer() error {
	if kc.producer != nil {
		return nil
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Version = sarama.V4_0_0_0 // or appropriate version

	producer, err := sarama.NewSyncProducer(kc.brokers, config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	kc.producer = producer
	return nil
}

func (kc *kafkaClient) Publish(ctx context.Context, topic string, data []byte) error {
	if err := kc.initProducer(); err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err := kc.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (kc *kafkaClient) Subscribe(ctx context.Context, topic string, handler HandlerFunc) error {
	config := sarama.NewConfig()
	config.Version = sarama.V4_0_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}

	client, err := sarama.NewConsumerGroup(kc.brokers, kc.groupID, config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	consumer := &consumerGroupHandler{
		handler: handler,
		ctx:     ctx,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer client.Close()

		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

		for {
			// `Consume` is blocking, so call in loop to handle rebalance
			if err := client.Consume(ctx, []string{topic}, consumer); err != nil {
				fmt.Fprintf(os.Stderr, "Error from consumer: %v\n", err)
				return
			}
			if ctx.Err() != nil {
				return
			}
			select {
			case <-sigchan:
				return
			default:
			}
		}
	}()

	<-done

	return nil
}

func (kc *kafkaClient) Close() error {
	if kc.producer != nil {
		return kc.producer.Close()
	}
	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler interface
type consumerGroupHandler struct {
	handler HandlerFunc
	ctx     context.Context
	// ready   chan bool
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	// close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-h.ctx.Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			h.handler(&kafkaMessage{data: msg.Value}, func() error {
				session.MarkMessage(msg, "consumed")
				return nil
			})

			session.Commit()
		}
	}
}
