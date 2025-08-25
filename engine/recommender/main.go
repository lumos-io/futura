package engine

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/engine/recommender/internal/config"
	"github.com/opisvigilant/futura/engine/recommender/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	pb "github.com/opisvigilant/futura/proto/gen/engine"
)

var recommenderCfg *config.Configuration

func main() {
	if err := setupConfiguration(); err != nil {
		panic(fmt.Errorf("configuration has not loaded correctly"))
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", recommenderCfg.Recommender.Endpoint)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to listen")
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	cs, err := server.NewRecommenderServer(recommenderCfg)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to create the recommender server")
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

		log.Logger.Info().Msg("Shutting down recommender...")
	}()

	pb.RegisterRecommendationServiceServer(grpcServer, cs)

	log.Logger.Info().Msg("🚀 gRPC server listening on :50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to serve")
		os.Exit(1)
	}
}

func setupConfiguration() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/opt/recommender")
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
	recommenderCfg = config.Fetch()
	if err := recommenderCfg.Validate(); err != nil {
		panic(err.Error())
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if recommenderCfg.Log != nil {
		switch recommenderCfg.Log.Level {
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
