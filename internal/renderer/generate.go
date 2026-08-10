package renderer

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"os"
	"regexp"
	"strings"

	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/logo"
	"github.com/scottrogowski/ariel/internal/mermaidjs"
	"github.com/scottrogowski/ariel/internal/renderadapter"
	"github.com/scottrogowski/ariel/internal/theme"
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
	DiagramType    string            `json:"diagram_type"`
	AnimateEdges   bool              `json:"animate_edges"`
	NodeLabels     map[string]string `json:"node_labels"`
	Steps          []jsStep          `json:"steps"`
}

type templateData struct {
	Title           string
	GitHubURL       string
	Sections        []jsSection
	LogoSVG         template.HTML
	FaviconURL      template.URL
	MermaidJSURL    template.URL
	ThemeCSS        template.CSS
	MermaidConfigJS template.JS
	RenderAdapterJS template.JS
	ThemeListener   template.JS
	WSSnippet       template.HTML
}

var tmpl = template.Must(
	template.New("ariel").Delims("[[", "]]").Parse(htmlTemplate),
)

// Generate renders a Walkthrough to a self-contained, server-free HTML string.
func Generate(w *dsl.Walkthrough, mode theme.Mode) (string, error) {
	return render(w, "", mode)
}

// RenderWatch renders a Walkthrough with the WebSocket client snippet injected.
func RenderWatch(w *dsl.Walkthrough, port int, mode theme.Mode) (string, error) {
	srv := &WatchServer{port: port}
	return render(w, srv.wsSnippet(), mode)
}

// render is the shared path for Generate and RenderWatch.
func render(w *dsl.Walkthrough, wsSnippet string, mode theme.Mode) (string, error) {
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

		analysis := dsl.AnalyzeDiagram(sec.MermaidDiagram)
		jsSections[i] = jsSection{
			Title:          sec.Title,
			MermaidDiagram: strings.TrimRight(sec.MermaidDiagram, "\n"),
			DiagramType:    string(analysis.Kind),
			AnimateEdges:   analysis.AnimateEdges,
			NodeLabels:     analysis.Nodes,
			Steps:          steps,
		}
	}

	data := templateData{
		Title:           w.Title,
		GitHubURL:       "https://github.com/scottrogowski/ariel",
		Sections:        jsSections,
		LogoSVG:         template.HTML(logo.SVG),
		FaviconURL:      template.URL("data:image/svg+xml;base64," + logo.FaviconBase64()),
		MermaidJSURL:    template.URL(mermaidjs.BrowserScriptURL()),
		ThemeCSS:        template.CSS(theme.HTMLRootCSS(mode)),
		MermaidConfigJS: template.JS(theme.HTMLMermaidConfigJS(mode)),
		RenderAdapterJS: template.JS(renderadapter.InlineJavaScript()),
		ThemeListener:   template.JS(theme.HTMLThemeListenerJS(mode)),
		WSSnippet:       template.HTML(wsSnippet),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return trimLineEndWhitespace(buf.String()), nil
}

func trimLineEndWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
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
