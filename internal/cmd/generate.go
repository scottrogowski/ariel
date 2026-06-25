package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottmrogowski/ariel/internal/dsl"
	"github.com/scottmrogowski/ariel/internal/renderer"
	"github.com/spf13/cobra"
)

var generateOutput string
var generateFormat string
var generateStepDuration int

var generateCmd = &cobra.Command{
	Use:   "generate <file.ariel.yaml>",
	Short: generateShort,
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

		switch generateFormat {
		case "mp4":
			outPath := generateOutput
			if outPath == "" {
				outPath = replaceExt(path, ".mp4")
			}
			if err := renderer.GenerateMP4(w, outPath, generateStepDuration); err != nil {
				fmt.Fprintf(os.Stderr, "generate: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("wrote %s\n", outPath)

		case "html", "":
			outPath := generateOutput
			if outPath == "" {
				outPath = replaceExt(path, ".html")
			}
			html, err := renderer.Generate(w)
			if err != nil {
				fmt.Fprintf(os.Stderr, "generate: %v\n", err)
				os.Exit(1)
			}
			if err := renderer.WriteFile(outPath, html); err != nil {
				fmt.Fprintf(os.Stderr, "generate: %v\n", err)
				os.Exit(3)
			}
			fmt.Printf("wrote %s\n", outPath)

		default:
			fmt.Fprintf(os.Stderr, "generate: unknown format %q (valid: html, mp4)\n", generateFormat)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", generateFlagOutputHelp)
	generateCmd.Flags().StringVar(&generateFormat, "format", "html", generateFlagFormatHelp)
	generateCmd.Flags().IntVar(&generateStepDuration, "step-duration", renderer.DefaultStepDuration, generateFlagStepDurationHelp)
	rootCmd.AddCommand(generateCmd)
}

// replaceExt replaces the last extension of path with newExt,
// e.g. "foo.ariel.yaml" → "foo.ariel.html".
func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + newExt
}
