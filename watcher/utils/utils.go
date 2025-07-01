package utils

import (
	"errors"
	"os"
	"strconv"
)

const (
	WatcherVersion string = "v0.0.1"
)

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, errors.New("getenv: environment variable empty")
	}
	return v, nil
}

func GetEnvBool(key string) (bool, error) {
	s, err := getenvStr(key)
	if err != nil {
		return false, err
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, err
	}
	return v, nil
}
