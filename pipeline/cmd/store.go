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

	"github.com/opisvigilant/futura/pipeline/internal/store"
	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if pipelineCfg == nil {
			panic(fmt.Errorf("configuration has not loaded correctly"))
		}

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		s, err := store.New(pipelineCfg)
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
	},
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
