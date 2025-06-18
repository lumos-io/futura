/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/opisvigilant/futura/pipeline/internal/collect"
	pb "github.com/opisvigilant/futura/proto/events/gen"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if pipelineCfg == nil {
			panic(fmt.Errorf("configuration has not loaded correctly"))
		}

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		address := fmt.Sprintf("%s:%s", pipelineCfg.Collect.Host, pipelineCfg.Collect.Port)
		lis, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
			os.Exit(1)
		}

		// start shutdown goroutine
		go func() {
			// capture sigterm and other system call here
			<-signalCh
			fmt.Println("Shutting down collecto stage...")
		}()

		grpcServer := grpc.NewServer()

		cs, err := collect.NewCollectServer(pipelineCfg)
		if err != nil {
			log.Fatalf("failed to create the collect server: %v", err)
			os.Exit(1)
		}

		pb.RegisterCollectServiceServer(grpcServer, cs)

		log.Println("🚀 gRPC server listening on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(collectCmd)
}
