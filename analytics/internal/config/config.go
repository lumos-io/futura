package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Config struct contains watcher configuration
type Configuration struct {
	Clickhouse *Clickhouse `toml:"clickhouse"`
	Analytics  *Analytics  `toml:"analytics"`
	Log        *Log        `toml:"log"`
}

type Clickhouse struct {
	Servers  []string `toml:"servers"`
	Database string   `toml:"database"`
	Username string   `toml:"username"`
	Password string   `toml:"password"`
}

type Analytics struct {
	Endpoint string `toml:"endpoint"`
}

type Log struct {
	Level string `toml:"level"`
}

func Fetch() *Configuration {
	return &Configuration{
		Clickhouse: &Clickhouse{
			Servers:  viper.GetStringSlice("clickhosue.servers"),
			Database: viper.GetString("clickhosue.database"),
			Username: viper.GetString("clickhosue.username"),
			Password: viper.GetString("clickhosue.password"),
		},
		Analytics: &Analytics{
			Endpoint: getStringOrDefault("analytics.endpoint", "localhost:50052"),
		},
		Log: &Log{
			Level: getStringOrDefault("log.level", "info"),
		},
	}
}

func (c *Configuration) Validate() error {
	if c.Analytics == nil {
		return errors.New("[analytics] entry is missing from the configuration")
	}
	if c.Clickhouse == nil {
		return errors.New("[clickhouse] entry is missing from the configuration")
	}
	if len(c.Clickhouse.Servers) == 0 {
		return errors.New("clickhouse server endpoint missing from the list")
	}
	if c.Clickhouse.Username == "" {
		return errors.New("clickhouse username is missing")
	}
	if c.Clickhouse.Password == "" {
		return errors.New("clickhouse password is missing")
	}
	if c.Clickhouse.Database == "" {
		return errors.New("clickhouse database is missing")
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
