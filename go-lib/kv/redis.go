package kv

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisKVStore struct {
	client redis.UniversalClient
}

func NewRedisKVStore(addrs []string) (KVStore, error) {
	client := &redisKVStore{
		client: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: addrs,
		}),
	}
	// test if it works or not
	if err := client.client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

func (r *redisKVStore) key(namespace, k string) string {
	if namespace == "" {
		return k
	}
	return fmt.Sprintf("%s:%s", namespace, k)
}

func (r *redisKVStore) Get(ctx context.Context, namespace, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, r.key(namespace, key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (r *redisKVStore) Put(ctx context.Context, namespace, key string, value []byte) error {
	return r.client.Set(ctx, r.key(namespace, key), value, 0).Err()
}

func (r *redisKVStore) Delete(ctx context.Context, namespace, key string) error {
	return r.client.Del(ctx, r.key(namespace, key)).Err()
}

// Close closes the Redis client
func (r *redisKVStore) Close() error {
	return r.client.Close()
}
