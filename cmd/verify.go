package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottmrogowski/ariel/dsl"
	"github.com/scottmrogowski/ariel/internal/mermaidjs"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <file>",
	Short: "Lint a walkthrough file for syntax and semantic errors",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		name := filepath.Base(path)
		exitCode := runVerify(path, name, true)
		os.Exit(exitCode)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

// runVerify executes all verify checks and returns an exit code.
// printResult controls whether output is printed (false when called as a sub-step of generate/watch).
func runVerify(path, displayName string, printResult bool) int {
	w, issues, err := dsl.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", displayName, err)
		return 2
	}

	if len(issues) > 0 {
		printIssues(displayName, issues)
		return 1
	}

	nodes, edges := dsl.ExtractGraph(w.MermaidDiagram)

	// Mermaid syntax validation via embedded JS runtime.
	if err := mermaidjs.Validate(w.MermaidDiagram); err != nil {
		issues = append(issues, dsl.Issue{
			Severity: dsl.SeverityError,
			Message:  fmt.Sprintf("mermaid_diagram: %v", err),
		})
	}

	semanticIssues := dsl.Verify(w, nodes, edges)
	issues = append(issues, semanticIssues...)

	if len(issues) > 0 {
		printIssues(displayName, issues)
		if hasErrors(issues) {
			return 1
		}
		return 0
	}

	if printResult {
		fmt.Printf("✓ %s is valid (%d steps, %d nodes, %d edges)\n",
			displayName, len(w.Steps), len(nodes), len(edges))
	}
	return 0
}

func printIssues(name string, issues []dsl.Issue) {
	for _, issue := range issues {
		if issue.Line > 0 {
			fmt.Printf("%s:%d: %s: %s\n", name, issue.Line, issue.Severity, issue.Message)
		} else {
			fmt.Printf("%s: %s: %s\n", name, issue.Severity, issue.Message)
		}
	}
}

func hasErrors(issues []dsl.Issue) bool {
	for _, i := range issues {
		if i.Severity == dsl.SeverityError {
			return true
		}
	}
	return false
}
