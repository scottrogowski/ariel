package dsl

type Walkthrough struct {
	Title string `yaml:"title"`
	// Multi-diagram format.
	Sections []Section `yaml:"sections,omitempty"`
	// Single-diagram convenience format; normalized to Sections via ToSections().
	MermaidDiagram string `yaml:"mermaid_diagram,omitempty"`
	Steps          []Step `yaml:"steps,omitempty"`
}

// ToSections returns the effective sections regardless of which format was used.
func (w *Walkthrough) ToSections() []Section {
	if len(w.Sections) > 0 {
		return w.Sections
	}
	return []Section{{MermaidDiagram: w.MermaidDiagram, Steps: w.Steps}}
}

type Section struct {
	Title          string `yaml:"title,omitempty"`
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
