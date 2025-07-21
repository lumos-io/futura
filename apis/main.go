package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/routes"
	"github.com/spf13/viper"
)

//go:embed public/*
var embeddedFiles embed.FS

var apisCfg *config.Configuration

func main() {
	// Load configuration
	if err := setupConfiguration(); err != nil {
		log.Fatalf("failed to load config.toml file: %v", err)
	}

	// setup feature flag
	if err := initializeUnleash(); err != nil {
		log.Fatalf("failed to initialize unleash: %v", err)
	}

	// Automigrate
	if err := models.AutoMigrate(apisCfg.Database); err != nil {
		log.Fatalf("failed to automigrate: %v", err)
	}

	// Setup router
	router, err := routes.SetupRouter(embeddedFiles, apisCfg)
	if err != nil {
		log.Fatalf("failed to define routes: %v", err)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Graceful shutdown goroutine
	go func() {
		<-signalCh
		fmt.Println("Shutting down api...")

		// close unleash http connection
		if err := unleash.Close(); err != nil {
			log.Fatalf("unleash failed to close: %v", err)
		}

		// Give the server 5 seconds to finish ongoing requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
	}()

	// Start server
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to serve: %v", err)
	}
}

func setupConfiguration() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/opt/apis")
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
	apisCfg = config.Fetch()
	if err := apisCfg.Validate(); err != nil {
		panic(err.Error())
	}
	return nil
}

func initializeUnleash() error {
	return unleash.Initialize(
		unleash.WithRefreshInterval(15*time.Second),
		unleash.WithEnvironment(apisCfg.Environment),
		unleash.WithAppName(apisCfg.Unleash.AppName),
		unleash.WithUrl(apisCfg.Unleash.URL),
		unleash.WithCustomHeaders(http.Header{"Authorization": {apisCfg.Unleash.APIToken}}),
	)
}
