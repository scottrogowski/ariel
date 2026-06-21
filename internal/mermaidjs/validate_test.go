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

func TestValidate_AuthFlowDiagram(t *testing.T) {
	diagram := `graph TD
    U([User]) -->|submits credentials| LF[Login Form]
    LF -->|POST /auth/login| API[Auth API]
    API -->|lookup| DB[(User DB)]
    DB -->|user record| API
    API --> PV{Password Valid?}
    PV -->|yes| TG[Token Generator]
    PV -->|no| ER[Error Response]
    TG --> SE[Set Cookie]
    SE --> DA[Dashboard]
    ER -->|401| LF`

	if err := Validate(diagram); err != nil {
		t.Errorf("expected auth flow diagram to be valid, got: %v", err)
	}
}
