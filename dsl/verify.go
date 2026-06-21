package dsl

import (
	"fmt"
	"strings"
)

// Verify runs semantic checks on a parsed walkthrough against its extracted graph.
// It does not re-parse the mermaid diagram; the caller supplies nodes and edges.
func Verify(w *Walkthrough, nodes map[string]string, edges [][2]string) []Issue {
	edgeSet := buildEdgeSet(edges)

	var issues []Issue
	for i, step := range w.Steps {
		stepNum := i + 1

		for _, id := range step.HighlightNodes {
			if _, ok := nodes[id]; !ok {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("step %d: highlight_nodes references unknown node ID %q", stepNum, id),
				})
			}
		}

		for _, id := range step.ActiveNodes {
			if _, ok := nodes[id]; !ok {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("step %d: active_nodes references unknown node ID %q", stepNum, id),
				})
			}
		}

		for _, edgeRef := range step.AnimateEdges {
			src, dst, ok := splitEdgeRef(edgeRef)
			if !ok {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("step %d: animate_edges entry %q is not in SOURCE_ID-TARGET_ID format", stepNum, edgeRef),
				})
				continue
			}
			if _, nodeOK := nodes[src]; !nodeOK {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("step %d: animate_edges references unknown node ID %q", stepNum, src),
				})
				continue
			}
			if _, nodeOK := nodes[dst]; !nodeOK {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("step %d: animate_edges references unknown node ID %q", stepNum, dst),
				})
				continue
			}
			if !edgeSet[[2]string{src, dst}] {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf(`step %d: animate_edges references edge %q which does not exist in mermaid_diagram`, stepNum, edgeRef),
				})
			}
		}

		if isEmpty(step) {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("step %d has no narration and no visual changes", stepNum),
			})
		}
	}

	return issues
}

// buildEdgeSet converts an edge list to a set for O(1) lookup.
func buildEdgeSet(edges [][2]string) map[[2]string]bool {
	set := make(map[[2]string]bool, len(edges))
	for _, e := range edges {
		set[e] = true
	}
	return set
}

// splitEdgeRef splits "SRC-DST" into ("SRC", "DST", true).
// Splits on the first hyphen; node IDs must not contain hyphens (per Mermaid spec).
func splitEdgeRef(ref string) (src, dst string, ok bool) {
	idx := strings.Index(ref, "-")
	if idx <= 0 || idx == len(ref)-1 {
		return "", "", false
	}
	return ref[:idx], ref[idx+1:], true
}

// isEmpty returns true if a step has neither narration nor any visual change.
func isEmpty(s Step) bool {
	return s.Narration == "" &&
		s.Label == "" &&
		len(s.HighlightNodes) == 0 &&
		len(s.ActiveNodes) == 0 &&
		len(s.AnimateEdges) == 0
}
