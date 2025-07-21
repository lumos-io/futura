package kv

import "context"

type Store interface {
	// Get returns the value for the given key or (nil, nil) if not found.
	Get(ctx context.Context, key string) ([]byte, error)

	// Put stores the value under the given key, replacing existing if present.
	Put(ctx context.Context, key string, value []byte) error

	// Delete removes the key (no-op if it does not exist).
	Delete(ctx context.Context, key string) error
}
