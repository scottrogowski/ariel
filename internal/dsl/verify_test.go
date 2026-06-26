package dsl

import (
	"testing"
)

func TestVerify_ValidWalkthrough(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)
	w := &Walkthrough{
		Title:          "Test",
		MermaidDiagram: authDiagram,
		Steps: []Step{
			{Label: "Overview", Narration: "Full system view."},
			{
				Label:          "Entry",
				HighlightNodes: []string{"U", "LF"},
			},
			{
				FocusNodes: []string{"API"},
			},
		},
	}

	issues := Verify(w.Steps, nodes, edges)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d: %+v", len(issues), issues)
	}
}

func TestVerify_UnknownNodeInHighlight(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)
	w := &Walkthrough{
		Steps: []Step{
			{Label: "Overview"},
			{HighlightNodes: []string{"BOGUS"}},
		},
	}

	issues := Verify(w.Steps, nodes, edges)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("expected error severity")
	}
}

func TestVerify_UnknownNodeInFocus(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)
	w := &Walkthrough{
		Steps: []Step{
			{Label: "Overview"},
			{FocusNodes: []string{"GHOST"}},
		},
	}

	issues := Verify(w.Steps, nodes, edges)
	if len(issues) != 1 || issues[0].Severity != SeverityError {
		t.Errorf("expected 1 error, got %+v", issues)
	}
}

func TestVerify_EmptyStepWarning(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)
	w := &Walkthrough{
		Steps: []Step{
			{}, // no fields at all
		},
	}

	issues := Verify(w.Steps, nodes, edges)
	if len(issues) != 1 || issues[0].Severity != SeverityWarning {
		t.Errorf("expected 1 warning, got %+v", issues)
	}
}

func TestVerify_DisconnectedHighlight(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)

	// U and DA have no path between them among their direct edges — should warn once.
	issues := Verify([]Step{
		{Label: "Overview"},
		{HighlightNodes: []string{"U", "DA"}},
	}, nodes, edges)
	if len(issues) != 1 || issues[0].Severity != SeverityWarning {
		t.Fatalf("expected 1 warning, got %+v", issues)
	}
	if issues[0].Message != "step 2: not all highlighted components are connected" {
		t.Errorf("unexpected message: %q", issues[0].Message)
	}

	// Directly connected nodes should not warn.
	issues = Verify([]Step{
		{Label: "Overview"},
		{HighlightNodes: []string{"TG", "SE"}},
	}, nodes, edges)
	for _, iss := range issues {
		if iss.Severity == SeverityWarning {
			t.Errorf("unexpected warning for directly connected nodes: %q", iss.Message)
		}
	}
}

func TestVerify_FirstStepNoVisuals(t *testing.T) {
	nodes, edges := ExtractGraph(authDiagram)
	for _, step := range []Step{
		{HighlightNodes: []string{"U"}},
		{FocusNodes: []string{"U"}},
	} {
		issues := Verify([]Step{step}, nodes, edges)
		var found bool
		for _, iss := range issues {
			if iss.Severity == SeverityError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error for step 0 with visual fields, got %+v", issues)
		}
	}
}

func TestParse_UnknownField(t *testing.T) {
	yaml := `
title: "Test"
mermaid_diagram: |
  graph TD
    A[Node A] --> B[Node B]
steps:
  - narration: "Hello"
    unknown_field: "bad"
`
	_, issues, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected IO error: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected issues for unknown field, got none")
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("expected error severity, got %q", issues[0].Severity)
	}
}

func TestParse_MissingTitle(t *testing.T) {
	yaml := `
mermaid_diagram: |
  graph TD
    A --> B
steps:
  - narration: "Hello"
`
	w, issues, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected IO error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %+v", issues)
	}
	if w.Title != "Ariel Walkthrough" {
		t.Errorf("expected default title, got %q", w.Title)
	}
}

func TestParse_ValidFile(t *testing.T) {
	yaml := `
title: "My Walkthrough"
mermaid_diagram: |
  graph TD
    A[Node A] --> B[Node B]
steps:
  - label: "Intro"
    narration: "This is a test."
    highlight_nodes: [A]
    focus_nodes: [B]
`
	w, issues, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected IO error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %+v", len(issues), issues)
	}
	if w.Title != "My Walkthrough" {
		t.Errorf("unexpected title: %q", w.Title)
	}
	if len(w.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(w.Steps))
	}
}
