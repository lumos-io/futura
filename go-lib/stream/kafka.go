package stream

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type kafkaClient struct {
	// producer it's lazy loaded when called
	// Publish for the first time
	producer *kafka.Producer

	// for the consumer since I lazy load it
	// only if there a call to Subscribe
	brokers []string
	groupID string
}

type kafkaMessage struct {
	data []byte
}

func (m *kafkaMessage) Data() []byte {
	return m.data
}

func NewKafkaClient(brokers []string, groupID string) (Stream, error) {
	return &kafkaClient{brokers: brokers, groupID: groupID}, nil
}

func (kc *kafkaClient) Publish(ctx context.Context, topic string, data []byte) error {
	if kc.producer == nil {
		producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": strings.Join(kc.brokers, ",")})
		if err != nil {
			return fmt.Errorf("producer init error: %w", err)
		}
		kc.producer = producer
	}

	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)

	err := kc.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("publish error: %w", err)
	}

	select {
	case ev := <-deliveryChan:
		m := ev.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (kc *kafkaClient) Subscribe(ctx context.Context, topic string, handler HandlerFunc) error {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kc.brokers,
		"group.id":          kc.groupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return fmt.Errorf("consumer init error: %w", err)
	}

	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		return fmt.Errorf("subscribe error: %w", err)
	}

	go func() {
		defer consumer.Close()

		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ctx.Done():
				return
			case <-sigchan:
				return
			default:
				ev := consumer.Poll(100)
				switch e := ev.(type) {
				case *kafka.Message:
					handler(&kafkaMessage{data: e.Value}, func() error {
						_, err := consumer.CommitMessage(e)
						return err
					})
				case kafka.Error:
					fmt.Fprintf(os.Stderr, "Kafka error: %v\n", e)
				}
			}
		}
	}()
	return nil
}

func (kc *kafkaClient) Close() error {
	kc.producer.Close()
	return nil
}
