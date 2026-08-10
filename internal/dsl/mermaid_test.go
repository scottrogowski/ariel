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

// This test prevents class syntax from producing missing or spurious walkthrough nodes.
func TestExtractGraph_ClassNodes(t *testing.T) {
	diagram := "" +
		"classDiagram\n" +
		"  class Service {\n" +
		"    +Run() error\n" +
		"  }\n" +
		"  class Store[\"Data Store\"]\n" +
		"  Store : +Save() error\n" +
		"  Queue : +Push(item Item)\n" +
		"  Service --> Store : writes\n" +
		"  Cache ..> Store : reads\n" +
		"  class `Job Runner`\n" +
		"  class List~Item~\n" +
		"  <<interface>> Repository\n" +
		"  namespace data {\n" +
		"    class Record\n" +
		"  }\n" +
		"  note for Service \"Coordinates work\"\n" +
		"  direction LR\n"

	nodes, _ := ExtractGraph(diagram)
	want := map[string]string{
		"Service":    "Service",
		"Store":      "Data Store",
		"Queue":      "Queue",
		"Cache":      "Cache",
		"Job Runner": "Job Runner",
		"List":       "List",
		"Repository": "Repository",
		"Record":     "Record",
	}
	if len(nodes) != len(want) {
		t.Fatalf("ExtractGraph() nodes = %#v, want %#v", nodes, want)
	}
	for id, wantLabel := range want {
		if got := nodes[id]; got != wantLabel {
			t.Errorf("node %q label = %q, want %q", id, got, wantLabel)
		}
	}
}

// This test prevents supported class relationship forms from breaking connectivity checks.
func TestExtractGraph_ClassRelationships(t *testing.T) {
	tests := []struct {
		name     string
		relation string
		source   string
		target   string
	}{
		{name: "inheritance", relation: "Animal <|-- Duck", source: "Animal", target: "Duck"},
		{name: "composition", relation: "Company *-- Department", source: "Company", target: "Department"},
		{name: "aggregation", relation: "Pond o-- Duck", source: "Pond", target: "Duck"},
		{name: "association", relation: "Owner -- Pet", source: "Owner", target: "Pet"},
		{name: "dependency", relation: "Service ..> Store", source: "Service", target: "Store"},
		{name: "realization", relation: "Service ..|> Runner", source: "Service", target: "Runner"},
		{name: "bidirectional", relation: "A <--> B", source: "A", target: "B"},
		{name: "lollipop", relation: "Service --() Port", source: "Service", target: "Port"},
		{name: "cardinality and label", relation: `Customer "1" --> "*" Order : places`, source: "Customer", target: "Order"},
		{name: "no spaces", relation: "Source-->Target", source: "Source", target: "Target"},
		{name: "backtick punctuation", relation: "`Job--Runner` --> `Data:Store`", source: "Job--Runner", target: "Data:Store"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, edges := ExtractGraph("classDiagram\n  " + tt.relation)
			if len(edges) != 1 {
				t.Fatalf("ExtractGraph() edges = %#v, want one edge", edges)
			}
			if edges[0] != [2]string{tt.source, tt.target} {
				t.Errorf("ExtractGraph() edge = %#v, want %q -> %q", edges[0], tt.source, tt.target)
			}
		})
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
		{"classDiagram\n  class Foo", "class"},
		{"classDiagram-v2\n  class Foo", "class"},
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
