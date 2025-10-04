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

func (r *redisKVStore) indexKey(namespace string) string {
	return fmt.Sprintf("kvindex:%s", namespace)
}

func (r *redisKVStore) Get(ctx context.Context, namespace, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, r.key(namespace, key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (r *redisKVStore) Put(ctx context.Context, namespace, key string, value []byte) error {
	fullKey := r.key(namespace, key)
	// store the blob
	if err := r.client.Set(ctx, fullKey, value, 0).Err(); err != nil {
		return err
	}
	// index the key for listing
	return r.client.SAdd(ctx, r.indexKey(namespace), key).Err()
}

func (r *redisKVStore) Delete(ctx context.Context, namespace, key string) error {
	fullKey := r.key(namespace, key)
	// delete the blob
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return err
	}
	// remove from index
	return r.client.SRem(ctx, r.indexKey(namespace), key).Err()
}

func (r *redisKVStore) List(ctx context.Context, namespace string) ([][]byte, error) {
	keys, err := r.client.SMembers(ctx, r.indexKey(namespace)).Result()
	if err != nil {
		return nil, err
	}

	var values [][]byte
	for _, k := range keys {
		val, err := r.Get(ctx, namespace, k)
		if err != nil {
			return nil, err
		}
		if val != nil {
			values = append(values, val)
		}
	}
	return values, nil
}

// Close closes the Redis client
func (r *redisKVStore) Close() error {
	return r.client.Close()
}
