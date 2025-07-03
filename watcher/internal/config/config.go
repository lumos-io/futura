package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Collect    *Collect    `toml:"collect"`
	Kubernetes *Kubernetes `toml:"kubernetes"`
	Log        *Log        `toml:"log"`
}

type Collect struct {
	Endpoint string `toml:"endpoint"`
	APIKey   string `toml:"apiKey"`
}

type Kubernetes struct {
	AuthType        string   `toml:"authType"`
	KubeContextName string   `toml:"kubeContextName"`
	Namespaces      []string `toml:"namespaces"`
	// Collection interval for metrics.
	CollectionInterval time.Duration `toml:"collectionInterval"`
	// Whether OpenShift support should be enabled or not.
	Distribution string `toml:"distribution"`
	// Collection interval for metadata.
	// Metadata of the particular entity in the cluster is collected when the entity changes.
	// In addition metadata of all entities is collected periodically even if no changes happen.
	// Setting the duration to 0 will disable periodic collection (however will not impact
	// metadata collection on changes).
	MetadataCollectionInterval time.Duration `toml:"metadataCollectionInterval"`
	LeaseName                  string        `toml:"leaseName"`
	LeaseNamespace             string        `toml:"leaseNamespace"`
	LeaseDuration              time.Duration `toml:"leaseDuration"`
	RenewDuration              time.Duration `toml:"renewDeadline"`
	RetryPeriod                time.Duration `toml:"retryPeriod"`
}

type Log struct {
	Level string `toml:"level"`
}

func Fetch() *Configuration {
	return &Configuration{
		Collect: &Collect{
			Endpoint: getStringOrDefault("collect.endpoint", "localhost:50051"),
			APIKey:   viper.GetString("collect.apiKey"),
		},
		Kubernetes: &Kubernetes{
			AuthType:                   viper.GetString("kubernetes.authType"),
			KubeContextName:            viper.GetString("kubernetes.kubeContextName"),
			Namespaces:                 viper.GetStringSlice("kubernetes.namespaces"),
			Distribution:               getStringOrDefault("kubernetes.distribution", "kubernetes"),
			CollectionInterval:         convertDurationStringToTime(getStringOrDefault("kubernetes.collectionInterval", "10")),
			MetadataCollectionInterval: convertDurationStringToTime(getStringOrDefault("kubernetes.metadataCollectionInterval", "30")),
			LeaseName:                  viper.GetString("kubernetes.leaseName"),
			LeaseNamespace:             viper.GetString("kubernetes.leaseNamespace"),
			LeaseDuration:              convertDurationStringToTime(getStringOrDefault("kubernetes.leaseDuration", "15")),
			RenewDuration:              convertDurationStringToTime(getStringOrDefault("kubernetes.renewDeadline", "10")),
			RetryPeriod:                convertDurationStringToTime(getStringOrDefault("kubernetes.retryPeriod", "2")),
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
	if c.Kubernetes == nil {
		return errors.New("[kubernetes] entry is missing from the configuration")
	}
	if c.Collect.APIKey == "" {
		return errors.New("apiKey field must be set with a valid key")
	}
	if c.Kubernetes.AuthType == "" && (c.Kubernetes.AuthType != "none" || c.Kubernetes.AuthType == "serviceAccount" || c.Kubernetes.AuthType == "kubeConfig") {
		return errors.New("authType must be set with either `none`, `serviceAccount` or `kubeConfig`")
	}
	if c.Kubernetes.AuthType == "kubeConfig" && c.Kubernetes.KubeContextName == "" {
		return errors.New("kubeContextName must be set authType is set to `kubeConfig`")
	}
	if c.Kubernetes.LeaseName == "" || c.Kubernetes.LeaseNamespace == "" {
		return errors.New("lease name and namespace must be set")
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

func convertDurationStringToTime(duration string) time.Duration {
	val, err := strconv.Atoi(duration)
	if err != nil {
		log.Logger.Fatal().Msgf("could not convert duration `%s` to integer", duration)
		os.Exit(1)
	}
	return time.Duration(val * int(time.Second))
}
