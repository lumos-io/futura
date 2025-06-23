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

	"github.com/joho/godotenv"
	"github.com/opisvigilant/futura/apis/models"
	"github.com/opisvigilant/futura/apis/routes"
)

//go:embed public/*
var embeddedFiles embed.FS

func main() {
	// Load env
	if err := godotenv.Load(); err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}

	// Automigrate
	if err := models.AutoMigrate(); err != nil {
		log.Fatalf("failed to automigrate: %v", err)
	}

	// Setup router
	router, err := routes.SetupRouter(embeddedFiles)
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
