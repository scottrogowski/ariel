package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ariel",
	Short: "Step-by-step Mermaid diagram walkthroughs from a YAML DSL",
	Long: `ariel generates annotated walkthroughs from a YAML file paired with a Mermaid diagram.
Each walkthrough defines a sequence of steps that highlight nodes, animate edges,
and display narration text — rendered as self-contained HTML (interactive, keyboard
navigable) or MP4 (for embedding in GitHub READMEs and docs).`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
