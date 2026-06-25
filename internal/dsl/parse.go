package dsl

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// errorf constructs a top-level (line 1) error Issue with a formatted message.
func errorf(msg string, args ...any) *Issue {
	return &Issue{Line: 1, Severity: SeverityError, Message: fmt.Sprintf(msg, args...)}
}

// lineNumRe matches "line N:" in yaml error strings.
var lineNumRe = regexp.MustCompile(`line (\d+):`)

// ParseFile reads and strictly parses a walkthrough YAML file.
// Unknown fields at any level are returned as errors (not silently ignored).
// Returns (nil, issues, nil) on parse/validation errors, (nil, nil, err) on IO error.
func ParseFile(path string) (*Walkthrough, []Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return Parse(data)
}

// Parse parses walkthrough YAML from bytes, applying strict unknown-field checking
// and default title injection. Same return semantics as ParseFile.
func Parse(data []byte) (*Walkthrough, []Issue, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var w Walkthrough
	if err := dec.Decode(&w); err != nil {
		return nil, yamlErrorsToIssues(err), nil
	}
	if w.Title == "" {
		w.Title = "Ariel Walkthrough"
	}

	var issues []Issue
	if err := requireTopLevelFields(&w); err != nil {
		issues = append(issues, *err)
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}

	return &w, nil, nil
}

// requireTopLevelFields validates that the walkthrough uses exactly one valid format
// (flat mermaid_diagram+steps, or sections) with all required fields present.
func requireTopLevelFields(w *Walkthrough) *Issue {
	hasSections := len(w.Sections) > 0
	hasFlat := w.MermaidDiagram != "" || len(w.Steps) > 0

	if hasSections && hasFlat {
		return errorf(`"sections" cannot be combined with "mermaid_diagram" or "steps"`)
	}
	if !hasSections && !hasFlat {
		return errorf(`missing content: use "sections" for multiple diagrams or "mermaid_diagram"+"steps" for a single diagram`)
	}

	if hasFlat {
		if w.MermaidDiagram == "" {
			return errorf(`missing required field "mermaid_diagram"`)
		}
		if len(w.Steps) == 0 {
			return errorf(`"steps" must contain at least one step`)
		}
	}

	for i, sec := range w.Sections {
		if sec.MermaidDiagram == "" {
			return errorf(`section %d: missing required field "mermaid_diagram"`, i+1)
		}
		if len(sec.Steps) == 0 {
			return errorf(`section %d: "steps" must contain at least one step`, i+1)
		}
	}

	return nil
}

// yamlErrorsToIssues converts a yaml decode error to a slice of Issues,
// extracting line numbers where available.
func yamlErrorsToIssues(err error) []Issue {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		issues := make([]Issue, 0, len(typeErr.Errors))
		for _, msg := range typeErr.Errors {
			issues = append(issues, parseYAMLErrorMsg(msg))
		}
		return issues
	}
	// Single error (e.g. "yaml: line N: ...")
	return []Issue{parseYAMLErrorMsg(err.Error())}
}

// parseYAMLErrorMsg parses a yaml error string into a line-numbered Issue,
// rephrasing low-readability messages into user-friendly form.
func parseYAMLErrorMsg(msg string) Issue {
	msg = strings.TrimPrefix(msg, "yaml: ")

	line := 0
	if m := lineNumRe.FindStringSubmatch(msg); m != nil {
		line, _ = strconv.Atoi(m[1])
		msg = strings.TrimSpace(lineNumRe.ReplaceAllString(msg, ""))
	}

	// Rephrase the "field X not found in type Y" message to be more user-friendly.
	msg = rephraseUnknownField(msg)

	return Issue{Line: line, Severity: SeverityError, Message: fmt.Sprintf("YAML parse error: %s", msg)}
}

var unknownFieldRe = regexp.MustCompile(`field (\w+) not found in type \S+`)

// rephraseUnknownField converts gopkg.in/yaml.v3's verbose "field X not found in type Y"
// into the user-facing "unknown field "X"".
func rephraseUnknownField(msg string) string {
	if m := unknownFieldRe.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("unknown field %q", m[1])
	}
	return msg
}
