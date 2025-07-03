package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Nats    *Nats    `toml:"nats"`
	Collect *Collect `toml:"collect"`
	Log     *Log     `toml:"log"`
}

type Nats struct {
	Servers   []string `toml:"servers"`
	APKBucket string   `toml:"apkBucket"`
}

type Log struct {
	Level string `toml:"level"`
}

type Collect struct {
	Endpoint string `toml:"endpoint"`
}

func Fetch() *Configuration {
	return &Configuration{
		Nats: &Nats{
			Servers:   viper.GetStringSlice("nats.servers"),
			APKBucket: viper.GetString("nats.apkBucket"),
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
	if c.Nats == nil {
		return errors.New("[nats] entry is missing from the configuration")
	}
	if len(c.Nats.Servers) == 0 {
		return errors.New("nats server endpoint missing from the list")
	}
	if c.Nats.APKBucket == "" {
		return errors.New("nats kv bucket is missing")
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
