package config

import (
	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Debug bool `toml:"debug"`
	Nats *Nats `toml:"nats"`
}

type Kubernetes struct {
	InCluster bool `toml:"inCluster"`
}

type Nats struct {
	Servers []string `tomls:"servers"`
}

func Fetch() *Configuration {
	return &Configuration{
		Debug: getBoolOrDefault("debug", true),
		Nats: &Nats{
			Servers: viper.GetStringSlice(""),
		}
	}
}

func (c *Configuration) Validate() error {
	return nil
}

func getStringOrDefault(key string, defaultValue string) string {
	value := viper.GetString(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func getBoolOrDefault(key string, defaultValue bool) bool {
	value := viper.GetBool(key)
	if !value {
		return value
	}
	return defaultValue
}

func getUInt64OrDefault(key string, defaultValue uint64) uint64 {
	value := viper.GetUint64(key)
	if value != defaultValue {
		return value
	}
	return defaultValue
}
