package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Redis   *Redis   `toml:"redis"`
	Collect *Collect `toml:"collect"`
	Log     *Log     `toml:"log"`
}

type Redis struct {
	Servers   []string `toml:"servers"`
	Namespace string   `toml:"namespace"`
}

type Log struct {
	Level string `toml:"level"`
}

type Collect struct {
	Endpoint string `toml:"endpoint"`
}

func Fetch() *Configuration {
	return &Configuration{
		Redis: &Redis{
			Servers:   viper.GetStringSlice("redis.servers"),
			Namespace: viper.GetString("redis.namespace"),
		},
		Collect: &Collect{
			Endpoint: getStringOrDefault("collect.endpoint", "localhost:50051"),
		},
		Log: &Log{
			Level: getStringOrDefault("log.level", "info"),
		},
	}
}

func (c *Configuration) Validate() error {
	if c.Collect == nil {
		return errors.New("[collect] entry is missing from the configuration")
	}
	if c.Redis == nil {
		return errors.New("[redis] entry is missing from the configuration")
	}
	if len(c.Redis.Servers) == 0 {
		return errors.New("redis server endpoint missing from the list")
	}
	if c.Redis.Namespace == "" {
		return errors.New("redis kv bucket is missing")
	}
	if c.Log != nil {
		if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
			return errors.New("invalid log level value")
		}
	}
	return nil
}

func getStringOrDefault(key string, defaultValue string) string {
	value := viper.GetString(key)
	if value != "" {
		return value
	}
	return defaultValue
}
