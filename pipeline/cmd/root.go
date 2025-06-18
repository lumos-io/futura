package cmd

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/opisvigilant/futura/pipeline/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pipelineCfg *config.Configuration

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "The pipeline absorbs the events coming from the watcher and handles them until the storage",
	Long: `This application is used to accepts all the Kubernetes events and metrics-server signals that are pushed by the watcher.
The events are sent through a pipeline with different stages before landing to the Storage where they can be queried.`,
	PersistentPreRunE: setupConfiguration,
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
	viper.AddConfigPath("/opt/pipeline")
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
	pipelineCfg = config.Fetch()
	if err := pipelineCfg.Validate(); err != nil {
		panic(err.Error())
	}
	return nil
}
