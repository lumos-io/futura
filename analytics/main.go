package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/analytics/internal/config"
	"github.com/opisvigilant/futura/analytics/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	pb "github.com/opisvigilant/futura/proto/gen/analytics"
)

var analyticsCfg *config.Configuration

func main() {
	if err := setupConfiguration(); err != nil {
		panic(fmt.Errorf("configuration has not loaded correctly"))
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", analyticsCfg.Analytics.Endpoint)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to listen")
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	cs, err := server.NewAnalyticsServer(analyticsCfg)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to create the analytics server")
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

		log.Logger.Info().Msg("Shutting down analytics service...")
	}()

	pb.RegisterAnalyticsServiceServer(grpcServer, cs)

	log.Logger.Info().Msg("🚀 gRPC server listening on :50061")
	if err := grpcServer.Serve(lis); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to serve")
		os.Exit(1)
	}
}

func setupConfiguration() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/opt/analytics")
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
	analyticsCfg = config.Fetch()
	if err := analyticsCfg.Validate(); err != nil {
		panic(err.Error())
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if analyticsCfg.Log != nil {
		switch analyticsCfg.Log.Level {
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
	return nil
}
