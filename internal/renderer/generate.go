package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/scottrogowski/ariel/internal/dsl"
)

var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

// renderNarration converts [text](url) markdown links to HTML <a> tags.
// All non-link text is HTML-escaped.
func renderNarration(text string) string {
	matches := mdLinkRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return html.EscapeString(text)
	}
	var out strings.Builder
	last := 0
	for _, m := range matches {
		out.WriteString(html.EscapeString(text[last:m[0]]))
		fmt.Fprintf(&out, `<a href="%s" target="_blank" rel="noopener">%s</a>`,
			html.EscapeString(text[m[4]:m[5]]),
			html.EscapeString(text[m[2]:m[3]]),
		)
		last = m[1]
	}
	out.WriteString(html.EscapeString(text[last:]))
	return out.String()
}

type jsStep struct {
	Label          string   `json:"label"`
	Narration      string   `json:"narration"`
	HighlightNodes []string `json:"highlight_nodes"`
	FocusNodes     []string `json:"focus_nodes"`
}

type jsSection struct {
	Title          string            `json:"title"`
	MermaidDiagram string            `json:"mermaid_diagram"`
	NodeLabels     map[string]string `json:"node_labels"`
	Steps          []jsStep          `json:"steps"`
}

type templateData struct {
	Title        string
	GitHubURL    string
	SectionsJSON string
	WSSnippet    string // empty for generate, populated for watch
}

var tmpl = template.Must(
	template.New("ariel").Delims("[[", "]]").Parse(htmlTemplate),
)

// Generate renders a Walkthrough to a self-contained, server-free HTML string.
func Generate(w *dsl.Walkthrough) (string, error) {
	return render(w, "")
}

// RenderWatch renders a Walkthrough with the WebSocket client snippet injected.
func RenderWatch(w *dsl.Walkthrough, port int) (string, error) {
	srv := &WatchServer{port: port}
	return render(w, srv.wsSnippet())
}

// render is the shared path for Generate and RenderWatch: serializes sections to JSON
// and executes the HTML template.
func render(w *dsl.Walkthrough, wsSnippet string) (string, error) {
	sections := w.ToSections()
	jsSections := make([]jsSection, len(sections))

	for i, sec := range sections {
		steps := make([]jsStep, len(sec.Steps))
		for j, s := range sec.Steps {
			steps[j] = jsStep{
				Label:          s.Label,
				Narration:      renderNarration(s.Narration),
				HighlightNodes: nonNil(s.HighlightNodes),
				FocusNodes:     nonNil(s.FocusNodes),
			}
		}

		nodeLabels, _ := dsl.ExtractGraph(sec.MermaidDiagram)
		jsSections[i] = jsSection{
			Title:          sec.Title,
			MermaidDiagram: strings.TrimRight(sec.MermaidDiagram, "\n"),
			NodeLabels:     nodeLabels,
			Steps:          steps,
		}
	}

	var jsonBuf bytes.Buffer
	enc := json.NewEncoder(&jsonBuf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(jsSections); err != nil {
		return "", fmt.Errorf("failed to marshal sections: %w", err)
	}

	data := templateData{
		Title:        w.Title,
		GitHubURL:    "https://github.com/scottrogowski/ariel",
		SectionsJSON: strings.TrimRight(jsonBuf.String(), "\n"),
		WSSnippet:    wsSnippet,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// WriteFile writes html to path, creating or truncating the file.
func WriteFile(path, html string) error {
	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// nonNil returns an empty slice instead of nil so JSON serializes as [] not null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
