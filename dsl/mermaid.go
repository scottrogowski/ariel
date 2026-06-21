package dsl

import (
	"regexp"
	"strings"
)

// nodeShapeRe matches a node ID followed by any Mermaid shape syntax.
// Order is significant: longer/more specific openers must come first.
var nodeShapeRe = regexp.MustCompile(
	`\b(\w+)` +
		`(?:` +
		`\[\[.*?\]\]` + // [[subroutine]]
		`|\[\(.*?\)\]` + // [(cylinder)]
		`|\(\[.*?\]\)` + // ([rounded])
		`|\(\(.*?\)\)` + // ((circle))
		`|>.*?\]` + // >asymmetric]
		`|\{[^}]*\}` + // {diamond}
		`|\[[^\]]*\]` + // [rectangle]
		`)`,
)

// edgeSyntaxRe matches any Mermaid edge connector.
var edgeSyntaxRe = regexp.MustCompile(`(?:--[->oOxX]|==+[>=]|-.->|<-->|<--)`)

// edgeLabelRe matches inline edge labels: |...|
var edgeLabelRe = regexp.MustCompile(`\|[^|]*\|`)

// nodeIDRe validates a bare node ID token.
var nodeIDRe = regexp.MustCompile(`^\w+$`)

// ExtractGraph parses a Mermaid diagram string and returns:
//   - nodes: map of node ID → display label (empty string for bare IDs)
//   - edges: list of [source, target] node ID pairs
func ExtractGraph(diagram string) (nodes map[string]string, edges [][2]string) {
	nodes = make(map[string]string)

	for _, line := range strings.Split(diagram, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Extract all node definitions with shapes on this line.
		for _, m := range nodeShapeRe.FindAllStringSubmatch(line, -1) {
			id := m[1]
			if _, exists := nodes[id]; !exists {
				nodes[id] = extractShapeLabel(m[0], id)
			}
		}

		if !edgeSyntaxRe.MatchString(line) {
			continue
		}

		// Simplify the line to extract edge source/target pairs:
		// replace each shaped node with just its ID, strip edge labels.
		simplified := nodeShapeRe.ReplaceAllStringFunc(line, func(match string) string {
			sub := nodeShapeRe.FindStringSubmatch(match)
			if sub != nil {
				return sub[1]
			}
			return match
		})
		simplified = edgeLabelRe.ReplaceAllString(simplified, " ")

		parts := edgeSyntaxRe.Split(simplified, -1)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		for i := 0; i+1 < len(parts); i++ {
			src, dst := parts[i], parts[i+1]
			if !nodeIDRe.MatchString(src) || !nodeIDRe.MatchString(dst) {
				continue
			}
			edges = append(edges, [2]string{src, dst})
			// Register bare IDs that may not have an explicit shape definition.
			if _, exists := nodes[src]; !exists {
				nodes[src] = ""
			}
			if _, exists := nodes[dst]; !exists {
				nodes[dst] = ""
			}
		}
	}

	return nodes, edges
}

// extractShapeLabel strips the node ID and shape delimiters to return the display label.
func extractShapeLabel(nodeStr, id string) string {
	shape := nodeStr[len(id):]
	for _, delims := range [][2]string{
		{"[[", "]]"}, {"[(", ")]"}, {"([", "])"}, {"((", "))"},
		{">", "]"}, {"{", "}"}, {"[", "]"},
	} {
		if strings.HasPrefix(shape, delims[0]) && strings.HasSuffix(shape, delims[1]) {
			return strings.TrimSpace(shape[len(delims[0]) : len(shape)-len(delims[1])])
		}
	}
	return ""
}
