package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Pipeline commands",
	Long:  `Pipeline commands for data ingestion from Kafka to ClickHouse`,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}

// Subcommands will be added from pipeline package
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Start the collect pipeline",
	Long:  `Start the Kafka consumer that collects events and processes them`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		// TODO: Implement collect logic
		return fmt.Errorf("collect command not yet implemented")
	},
}

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Start the store pipeline",
	Long:  `Start the pipeline that stores events to ClickHouse`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		// TODO: Implement store logic
		return fmt.Errorf("store command not yet implemented")
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Start the validate pipeline",
	Long:  `Start the pipeline that validates events`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		// TODO: Implement validate logic
		return fmt.Errorf("validate command not yet implemented")
	},
}

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Start the enrich pipeline",
	Long:  `Start the pipeline that enriches events with additional data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ValidatePipeline(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		// TODO: Implement enrich logic
		return fmt.Errorf("enrich command not yet implemented")
	},
}

func init() {
	pipelineCmd.AddCommand(collectCmd)
	pipelineCmd.AddCommand(storeCmd)
	pipelineCmd.AddCommand(validateCmd)
	pipelineCmd.AddCommand(enrichCmd)
}
