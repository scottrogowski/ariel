package dsl

import "strings"

// DiagramType returns the Mermaid diagram family ("flowchart" or "sequence").
// Only these two types support highlight_nodes and focus_nodes.
func DiagramType(diagram string) string {
	return detectDiagramType(diagram)
}

func detectDiagramType(diagram string) string {
	for _, line := range strings.Split(diagram, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if strings.HasPrefix(line, "sequencediagram") {
			return "sequence"
		}
		if strings.HasPrefix(line, "graph ") || strings.HasPrefix(line, "graph\t") ||
			line == "graph" || strings.HasPrefix(line, "flowchart ") || strings.HasPrefix(line, "flowchart\t") {
			return "flowchart"
		}
		return "unsupported"
	}
	return "flowchart"
}

// ExtractGraph parses a Mermaid diagram and returns:
//   - nodes: map of node ID → display label (empty string for bare IDs)
//   - edges: list of [source, target] node ID pairs
//
// Returns empty maps for unsupported diagram types.
func ExtractGraph(diagram string) (map[string]string, [][2]string) {
	lines := strings.Split(diagram, "\n")
	switch detectDiagramType(diagram) {
	case "sequence":
		return extractSequenceGraph(lines)
	case "flowchart":
		return extractFlowchartGraph(lines)
	default:
		return make(map[string]string), nil
	}
}
