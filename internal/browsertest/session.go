// Package browsertest drives generated ariel HTML output in headless Chrome and
// reads back computed geometry, navigation state, and highlight state.
package browsertest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// WindowW and WindowH are the browser viewport dimensions used by all sessions.
// At 1200×800, the diagram pane is approximately 859px wide (1200 − 340 narration − 1 border).
const (
	WindowW = 1200
	WindowH = 800
)

// Session is a headless Chrome session driving a single ariel HTML page.
// Call Open to create one; it is automatically closed when the test ends.
type Session struct {
	ctx    context.Context
	cancel context.CancelFunc
	t      *testing.T
}

// ViewportState captures the computed geometry of the diagram at the current step.
type ViewportState struct {
	NaturalW   float64 `json:"naturalW"`
	ContainerW float64 `json:"containerW"`
	ContainerH float64 `json:"containerH"`
	SVGWidth   float64 `json:"svgWidth"`
	SVGHeight  float64 `json:"svgHeight"`
	TransformX float64 `json:"transformX"`
	TransformY float64 `json:"transformY"`
	// CurrentStep is the zero-based JS currentStep index.
	CurrentStep int `json:"currentStep"`
}

// Open navigates to the given HTML file (absolute path) in headless Chrome and
// waits for the page to signal readiness via the #ariel-ready element.
// The session is automatically cancelled when the test ends.
func Open(t *testing.T, htmlPath string) *Session {
	t.Helper()

	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("no-sandbox", true),
			chromedp.WindowSize(WindowW, WindowH),
		)...,
	)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...any) {}))
	ctx, timeoutCancel := context.WithTimeout(ctx, 60*time.Second)

	s := &Session{ctx: ctx, t: t, cancel: func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
	}}
	t.Cleanup(s.cancel)

	if err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.WaitVisible("#ariel-ready", chromedp.ByID),
	); err != nil {
		t.Fatalf("browsertest.Open: %v", err)
	}

	return s
}

// Next advances to the next step (calls nextStep() in JS).
func (s *Session) Next() {
	s.t.Helper()
	s.exec("nextStep()")
}

// Back goes to the previous step (calls prevStep() in JS).
func (s *Session) Back() {
	s.t.Helper()
	s.exec("prevStep()")
}

// GoToStep jumps directly to step n (zero-based) in the current section.
func (s *Session) GoToStep(n int) {
	s.t.Helper()
	s.exec(fmt.Sprintf("goToStep(%d)", n))
}

// GetViewport returns the current computed geometry for the diagram area.
func (s *Session) GetViewport() ViewportState {
	s.t.Helper()
	raw := s.evalString(`(function() {
		var svg = document.querySelector('#mermaid-container svg');
		var c = document.getElementById('mermaid-container');
		if (!svg || !c) return '{}';
		var sw = parseFloat(svg.style.width) || 0;
		var sh = parseFloat(svg.style.height) || 0;
		var tx = 0, ty = 0;
		var m = (svg.style.transform || '').match(/translate\(([^,]+)px,\s*([^)]+)px\)/);
		if (m) { tx = parseFloat(m[1]); ty = parseFloat(m[2]); }
		return JSON.stringify({
			naturalW:   diagramNaturalW,
			containerW: c.clientWidth,
			containerH: c.clientHeight,
			svgWidth:   sw,
			svgHeight:  sh,
			transformX: tx,
			transformY: ty,
			currentStep: currentStep
		});
	})()`)

	var st ViewportState
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		s.t.Fatalf("browsertest.GetViewport: unmarshal %q: %v", raw, err)
	}
	return st
}

// BBoxCenterError returns the pixel distance between the union bounding box
// center of nodeIDs and the container center, after the current applyPanZoom
// transform. Both X and Y errors are returned. A correctly centered diagram
// returns (0, 0); tolerance is typically ≤5px.
func (s *Session) BBoxCenterError(nodeIDs []string) (errX, errY float64) {
	s.t.Helper()

	idsJSON, _ := json.Marshal(nodeIDs)
	js := fmt.Sprintf(`(function(ids) {
		var svg = document.querySelector('#mermaid-container svg');
		var c = document.getElementById('mermaid-container');
		if (!svg || !c) return JSON.stringify({errX:999, errY:999, reason:'no svg or container'});

		var vb = svg.viewBox.baseVal;
		if (!vb || vb.width === 0) return JSON.stringify({errX:999, errY:999, reason:'no viewBox'});

		var svgW = parseFloat(svg.style.width) || 0;
		var svgH = parseFloat(svg.style.height) || 0;
		var tx = 0, ty = 0;
		var m = (svg.style.transform || '').match(/translate\(([^,]+)px,\s*([^)]+)px\)/);
		if (m) { tx = parseFloat(m[1]); ty = parseFloat(m[2]); }

		var cW = c.clientWidth, cH = c.clientHeight;
		// getCTM() returns SVG viewport coords scaled by viewBox-to-viewport ratio.
		// currentScale converts back to viewBox units (matches applyPanZoom's coordinate space).
		var currentScale = (svgW > 0) ? svgW / vb.width : 1;
		var x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
		for (var i = 0; i < ids.length; i++) {
			var groups = nodeMap[ids[i]];
			if (!groups) continue;
			for (var j = 0; j < groups.length; j++) {
				try {
					var lb = groups[j].getBBox();
					var m = groups[j].getCTM();
					if (!m) continue;
					var corners = [
						[lb.x, lb.y], [lb.x + lb.width, lb.y],
						[lb.x, lb.y + lb.height], [lb.x + lb.width, lb.y + lb.height]
					];
					for (var k = 0; k < corners.length; k++) {
						var lx = corners[k][0], ly = corners[k][1];
						var vpx = m.a*lx + m.c*ly + m.e;
						var vpy = m.b*lx + m.d*ly + m.f;
						var vbx = vpx / currentScale + vb.x;
						var vby = vpy / currentScale + vb.y;
						x0 = Math.min(x0, vbx); y0 = Math.min(y0, vby);
						x1 = Math.max(x1, vbx); y1 = Math.max(y1, vby);
					}
				} catch(e) {}
			}
		}

		if (x0 === Infinity) return JSON.stringify({errX:999, errY:999, reason:'no bbox for nodes'});

		var cx = (x0 + x1) / 2;
		var cy = (y0 + y1) / 2;
		// Map viewBox center to CSS pixels within the SVG element.
		var cxPx = (cx - vb.x) / vb.width * svgW;
		var cyPx = (cy - vb.y) / vb.height * svgH;
		// Flex positions SVG top-left at ((cW-svgW)/2, (cH-svgH)/2).
		// Transform shifts visually. Bbox center in container coords:
		var actualX = (cW - svgW) / 2 + tx + cxPx;
		var actualY = (cH - svgH) / 2 + ty + cyPx;
		return JSON.stringify({
			errX: Math.abs(actualX - cW / 2),
			errY: Math.abs(actualY - cH / 2)
		});
	})(%s)`, string(idsJSON))

	raw := s.evalString(js)
	var result struct {
		ErrX   float64 `json:"errX"`
		ErrY   float64 `json:"errY"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		s.t.Fatalf("browsertest.BBoxCenterError: unmarshal %q: %v", raw, err)
	}
	if result.Reason != "" {
		s.t.Fatalf("browsertest.BBoxCenterError: %s", result.Reason)
	}
	return result.ErrX, result.ErrY
}

// exec runs a JS void expression and waits briefly for DOM to settle.
func (s *Session) exec(js string) {
	s.t.Helper()
	if err := chromedp.Run(s.ctx,
		chromedp.Evaluate(js, nil),
		chromedp.Sleep(50*time.Millisecond),
	); err != nil {
		s.t.Fatalf("browsertest.exec %q: %v", js, err)
	}
}

// evalString evaluates a JS expression that returns a string and returns it.
func (s *Session) evalString(js string) string {
	s.t.Helper()
	var result string
	if err := chromedp.Run(s.ctx, chromedp.Evaluate(js, &result)); err != nil {
		snippet := js
		if len(snippet) > 120 {
			snippet = snippet[:120] + "..."
		}
		s.t.Fatalf("browsertest.evalString: %v\nJS: %s", err, snippet)
	}
	return result
}
