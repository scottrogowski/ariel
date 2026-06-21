package cmd

import (
	"fmt"

	"github.com/scottmrogowski/ariel/guide"
	"github.com/spf13/cobra"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Print a brief DSL reference and authoring tips (Agents: run this first)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(guide.Reference)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guideCmd)
}
