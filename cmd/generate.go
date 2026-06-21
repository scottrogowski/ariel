package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottmrogowski/ariel/dsl"
	"github.com/scottmrogowski/ariel/renderer"
	"github.com/spf13/cobra"
)

var generateOutput string

var generateCmd = &cobra.Command{
	Use:   "generate <file>",
	Short: "Render a walkthrough file to a self-contained HTML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		name := filepath.Base(path)

		exitCode := runVerify(path, name, false)
		if exitCode != 0 {
			os.Exit(exitCode)
		}

		w, _, err := dsl.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", name, err)
			os.Exit(2)
		}

		html, err := renderer.Generate(w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate: %v\n", err)
			os.Exit(1)
		}

		outPath := generateOutput
		if outPath == "" {
			outPath = replaceExt(path, ".html")
		}

		if err := renderer.WriteFile(outPath, html); err != nil {
			fmt.Fprintf(os.Stderr, "generate: %v\n", err)
			os.Exit(3)
		}

		fmt.Printf("wrote %s\n", outPath)
		return nil
	},
}

func init() {
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "output path (default: input path with .html extension)")
	rootCmd.AddCommand(generateCmd)
}

// replaceExt replaces the file extension of path with newExt.
func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + newExt
}
