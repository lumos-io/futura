package mocks

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
	"github.com/stretchr/testify/require"
)

// SetupMockRedis creates an in-memory Redis server and returns its configuration
func SetupMockRedis(t *testing.T) (*miniredis.Miniredis, *config.Redis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisConfig := &config.Redis{
		Servers:   []string{""},
		Namespace: "",
	}

	return mr, redisConfig
}

// CleanupMockRedis closes the mock Redis server
func CleanupMockRedis(mr *miniredis.Miniredis) {
	mr.Close()
}
