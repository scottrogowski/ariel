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

// Parse parses walkthrough YAML from a byte slice.
func Parse(data []byte) (*Walkthrough, []Issue, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var w Walkthrough
	if err := dec.Decode(&w); err != nil {
		return nil, yamlErrorsToIssues(err), nil
	}

	var issues []Issue
	if err := requireTopLevelFields(&w, data); err != nil {
		issues = append(issues, *err)
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}

	return &w, nil, nil
}

// requireTopLevelFields checks that mandatory top-level fields are present and non-empty.
func requireTopLevelFields(w *Walkthrough, data []byte) *Issue {
	if w.Title == "" {
		return &Issue{Line: 1, Severity: SeverityError, Message: `missing required field "title"`}
	}
	if w.MermaidDiagram == "" {
		return &Issue{Line: 1, Severity: SeverityError, Message: `missing required field "mermaid_diagram"`}
	}
	if len(w.Steps) == 0 {
		return &Issue{Line: 1, Severity: SeverityError, Message: `"steps" must contain at least one step`}
	}
	return nil
}

// yamlErrorsToIssues converts a yaml decode error into a slice of Issues with line numbers.
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

// parseYAMLErrorMsg extracts line number and message from a yaml error string.
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

func rephraseUnknownField(msg string) string {
	if m := unknownFieldRe.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("unknown field %q", m[1])
	}
	return msg
}
