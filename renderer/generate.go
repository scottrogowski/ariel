package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/scottmrogowski/ariel/dsl"
)

// jsStep is the JSON shape consumed by the browser step player.
type jsStep struct {
	Label          string   `json:"label"`
	Narration      string   `json:"narration"`
	HighlightNodes []string `json:"highlight_nodes"`
	ActiveNodes    []string `json:"active_nodes"`
	AnimateEdges   []string `json:"animate_edges"`
}

type templateData struct {
	Title          string
	MermaidDiagram string
	StepsJSON      string
	NodeLabelsJSON string
	WSSnippet      string // empty for generate, populated for watch
}

var tmpl = template.Must(
	template.New("ariel").Delims("[[", "]]").Parse(htmlTemplate),
)

// Generate renders a Walkthrough to a self-contained HTML string.
func Generate(w *dsl.Walkthrough) (string, error) {
	return render(w, "")
}

// RenderWatch renders a Walkthrough with the websocket client snippet injected.
func RenderWatch(w *dsl.Walkthrough, port int) (string, error) {
	srv := &WatchServer{port: port}
	return render(w, srv.wsSnippet())
}

// render is the shared rendering path for both generate and watch.
// wsSnippet is injected before </body> for the watch websocket client.
func render(w *dsl.Walkthrough, wsSnippet string) (string, error) {
	nodes, _ := dsl.ExtractGraph(w.MermaidDiagram)

	nodeLabelsJSON, err := json.Marshal(nodes)
	if err != nil {
		return "", fmt.Errorf("failed to marshal node labels: %w", err)
	}

	steps := make([]jsStep, len(w.Steps))
	for i, s := range w.Steps {
		steps[i] = jsStep{
			Label:          s.Label,
			Narration:      s.Narration,
			HighlightNodes: nonNil(s.HighlightNodes),
			ActiveNodes:    nonNil(s.ActiveNodes),
			AnimateEdges:   nonNil(s.AnimateEdges),
		}
	}

	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return "", fmt.Errorf("failed to marshal steps: %w", err)
	}

	data := templateData{
		Title:          w.Title,
		MermaidDiagram: strings.TrimRight(w.MermaidDiagram, "\n"),
		StepsJSON:      string(stepsJSON),
		NodeLabelsJSON: string(nodeLabelsJSON),
		WSSnippet:      wsSnippet,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// WriteFile writes generated HTML to path, creating or truncating the file.
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
