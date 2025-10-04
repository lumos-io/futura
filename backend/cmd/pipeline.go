package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/opisvigilant/futura/backend/internal/pipeline/collect"
	"github.com/opisvigilant/futura/backend/internal/pipeline/enrich"
	"github.com/opisvigilant/futura/backend/internal/pipeline/store"
	"github.com/opisvigilant/futura/backend/internal/pipeline/validate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pb "github.com/opisvigilant/futura/proto/gen/services"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Pipeline commands",
	Long:  `Pipeline commands for data ingestion from Kafka to ClickHouse`,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}

// Subcommands will be added from pipeline package
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Start the collect pipeline",
	Long:  `Start the Kafka consumer that collects events and processes them`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		lis, err := net.Listen("tcp", cfg.Collect.Endpoint)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
			os.Exit(1)
		}

		grpcServer := grpc.NewServer()

		cs, err := collect.NewCollectServer(cfg)
		if err != nil {
			log.Fatalf("failed to create the collect server: %v", err)
			os.Exit(1)
		}

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			if err := cs.Close(); err != nil {
				panic(err)
			}
			grpcServer.GracefulStop()

			fmt.Println("Shutting down collector stage...")
		}()

		pb.RegisterCollectServiceServer(grpcServer, cs)

		log.Println("🚀 gRPC server listening on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
			os.Exit(1)
		}
		return nil
	},
}

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Start the store pipeline",
	Long:  `Start the pipeline that stores events to ClickHouse`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		s, err := store.New(cfg)
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			signal.Stop(signalCh)
			cancel()
			fmt.Println("Shutting down store stage...")
		}()

		if err := s.Start(ctx); err != nil {
			panic(err)
		}
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Start the validate pipeline",
	Long:  `Start the pipeline that validates events`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		v, err := validate.New(cfg)
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			signal.Stop(signalCh)
			cancel()
			fmt.Println("Shutting down validate stage...")
		}()

		if err := v.Start(ctx); err != nil {
			panic(err)
		}
		return nil
	},
}

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Start the enrich pipeline",
	Long:  `Start the pipeline that enriches events with additional data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		e, err := enrich.New(cfg)
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			signal.Stop(signalCh)
			cancel()
			fmt.Println("Shutting down enrichment stage...")
		}()

		if err := e.Start(ctx); err != nil {
			panic(err)
		}
		return nil
	},
}

func init() {
	pipelineCmd.AddCommand(collectCmd)
	pipelineCmd.AddCommand(storeCmd)
	pipelineCmd.AddCommand(validateCmd)
	pipelineCmd.AddCommand(enrichCmd)
}
