package svgformat

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/scottmrogowski/ariel/internal/dsl"
)

const (
	outputWidth     = 1200
	narrationWidth  = 300
	diagramWidth    = outputWidth - narrationWidth // 900
	colHeaderHeight = 44                           // title bar at top of each column
	navHeight       = 60
	svgTimeout      = 5 * time.Minute
	browserWidth    = diagramWidth + 20 // slightly wider than diagramWidth to avoid a scrollbar
	browserHeight   = 2000
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

	// The diagram source is identical across all steps (narration is rendered in the
	// right panel, not as a Mermaid node), so write the extraction HTML once.
	htmlPath := filepath.Join(tmpDir, "diagram.html")
	if err := os.WriteFile(htmlPath, []byte(renderExtractionHTML(sec.MermaidDiagram)), 0644); err != nil {
		return fmt.Errorf("write extraction HTML: %w", err)
	}

	n := len(sec.Steps)
	stepSVGs := make([]string, n)
	narrations := make([]string, n)
	stepHeaders := make([]string, n)
	var maxDiagramHeight int

	for i, step := range sec.Steps {
		narrations[i] = step.Narration
		stepHeaders[i] = formatStepHeader(i, n, step.Label)

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
		// without the closing slash, which is valid HTML but invalid XML.
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

	totalHeight := colHeaderHeight + maxDiagramHeight + navHeight
	out := buildOutputSVG(outputWidth, totalHeight, diagramWidth, colHeaderHeight, maxDiagramHeight, navHeight,
		w.Title, stepSVGs, narrations, stepHeaders)
	return os.WriteFile(outPath, []byte(out), 0644)
}

// formatStepHeader returns the step label string shown at the top of the narration panel.
// Step 0 (overview) shows its label directly; later steps show "N of M — label".
func formatStepHeader(i, total int, label string) string {
	if i == 0 {
		return label
	}
	h := fmt.Sprintf("%d of %d", i, total-1)
	if label != "" {
		h += " — " + label // em dash
	}
	return h
}

func buildOutputSVG(width, totalHeight, diagW, colHeaderH, diagH, navH int,
	title string, stepSVGs, narrations, stepHeaders []string) string {

	n := len(stepSVGs)
	narW := width - diagW
	var b strings.Builder

	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n", width, totalHeight)
	fmt.Fprintf(&b, `<foreignObject width="%d" height="%d">`+"\n", width, totalHeight)
	fmt.Fprintf(&b, `<div xmlns="http://www.w3.org/1999/xhtml" style="width:%dpx;height:%dpx;display:flex;flex-direction:column;font-family:Inter,system-ui,sans-serif;background:#0f1117;">`+"\n",
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

	// Content row: diagram column (left) + narration column (right).
	b.WriteString(`<div class="content">` + "\n")

	// Diagram column: static title + per-step SVGs.
	fmt.Fprintf(&b, `<div class="diagram-col" style="width:%dpx;">`+"\n", diagW)
	fmt.Fprintf(&b, `<div class="col-title">%s</div>`+"\n", html.EscapeString(title))
	b.WriteString(`<div class="diagrams">` + "\n")
	for i, svgStr := range stepSVGs {
		fmt.Fprintf(&b, `<div class="step step-%d">`+"\n", i)
		b.WriteString(svgStr)
		b.WriteString("\n</div>\n")
	}
	b.WriteString("</div>\n") // end .diagrams
	b.WriteString("</div>\n") // end .diagram-col

	// Narration column: per-step header + text, panel always visible.
	fmt.Fprintf(&b, `<div class="narrations" style="width:%dpx;">`+"\n", narW)
	for i := range stepSVGs {
		header := html.EscapeString(stepHeaders[i])
		text := html.EscapeString(narrations[i])
		text = strings.ReplaceAll(text, "\n", "<br/>")
		fmt.Fprintf(&b, `<div class="narration n-%d"><div class="narr-header">%s</div><div class="narr-text">%s</div></div>`+"\n",
			i, header, text)
	}
	b.WriteString("</div>\n") // end .narrations

	b.WriteString("</div>\n") // end .content

	// Bottom bar: CTA label on step 0 (navigates to step 1), nav controls on step 1+.
	b.WriteString(`<div class="bottom">` + "\n")
	if n > 1 {
		b.WriteString(`<label class="cta" for="s1">&#x25B6; Click for walkthrough</label>` + "\n")
	}
	b.WriteString(`<div class="nav-controls">` + "\n")
	b.WriteString(`<div class="nav-prev">` + "\n")
	for i := 1; i < n; i++ {
		fmt.Fprintf(&b, `<label class="prev prev-%d" for="s%d">&#x25C0;</label>`+"\n", i, i-1)
	}
	b.WriteString("</div>\n")
	b.WriteString(`<div class="nav-dots">` + "\n")
	for i := range stepSVGs {
		fmt.Fprintf(&b, `<label class="dot dot-%d" for="s%d"></label>`+"\n", i, i)
	}
	b.WriteString("</div>\n")
	b.WriteString(`<div class="nav-next">` + "\n")
	for i := 0; i < n-1; i++ {
		fmt.Fprintf(&b, `<label class="next next-%d" for="s%d">&#x25B6;</label>`+"\n", i, i+1)
	}
	b.WriteString("</div>\n")
	b.WriteString("</div>\n") // end .nav-controls
	b.WriteString("</div>\n") // end .bottom

	b.WriteString("</div>\n")
	b.WriteString("</foreignObject>\n")
	b.WriteString("</svg>\n")

	return b.String()
}

// buildNavCSS generates the CSS that drives step navigation via :checked selectors.
// All rules are statically emitted for N steps.
func buildNavCSS(n int) string {
	var b strings.Builder

	b.WriteString(`*{box-sizing:border-box;margin:0;padding:0;}` + "\n")
	b.WriteString(`input[type="radio"]{display:none;}` + "\n")

	// Content row: diagram column left, narration column right.
	b.WriteString(`.content{flex:1;display:flex;flex-direction:row;overflow:hidden;}` + "\n")

	// Diagram column: flex column, title bar + diagram area.
	b.WriteString(`.diagram-col{display:flex;flex-direction:column;}` + "\n")
	b.WriteString(`.col-title{flex-shrink:0;height:44px;line-height:44px;padding:0 20px;font-size:13px;font-weight:600;color:#e8eaf0;border-bottom:1px solid #1e2130;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}` + "\n")
	b.WriteString(`.diagrams{flex:1;overflow:hidden;display:flex;align-items:center;justify-content:center;}` + "\n")
	b.WriteString(`.step{display:none;}` + "\n")
	b.WriteString(`.step>svg{display:block;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .step-%d{display:block;}`+"\n", i, i)
	}

	// Narration column: always visible, flex column, one narration visible at a time.
	b.WriteString(`.narrations{display:flex;flex-direction:column;border-left:1px solid #1e2130;}` + "\n")
	b.WriteString(`.narration{display:none;flex-direction:column;flex:1;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .n-%d{display:flex;}`+"\n", i, i)
	}
	b.WriteString(`.narr-header{flex-shrink:0;height:44px;line-height:44px;padding:0 20px;font-size:11px;font-weight:600;color:#5b8dee;letter-spacing:0.05em;text-transform:uppercase;border-bottom:1px solid #1e2130;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}` + "\n")
	b.WriteString(`.narr-text{flex:1;padding:20px;font-size:13px;line-height:1.65;color:#c0c4d0;overflow-y:auto;}` + "\n")

	// Bottom bar.
	b.WriteString(`.bottom{height:60px;flex-shrink:0;background:#0f1117;border-top:1px solid #1e2130;display:flex;align-items:center;justify-content:center;}` + "\n")

	// CTA: shown on step 0 only.
	b.WriteString(`.cta{display:none;font-size:13px;font-weight:600;color:#5b8dee;cursor:pointer;letter-spacing:0.04em;padding:10px 28px;border:1px solid #2a3a5a;border-radius:8px;align-items:center;gap:8px;}` + "\n")
	b.WriteString(`.cta:hover{color:#7da9f0;border-color:#5b8dee;}` + "\n")
	if n > 1 {
		b.WriteString(`#s0:checked~.bottom .cta{display:flex;}` + "\n")
	}

	// Nav controls: shown on step 1+.
	b.WriteString(`.nav-controls{display:none;width:100%;height:100%;align-items:center;justify-content:center;gap:12px;}` + "\n")
	for i := 1; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.bottom .nav-controls{display:flex;}`+"\n", i)
	}

	// 3-zone stable nav: zones always the same size so dots never shift.
	b.WriteString(`.nav-prev,.nav-next{position:relative;width:32px;height:32px;flex-shrink:0;}` + "\n")
	b.WriteString(`.nav-dots{display:flex;align-items:center;gap:8px;}` + "\n")
	b.WriteString(`.prev,.next{position:absolute;top:0;left:0;display:none;width:32px;height:32px;background:#1a2744;border:1px solid #2a3a5a;border-radius:6px;color:#e8eaf0;font-size:13px;align-items:center;justify-content:center;cursor:pointer;}` + "\n")
	b.WriteString(`.prev:hover,.next:hover{background:#243a6e;border-color:#5b8dee;}` + "\n")
	for i := 0; i < n; i++ {
		if i > 0 {
			fmt.Fprintf(&b, `#s%d:checked~.bottom .nav-controls .nav-prev .prev-%d{display:inline-flex;}`+"\n", i, i)
		}
		if i < n-1 {
			fmt.Fprintf(&b, `#s%d:checked~.bottom .nav-controls .nav-next .next-%d{display:inline-flex;}`+"\n", i, i)
		}
	}

	// Step dots.
	b.WriteString(`.dot{width:8px;height:8px;background:#2a2d3a;border-radius:50%;cursor:pointer;display:inline-block;}` + "\n")
	b.WriteString(`.dot:hover{background:#4a5a7a;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.bottom .nav-controls .nav-dots .dot-%d{background:#5b8dee;}`+"\n", i, i)
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
