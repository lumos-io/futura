package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Configuration contains all backend services configuration
type Configuration struct {
	Environment string      `toml:"environment"`
	Database    *Database   `toml:"database"`
	OAuth       *OAuth      `toml:"oauth"`
	Secrets     *Secrets    `toml:"secrets"`
	Frontend    *Frontend   `toml:"frontend"`
	Redis       *Redis      `toml:"redis"`
	Unleash     *Unleash    `toml:"unleash"`
	Clickhouse  *Clickhouse `toml:"clickhouse"`
	Kafka       *Kafka      `toml:"kafka"`
	Collect     *Collect    `toml:"collect"`
	Log         *Log        `toml:"log"`
}

type Database struct {
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Name     string `toml:"name"`
	SSLMode  string `toml:"sslmode"`
}

type OAuth struct {
	GitHub *OAuthProvider `toml:"github"`
	Google *OAuthProvider `toml:"google"`
}

type OAuthProvider struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	CallbackURL  string `toml:"callback_url"`
}

type Secrets struct {
	JWTSecret         string `toml:"jwt_secret"`
	CookieStoreSecret string `toml:"cookie_store_secret"`
	CSFRSecret        string `toml:"csfr_secret"`
}

type Frontend struct {
	URL string `toml:"url"`
}

type Redis struct {
	Servers   []string `toml:"servers"`
	Namespace string   `toml:"namespace"`
}

type Unleash struct {
	AppName  string `toml:"app_name"`
	APIToken string `toml:"api_token"`
	URL      string `toml:"url"`
}

type Clickhouse struct {
	Servers  []string `toml:"servers"`
	Database string   `toml:"database"`
	Username string   `toml:"username"`
	Password string   `toml:"password"`
}

type Kafka struct {
	Brokers []string `toml:"brokers"`
}

type Collect struct {
	Endpoint string `toml:"endpoint"`
}

type Log struct {
	Level string `toml:"level"`
}

func Fetch() *Configuration {
	return &Configuration{
		Environment: getStringOrDefault("environment", "development"),
		Database: &Database{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetString("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			Name:     viper.GetString("database.name"),
			SSLMode:  viper.GetString("database.sslmode"),
		},
		OAuth: &OAuth{
			GitHub: &OAuthProvider{
				ClientID:     viper.GetString("oauth.github.client_id"),
				ClientSecret: viper.GetString("oauth.github.client_secret"),
				CallbackURL:  viper.GetString("oauth.github.callback_url"),
			},
			Google: &OAuthProvider{
				ClientID:     viper.GetString("oauth.google.client_id"),
				ClientSecret: viper.GetString("oauth.google.client_secret"),
				CallbackURL:  viper.GetString("oauth.google.callback_url"),
			},
		},
		Secrets: &Secrets{
			JWTSecret:         viper.GetString("secrets.jwt_secret"),
			CookieStoreSecret: viper.GetString("secrets.cookie_store_secret"),
			CSFRSecret:        viper.GetString("secrets.csfr_secret"),
		},
		Frontend: &Frontend{
			URL: viper.GetString("frontend.url"),
		},
		Redis: &Redis{
			Servers:   viper.GetStringSlice("redis.servers"),
			Namespace: viper.GetString("redis.namespace"),
		},
		Unleash: &Unleash{
			AppName:  viper.GetString("unleash.app_name"),
			APIToken: viper.GetString("unleash.api_token"),
			URL:      viper.GetString("unleash.url"),
		},
		Clickhouse: &Clickhouse{
			Servers:  viper.GetStringSlice("clickhouse.servers"),
			Database: viper.GetString("clickhouse.database"),
			Username: viper.GetString("clickhouse.username"),
			Password: viper.GetString("clickhouse.password"),
		},
		Kafka: &Kafka{
			Brokers: viper.GetStringSlice("kafka.brokers"),
		},
		Collect: &Collect{
			Endpoint: getStringOrDefault("collect.endpoint", "localhost:50051"),
		},
		Log: &Log{
			Level: getStringOrDefault("log.level", "info"),
		},
	}
}

// ValidateAPIs validates configuration for APIs service
func (c *Configuration) ValidateAPIs() error {
	if c.Database == nil {
		return errors.New("[database] entry is missing from the configuration")
	}
	if c.Database.Host == "" || c.Database.Name == "" || c.Database.Password == "" || c.Database.SSLMode == "" || c.Database.User == "" || c.Database.Port == "" {
		return errors.New("[database] entries are incorrect because some are empty")
	}
	if c.OAuth == nil {
		return errors.New("[oauth] entry is missing from the configuration")
	}
	if c.OAuth.GitHub.ClientID == "" || c.OAuth.GitHub.ClientSecret == "" || c.OAuth.GitHub.CallbackURL == "" {
		return errors.New("[oauth.github] entry is not properly configured because entries are empty")
	}
	if c.OAuth.Google.ClientID == "" || c.OAuth.Google.ClientSecret == "" || c.OAuth.Google.CallbackURL == "" {
		return errors.New("[oauth.google] entry is not properly configured because entries are empty")
	}
	if c.Secrets == nil {
		return errors.New("[secrets] entry is missing from the configuration")
	}
	if c.Secrets.CSFRSecret == "" || c.Secrets.CookieStoreSecret == "" || c.Secrets.JWTSecret == "" {
		return errors.New("[secrets] entries are incorrect because some are empty")
	}
	if c.Frontend == nil || c.Frontend.URL == "" {
		return errors.New("[frontend] entry is missing from the configuration or `url` entry is empty")
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
	if c.Unleash == nil {
		return errors.New("[unleash] entry is missing from the configuration")
	}
	if c.Unleash.APIToken == "" || c.Unleash.URL == "" || c.Unleash.AppName == "" {
		return errors.New("[unleash] entries are incorrect because some are empty")
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
	return c.validateLog()
}

// ValidatePipeline validates configuration for Pipeline service
func (c *Configuration) ValidatePipeline() error {
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
	if c.Kafka == nil {
		return errors.New("[kafka] entry is missing from the configuration")
	}
	if len(c.Kafka.Brokers) == 0 {
		return errors.New("kafka brokers endpoints missing from the list")
	}
	return c.validateLog()
}

func (c *Configuration) validateLog() error {
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
