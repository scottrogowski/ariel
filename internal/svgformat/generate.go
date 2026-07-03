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
	"github.com/scottrogowski/ariel/internal/dsl"
)

const (
	narrationWidth   = 300
	diagAreaW        = 900
	diagAreaH        = 686 // pageHeader(60) + diagAreaH(686) + overhead(104=narr-header+nav) = totalH(850)
	totalW           = diagAreaW + narrationWidth // 1200
	totalH           = 850
	pageHeaderHeight = 60
	navHeight        = 60
	// bboxMargin is the fractional padding added around the highlighted node bounding box
	// when computing the scale that fits all highlighted nodes into the viewport.
	bboxMargin     = 0.15
	svgTimeout     = 5 * time.Minute
	browserWidth   = 4000 // generous width so diagrams of any size render at natural pixel dimensions
	browserHeight  = 2000
	arielGitHubURL = "https://github.com/scottrogowski/ariel"
)

type sectionMeta struct {
	title string
	start int // global index of first step in this section
	count int // total steps in this section
}

type stepTransform struct {
	scale float64
	tx    float64
	ty    float64
}

type nodeBBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Generate renders a Walkthrough as an interactive SVG file at outPath.
// The output SVG uses foreignObject + CSS :checked for step navigation — interactive
// when opened in GitHub's SVG viewer, static when embedded as <img>.
// Multi-section walkthroughs are supported; sections are navigable via section dots.
//
// The initial "Click for walkthrough" CTA (shown at s0) is a one-way entry point:
// the Back button and all dot navigation start from s1, making s0 unreachable once
// the user has clicked through.
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
	naturalWs := make([]int, totalSteps)
	naturalHs := make([]int, totalSteps)
	transforms := make([]stepTransform, totalSteps)
	stepSecIdx := make([]int, totalSteps) // which section each global step belongs to

	htmlPath := filepath.Join(tmpDir, "diagram.html")
	globalIdx := 0

	secsMeta := make([]sectionMeta, len(sections))
	for si, sec := range sections {
		secsMeta[si] = sectionMeta{title: sec.Title, start: globalIdx, count: len(sec.Steps)}

		nodeLabels, _ := dsl.ExtractGraph(sec.MermaidDiagram)
		if err := os.WriteFile(htmlPath, []byte(renderExtractionHTML(sec.MermaidDiagram, nodeLabels)), 0644); err != nil {
			return fmt.Errorf("write extraction HTML: %w", err)
		}

		for secStepIdx, step := range sec.Steps {
			narrations[globalIdx] = step.Narration
			stepHeaders[globalIdx] = formatStepHeader(sec.Title, secStepIdx, len(sec.Steps), step.Label, multiSection)
			stepSecIdx[globalIdx] = si

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
			if dims.W <= 0 || dims.H <= 0 {
				return fmt.Errorf("step %d: diagram rendered with zero dimensions — Mermaid may have failed to parse the diagram", globalIdx)
			}
			naturalWs[globalIdx] = dims.W
			naturalHs[globalIdx] = dims.H

			var bboxes []nodeBBox
			allNodes := append(strSlice(step.HighlightNodes), strSlice(step.FocusNodes)...)
			if len(allNodes) > 0 {
				nodesJSON, _ := json.Marshal(allNodes)
				var bboxJSON string
				if err := chromedp.Run(ctx, chromedp.Evaluate(
					fmt.Sprintf(`getNodeBBoxes(%s)`, nodesJSON), &bboxJSON,
				)); err != nil {
					return fmt.Errorf("step %d: getNodeBBoxes: %w", globalIdx, err)
				}
				var bboxMap map[string]nodeBBox
				if err := json.Unmarshal([]byte(bboxJSON), &bboxMap); err != nil {
					return fmt.Errorf("step %d: parse bboxes: %w", globalIdx, err)
				}
				for _, bb := range bboxMap {
					bboxes = append(bboxes, bb)
				}
			}
			transforms[globalIdx] = computeStepTransform(dims.W, dims.H, bboxes)

			globalIdx++
		}
	}

	out := buildOutputSVG(w.Title, stepSVGs, narrations, stepHeaders,
		stepSecIdx, secsMeta, naturalWs, naturalHs, transforms)
	return os.WriteFile(outPath, []byte(out), 0644)
}

// formatStepHeader returns the narration panel header for a step.
// Matches HTML renderer format: "SECTION · N of M — label" for content steps,
// or the section title alone for overview steps (secStepIdx == 0).
func formatStepHeader(sectionTitle string, secStepIdx, secTotal int, stepLabel string, multiSection bool) string {
	if secStepIdx == 0 {
		if multiSection && sectionTitle != "" {
			return sectionTitle
		}
		return stepLabel
	}
	h := fmt.Sprintf("%d of %d", secStepIdx, secTotal-1)
	if stepLabel != "" {
		h += " — " + stepLabel // em dash
	}
	if multiSection && sectionTitle != "" {
		h = sectionTitle + " · " + h // middle dot
	}
	return h
}

func buildOutputSVG(title string, stepSVGs, narrations, stepHeaders []string,
	stepSecIdx []int, secsMeta []sectionMeta,
	naturalWs, naturalHs []int, transforms []stepTransform) string {

	n := len(stepSVGs)
	narW := narrationWidth
	multiSection := len(secsMeta) > 1
	var b strings.Builder

	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n", totalW, totalH)
	fmt.Fprintf(&b, `<foreignObject width="%d" height="%d">`+"\n", totalW, totalH)
	// position:relative is required so the cta-overlay label can be positioned absolutely over the full area.
	fmt.Fprintf(&b, `<div xmlns="http://www.w3.org/1999/xhtml" style="position:relative;width:%dpx;height:%dpx;display:flex;flex-direction:column;font-family:Inter,system-ui,sans-serif;background:#0f1117;">`+"\n",
		totalW, totalH)

	b.WriteString("<style>\n")
	b.WriteString(buildNavCSS(n, stepSecIdx, secsMeta, naturalWs, naturalHs, transforms))
	b.WriteString("</style>\n")

	// Radio inputs must precede all elements they control via the ~ combinator.
	for i := range stepSVGs {
		checked := ""
		if i == 0 {
			checked = ` checked="checked"`
		}
		fmt.Fprintf(&b, `<input type="radio" name="s" id="s%d"%s/>`+"\n", i, checked)
	}

	// Page header: title (with per-step section name for multi-section) + ariel logo.
	b.WriteString(`<div class="page-header">` + "\n")
	b.WriteString(`<div class="page-title">` + html.EscapeString(title))
	if multiSection {
		b.WriteString(`<span class="page-sep"> | </span>`)
		for si, sec := range secsMeta {
			fmt.Fprintf(&b, `<span class="sec-title sec-title-%d">%s</span>`, si, html.EscapeString(sec.title))
		}
	}
	b.WriteString("</div>\n")
	// Logo links to the ariel GitHub page (functional in SVG viewer, not in <img> embed).
	fmt.Fprintf(&b,
		`<a class="ariel-link" href="%s">`+
			`<span class="ariel-logo">`+
			`<span class="ariel-emoji">&#x1F9DC;&#x1F3FB;&#x200D;&#x2640;&#xFE0F;</span>`+
			`<svg class="ariel-flowchart" viewBox="0 0 44 44" fill="none" stroke="currentColor" xmlns="http://www.w3.org/2000/svg">`+
			`<circle cx="22" cy="22" r="20" stroke-width="1.2"/>`+
			`<rect x="7.86" y="7.86" width="28.28" height="28.28" stroke-width="1.2"/>`+
			`<polygon points="22,2 42,22 22,42 2,22" stroke-width="1.2"/>`+
			`</svg></span></a>`+"\n", arielGitHubURL)
	b.WriteString("</div>\n") // end .page-header

	// Content row: diagram column (left) + narration column (right).
	b.WriteString(`<div class="content">` + "\n")

	// Diagram column.
	fmt.Fprintf(&b, `<div class="diagram-col" style="width:%dpx;">`+"\n", diagAreaW)
	b.WriteString(`<div class="diagrams">` + "\n")
	for i, svgStr := range stepSVGs {
		fmt.Fprintf(&b, `<div class="step step-%d">`+"\n", i)
		b.WriteString(svgStr)
		b.WriteString("\n</div>\n")
	}
	b.WriteString("</div>\n") // end .diagrams
	b.WriteString("</div>\n") // end .diagram-col

	// Narration column: per-step narrations (each including their own progress dots),
	// followed by nav controls pinned to the bottom.
	fmt.Fprintf(&b, `<div class="narrations" style="width:%dpx;">`+"\n", narW)

	for i := range stepSVGs {
		si := stepSecIdx[i]
		header := html.EscapeString(stepHeaders[i])
		text := html.EscapeString(narrations[i])
		text = strings.ReplaceAll(text, "\n", "<br/>")

		fmt.Fprintf(&b, `<div class="narration n-%d">`+"\n", i)
		fmt.Fprintf(&b, `<div class="narr-header">%s</div>`+"\n", header)
		fmt.Fprintf(&b, `<div class="narr-text">%s</div>`+"\n", text)

		// Progress dots inside each narration so they flow right below the text,
		// matching HTML renderer layout (dots are inside .narration-area, not pinned to bottom).
		b.WriteString(`<div class="progress-area">` + "\n")
		if multiSection {
			b.WriteString(`<div class="section-track">` + "\n")
			for si2, sec := range secsMeta {
				// Section 0's dot targets s1 (not s0/CTA) so the CTA is unreachable via navigation.
				target := sec.start
				if si2 == 0 && n > 1 {
					target = 1
				}
				fmt.Fprintf(&b, `<label class="sec-dot sec-dot-%d" for="s%d" title="%s"></label>`+"\n",
					si2, target, html.EscapeString(sec.title))
			}
			b.WriteString("</div>\n") // end .section-track
		}
		// One step-track per section. Visibility controlled by CSS :checked rules.
		for si2, sec := range secsMeta {
			fmt.Fprintf(&b, `<div class="step-track sec-steps-%d">`+"\n", si2)
			// Section 0 skips s0 (CTA state) when n > 1; all other sections start at their first step.
			startJ := 0
			if si2 == 0 && n > 1 {
				startJ = 1
			}
			for j := startJ; j < sec.count; j++ {
				globalI := sec.start + j
				introCls := ""
				if j == startJ {
					introCls = " intro-dot"
				}
				fmt.Fprintf(&b, `<label class="dot dot-%d%s" for="s%d"></label>`+"\n", globalI, introCls, globalI)
			}
			b.WriteString("</div>\n") // end .sec-steps-N
		}
		b.WriteString("</div>\n") // end .progress-area
		_ = si                    // si used above; suppress unused warning
		b.WriteString("</div>\n") // end .narration
	}

	// Nav controls: Back + Next buttons. Always at the bottom via margin-top:auto.
	b.WriteString(`<div class="controls">` + "\n")
	b.WriteString(`<div class="nav-prev">` + "\n")
	// Back buttons start at s2: s0 is CTA state, s1 is first real step with no predecessor.
	// This makes the CTA unreachable via Back navigation.
	for i := 2; i < n; i++ {
		fmt.Fprintf(&b, `<label class="prev prev-%d" for="s%d">&#x2190; Back</label>`+"\n", i, i-1)
	}
	b.WriteString("</div>\n")
	b.WriteString(`<div class="nav-next">` + "\n")
	// Next buttons for steps 0..N-2. Last step shows Done (targets itself = no-op click, disabled-styled).
	for i := 0; i < n-1; i++ {
		fmt.Fprintf(&b, `<label class="next next-%d" for="s%d">Next &#x2192;</label>`+"\n", i, i+1)
	}
	fmt.Fprintf(&b, `<label class="next next-done next-%d" for="s%d">Done</label>`+"\n", n-1, n-1)
	b.WriteString("</div>\n")
	b.WriteString("</div>\n") // end .controls

	b.WriteString("</div>\n") // end .narrations
	b.WriteString("</div>\n") // end .content

	// CTA overlay: covers full output on step 0, advances to step 1 on click.
	// This is a one-way entry point — the Back button and dot navigation never return to s0.
	if n > 1 {
		b.WriteString(`<label class="cta-overlay" for="s1"><div class="cta-btn">&#x25B6; Click for walkthrough</div></label>` + "\n")
	}

	b.WriteString("</div>\n")
	b.WriteString("</foreignObject>\n")
	b.WriteString("</svg>\n")

	return b.String()
}

// buildNavCSS generates all CSS for the SVG output: layout, theming, and the
// :checked-based rules that drive step navigation, dot highlighting, and section titles.
func buildNavCSS(n int, stepSecIdx []int, secsMeta []sectionMeta,
	naturalWs, naturalHs []int, transforms []stepTransform) string {
	var b strings.Builder
	multiSection := len(secsMeta) > 1

	b.WriteString(`*{box-sizing:border-box;margin:0;padding:0;}` + "\n")
	b.WriteString(`input[type="radio"]{display:none;}` + "\n")

	// Page header.
	b.WriteString(`.page-header{position:relative;flex-shrink:0;height:60px;display:flex;align-items:center;justify-content:center;border-bottom:1px solid #1e2130;}` + "\n")
	b.WriteString(`.page-title{font-size:22px;font-weight:600;color:#e8eaf0;text-align:center;}` + "\n")
	if multiSection {
		b.WriteString(`.page-sep{margin:0 10px;color:#6b7280;font-weight:300;}` + "\n")
		b.WriteString(`.sec-title{display:none;font-size:22px;font-weight:400;color:#e8eaf0;}` + "\n")
		// Show the current section's title span in the header.
		for i := 0; i < n; i++ {
			si := stepSecIdx[i]
			fmt.Fprintf(&b, `#s%d:checked~.page-header .sec-title-%d{display:inline;}`+"\n", i, si)
		}
	}
	// Logo.
	b.WriteString(`.ariel-link{position:absolute;right:32px;opacity:0.7;text-decoration:none;color:#6b7280;}` + "\n")
	b.WriteString(`.ariel-link:hover{opacity:1;}` + "\n")
	b.WriteString(`.ariel-logo{position:relative;width:36px;height:36px;display:block;}` + "\n")
	b.WriteString(`.ariel-emoji{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);font-size:20px;line-height:1;z-index:1;}` + "\n")
	b.WriteString(`.ariel-flowchart{position:absolute;top:0;left:0;width:36px;height:36px;}` + "\n")

	// Content row.
	b.WriteString(`.content{flex:1;display:flex;flex-direction:row;overflow:hidden;}` + "\n")

	// Diagram column: fixed viewport with overflow:hidden; diagrams pan/zoom within it via CSS transforms.
	b.WriteString(`.diagram-col{display:flex;flex-direction:column;}` + "\n")
	fmt.Fprintf(&b, `.diagrams{height:%dpx;width:%dpx;overflow:hidden;position:relative;}`+"\n", diagAreaH, diagAreaW)
	b.WriteString(`.step{display:none;position:absolute;top:0;left:0;}` + "\n")
	b.WriteString(`.step>svg{display:block;position:absolute;top:0;left:0;transform-origin:0 0;}` + "\n")
	// Per-step: pin SVG to natural pixel dimensions and apply precomputed pan/zoom transform.
	for i, t := range transforms {
		fmt.Fprintf(&b, `.step-%d>svg{width:%dpx!important;height:%dpx!important;transform:translate(%.2fpx,%.2fpx) scale(%.6f);}`+"\n",
			i, naturalWs[i], naturalHs[i], t.tx, t.ty, t.scale)
	}
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .step-%d{display:block;}`+"\n", i, i)
	}

	// Narration column: flex column; controls pin to bottom via margin-top:auto.
	b.WriteString(`.narrations{display:flex;flex-direction:column;background:#141720;border-left:1px solid #1e2130;}` + "\n")
	// .narration takes natural height (no flex:1) so progress dots flow right below the text.
	b.WriteString(`.narration{display:none;flex-direction:column;}` + "\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `#s%d:checked~.content .n-%d{display:flex;}`+"\n", i, i)
	}
	b.WriteString(`.narr-header{flex-shrink:0;padding:16px 20px;font-size:11px;font-weight:600;color:#5b8dee;letter-spacing:0.05em;text-transform:uppercase;border-bottom:1px solid #1e2130;}` + "\n")
	// narr-text: cap height so long narrations don't push dots/controls out of view.
	maxNarrTextH := totalH - pageHeaderHeight - navHeight - 120 // subtract header, controls, approx progress+narr-header
	fmt.Fprintf(&b, `.narr-text{padding:20px;font-size:17px;line-height:1.65;color:#e8eaf0;overflow-y:auto;max-height:%dpx;}`+"\n", maxNarrTextH)

	// Progress area: flows immediately below narration text (dots are inside each .narration div).
	b.WriteString(`.progress-area{padding:12px 20px;display:flex;flex-direction:column;gap:8px;}` + "\n")

	// Section dots (multi-section only). Clicking navigates to that section's first real step.
	if multiSection {
		b.WriteString(`.section-track{display:flex;gap:8px;align-items:center;}` + "\n")
		b.WriteString(`.sec-dot{width:8px;height:8px;border-radius:50%;background:#2a2d3a;cursor:pointer;display:inline-block;transition:all 0.3s;}` + "\n")
		b.WriteString(`.sec-dot:hover{background:#4ecdc4;opacity:0.6;}` + "\n")
		// Active section dot: teal pill matching HTML's .section-dot.active.
		for i := 0; i < n; i++ {
			si := stepSecIdx[i]
			fmt.Fprintf(&b, `#s%d:checked~.content .sec-dot-%d{background:#4ecdc4;width:24px;border-radius:3px;}`+"\n", i, si)
		}
	}

	// Per-section step-track rows. Only the current section's row is visible.
	b.WriteString(`.step-track{display:none;gap:6px;align-items:center;}` + "\n")
	for i := 0; i < n; i++ {
		si := stepSecIdx[i]
		fmt.Fprintf(&b, `#s%d:checked~.content .sec-steps-%d{display:flex;}`+"\n", i, si)
	}
	b.WriteString(`.dot{width:6px;height:6px;border-radius:50%;background:#2a2d3a;cursor:pointer;display:inline-block;transition:all 0.3s;}` + "\n")
	b.WriteString(`.dot:hover{background:#4a5a7a;}` + "\n")
	// Intro dot (first visible dot of each section): accent color, slightly transparent.
	b.WriteString(`.intro-dot{background:#5b8dee;opacity:0.3;}` + "\n")
	// Active step dot per step. s0 has no dot when n > 1 (CTA state, unreachable).
	for i := 0; i < n; i++ {
		if i == 0 && n > 1 {
			continue // no dot generated for CTA step
		}
		si := stepSecIdx[i]
		// First dot of each section accounts for the CTA skip in section 0.
		firstDot := secsMeta[si].start
		if si == 0 && n > 1 {
			firstDot = 1
		}
		isIntro := i == firstDot
		if isIntro {
			// Active intro dot: stays circular, full opacity.
			fmt.Fprintf(&b, `#s%d:checked~.content .dot-%d{background:#5b8dee;opacity:1;width:6px;border-radius:50%%;}`+"\n", i, i)
		} else {
			// Active regular dot: pill shape.
			fmt.Fprintf(&b, `#s%d:checked~.content .dot-%d{background:#5b8dee;opacity:1;width:20px;border-radius:3px;}`+"\n", i, i)
		}
	}

	// Nav controls: pinned to bottom of narrations column via margin-top:auto.
	b.WriteString(`.controls{flex-shrink:0;margin-top:auto;height:60px;border-top:1px solid #1e2130;padding:0 20px;display:flex;align-items:center;gap:12px;}` + "\n")
	b.WriteString(`.nav-next{flex:1;}` + "\n")
	b.WriteString(`.prev,.next{display:none;padding:10px 20px;border-radius:6px;font-size:13px;font-weight:500;cursor:pointer;white-space:nowrap;}` + "\n")
	b.WriteString(`.prev{background:transparent;color:#6b7280;border:1px solid #2a2d3a;}` + "\n")
	b.WriteString(`.prev:hover{color:#e8eaf0;border-color:#6b7280;}` + "\n")
	b.WriteString(`.next{background:#5b8dee;color:white;width:100%;text-align:center;}` + "\n")
	b.WriteString(`.next:hover{background:#4a7de0;}` + "\n")
	b.WriteString(`.next-done{opacity:0.3;cursor:not-allowed;}` + "\n")
	b.WriteString(`.next-done:hover{background:#5b8dee;}` + "\n")
	for i := 0; i < n; i++ {
		// Back: shown from s2 onward; s0=CTA, s1=first real step (no predecessor).
		if i >= 2 {
			fmt.Fprintf(&b, `#s%d:checked~.content .prev-%d{display:block;}`+"\n", i, i)
		}
		if i < n-1 {
			fmt.Fprintf(&b, `#s%d:checked~.content .next-%d{display:block;}`+"\n", i, i)
		} else {
			fmt.Fprintf(&b, `#s%d:checked~.content .next-done{display:block;}`+"\n", i)
		}
	}

	// CTA overlay.
	b.WriteString(`.cta-overlay{position:absolute;top:0;left:0;right:0;bottom:0;display:none;align-items:center;justify-content:center;cursor:pointer;background:rgba(15,17,23,0.45);}` + "\n")
	if n > 1 {
		b.WriteString(`#s0:checked~.cta-overlay{display:flex;}` + "\n")
	}
	b.WriteString(`.cta-btn{background:#0f1117;border:2px solid #5b8dee;border-radius:12px;padding:32px 72px;font-size:24px;font-weight:700;color:#5b8dee;letter-spacing:0.03em;}` + "\n")
	b.WriteString(`.cta-overlay:hover .cta-btn{background:#1a2744;border-color:#7da9f0;color:#7da9f0;}` + "\n")

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

// computeStepTransform returns the CSS transform (scale + translation) for one diagram step.
// bboxes holds natural-pixel bounding boxes of all highlighted and focused nodes;
// empty bboxes signals an overview step: scale the full diagram to fit, centered.
// Scale is capped at 1.0 so diagram text never exceeds narration text size.
func computeStepTransform(naturalW, naturalH int, bboxes []nodeBBox) stepTransform {
	vw, vh := float64(diagAreaW), float64(diagAreaH)
	nw, nh := float64(naturalW), float64(naturalH)

	if len(bboxes) == 0 {
		s := math.Min(vw/nw, vh/nh)
		if s > 1.0 {
			s = 1.0
		}
		return stepTransform{scale: s, tx: (vw - nw*s) / 2, ty: (vh - nh*s) / 2}
	}

	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, bb := range bboxes {
		if bb.X < x0 {
			x0 = bb.X
		}
		if bb.Y < y0 {
			y0 = bb.Y
		}
		if bb.X+bb.W > x1 {
			x1 = bb.X + bb.W
		}
		if bb.Y+bb.H > y1 {
			y1 = bb.Y + bb.H
		}
	}

	cx, cy := (x0+x1)/2, (y0+y1)/2
	sw := vw / ((x1 - x0) * (1 + bboxMargin))
	sh := vh / ((y1 - y0) * (1 + bboxMargin))
	s := math.Min(math.Min(sw, sh), 1.0)

	return stepTransform{scale: s, tx: vw/2 - cx*s, ty: vh/2 - cy*s}
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
