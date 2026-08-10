package dsl

import (
	"regexp"
	"strings"
)

var (
	classDeclarationRe    = regexp.MustCompile(`^class\s+(` + classNamePattern + `)(?:\s*\["([^"]*)"\])?`)
	classMemberRe         = regexp.MustCompile(`^(` + classNamePattern + `)\s*:\s*.+$`)
	classAnnotationRe     = regexp.MustCompile(`^<<[^>]+>>\s*(` + classNamePattern + `)\s*$`)
	classNameRe           = regexp.MustCompile(`^` + classNamePattern + `$`)
	classRelationRe       = regexp.MustCompile(`(?:<\||\(\)|[<*o])?(?:--|\.\.)(?:\|>|\(\)|[>*o])?`)
	trailingCardinalityRe = regexp.MustCompile(`\s*"[^"]*"\s*$`)
	leadingCardinalityRe  = regexp.MustCompile(`^\s*"[^"]*"\s*`)
)

const classNamePattern = "(?:`[^`]+`|[^\\s\\[\\]{}:]+(?:~[^~\\r\\n]+~)?)"

func extractClassGraph(lines []string) (map[string]string, [][2]string) {
	nodes := make(map[string]string)
	var edges [][2]string
	classBodyDepth := 0
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if classBodyDepth > 0 {
			classBodyDepth += strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}
		if match := classDeclarationRe.FindStringSubmatch(line); match != nil {
			id := normalizeClassName(match[1])
			registerClass(nodes, id, strings.TrimSpace(match[2]))
			classBodyDepth = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}
		if source, target, ok := classRelationship(line); ok {
			registerClass(nodes, source, "")
			registerClass(nodes, target, "")
			edges = append(edges, [2]string{source, target})
			continue
		}
		if match := classMemberRe.FindStringSubmatch(line); match != nil {
			registerClass(nodes, normalizeClassName(match[1]), "")
			continue
		}
		if match := classAnnotationRe.FindStringSubmatch(line); match != nil {
			registerClass(nodes, normalizeClassName(match[1]), "")
		}
	}

	return nodes, edges
}

func classRelationship(line string) (string, string, bool) {
	operator := classRelationRe.FindStringIndex(maskBacktickContent(line))
	if operator == nil {
		return "", "", false
	}
	left := trailingCardinalityRe.ReplaceAllString(line[:operator[0]], "")
	right := leadingCardinalityRe.ReplaceAllString(line[operator[1]:], "")
	if labelStart := strings.Index(maskBacktickContent(right), ":"); labelStart >= 0 {
		right = right[:labelStart]
	}

	source, sourceOK := parseClassName(left)
	target, targetOK := parseClassName(right)
	if !sourceOK || !targetOK {
		return "", "", false
	}
	return source, target, true
}

func maskBacktickContent(value string) string {
	masked := []byte(value)
	insideBackticks := false
	for i := range masked {
		if masked[i] == '`' {
			insideBackticks = !insideBackticks
			continue
		}
		if insideBackticks {
			masked[i] = ' '
		}
	}
	return string(masked)
}

func parseClassName(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	match := classNameRe.FindString(value)
	if match == "" {
		return "", false
	}
	return normalizeClassName(match), true
}

func normalizeClassName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	if genericStart := strings.Index(value, "~"); genericStart >= 0 {
		value = value[:genericStart]
	}
	return value
}

func registerClass(nodes map[string]string, id, label string) {
	if id == "" {
		return
	}
	if label != "" {
		nodes[id] = label
		return
	}
	if _, exists := nodes[id]; !exists {
		nodes[id] = id
	}
}
