package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottmrogowski/ariel/internal/dsl"
	"github.com/scottmrogowski/ariel/internal/mermaidjs"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <file.ariel.yaml>",
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

// runVerify parses, fully verifies, and optionally prints a ✓ success line.
// printResult should be false when called as a pre-flight step inside generate/watch.
// Returns 0 (valid), 1 (errors), or 2 (IO error / file not found).
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

	issues = verifyWalkthrough(w)
	if len(issues) > 0 {
		printIssues(displayName, issues)
		if hasErrors(issues) {
			return 1
		}
		return 0
	}

	if printResult {
		sections := w.ToSections()
		var totalNodes, totalEdges, totalSteps int
		for _, sec := range sections {
			nodes, edges := dsl.ExtractGraph(sec.MermaidDiagram)
			totalNodes += len(nodes)
			totalEdges += len(edges)
			totalSteps += len(sec.Steps)
		}
		if len(sections) > 1 {
			fmt.Printf("✓ %s is valid (%d sections, %d steps, %d nodes, %d edges)\n",
				displayName, len(sections), totalSteps, totalNodes, totalEdges)
		} else {
			fmt.Printf("✓ %s is valid (%d steps, %d nodes, %d edges)\n",
				displayName, totalSteps, totalNodes, totalEdges)
		}
	}
	return 0
}

// verifyWalkthrough runs full semantic verification on a parsed walkthrough:
// mermaid syntax per section and step/node/edge validity.
// Assumes the walkthrough has already been parsed cleanly (ParseFile returned no issues).
func verifyWalkthrough(w *dsl.Walkthrough) []dsl.Issue {
	sections := w.ToSections()
	multi := len(sections) > 1
	var issues []dsl.Issue
	for i, sec := range sections {
		nodes, edges := dsl.ExtractGraph(sec.MermaidDiagram)
		if err := mermaidjs.Validate(sec.MermaidDiagram); err != nil {
			msg := fmt.Sprintf("mermaid_diagram: %v", err)
			if multi {
				msg = fmt.Sprintf("section %d %s", i+1, msg)
			}
			issues = append(issues, dsl.Issue{Severity: dsl.SeverityError, Message: msg})
		}
		for _, issue := range dsl.Verify(sec.Steps, nodes, edges) {
			if multi {
				issue.Message = fmt.Sprintf("section %d: %s", i+1, issue.Message)
			}
			issues = append(issues, issue)
		}
	}
	return issues
}

// printIssues writes issues to stdout in "file:line: severity: message" format.
func printIssues(name string, issues []dsl.Issue) {
	for _, issue := range issues {
		if issue.Line > 0 {
			fmt.Printf("%s:%d: %s: %s\n", name, issue.Line, issue.Severity, issue.Message)
		} else {
			fmt.Printf("%s: %s: %s\n", name, issue.Severity, issue.Message)
		}
	}
}

// hasErrors reports whether any issue in the slice has error severity.
func hasErrors(issues []dsl.Issue) bool {
	for _, i := range issues {
		if i.Severity == dsl.SeverityError {
			return true
		}
	}
	return false
}
