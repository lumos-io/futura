package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/controller"
	"github.com/opisvigilant/futura/watcher/internal/handlers"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var watcherCfg *config.Configuration

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "watcher",
	Short: "Watches all the available Kubernetes events",
	Long: `This application is used to watch all the Kubernetes events that are available.
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

		if watcherCfg.EnablePprof {
			go func() {
				pprofAddr := "localhost:6060"
				logger.Logger().Info().Msgf("initializing pprof %s", pprofAddr)
				err := http.ListenAndServe(pprofAddr, nil)
				if err != nil {
					logger.Logger().Error().Err(err).Msg("failed to initialize pprof")
				}
			}()
		}

		eventHandler, err := handlers.New(watcherCfg)
		if err != nil {
			panic(fmt.Errorf("initHandler failed"))
		}

		controller.Start(watcherCfg, eventHandler)
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
