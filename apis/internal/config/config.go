package config

import (
	"errors"

	"github.com/spf13/viper"
)

type Configuration struct {
	Environment string    `toml:"environment"`
	Database    *Database `toml:"database"`
	OAuth       *OAuth    `toml:"oauth"`
	Secrets     *Secrets  `toml:"secrets"`
	Frontend    *Frontend `toml:"frontend"`
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
	}
}

func (c *Configuration) Validate() error {
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
	return nil
}

func getStringOrDefault(key string, defaultValue string) string {
	value := viper.GetString(key)
	if value != "" {
		return value
	}
	return defaultValue
}
