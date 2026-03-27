package main

import (
	"os"

	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/pipeline"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "shiplog",
	Short:   "Auto-generate Notion changelog from git history",
	Version: version,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Analyze commits and push changelog to Notion",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = os.Getenv("SHIPLOG_CONFIG")
		}
		if configPath == "" {
			configPath = ".shiplog.yml"
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		since, _ := cmd.Flags().GetString("since")
		last, _ := cmd.Flags().GetInt("last")
		output, _ := cmd.Flags().GetString("output")
		quiet, _ := cmd.Flags().GetBool("quiet")

		return pipeline.Run(cfg, pipeline.RunOptions{
			DryRun: dryRun,
			Since:  since,
			Last:   last,
			Output: output,
			Quiet:  quiet,
		})
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup: create Notion DB + .shiplog.yml",
	RunE: func(cmd *cobra.Command, args []string) error {
		notionSetup, _ := cmd.Flags().GetBool("notion-setup")
		return runInit(notionSetup)
	},
}

func main() {
	runCmd.Flags().Bool("dry-run", false, "Preview without writing to Notion")
	runCmd.Flags().String("since", "", "Analyze from specific commit SHA")
	runCmd.Flags().Int("last", 0, "Analyze last N commits")
	runCmd.Flags().String("config", "", "Path to .shiplog.yml (default: .shiplog.yml)")
	runCmd.Flags().String("output", "table", "Output format: table or json")
	runCmd.Flags().Bool("quiet", false, "Suppress non-error output")

	initCmd.Flags().Bool("notion-setup", false, "Guided Notion integration creation")

	rootCmd.AddCommand(runCmd, initCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
