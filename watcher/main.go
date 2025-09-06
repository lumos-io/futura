package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/watcher/internal/collector"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var watcherCfg *config.Configuration

func main() {
	if err := setupConfiguration(); err != nil {
		panic(fmt.Errorf("configuration has not loaded correctly"))
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if watcherCfg.Log != nil {
		switch watcherCfg.Log.Level {
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	}

	debug.SetGCPercent(80)
	ctx, cancel := context.WithCancel(context.Background())

	collector, err := collector.New(watcherCfg)
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		signal.Stop(c)
		log.Logger.Info().Msg("Shutdown signal received...")
		if err := collector.Shutdown(ctx); err != nil {
			log.Logger.Error().Err(err).Msg("error during shutdown")
		}
		cancel()
	}()

	if err := collector.Start(ctx); err != nil {
		panic(err)
	}

	log.Logger.Info().Msg("Collector started. Waiting for shutdown signal...")
	<-ctx.Done()

	log.Logger.Info().Msg("Collector done")
	log.Logger.Info().Msg("Futura exiting...")
}

func setupConfiguration() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/opt/watcher")
	if err := viper.ReadInConfig(); err != nil {
		if e, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("config.toml not found")
		} else {
			// Config file was found but another error was produced
			fmt.Println(e.Error())
		}
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})

	viper.WatchConfig()

	// fetch and validate configuration file
	watcherCfg = config.Fetch()
	if err := watcherCfg.Validate(); err != nil {
		panic(err.Error())
	}
	return nil
}
