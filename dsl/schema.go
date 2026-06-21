package dsl

type Walkthrough struct {
	Title          string `yaml:"title"`
	MermaidDiagram string `yaml:"mermaid_diagram"`
	Steps          []Step `yaml:"steps"`
}

type Step struct {
	Label          string   `yaml:"label"`
	Narration      string   `yaml:"narration"`
	HighlightNodes []string `yaml:"highlight_nodes"`
	ActiveNodes    []string `yaml:"active_nodes"`
	AnimateEdges   []string `yaml:"animate_edges"`
}

type IssueSeverity string

const (
	SeverityError   IssueSeverity = "error"
	SeverityWarning IssueSeverity = "warning"
)

type Issue struct {
	Line     int
	Severity IssueSeverity
	Message  string
}
