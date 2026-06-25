package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/scottmrogowski/ariel/internal/assets"
	"github.com/scottmrogowski/ariel/internal/dsl"
)

type jsStep struct {
	Label          string   `json:"label"`
	Narration      string   `json:"narration"`
	HighlightNodes []string `json:"highlight_nodes"`
	ActiveNodes    []string `json:"active_nodes"`
	AnimateEdges   []string `json:"animate_edges"`
}

type jsSection struct {
	Title          string   `json:"title"`
	MermaidDiagram string   `json:"mermaid_diagram"`
	Steps          []jsStep `json:"steps"`
}

type templateData struct {
	Title        string
	GitHubURL    string
	LogoDataURI  string
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
				Narration:      s.Narration,
				HighlightNodes: nonNil(s.HighlightNodes),
				ActiveNodes:    nonNil(s.ActiveNodes),
				AnimateEdges:   nonNil(s.AnimateEdges),
			}
		}

		jsSections[i] = jsSection{
			Title:          sec.Title,
			MermaidDiagram: strings.TrimRight(sec.MermaidDiagram, "\n"),
			Steps:          steps,
		}
	}

	sectionsJSON, err := json.Marshal(jsSections)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sections: %w", err)
	}

	data := templateData{
		Title:        w.Title,
		GitHubURL:    "https://github.com/scottmrogowski/ariel",
		LogoDataURI:  assets.ArielLogoDataURI,
		SectionsJSON: string(sectionsJSON),
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
