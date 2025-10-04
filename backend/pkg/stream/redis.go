package stream

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisStreamClient implements the Stream interface
type redisStreamClient struct {
	client           redis.UniversalClient
	consumers        sync.Map // stream -> cancel func
	blockTime        time.Duration
	selectedDB       int
	groupNamePrefix  string
	maxDeliveries    int64
	deadLetterSuffix string
}

type RedisOption func(*redisStreamClient)

func WithBlockTime(d time.Duration) RedisOption {
	return func(c *redisStreamClient) {
		c.blockTime = d
	}
}

func WithGroupNamePrefix(prefix string) RedisOption {
	return func(c *redisStreamClient) {
		c.groupNamePrefix = prefix
	}
}

func WithMaxDeliveries(max int64) RedisOption {
	return func(c *redisStreamClient) {
		c.maxDeliveries = max
	}
}

func WithDeadLetterStreamSuffix(suffix string) RedisOption {
	return func(c *redisStreamClient) {
		c.deadLetterSuffix = suffix
	}
}

// NewRedisStreamClient creates a new Stream backed by Redis Streams
func NewRedisStreamClient(addrs []string, opts ...RedisOption) (Stream, error) {
	client := &redisStreamClient{
		blockTime:        5 * time.Second,
		groupNamePrefix:  "group-",
		consumers:        sync.Map{},
		deadLetterSuffix: "dlq",
		client: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: addrs,
		}),
	}
	for _, opt := range opts {
		opt(client)
	}
	// test if it works or not
	if err := client.client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

// redisMessage holds the data for a message
type redisMessage struct {
	data []byte
}

func (r *redisMessage) Data() []byte {
	return r.data
}

// Publish adds a message to the Redis stream under the given stream
// `data“ needs to be a JSON Marshalled value in []byte
func (r *redisStreamClient) Publish(ctx context.Context, stream string, data []byte) error {
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": data},
	}).Err()
}

// Subscribe starts consuming messages from the stream using a consumer group
func (r *redisStreamClient) Subscribe(ctx context.Context, stream string, handler HandlerFunc) error {
	group := r.groupNamePrefix + stream
	consumer := "consumer-" + time.Now().Format("20060102150405.000")

	// Ensure stream & group exist
	err := r.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP Consumer Group name already exists") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	r.consumers.Store(stream, cancel)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    group,
					Consumer: consumer,
					Streams:  []string{stream, ">"},
					Block:    r.blockTime,
					Count:    1,
				}).Result()

				if err == redis.Nil {
					continue
				}
				if err != nil {
					time.Sleep(time.Second)
					continue
				}

				for _, st := range streams {
					for _, msg := range st.Messages {
						msgID := msg.ID
						deliveryCount := r.getDeliveryCount(ctx, stream, group, msgID)

						if r.maxDeliveries > 0 && deliveryCount > r.maxDeliveries {
							// Move to DLQ
							r.moveToDeadLetter(ctx, stream, msg)
							_ = r.client.XAck(ctx, stream, group, msgID)
							continue
						}

						raw, _ := msg.Values["payload"].(string)
						ack := func() error {
							return r.client.XAck(ctx, stream, group, msgID).Err()
						}
						handler(&redisMessage{data: []byte(raw)}, ack)
					}
				}
			}
		}
	}()

	return nil
}

func (r *redisStreamClient) getDeliveryCount(ctx context.Context, stream, group, msgID string) int64 {
	res, err := r.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  msgID,
		End:    msgID,
		Count:  1,
	}).Result()

	if err != nil || len(res) == 0 {
		return 0
	}
	return res[0].RetryCount + 1 // retry count starts at 0
}

func (r *redisStreamClient) moveToDeadLetter(ctx context.Context, stream string, msg redis.XMessage) {
	if r.deadLetterSuffix == "" {
		return
	}
	dlqStream := stream + "." + r.deadLetterSuffix

	// Flatten all values into string format
	values := make(map[string]interface{})
	maps.Copy(values, msg.Values)
	values["original_id"] = msg.ID

	_ = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: values,
	}).Err()
}

// Close shuts down all active consumers and closes the Redis client
func (r *redisStreamClient) Close() error {
	r.consumers.Range(func(key, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})
	return r.client.Close()
}
