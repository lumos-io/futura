/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// enrichCmd represents the enrich command
var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if pipelineCfg == nil {
			panic(fmt.Errorf("configuration has not loaded correctly"))
		}

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		// address := fmt.Sprintf("%s:%s", pipelineCfg.Collect.Host, pipelineCfg.Collect.Port)
		// lis, err := net.Listen("tcp", address)
		// if err != nil {
		// 	log.Fatalf("failed to listen: %v", err)
		// 	os.Exit(1)
		// }

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			fmt.Println("Shutting down collecto stage...")
		}()
	},
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// enrichCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// enrichCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
