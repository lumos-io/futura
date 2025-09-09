/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/opisvigilant/futura/pipeline/internal/validate"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if pipelineCfg == nil {
			panic(fmt.Errorf("configuration has not loaded correctly"))
		}

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		v, err := validate.New(pipelineCfg)
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
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
