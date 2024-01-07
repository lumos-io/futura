package config

import "github.com/spf13/viper"

// Config struct contains watcher configuration
type Configuration struct {
	Debug       bool `toml:"debug"`
	EnablePprof bool `toml:"enablePprof"`
	// Handlers know how to send notifications to specific services.
	Handler *Handler `toml:"handler"`
}

// Handler contains Handler configuration
type Handler struct {
	Console *Console `toml:"console"`
	Webhook *Webhook `toml:"webhook"`
}

// Console contains the stdoutput configuration
type Console struct {
	Color bool `toml:"color"`
}

// Webhook contains Webhook configuration
type Webhook struct {
	URL     string `toml:"url"`
	Cert    string `toml:"cert"`
	TlsSkip bool   `toml:"tlsSkip"`
}

func Fetch() *Configuration {
	return &Configuration{
		Debug:       getBoolOrDefault("debug", true),
		EnablePprof: getBoolOrDefault("enablePprof", false),
		Handler: &Handler{
			Console: &Console{
				Color: getBoolOrDefault("handler.console", true),
			},
			Webhook: &Webhook{
				URL:     getStringOrDefault("handler.url", ""),
				Cert:    getStringOrDefault("handler.cert", ""),
				TlsSkip: getBoolOrDefault("handler.tlsSkip", true),
			},
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

func getBoolOrDefault(key string, defaultValue bool) bool {
	value := viper.GetBool(key)
	if !value {
		return value
	}
	return defaultValue
}
