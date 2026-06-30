package svgformat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/scottmrogowski/ariel/internal/dsl"
)

const (
	outputWidth  = 960
	ctaHeight    = 40
	navHeight    = 60
	svgTimeout   = 5 * time.Minute
	browserWidth = 980 // slightly wider than outputWidth to avoid triggering a scrollbar
	browserHeight = 2000
)

// Generate renders a single-section Walkthrough as an interactive SVG file at outPath.
// The output SVG uses foreignObject + CSS :checked for step navigation — interactive
// when opened in GitHub's SVG viewer, static when embedded as <img>.
func Generate(w *dsl.Walkthrough, outPath string) error {
	sections := w.ToSections()
	if len(sections) > 1 {
		return fmt.Errorf("svg format does not support multi-section walkthroughs; flatten to a single section first")
	}
	sec := sections[0]

	tmpDir, err := os.MkdirTemp("", "ariel-svg-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := newBrowserCtx()
	defer cancel()

	stepSVGs := make([]string, len(sec.Steps))
	var maxDiagramHeight int

	for i, step := range sec.Steps {
		diagram := appendNarrationNode(sec.MermaidDiagram, step.Narration)
		htmlPath := filepath.Join(tmpDir, fmt.Sprintf("step%d.html", i))
		if err := os.WriteFile(htmlPath, []byte(renderExtractionHTML(diagram)), 0644); err != nil {
			return fmt.Errorf("step %d: write extraction HTML: %w", i, err)
		}

		if err := chromedp.Run(ctx,
			chromedp.Navigate("file://"+htmlPath),
			chromedp.WaitVisible("#ready", chromedp.ByID),
		); err != nil {
			return fmt.Errorf("step %d: load: %w", i, err)
		}

		hJSON, _ := json.Marshal(strSlice(step.HighlightNodes))
		fJSON, _ := json.Marshal(strSlice(step.FocusNodes))
		if err := chromedp.Run(ctx, chromedp.Evaluate(
			fmt.Sprintf(`applyStep(%s,%s)`, hJSON, fJSON), nil,
		)); err != nil {
			return fmt.Errorf("step %d: applyStep: %w", i, err)
		}

		var svgStr string
		if err := chromedp.Run(ctx, chromedp.Evaluate(`getSVG()`, &svgStr)); err != nil {
			return fmt.Errorf("step %d: getSVG: %w", i, err)
		}
		if !strings.HasPrefix(svgStr, "<svg") {
			return fmt.Errorf("step %d: getSVG returned unexpected content (first 60 chars): %q", i, truncate(svgStr, 60))
		}
		// Mermaid renders HTML void elements (e.g. <br>) inside foreignObject
		// without the closing slash, which is valid HTML but invalid XML. The
		// output file is parsed as XML, so fix them up here.
		svgStr = strings.ReplaceAll(svgStr, "<br>", "<br/>")
		stepSVGs[i] = svgStr

		var dimsJSON string
		if err := chromedp.Run(ctx, chromedp.Evaluate(`getDimensions()`, &dimsJSON)); err != nil {
			return fmt.Errorf("step %d: getDimensions: %w", i, err)
		}
		var dims struct {
			H int `json:"h"`
		}
		if err := json.Unmarshal([]byte(dimsJSON), &dims); err != nil {
			return fmt.Errorf("step %d: parse dimensions: %w", i, err)
		}
		if dims.H <= 0 {
			return fmt.Errorf("step %d: diagram rendered with zero height — Mermaid may have failed to parse the diagram", i)
		}
		if dims.H > maxDiagramHeight {
			maxDiagramHeight = dims.H
		}
	}

	totalHeight := ctaHeight + maxDiagramHeight + navHeight
	out := buildOutputSVG(outputWidth, totalHeight, ctaHeight, maxDiagramHeight, navHeight, stepSVGs)
	return os.WriteFile(outPath, []byte(out), 0644)
}

// appendNarrationNode appends an unconnected Mermaid node carrying the narration
// text to flowchart/graph diagrams. Mermaid's layout engine places unconnected
// nodes in the emptiest available space. Other diagram types are returned unchanged.
func appendNarrationNode(diagram, narration string) string {
	if narration == "" {
		return diagram
	}
	trimmed := strings.TrimLeft(diagram, " \t\n")
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "flowchart") && !strings.HasPrefix(lower, "graph") {
		return diagram
	}
	// Replace double quotes with single quotes so the label parses correctly,
	// then wrap at word boundaries so the node doesn't render as one long line.
	label := wrapNarration(strings.ReplaceAll(narration, `"`, `'`), 40)
	return strings.TrimRight(diagram, "\n") +
		"\n    _narration_[\"" + label + "\"]\n" +
		"    style _narration_ fill:#1a2744,color:#e8eaf0,stroke:#5b8dee,stroke-width:2px"
}

// wrapNarration splits text at word boundaries, joining lines with Mermaid's
// <br/> line-break syntax for use inside ["..."] node labels.
func wrapNarration(text string, charsPerLine int) string {
	words := strings.Fields(text)
	var lines []string
	var current strings.Builder
	for _, word := range words {
		if current.Len() > 0 && current.Len()+1+len(word) > charsPerLine {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "<br/>")
}

func buildOutputSVG(width, totalHeight, ctaH, diagH, navH int, stepSVGs []string) string {
	n := len(stepSVGs)
	var b strings.Builder

	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n", width, totalHeight)
	fmt.Fprintf(&b, `<foreignObject width="%d" height="%d">`+"\n", width, totalHeight)
	fmt.Fprintf(&b, `<div xmlns="http://www.w3.org/1999/xhtml" style="width:%dpx;height:%dpx;overflow:hidden;font-family:Inter,system-ui,sans-serif;background:#0f1117;">`+"\n",
		width, totalHeight)

	b.WriteString("<style>\n")
	b.WriteString(buildNavCSS(n))
	b.WriteString("</style>\n")

	// Radio inputs must precede all elements they control via the ~ combinator.
	for i := range stepSVGs {
		checked := ""
		if i == 0 {
			checked = ` checked="checked"`
		}
		fmt.Fprintf(&b, `<input type="radio" name="s" id="s%d"%s/>`+"\n", i, checked)
	}

	// CTA bar — only shown on step 0.
	b.WriteString(`<div class="cta">&#x25B6; Click for walkthrough</div>` + "\n")

	// Diagram area — one pre-rendered SVG per step; only the active one is shown.
	fmt.Fprintf(&b, `<div class="diagrams" style="width:%dpx;height:%dpx;overflow:hidden;">`+"\n", width, diagH)
	for i, svgStr := range stepSVGs {
		fmt.Fprintf(&b, `<div class="step step-%d" style="width:%dpx;height:%dpx;overflow:hidden;">`+"\n", i, width, diagH)
		b.WriteString(svgStr)
		b.WriteString("\n</div>\n")
	}
	b.WriteString("</div>\n")

	// Nav bar — prev buttons, step dots, next buttons.
	b.WriteString(`<div class="nav">` + "\n")
	for i := 1; i < n; i++ {
		fmt.Fprintf(&b, `<label class="prev prev-%d" for="s%d">&#x25C0;</label>`+"\n", i, i-1)
	}
	for i := range stepSVGs {
		fmt.Fprintf(&b, `<label class="dot dot-%d" for="s%d"></label>`+"\n", i, i)
	}
	for i := 0; i < n-1; i++ {
		fmt.Fprintf(&b, `<label class="next next-%d" for="s%d">&#x25B6;</label>`+"\n", i, i+1)
	}
	b.WriteString("</div>\n")

	b.WriteString("</div>\n")
	b.WriteString("</foreignObject>\n")
	b.WriteString("</svg>\n")

	return b.String()
}

// buildNavCSS generates the CSS that drives navigation state via :checked selectors.
// All rules are statically emitted for N steps.
func buildNavCSS(n int) string {
	var b strings.Builder

	b.WriteString(`*{box-sizing:border-box;margin:0;padding:0;}` + "\n")
	b.WriteString(`input[type="radio"]{display:none;}` + "\n")

	// CTA bar.
	b.WriteString(`.cta{display:none;height:40px;line-height:40px;text-align:center;font-size:13px;font-weight:600;color:#5b8dee;background:#0f1117;border-bottom:1px solid #2a2d3a;cursor:pointer;letter-spacing:0.04em;}` + "\n")
	b.WriteString(`.cta:hover{color:#7da9f0;}` + "\n")
	b.WriteString(`#s0:checked~.cta{display:block;}` + "\n")

	// Step SVG visibility.
	b.WriteString(`.step{display:none;}` + "\n")
	b.WriteString(`.step>svg{width:960px !important;display:block;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.diagrams .step-%d{display:block;}`+"\n", i, i)
	}

	// Nav bar base.
	b.WriteString(`.nav{height:60px;background:#0f1117;border-top:1px solid #2a2d3a;display:flex;align-items:center;justify-content:center;gap:10px;}` + "\n")

	// Prev/next buttons.
	b.WriteString(`.prev,.next{display:none;width:32px;height:32px;background:#1a2744;border:1px solid #2a3a5a;border-radius:6px;color:#e8eaf0;font-size:13px;align-items:center;justify-content:center;cursor:pointer;}` + "\n")
	b.WriteString(`.prev:hover,.next:hover{background:#243a6e;border-color:#5b8dee;}` + "\n")
	for i := 0; i < n; i++ {
		if i > 0 {
			fmt.Fprintf(&b, `#s%d:checked~.nav .prev-%d{display:inline-flex;}`+"\n", i, i)
		}
		if i < n-1 {
			fmt.Fprintf(&b, `#s%d:checked~.nav .next-%d{display:inline-flex;}`+"\n", i, i)
		}
	}

	// Step dots.
	b.WriteString(`.dot{width:8px;height:8px;background:#2a2d3a;border-radius:50%;cursor:pointer;display:inline-block;}` + "\n")
	b.WriteString(`.dot:hover{background:#4a5a7a;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.nav .dot-%d{background:#5b8dee;}`+"\n", i, i)
	}

	return b.String()
}

func newBrowserCtx() (context.Context, context.CancelFunc) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("no-sandbox", true),
			chromedp.WindowSize(browserWidth, browserHeight),
		)...,
	)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...any) {}))
	ctx, timeoutCancel := context.WithTimeout(ctx, svgTimeout)
	return ctx, func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
	}
}

func strSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
