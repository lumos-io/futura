package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/handlers"
	"github.com/opisvigilant/futura/watcher/internal/kubernetes"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var watcherCfg *config.Configuration

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "watcher",
	Short: "Watches all the available Kubernetes events and metrics-server signals",
	Long: `This application is used to watch all the Kubernetes events and metrics-server signals that are available.
The events are batched and then sent to either STDOUT or to a defined Webhook. The former
should be used for debugging while the latter for production and to actually send the 
events to the backend`,
	PersistentPreRunE: setupConfiguration,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if watcherCfg == nil {
			panic(fmt.Errorf("configuration has not loaded correctly"))
		}

		debug.SetGCPercent(80)
		ctx, cancel := context.WithCancel(context.Background())

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-c
			signal.Stop(c)
			cancel()
		}()

		// var nsFilterRx *regexp.Regexp
		// if os.Getenv("EXCLUDE_NAMESPACES") != "" {
		// 	nsFilterRx = regexp.MustCompile(os.Getenv("EXCLUDE_NAMESPACES"))
		// }

		// var nsFilterStr string
		// if nsFilterRx != nil {
		// 	nsFilterStr = nsFilterRx.String()
		// }

		// Kubernetes events
		var kubernetesCollector *kubernetes.Collector
		kuberneteEvents := make(chan any, 1000)

		var err error
		kubernetesCollector, err = kubernetes.New(watcherCfg, ctx)
		if err != nil {
			panic(err)
		}
		k8sVersion := kubernetesCollector.GetK8sVersion()
		logger.Logger().Info().Msgf("Current Kubernetes version %s", k8sVersion)
		go kubernetesCollector.Start(kuberneteEvents)

		// where to route the events
		eventHandler, err := handlers.New(watcherCfg)
		if err != nil {
			panic(fmt.Errorf("initHandler failed"))
		}

		go eventHandler.HandleKubernetesEvent()

		<-kubernetesCollector.Done()
		logger.Logger().Info().Msg("Collector done")
		logger.Logger().Info().Msg("Futura exiting...")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Disable Help subcommand
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

func setupConfiguration(cmd *cobra.Command, args []string) error {
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
