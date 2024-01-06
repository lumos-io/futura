package cmd

import (
	"net/http"
	"os"

	"github.com/opisvigilant/futura/pkg/logger"
	"github.com/opisvigilant/futura/watcher/pkg/controller"
	"github.com/opisvigilant/futura/watcher/pkg/webhook"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "watcher",
	Short: "Watches all the available Kubernetes events",
	Long: `This application is used to watch all the Kubernetes events that are available.
The events are batched and then sent to either STDOUT or to a defined Webhook. The former
should be used for debugging while the latter for production and to actually send the 
events to the backend`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		wh := webhook.Webhook{}
		controller.Start(wh)
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
	if os.Getenv("ENABLE_PPROF") == "1" {
		go func() {
			pprofAddr := "localhost:6060"
			logger.Logger().Info().Msgf("Initializing pprof %s", pprofAddr)
			err := http.ListenAndServe(pprofAddr, nil)
			if err != nil {
				logger.Logger().Error().Err(err).Msg("Failed to initialize pprof")
			}
		}()
	}
}
