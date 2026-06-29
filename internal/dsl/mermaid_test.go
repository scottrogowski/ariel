package dsl

import (
	"testing"
)

const authDiagram = `graph TD
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

func TestExtractGraph_Nodes(t *testing.T) {
	nodes, _ := ExtractGraph(authDiagram)

	want := map[string]string{
		"U":   "User",
		"LF":  "Login Form",
		"API": "Auth API",
		"DB":  "User DB",
		"PV":  "Password Valid?",
		"TG":  "Token Generator",
		"ER":  "Error Response",
		"SE":  "Set Cookie",
		"DA":  "Dashboard",
	}

	for id, wantLabel := range want {
		gotLabel, ok := nodes[id]
		if !ok {
			t.Errorf("node %q not found", id)
			continue
		}
		if gotLabel != wantLabel {
			t.Errorf("node %q: got label %q, want %q", id, gotLabel, wantLabel)
		}
	}

	// No spurious entries from keywords or edge syntax.
	for id := range nodes {
		if _, ok := want[id]; !ok {
			t.Errorf("unexpected node %q in result", id)
		}
	}
}

func TestExtractGraph_Edges(t *testing.T) {
	_, edges := ExtractGraph(authDiagram)

	wantEdges := map[[2]string]bool{
		{"U", "LF"}:   true,
		{"LF", "API"}: true,
		{"API", "DB"}: true,
		{"DB", "API"}: true,
		{"API", "PV"}: true,
		{"PV", "TG"}:  true,
		{"PV", "ER"}:  true,
		{"TG", "SE"}:  true,
		{"SE", "DA"}:  true,
		{"ER", "LF"}:  true,
	}

	got := make(map[[2]string]bool)
	for _, e := range edges {
		got[e] = true
	}

	for e := range wantEdges {
		if !got[e] {
			t.Errorf("missing edge %v->%v", e[0], e[1])
		}
	}
	for e := range got {
		if !wantEdges[e] {
			t.Errorf("unexpected edge %v->%v", e[0], e[1])
		}
	}
}

func TestExtractGraph_Shapes(t *testing.T) {
	diagram := `graph TD
    A[rectangle]
    B{diamond}
    C([rounded])
    D[(cylinder)]
    E((circle))
    F[[subroutine]]
    A --> B
    B --> C`

	nodes, _ := ExtractGraph(diagram)

	want := map[string]string{
		"A": "rectangle",
		"B": "diamond",
		"C": "rounded",
		"D": "cylinder",
		"E": "circle",
		"F": "subroutine",
	}

	for id, wantLabel := range want {
		gotLabel, ok := nodes[id]
		if !ok {
			t.Errorf("node %q not found", id)
			continue
		}
		if gotLabel != wantLabel {
			t.Errorf("node %q: got label %q, want %q", id, gotLabel, wantLabel)
		}
	}
}

func TestExtractGraph_BareTargetNode(t *testing.T) {
	// DB --> API where API appears earlier with a shape, and bare on this line.
	diagram := `graph TD
    API[Auth API]
    DB[(User DB)]
    DB --> API`

	nodes, edges := ExtractGraph(diagram)

	if _, ok := nodes["API"]; !ok {
		t.Error("API node not found")
	}
	if _, ok := nodes["DB"]; !ok {
		t.Error("DB node not found")
	}

	found := false
	for _, e := range edges {
		if e[0] == "DB" && e[1] == "API" {
			found = true
		}
	}
	if !found {
		t.Error("DB->API edge not found")
	}
}

const seqDiagram = `sequenceDiagram
  participant HP as Human Prompter
  participant C as Claude
  participant CLI as ariel CLI
  participant HR as Human Reviewer

  HP->>C: Prompt
  C->>C: Write code
  C->>CLI: ariel guide
  CLI-->>C: DSL reference
  C->>CLI: ariel verify / generate
  CLI-->>HP: ariel walkthrough
  HP->>HR: ariel walkthrough`

func TestExtractGraph_SequenceNodes(t *testing.T) {
	nodes, _ := ExtractGraph(seqDiagram)

	want := map[string]string{
		"HP":  "Human Prompter",
		"C":   "Claude",
		"CLI": "ariel CLI",
		"HR":  "Human Reviewer",
	}

	for id, wantLabel := range want {
		gotLabel, ok := nodes[id]
		if !ok {
			t.Errorf("node %q not found", id)
			continue
		}
		if gotLabel != wantLabel {
			t.Errorf("node %q: got label %q, want %q", id, gotLabel, wantLabel)
		}
	}

	for id := range nodes {
		if _, ok := want[id]; !ok {
			t.Errorf("unexpected node %q in result", id)
		}
	}
}

func TestExtractGraph_SequenceEdges(t *testing.T) {
	_, edges := ExtractGraph(seqDiagram)

	wantEdges := map[[2]string]bool{
		{"HP", "C"}:   true,
		{"C", "C"}:    true,
		{"C", "CLI"}:  true,
		{"CLI", "C"}:  true,
		{"C", "CLI"}:  true,
		{"CLI", "HP"}: true,
		{"HP", "HR"}:  true,
	}

	got := make(map[[2]string]bool)
	for _, e := range edges {
		got[e] = true
	}

	for e := range wantEdges {
		if !got[e] {
			t.Errorf("missing edge %v->%v", e[0], e[1])
		}
	}
}

func TestExtractGraph_SequenceImplicitParticipants(t *testing.T) {
	// Participants discovered from message lines (no explicit declarations).
	diagram := `sequenceDiagram
  A->>B: hello
  B-->>A: world`

	nodes, edges := ExtractGraph(diagram)

	for _, id := range []string{"A", "B"} {
		if label, ok := nodes[id]; !ok {
			t.Errorf("node %q not found", id)
		} else if label != id {
			t.Errorf("node %q: got label %q, want %q", id, label, id)
		}
	}

	if len(edges) != 2 {
		t.Errorf("got %d edges, want 2", len(edges))
	}
}

func TestDetectDiagramType(t *testing.T) {
	cases := []struct {
		diagram string
		want    string
	}{
		{"sequenceDiagram\n  A->>B: hi", "sequence"},
		{"graph TD\n  A-->B", "flowchart"},
		{"flowchart LR\n  A-->B", "flowchart"},
		{"%% comment\nsequenceDiagram\n  A->>B: hi", "sequence"},
		{"pie title Pets\n  \"Dogs\" : 386", "unsupported"},
		{"classDiagram\n  class Foo", "unsupported"},
		{"stateDiagram-v2\n  s1 --> s2", "unsupported"},
		{"erDiagram\n  FOO ||--o{ BAR : has", "unsupported"},
		{"gantt\n  title A", "unsupported"},
		{"gitGraph\n  commit", "unsupported"},
	}
	for _, c := range cases {
		if got := detectDiagramType(c.diagram); got != c.want {
			t.Errorf("detectDiagramType(%q): got %q, want %q", c.diagram[:20], got, c.want)
		}
	}
}
