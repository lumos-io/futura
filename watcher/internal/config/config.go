package config

import (
	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Debug      bool        `toml:"debug"`
	NodeName   string      `toml:"nodeName"`
	Tag        string      `toml:"tag"`
	Collect    *Collect    `toml:"collect"`
	Kubernetes *Kubernetes `toml:"kubernetes"`
}

type Collect struct {
	Host string `toml:"host"`
	Port string `toml:"port"`
}

type Kubernetes struct {
	InCluster  bool     `toml:"inCluster"`
	Namespaces []string `toml:"namespaces"`
}

func Fetch() *Configuration {
	return &Configuration{
		Debug:    viper.GetBool("debug"),
		NodeName: getStringOrDefault("nodeName", "localhost"),
		Tag:      getStringOrDefault("tag", "v0.0.1"),
		Collect: &Collect{
			Host: getStringOrDefault("collect.host", ""),
			Port: getStringOrDefault("collect.port", "50051"),
		},
		Kubernetes: &Kubernetes{
			InCluster:  viper.GetBool("kubernetes.inCluster"),
			Namespaces: viper.GetStringSlice("kubernetes.namespaces"),
		},
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
