package dsl

import (
	"regexp"
	"strings"
)

// seqParticipantRe matches participant/actor declarations.
// Group 1: ID, Group 2: alias label (may be empty).
var seqParticipantRe = regexp.MustCompile(`(?i)^\s*(?:participant|actor)\s+(\w+)(?:\s+as\s+(.+))?$`)

// seqMsgRe matches sequence diagram message lines. Arrow tokens ordered longest-first to avoid
// partial matches (e.g. "-->" must not match before "-->>" is tried).
// Group 1: sender ID, Group 2: receiver ID.
var seqMsgRe = regexp.MustCompile(`^\s*(\w+)\s*(?:-->>|--x|--\)|-->|->>|-x|-\)|->)\+?\s*(\w+)\s*:`)

// extractSequenceGraph parses sequenceDiagram syntax.
func extractSequenceGraph(lines []string) (map[string]string, [][2]string) {
	nodes := make(map[string]string)
	var edges [][2]string

	for _, line := range lines {
		if m := seqParticipantRe.FindStringSubmatch(line); m != nil {
			id := m[1]
			label := strings.TrimSpace(m[2])
			if label == "" {
				label = id
			}
			if _, exists := nodes[id]; !exists {
				nodes[id] = label
			}
			continue
		}
		if m := seqMsgRe.FindStringSubmatch(line); m != nil {
			src, dst := m[1], m[2]
			edges = append(edges, [2]string{src, dst})
			if _, exists := nodes[src]; !exists {
				nodes[src] = src
			}
			if _, exists := nodes[dst]; !exists {
				nodes[dst] = dst
			}
		}
	}
	return nodes, edges
}
