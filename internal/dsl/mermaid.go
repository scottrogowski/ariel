package dsl

import "strings"

// DiagramKind identifies a Mermaid diagram family.
type DiagramKind string

const (
	DiagramKindFlowchart   DiagramKind = "flowchart"
	DiagramKindSequence    DiagramKind = "sequence"
	DiagramKindClass       DiagramKind = "class"
	DiagramKindUnsupported DiagramKind = "unsupported"
)

// DiagramAnalysis contains the walkthrough targets and relationships in a Mermaid diagram.
type DiagramAnalysis struct {
	Kind         DiagramKind
	Nodes        map[string]string
	Edges        [][2]string
	AnimateEdges bool
}

type graphExtractor func([]string) (map[string]string, [][2]string)

type diagramAdapter struct {
	kind         DiagramKind
	matches      func(string) bool
	extractor    graphExtractor
	animateEdges bool
}

var diagramAdapters = []diagramAdapter{
	{
		kind: DiagramKindSequence,
		matches: func(header string) bool {
			return strings.HasPrefix(header, "sequencediagram")
		},
		extractor: extractSequenceGraph,
	},
	{
		kind: DiagramKindClass,
		matches: func(header string) bool {
			return header == "classdiagram" || header == "classdiagram-v2"
		},
		extractor:    extractClassGraph,
		animateEdges: true,
	},
	{
		kind: DiagramKindFlowchart,
		matches: func(header string) bool {
			return strings.HasPrefix(header, "graph ") || strings.HasPrefix(header, "graph\t") ||
				header == "graph" || strings.HasPrefix(header, "flowchart ") || strings.HasPrefix(header, "flowchart\t")
		},
		extractor:    extractFlowchartGraph,
		animateEdges: true,
	},
}

// DiagramType returns the Mermaid diagram family.
func DiagramType(diagram string) string {
	return string(AnalyzeDiagram(diagram).Kind)
}

func detectDiagramType(diagram string) string {
	return DiagramType(diagram)
}

// AnalyzeDiagram selects the diagram adapter and extracts its graph.
func AnalyzeDiagram(diagram string) DiagramAnalysis {
	lines := strings.Split(diagram, "\n")
	header := firstDiagramLine(lines)
	if header == "" {
		return DiagramAnalysis{Kind: DiagramKindFlowchart, Nodes: make(map[string]string)}
	}

	for _, adapter := range diagramAdapters {
		if !adapter.matches(header) {
			continue
		}
		nodes, edges := adapter.extractor(lines)
		return DiagramAnalysis{Kind: adapter.kind, Nodes: nodes, Edges: edges, AnimateEdges: adapter.animateEdges}
	}

	return DiagramAnalysis{Kind: DiagramKindUnsupported, Nodes: make(map[string]string)}
}

func firstDiagramLine(lines []string) string {
	for _, line := range lines {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		return line
	}
	return ""
}

// ExtractGraph parses a Mermaid diagram and returns:
//   - nodes: map of node ID → display label (empty string for bare IDs)
//   - edges: list of [source, target] node ID pairs
//
// Returns an empty node map for unsupported diagram types.
func ExtractGraph(diagram string) (map[string]string, [][2]string) {
	analysis := AnalyzeDiagram(diagram)
	return analysis.Nodes, analysis.Edges
}
