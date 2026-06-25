package cmd

import (
	"fmt"

	"github.com/scottmrogowski/ariel/internal/guide"
	"github.com/spf13/cobra"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: guideShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(guide.Reference)
		return nil
	},
}

var singleDiagramExampleCmd = &cobra.Command{
	Use:   "single-diagram-example",
	Short: singleDiagramExampleShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(guide.SingleDiagramExample)
		return nil
	},
}

var multipleDiagramExampleCmd = &cobra.Command{
	Use:   "multiple-diagram-example",
	Short: multipleDiagramExampleShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(guide.MultipleDiagramExample)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guideCmd)
	rootCmd.AddCommand(singleDiagramExampleCmd)
	rootCmd.AddCommand(multipleDiagramExampleCmd)
}
