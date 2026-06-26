package dsl

import "fmt"

// Verify runs semantic checks against the extracted graph.
// The caller supplies nodes and edges — this does not re-parse the diagram.
func Verify(steps []Step, nodes map[string]string, edges [][2]string) []Issue {
	edgeSet := buildEdgeSet(edges)

	var issues []Issue
	for i, step := range steps {
		stepNum := i + 1

		if i == 0 && (len(step.HighlightNodes) > 0 || len(step.FocusNodes) > 0) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Message:  `step 1: the first step of each section is the overview — it may only use "label" and "narration"`,
			})
		}

		issues = append(issues, verifyNodeRefs("highlight_nodes", step.HighlightNodes, stepNum, nodes)...)
		issues = append(issues, verifyNodeRefs("focus_nodes", step.FocusNodes, stepNum, nodes)...)
		issues = append(issues, disconnectedHighlightWarning(step, stepNum, nodes, edgeSet)...)

		if isEmpty(step) {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("step %d has no narration and no visual changes", stepNum),
			})
		}
	}

	return issues
}

// verifyNodeRefs reports an error for each ID in ids that does not exist in nodes.
func verifyNodeRefs(field string, ids []string, stepNum int, nodes map[string]string) []Issue {
	var issues []Issue
	for _, id := range ids {
		if _, ok := nodes[id]; !ok {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Message:  fmt.Sprintf("step %d: %s references unknown node ID %q", stepNum, field, id),
			})
		}
	}
	return issues
}

// disconnectedHighlightWarning returns a single warning if the nodes referenced by a
// step's highlight_nodes and focus_nodes do not form a single connected component when
// traversing direct diagram edges in either direction.
// Skips unknown node IDs (already reported as errors by other checks).
func disconnectedHighlightWarning(step Step, stepNum int, nodes map[string]string, edgeSet map[[2]string]bool) []Issue {
	seen := make(map[string]bool)
	var refs []string

	add := func(id string) {
		if _, ok := nodes[id]; ok && !seen[id] {
			refs = append(refs, id)
			seen[id] = true
		}
	}
	for _, id := range step.HighlightNodes {
		add(id)
	}
	for _, id := range step.FocusNodes {
		add(id)
	}

	if len(refs) < 2 {
		return nil
	}

	// BFS over the referenced nodes using undirected diagram edges.
	visited := make(map[string]bool)
	queue := []string{refs[0]}
	visited[refs[0]] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, other := range refs {
			if !visited[other] && (edgeSet[[2]string{cur, other}] || edgeSet[[2]string{other, cur}]) {
				visited[other] = true
				queue = append(queue, other)
			}
		}
	}

	if len(visited) < len(refs) {
		return []Issue{{
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("step %d: not all highlighted components are connected", stepNum),
		}}
	}
	return nil
}

// buildEdgeSet converts an edge list to a map for O(1) lookup.
func buildEdgeSet(edges [][2]string) map[[2]string]bool {
	set := make(map[[2]string]bool, len(edges))
	for _, e := range edges {
		set[e] = true
	}
	return set
}

// isEmpty reports true when a step has no content — no label, narration, or visual changes.
func isEmpty(s Step) bool {
	return s.Narration == "" &&
		s.Label == "" &&
		len(s.HighlightNodes) == 0 &&
		len(s.FocusNodes) == 0
}
