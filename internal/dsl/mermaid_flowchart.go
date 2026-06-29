package dsl

import (
	"regexp"
	"strings"
)

// nodeShapeRe matches a node ID followed by any Mermaid shape syntax.
// Order matters: longer/more specific openers must come before shorter overlapping ones.
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

var nodeIDRe = regexp.MustCompile(`^\w+$`)

// extractFlowchartGraph parses flowchart/graph diagram syntax.
func extractFlowchartGraph(lines []string) (map[string]string, [][2]string) {
	nodes := make(map[string]string)
	var edges [][2]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		for _, m := range nodeShapeRe.FindAllStringSubmatch(line, -1) {
			id := m[1]
			if _, exists := nodes[id]; !exists {
				nodes[id] = extractShapeLabel(m[0], id)
			}
		}

		for _, e := range edgesFromLine(line) {
			edges = append(edges, e)
			// Register bare node IDs that have no explicit shape definition.
			if _, exists := nodes[e[0]]; !exists {
				nodes[e[0]] = ""
			}
			if _, exists := nodes[e[1]]; !exists {
				nodes[e[1]] = ""
			}
		}
	}
	return nodes, edges
}

// edgesFromLine extracts source→target node ID pairs from a line containing edge syntax.
// Replaces shaped nodes with bare IDs and strips edge labels before splitting on connectors.
func edgesFromLine(line string) [][2]string {
	if !edgeSyntaxRe.MatchString(line) {
		return nil
	}
	simplified := nodeShapeRe.ReplaceAllStringFunc(line, func(match string) string {
		if sub := nodeShapeRe.FindStringSubmatch(match); sub != nil {
			return sub[1]
		}
		return match
	})
	simplified = edgeLabelRe.ReplaceAllString(simplified, " ")

	parts := edgeSyntaxRe.Split(simplified, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	var pairs [][2]string
	for i := 0; i+1 < len(parts); i++ {
		src, dst := parts[i], parts[i+1]
		if nodeIDRe.MatchString(src) && nodeIDRe.MatchString(dst) {
			pairs = append(pairs, [2]string{src, dst})
		}
	}
	return pairs
}

// extractShapeLabel strips the shape delimiters from a matched node string to return
// the inner display label, e.g. "API[Auth API]" → "Auth API".
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
