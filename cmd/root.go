package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ariel",
	Short: "ariel — animated diagram walkthroughs",
	Long: `ariel — animated diagram walkthroughs

Usage:
  ariel <command> [arguments]

Commands:
  guide        Print a brief DSL reference and authoring tips (Agents: run this first)
  verify       Lint a walkthrough file for syntax and semantic errors
  generate     Render a walkthrough file to a self-contained HTML file
  watch        Serve a live-reloading browser preview of a walkthrough file

Run 'ariel <command> --help' for command-specific usage.`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
