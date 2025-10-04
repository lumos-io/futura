package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/opisvigilant/futura/backend/internal/apis/models"
	"github.com/opisvigilant/futura/backend/internal/apis/routes"
	"github.com/opisvigilant/futura/backend/internal/apis/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var apisCmd = &cobra.Command{
	Use:   "apis",
	Short: "Start the APIs server",
	Long:  `Start the HTTP/REST API server with SSE support for the Futura platform`,
	RunE:  runAPIs,
}

func init() {
	rootCmd.AddCommand(apisCmd)
	apisCmd.Flags().IntP("port", "p", 8080, "Port to run the APIs server on")
}

func runAPIs(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	// Validate config for APIs
	if err := cfg.ValidateAPIs(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Setup feature flags
	if err := initializeUnleash(); err != nil {
		return fmt.Errorf("failed to initialize unleash: %w", err)
	}
	defer unleash.Close()

	// Automigrate database
	if err := models.AutoMigrate(cfg.Database); err != nil {
		return fmt.Errorf("failed to automigrate: %w", err)
	}

	// Setup router
	router, err := routes.SetupRouter(cfg)
	if err != nil {
		return fmt.Errorf("failed to define routes: %w", err)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Initialize workflow manager
	wf, err := workflow.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize workflow manager: %w", err)
	}

	go func() {
		if err := wf.StartWorkers(); err != nil {
			log.Fatal().Msgf("failed to start workflow worker: %v", err)
		}
	}()

	// Signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Graceful shutdown goroutine
	go func() {
		<-signalCh
		log.Info().Msg("Shutting down APIs server...")

		if err := wf.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop workflow worker")
		}

		// Give the server 5 seconds to finish ongoing requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Msgf("Server forced to shutdown: %v", err)
		}
	}()

	// Start server
	log.Info().Msgf("APIs server listening on port %d", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func initializeUnleash() error {
	return unleash.Initialize(
		unleash.WithRefreshInterval(15*time.Second),
		unleash.WithEnvironment(cfg.Environment),
		unleash.WithAppName(cfg.Unleash.AppName),
		unleash.WithUrl(cfg.Unleash.URL),
		unleash.WithCustomHeaders(http.Header{"Authorization": {cfg.Unleash.APIToken}}),
	)
}
