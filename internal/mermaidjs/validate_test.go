package mermaidjs

import (
	"testing"
)

func TestValidate_ValidDiagram(t *testing.T) {
	diagram := `graph TD
    A[Node A] --> B[Node B]
    B --> C[Node C]`

	if err := Validate(diagram); err != nil {
		t.Errorf("expected valid diagram to pass, got: %v", err)
	}
}

func TestValidate_InvalidDiagram(t *testing.T) {
	// Missing closing bracket on node definition.
	diagram := `graph TD
    A[Node A ---> B[Node B]`

	if err := Validate(diagram); err == nil {
		t.Error("expected invalid diagram to fail, got nil error")
	}
}
