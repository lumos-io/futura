package workflowclusters

import (
	"crypto/rand"
	"math/big"
)

const (
	charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	keyLength = 32
	prefix    = "cluster_"
)

func GenerateAPIKey() (string, error) {
	key := make([]byte, keyLength)
	for i := range key {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		key[i] = charset[n.Int64()]
	}
	return prefix + string(key), nil
}
