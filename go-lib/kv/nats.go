package kv

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

type NATSStore struct {
	apkKV nats.KeyValue
}

func NewNATSStore(ctx context.Context, servers []string, bucket string) (*NATSStore, error) {
	nc, err := nats.Connect(strings.Join(servers, ","))
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream context error: %w", err)
	}

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to open KV bucket: %w", err)
	}
	return &NATSStore{apkKV: kv}, nil
}

func (s *NATSStore) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := s.apkKV.Get(key)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return entry.Value(), nil
}

func (s *NATSStore) Put(ctx context.Context, key string, value []byte) error {
	_, err := s.apkKV.Put(key, value)
	return err
}

func (s *NATSStore) Delete(ctx context.Context, key string) error {
	return s.apkKV.Delete(key)
}
