package svgformat

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/scottmrogowski/ariel/internal/dsl"
)

const (
	outputWidth      = 1200
	narrationWidth   = 300
	diagramWidth     = outputWidth - narrationWidth // 900
	pageHeaderHeight = 60                           // full-width title bar above both columns
	colHeaderHeight  = 44                           // narration step-header height
	navHeight        = 60
	maxOutputHeight  = 850
	// diagramSidePad: fraction of diagram column width used as padding on each side (5% × 2 = 10% total).
	diagramSidePad = 0.05
	// maxScaleUp: diagrams are scaled up by at most this factor from their natural Mermaid size.
	maxScaleUp   = 2.0
	svgTimeout   = 5 * time.Minute
	browserWidth = diagramWidth + 20 // slightly wider than diagramWidth to avoid a scrollbar
	browserHeight = 2000
)

// Generate renders a Walkthrough as an interactive SVG file at outPath.
// The output SVG uses foreignObject + CSS :checked for step navigation — interactive
// when opened in GitHub's SVG viewer, static when embedded as <img>.
// Multi-section walkthroughs are supported: sections are flattened into a single
// step sequence, each step rendered from its section's diagram.
func Generate(w *dsl.Walkthrough, outPath string) error {
	sections := w.ToSections()

	tmpDir, err := os.MkdirTemp("", "ariel-svg-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := newBrowserCtx()
	defer cancel()

	multiSection := len(sections) > 1
	totalSteps := 0
	for _, sec := range sections {
		totalSteps += len(sec.Steps)
	}

	stepSVGs := make([]string, totalSteps)
	narrations := make([]string, totalSteps)
	stepHeaders := make([]string, totalSteps)
	// effectiveWidths[i]: the rendered width of step i's diagram in the output SVG.
	// Capped at min(2× natural Mermaid width, available column width with 10% padding).
	effectiveWidths := make([]int, totalSteps)
	availableW := int(float64(diagramWidth) * (1.0 - 2*diagramSidePad)) // 810px
	var maxEffectiveH int

	htmlPath := filepath.Join(tmpDir, "diagram.html")
	globalIdx := 0

	for _, sec := range sections {
		// Each section has its own Mermaid diagram; write extraction HTML once per section.
		if err := os.WriteFile(htmlPath, []byte(renderExtractionHTML(sec.MermaidDiagram)), 0644); err != nil {
			return fmt.Errorf("write extraction HTML: %w", err)
		}

		for _, step := range sec.Steps {
			narrations[globalIdx] = step.Narration

			// In multi-section files, prefix the step label with the section title so
			// the narration panel shows which section the step belongs to.
			label := step.Label
			if multiSection && sec.Title != "" {
				if label != "" {
					label = sec.Title + " — " + label
				} else {
					label = sec.Title
				}
			}
			stepHeaders[globalIdx] = formatStepHeader(globalIdx, totalSteps, label)

			if err := chromedp.Run(ctx,
				chromedp.Navigate("file://"+htmlPath),
				chromedp.WaitVisible("#ready", chromedp.ByID),
			); err != nil {
				return fmt.Errorf("step %d: load: %w", globalIdx, err)
			}

			hJSON, _ := json.Marshal(strSlice(step.HighlightNodes))
			fJSON, _ := json.Marshal(strSlice(step.FocusNodes))
			if err := chromedp.Run(ctx, chromedp.Evaluate(
				fmt.Sprintf(`applyStep(%s,%s)`, hJSON, fJSON), nil,
			)); err != nil {
				return fmt.Errorf("step %d: applyStep: %w", globalIdx, err)
			}

			var svgStr string
			if err := chromedp.Run(ctx, chromedp.Evaluate(`getSVG()`, &svgStr)); err != nil {
				return fmt.Errorf("step %d: getSVG: %w", globalIdx, err)
			}
			if !strings.HasPrefix(svgStr, "<svg") {
				return fmt.Errorf("step %d: getSVG returned unexpected content (first 60 chars): %q", globalIdx, truncate(svgStr, 60))
			}
			// Mermaid renders HTML void elements (e.g. <br>) inside foreignObject
			// without the closing slash, which is valid HTML but invalid XML.
			svgStr = strings.ReplaceAll(svgStr, "<br>", "<br/>")
			stepSVGs[globalIdx] = svgStr

			var dimsJSON string
			if err := chromedp.Run(ctx, chromedp.Evaluate(`getDimensions()`, &dimsJSON)); err != nil {
				return fmt.Errorf("step %d: getDimensions: %w", globalIdx, err)
			}
			var dims struct {
				W  int `json:"w"`
				H  int `json:"h"`
				NW int `json:"nw"` // natural Mermaid width (before any CSS override)
			}
			if err := json.Unmarshal([]byte(dimsJSON), &dims); err != nil {
				return fmt.Errorf("step %d: parse dimensions: %w", globalIdx, err)
			}
			if dims.H <= 0 || dims.NW <= 0 {
				return fmt.Errorf("step %d: diagram rendered with zero dimensions — Mermaid may have failed to parse the diagram", globalIdx)
			}
			// Cap scale-up at maxScaleUp× natural width, then further cap at available column width.
			effectiveW := int(math.Round(float64(dims.NW) * maxScaleUp))
			if effectiveW > availableW {
				effectiveW = availableW
			}
			effectiveWidths[globalIdx] = effectiveW
			// Effective height scales proportionally from the measured natural height.
			effectiveH := int(math.Round(float64(dims.H) * float64(effectiveW) / float64(dims.W)))
			if effectiveH > maxEffectiveH {
				maxEffectiveH = effectiveH
			}

			globalIdx++
		}
	}

	// Natural layout dimensions (internal coordinate space).
	naturalH := pageHeaderHeight + colHeaderHeight + maxEffectiveH + navHeight

	// Scale entire SVG down proportionally if it exceeds maxOutputHeight.
	// Width is always outputWidth (1200); only height can exceed the limit.
	scale := 1.0
	if naturalH > maxOutputHeight {
		scale = float64(maxOutputHeight) / float64(naturalH)
	}
	finalW := int(math.Round(float64(outputWidth) * scale))
	finalH := int(math.Round(float64(naturalH) * scale))
	if finalH > maxOutputHeight {
		finalH = maxOutputHeight
	}

	out := buildOutputSVG(finalW, finalH, outputWidth, naturalH, diagramWidth, pageHeaderHeight, colHeaderHeight, maxEffectiveH, navHeight,
		effectiveWidths, w.Title, stepSVGs, narrations, stepHeaders)
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

func buildOutputSVG(finalW, finalH, naturalW, naturalH, diagW, pageHeaderH, colHeaderH, diagH, navH int,
	effectiveWidths []int, title string, stepSVGs, narrations, stepHeaders []string) string {

	n := len(stepSVGs)
	narW := naturalW - diagW
	var b strings.Builder

	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	// viewBox maps the natural coordinate space to the clamped rendered dimensions.
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		finalW, finalH, naturalW, naturalH)
	fmt.Fprintf(&b, `<foreignObject width="%d" height="%d">`+"\n", naturalW, naturalH)
	// position:relative is required so the cta-overlay label can be positioned absolutely over the full area.
	fmt.Fprintf(&b, `<div xmlns="http://www.w3.org/1999/xhtml" style="position:relative;width:%dpx;height:%dpx;display:flex;flex-direction:column;font-family:Inter,system-ui,sans-serif;background:#0f1117;">`+"\n",
		naturalW, naturalH)

	b.WriteString("<style>\n")
	b.WriteString(buildNavCSS(n, effectiveWidths))
	b.WriteString("</style>\n")

	// Radio inputs must precede all elements they control via the ~ combinator.
	for i := range stepSVGs {
		checked := ""
		if i == 0 {
			checked = ` checked="checked"`
		}
		fmt.Fprintf(&b, `<input type="radio" name="s" id="s%d"%s/>`+"\n", i, checked)
	}

	// Full-width page header with walkthrough title, matching HTML renderer layout.
	fmt.Fprintf(&b, `<div class="page-header"><div class="page-title">%s</div></div>`+"\n", html.EscapeString(title))

	// Content row: diagram column (left) + narration column (right).
	b.WriteString(`<div class="content">` + "\n")

	// Diagram column: per-step SVGs only (title is in page-header above).
	fmt.Fprintf(&b, `<div class="diagram-col" style="width:%dpx;">`+"\n", diagW)
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

	// CTA overlay: covers full width, shown on step 0 only.
	if n > 1 {
		b.WriteString(`<label class="cta-overlay" for="s1"><div class="cta-btn">&#x25B6; Click for walkthrough</div></label>` + "\n")
	}

	// Bottom bar: empty on step 0, nav controls on step 1+.
	b.WriteString(`<div class="bottom">` + "\n")
	b.WriteString(`<div class="nav-controls">` + "\n")
	b.WriteString(`<div class="nav-prev">` + "\n")
	// Start at 2: step 1 has no back button since s0 is the CTA-only pre-step.
	for i := 2; i < n; i++ {
		fmt.Fprintf(&b, `<label class="prev prev-%d" for="s%d">&#x25C0;</label>`+"\n", i, i-1)
	}
	b.WriteString("</div>\n")
	b.WriteString(`<div class="nav-dots">` + "\n")
	// Start at 1: no dot for s0 (the CTA pre-step cannot be returned to).
	for i := 1; i < n; i++ {
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
// effectiveWidths[i] is the rendered width of step i's diagram: capped at 2× natural Mermaid
// size and at 90% of the diagram column (10% total horizontal padding).
func buildNavCSS(n int, effectiveWidths []int) string {
	var b strings.Builder

	b.WriteString(`*{box-sizing:border-box;margin:0;padding:0;}` + "\n")
	b.WriteString(`input[type="radio"]{display:none;}` + "\n")

	// Full-width page header with walkthrough title.
	b.WriteString(`.page-header{flex-shrink:0;height:60px;display:flex;align-items:center;justify-content:center;border-bottom:1px solid #1e2130;}` + "\n")
	b.WriteString(`.page-title{font-size:22px;font-weight:600;color:#e8eaf0;text-align:center;}` + "\n")

	// Content row: diagram column left, narration column right.
	b.WriteString(`.content{flex:1;display:flex;flex-direction:row;overflow:hidden;}` + "\n")

	// Diagram column: flex column containing only the diagram area (no per-column title).
	b.WriteString(`.diagram-col{display:flex;flex-direction:column;}` + "\n")
	// 10% horizontal padding (5% each side); flex+center for vertical centering within the column.
	b.WriteString(`.diagrams{flex:1;overflow:hidden;padding:0 5%;display:flex;flex-direction:column;align-items:center;justify-content:center;}` + "\n")
	b.WriteString(`.step{display:none;width:100%;}` + "\n")
	// Per-step max-width enforces the 2× natural-width cap; margin:auto centers within the step.
	b.WriteString(`.step>svg{display:block;width:100%!important;height:auto!important;margin:0 auto;}` + "\n")
	for i, w := range effectiveWidths {
		fmt.Fprintf(&b, `.step-%d>svg{max-width:%dpx!important;}`+"\n", i, w)
	}
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .step-%d{display:block;}`+"\n", i, i)
	}

	// Narration column: always visible, flex column, one narration visible at a time.
	b.WriteString(`.narrations{display:flex;flex-direction:column;border-left:1px solid #1e2130;}` + "\n")
	b.WriteString(`.narration{display:none;flex-direction:column;flex:1;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .n-%d{display:flex;}`+"\n", i, i)
	}
	b.WriteString(`.narr-header{flex-shrink:0;height:44px;line-height:44px;padding:0 20px;font-size:11px;font-weight:600;color:#5b8dee;letter-spacing:0.05em;text-transform:uppercase;border-bottom:1px solid #1e2130;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;text-align:center;}` + "\n")
	b.WriteString(`.narr-text{flex:1;padding:20px;font-size:13px;line-height:1.65;color:#c0c4d0;overflow-y:auto;}` + "\n")

	// CTA overlay: covers full output width/height, shown on step 0 only.
	// Positioned relative to the root wrapper div (which has position:relative).
	b.WriteString(`.cta-overlay{position:absolute;top:0;left:0;right:0;bottom:0;display:none;align-items:center;justify-content:center;cursor:pointer;background:rgba(15,17,23,0.45);}` + "\n")
	if n > 1 {
		b.WriteString(`#s0:checked~.cta-overlay{display:flex;}` + "\n")
	}
	b.WriteString(`.cta-btn{background:#0f1117;border:2px solid #5b8dee;border-radius:12px;padding:32px 72px;font-size:24px;font-weight:700;color:#5b8dee;letter-spacing:0.03em;}` + "\n")
	b.WriteString(`.cta-overlay:hover .cta-btn{background:#1a2744;border-color:#7da9f0;color:#7da9f0;}` + "\n")

	// Bottom bar.
	b.WriteString(`.bottom{height:60px;flex-shrink:0;background:#0f1117;border-top:1px solid #1e2130;display:flex;align-items:center;justify-content:center;}` + "\n")

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
