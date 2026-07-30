package renderer_test

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"

	"github.com/scottrogowski/ariel/internal/browsertest"
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/renderer"
	"github.com/scottrogowski/ariel/internal/theme"
)

const centerTolerance = 5.0 // acceptable pixel error for bbox centering

// generateHTML parses fixturePath and writes a generated HTML file to a temp
// directory, returning its absolute path.
func generateHTML(t *testing.T, fixturePath string) string {
	t.Helper()
	abs, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	w, issues, err := dsl.ParseFile(abs)
	if err != nil {
		t.Fatalf("dsl.ParseFile: %v", err)
	}
	for _, iss := range issues {
		if iss.Severity == dsl.SeverityError {
			t.Fatalf("fixture has error: %v", iss.Message)
		}
	}
	html, err := renderer.Generate(w, theme.ModeDark)
	if err != nil {
		t.Fatalf("renderer.Generate: %v", err)
	}
	out := filepath.Join(t.TempDir(), "test.html")
	if err := renderer.WriteFile(out, html); err != nil {
		t.Fatalf("renderer.WriteFile: %v", err)
	}
	return out
}

func assertViewportMatches(
	t *testing.T,
	got browsertest.ViewportState,
	want browsertest.ViewportState,
	context string,
) {
	t.Helper()
	values := []struct {
		name string
		got  float64
		want float64
	}{
		{name: "width", got: got.SVGWidth, want: want.SVGWidth},
		{name: "height", got: got.SVGHeight, want: want.SVGHeight},
		{name: "X translation", got: got.TransformX, want: want.TransformX},
		{name: "Y translation", got: got.TransformY, want: want.TransformY},
	}
	for _, value := range values {
		if math.Abs(value.got-value.want) > 0.5 {
			t.Errorf("%s %s = %.1f, want %.1f", context, value.name, value.got, value.want)
		}
	}
}

// TestPanZoom_FitsDiagram_NeverMoves asserts that automatic framing stays
// centered at natural size for each step when the diagram fits.
func TestPanZoom_FitsDiagram_NeverMoves(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st0 := s.GetViewport()
	if st0.NaturalW > st0.ContainerW {
		t.Skipf("fits fixture is not small enough: naturalW=%.0fpx containerW=%.0fpx", st0.NaturalW, st0.ContainerW)
	}

	// Step 0 (overview), step 1 (highlight A), step 2 (highlight B) — all must be static.
	for i := 0; i < 3; i++ {
		st := s.GetViewport()
		t.Logf("step %d: naturalW=%.1f svgW=%.1f tx=%.1f ty=%.1f", i, st.NaturalW, st.SVGWidth, st.TransformX, st.TransformY)

		if math.Abs(st.SVGWidth-st.NaturalW) > 1.0 {
			t.Errorf("step %d: SVGWidth(%.1f) should equal NaturalW(%.1f)", i, st.SVGWidth, st.NaturalW)
		}
		if math.Abs(st.TransformX) > 0.5 || math.Abs(st.TransformY) > 0.5 {
			t.Errorf("step %d: expected no transform, got translate(%.1f, %.1f)", i, st.TransformX, st.TransformY)
		}

		if i < 2 {
			s.Next()
		}
	}
}

// TestPanZoom_OverflowDiagram_OverviewScalesToFit asserts that an overflowing
// diagram is scaled to fit the container on the overview step with no transform.
func TestPanZoom_OverflowDiagram_OverviewScalesToFit(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	t.Logf("overview: naturalW=%.1f containerW=%.1f svgW=%.1f tx=%.1f ty=%.1f", st.NaturalW, st.ContainerW, st.SVGWidth, st.TransformX, st.TransformY)

	if st.NaturalW <= st.ContainerW {
		t.Skipf("overflows fixture is not large enough: naturalW=%.0fpx containerW=%.0fpx", st.NaturalW, st.ContainerW)
	}
	if st.SVGWidth >= st.NaturalW {
		t.Errorf("overview: SVGWidth(%.1f) should be less than NaturalW(%.1f)", st.SVGWidth, st.NaturalW)
	}
	if st.SVGWidth > st.ContainerW+1.0 {
		t.Errorf("overview: SVGWidth(%.1f) exceeds ContainerW(%.1f)", st.SVGWidth, st.ContainerW)
	}
	if math.Abs(st.TransformX) > 0.5 || math.Abs(st.TransformY) > 0.5 {
		t.Errorf("overview: expected no transform, got translate(%.1f, %.1f)", st.TransformX, st.TransformY)
	}
}

// TestPanZoom_OverflowDiagram_Step1Centered asserts that step 1 (highlight A, B —
// left side of the chain) centers the union bbox within the container.
func TestPanZoom_OverflowDiagram_Step1Centered(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	if st.NaturalW <= st.ContainerW {
		t.Skipf("overflows fixture is not large enough: naturalW=%.0fpx containerW=%.0fpx", st.NaturalW, st.ContainerW)
	}

	s.Next() // → step 1: highlight A, B

	st1 := s.GetViewport()
	t.Logf("step 1: svgW=%.1f tx=%.1f ty=%.1f", st1.SVGWidth, st1.TransformX, st1.TransformY)

	errX, errY := s.BBoxCenterError([]string{"A", "B"})
	t.Logf("step 1: bbox center error errX=%.1fpx errY=%.1fpx", errX, errY)

	if errX > centerTolerance || errY > centerTolerance {
		t.Errorf("step 1: [A,B] bbox center off by (%.1fpx, %.1fpx), tolerance %.1fpx", errX, errY, centerTolerance)
	}
}

// TestPanZoom_OverflowDiagram_Step2Centered asserts that step 2 (highlight K, L —
// right side of the chain) centers the union bbox within the container.
func TestPanZoom_OverflowDiagram_Step2Centered(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	if st.NaturalW <= st.ContainerW {
		t.Skipf("overflows fixture is not large enough: naturalW=%.0fpx containerW=%.0fpx", st.NaturalW, st.ContainerW)
	}

	s.Next() // → step 1
	s.Next() // → step 2: highlight K, L

	st2 := s.GetViewport()
	t.Logf("step 2: svgW=%.1f tx=%.1f ty=%.1f", st2.SVGWidth, st2.TransformX, st2.TransformY)

	errX, errY := s.BBoxCenterError([]string{"K", "L"})
	t.Logf("step 2: bbox center error errX=%.1fpx errY=%.1fpx", errX, errY)

	if errX > centerTolerance || errY > centerTolerance {
		t.Errorf("step 2: [K,L] bbox center off by (%.1fpx, %.1fpx), tolerance %.1fpx", errX, errY, centerTolerance)
	}
}

// TestPanZoom_FitsDiagram_ContainerHasHeight is a regression guard: asserts
// that the mermaid-container has non-zero height on the very first render
// (before any user interaction). A zero height would clip the SVG entirely
// via overflow:hidden, making the first section's overview invisible.
func TestPanZoom_FitsDiagram_ContainerHasHeight(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	t.Logf("step 0: containerW=%.0f containerH=%.0f svgW=%.1f", st.ContainerW, st.ContainerH, st.SVGWidth)

	if st.ContainerH == 0 {
		t.Error("container height is 0 on initial render: SVG would be clipped by overflow:hidden")
	}
	if st.SVGWidth == 0 {
		t.Error("SVG has no width on initial render: diagram is invisible")
	}
}

// TestPanZoom_MultiSection_FirstSectionOverviewVisible is a regression guard:
// when a walkthrough has multiple sections, the initial load renders the first
// section's overview (step 0) correctly. Previously, the body layout might not
// have settled before applyPanZoom read container.clientHeight, causing a zero
// height and a fully-clipped SVG.
func TestPanZoom_MultiSection_FirstSectionOverviewVisible(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/multi-section.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	// Initial state: section 1, step 0 (overview, no highlights).
	st0 := s.GetViewport()
	t.Logf("section 1 step 0: containerH=%.0f svgW=%.1f svgH=%.1f tx=%.1f ty=%.1f",
		st0.ContainerH, st0.SVGWidth, st0.SVGHeight, st0.TransformX, st0.TransformY)

	if st0.ContainerH == 0 {
		t.Fatal("container height is 0 on initial render: first section overview is invisible")
	}
	if st0.SVGWidth == 0 {
		t.Fatal("SVG width is 0 on initial render: first section overview diagram not rendered")
	}
	// Step 0 is a fits diagram — no transform.
	if math.Abs(st0.TransformX) > 0.5 || math.Abs(st0.TransformY) > 0.5 {
		t.Errorf("step 0 should have no transform, got translate(%.1f, %.1f)", st0.TransformX, st0.TransformY)
	}

	// Navigate into section 1 highlights and back to overview — verify consistency.
	s.Next() // → step 1: highlight A
	st1 := s.GetViewport()
	t.Logf("section 1 step 1: svgW=%.1f tx=%.1f", st1.SVGWidth, st1.TransformX)

	if math.Abs(st1.SVGWidth-st0.SVGWidth) > 1.0 {
		t.Errorf("fits diagram: SVGWidth changed between steps (step0=%.1f step1=%.1f)", st0.SVGWidth, st1.SVGWidth)
	}
}

// TestPanZoom_FitsDiagram_NoMoveOnHighlightSteps guards automatic framing for
// diagrams that fit at natural scale, including highlighted and focused steps.
func TestPanZoom_FitsDiagram_NoMoveOnHighlightSteps(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st0 := s.GetViewport()
	if st0.NaturalW > st0.ContainerW {
		t.Skipf("fits fixture is not small enough: naturalW=%.0fpx containerW=%.0fpx", st0.NaturalW, st0.ContainerW)
	}

	// Step 0: overview (no highlights). Step 1, 2: highlight steps.
	for step := 0; step < 3; step++ {
		st := s.GetViewport()
		t.Logf("step %d: svgW=%.1f tx=%.1f ty=%.1f", step, st.SVGWidth, st.TransformX, st.TransformY)

		if st.ContainerH == 0 {
			t.Errorf("step %d: container height is 0", step)
		}
		if math.Abs(st.SVGWidth-st.NaturalW) > 1.0 {
			t.Errorf("step %d: SVGWidth(%.1f) != NaturalW(%.1f); fits diagram must not resize", step, st.SVGWidth, st.NaturalW)
		}
		if math.Abs(st.TransformX) > 0.5 || math.Abs(st.TransformY) > 0.5 {
			t.Errorf("step %d: fits diagram must not pan/zoom, got translate(%.1f, %.1f)", step, st.TransformX, st.TransformY)
		}

		if step < 2 {
			s.Next()
		}
	}
}

// TestPanZoom_OverflowDiagram_HighlightStepsDiffer asserts that step 1 (left
// nodes) and step 2 (right nodes) produce meaningfully different transforms.
// This is a regression guard against "only the first step changes."
func TestPanZoom_OverflowDiagram_HighlightStepsDiffer(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	if st.NaturalW <= st.ContainerW {
		t.Skipf("overflows fixture is not large enough: naturalW=%.0fpx containerW=%.0fpx", st.NaturalW, st.ContainerW)
	}

	s.Next() // → step 1: highlight A, B (left)
	st1 := s.GetViewport()

	s.Next() // → step 2: highlight K, L (right)
	st2 := s.GetViewport()

	t.Logf("step 1 tx=%.1f  step 2 tx=%.1f  diff=%.1f", st1.TransformX, st2.TransformX, math.Abs(st1.TransformX-st2.TransformX))

	// A and B are on the far left; K and L are on the far right. The translate X
	// values must differ by a substantial margin (20px minimum).
	if math.Abs(st1.TransformX-st2.TransformX) < 20 {
		t.Errorf("steps 1 and 2 have nearly identical X transforms (%.1f vs %.1f); expected significant difference for left vs right nodes", st1.TransformX, st2.TransformX)
	}
}

// This test prevents pointer dragging from moving the page without moving the diagram.
func TestPanZoom_PointerDragPansDiagramAndChangesCursor(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	before := s.GetViewport()
	cursorStates := s.Eval(`(function() {
		var c = document.getElementById('mermaid-container');
		var initial = getComputedStyle(c).cursor;
		c.dispatchEvent(new PointerEvent('pointerdown', {
			pointerId: 1, button: 0, clientX: 100, clientY: 100, bubbles: true
		}));
		var dragging = getComputedStyle(c).cursor;
		c.dispatchEvent(new PointerEvent('pointerup', {
			pointerId: 1, button: 0, clientX: 100, clientY: 100, bubbles: true
		}));
		return JSON.stringify([initial, dragging, getComputedStyle(c).cursor]);
	})()`)
	s.DragDiagram(45, 25)
	after := s.GetViewport()

	if cursorStates != `["grab","grabbing","grab"]` {
		t.Errorf("cursor states = %s, want grab, grabbing, grab", cursorStates)
	}
	if math.Abs(after.TransformX-before.TransformX-45) > 0.5 {
		t.Errorf("drag X translation = %.1f, want 45", after.TransformX-before.TransformX)
	}
	if math.Abs(after.TransformY-before.TransformY-25) > 0.5 {
		t.Errorf("drag Y translation = %.1f, want 25", after.TransformY-before.TransformY)
	}
}

// This test prevents dragging a navigable node from changing the walkthrough step.
func TestPanZoom_NodeDragDoesNotReplaceNodeClickNavigation(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	stepAfterDrag := s.Eval(`(function() {
		var node = nodeMap.A[0];
		node.dispatchEvent(new PointerEvent('pointerdown', {
			pointerId: 1, button: 0, clientX: 100, clientY: 100, bubbles: true
		}));
		node.dispatchEvent(new PointerEvent('pointermove', {
			pointerId: 1, buttons: 1, clientX: 145, clientY: 125, bubbles: true
		}));
		node.dispatchEvent(new PointerEvent('pointerup', {
			pointerId: 1, button: 0, clientX: 145, clientY: 125, bubbles: true
		}));
		node.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
		return String(currentStep);
	})()`)
	s.ClickFlowchartNode("A")
	stepAfterClick := s.GetViewport().CurrentStep

	if stepAfterDrag != "0" {
		t.Errorf("step after node drag = %s, want 0", stepAfterDrag)
	}
	if stepAfterClick != 1 {
		t.Errorf("step after node click = %d, want 1", stepAfterClick)
	}
}

// This test prevents trackpad pinch handling from using discrete or capped zoom levels.
func TestPanZoom_TrackpadPinchZoomsContinuouslyWithoutFixedLimit(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	before := s.GetViewport()
	anchorX, anchorY := s.ElementCenter(`[id^="flowchart-A-"]`)
	s.PinchDiagramAt(-20, anchorX, anchorY)
	firstZoom := s.GetViewport()
	zoomedAnchorX, zoomedAnchorY := s.ElementCenter(`[id^="flowchart-A-"]`)

	s.PinchDiagramAt(-180, zoomedAnchorX, zoomedAnchorY)
	secondZoom := s.GetViewport()

	if firstZoom.SVGWidth <= before.SVGWidth {
		t.Errorf("first pinch width = %.1f, want greater than %.1f", firstZoom.SVGWidth, before.SVGWidth)
	}
	if math.Abs(zoomedAnchorX-anchorX) > 1 || math.Abs(zoomedAnchorY-anchorY) > 1 {
		t.Errorf(
			"pinch anchor moved from (%.1f, %.1f) to (%.1f, %.1f)",
			anchorX,
			anchorY,
			zoomedAnchorX,
			zoomedAnchorY,
		)
	}
	if secondZoom.SVGWidth <= before.SVGWidth*5 {
		t.Errorf("continuous pinch width = %.1f, want greater than %.1f", secondZoom.SVGWidth, before.SVGWidth*5)
	}
}

// This test prevents manual viewport changes from replacing each step's automatic initial framing.
func TestPanZoom_StepNavigationResetsManualViewport(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	s.Next()
	initialStepOne := s.GetViewport()
	s.DragDiagram(50, 30)
	s.ClickElement("#btn-zoom-in")
	manualStepOne := s.GetViewport()
	s.Next()
	s.Back()
	resetStepOne := s.GetViewport()

	if math.Abs(manualStepOne.TransformX-initialStepOne.TransformX) < 20 {
		t.Errorf("manual translation did not change: before %.1f, after %.1f", initialStepOne.TransformX, manualStepOne.TransformX)
	}
	if manualStepOne.SVGWidth <= initialStepOne.SVGWidth {
		t.Errorf("manual width = %.1f, want greater than %.1f", manualStepOne.SVGWidth, initialStepOne.SVGWidth)
	}
	assertViewportMatches(t, resetStepOne, initialStepOne, "step reset")
}

// This test prevents manual viewport state from surviving a browser refresh.
func TestPanZoom_BrowserRefreshRestoresInitialViewport(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	initial := s.GetViewport()
	s.DragDiagram(50, 30)
	s.ClickElement("#btn-zoom-in")
	manual := s.GetViewport()
	s.Reload()
	refreshed := s.GetViewport()

	if math.Abs(manual.TransformX-initial.TransformX) < 20 {
		t.Errorf("manual translation did not change: before %.1f, after %.1f", initial.TransformX, manual.TransformX)
	}
	if manual.SVGWidth <= initial.SVGWidth {
		t.Errorf("manual width = %.1f, want greater than %.1f", manual.SVGWidth, initial.SVGWidth)
	}
	assertViewportMatches(t, refreshed, initial, "refresh reset")
}

// This test prevents the floating zoom controls from moving or changing order.
func TestPanZoom_ButtonsFloatAtDiagramBottomRight(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	raw := s.Eval(`(function() {
		var pane = document.querySelector('.diagram-pane').getBoundingClientRect();
		var controls = document.querySelector('.diagram-zoom-controls');
		var zoomIn = document.getElementById('btn-zoom-in').getBoundingClientRect();
		var zoomOut = document.getElementById('btn-zoom-out').getBoundingClientRect();
		return JSON.stringify({
			position: getComputedStyle(controls).position,
			rightStyle: getComputedStyle(controls).right,
			bottomStyle: getComputedStyle(controls).bottom,
			gap: zoomOut.top - zoomIn.bottom,
			right: pane.right - zoomOut.right,
			bottom: pane.bottom - zoomOut.bottom,
			zoomInText: document.getElementById('btn-zoom-in').textContent,
			zoomOutText: document.getElementById('btn-zoom-out').textContent
		});
	})()`)
	var layout struct {
		Position    string  `json:"position"`
		RightStyle  string  `json:"rightStyle"`
		BottomStyle string  `json:"bottomStyle"`
		Gap         float64 `json:"gap"`
		Right       float64 `json:"right"`
		Bottom      float64 `json:"bottom"`
		ZoomInText  string  `json:"zoomInText"`
		ZoomOutText string  `json:"zoomOutText"`
	}
	if err := json.Unmarshal([]byte(raw), &layout); err != nil {
		t.Fatalf("unmarshal zoom control layout %q: %v", raw, err)
	}

	if layout.Position != "absolute" {
		t.Errorf("control position = %q, want absolute", layout.Position)
	}
	if math.Abs(layout.Gap-8) > 0.5 {
		t.Errorf("control gap = %.1f, want 8", layout.Gap)
	}
	if layout.RightStyle != "20px" || layout.BottomStyle != "20px" {
		t.Errorf("control inset = %q and %q, want 20px", layout.RightStyle, layout.BottomStyle)
	}
	if layout.Right < 0 || layout.Bottom < 0 {
		t.Errorf("controls exceed pane by right %.1f or bottom %.1f", layout.Right, layout.Bottom)
	}
	if layout.ZoomInText != "+" || layout.ZoomOutText != "−" {
		t.Errorf("control text = %q and %q, want + and −", layout.ZoomInText, layout.ZoomOutText)
	}
}

// This test prevents zoom buttons from drifting away from reciprocal center scaling.
func TestPanZoom_ButtonsZoomByReciprocalFactors(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	initial := s.GetViewport()
	s.ClickElement("#btn-zoom-in")
	zoomedIn := s.GetViewport()
	s.ClickElement("#btn-zoom-out")
	zoomedOut := s.GetViewport()

	if math.Abs(zoomedIn.SVGWidth/initial.SVGWidth-1.25) > 0.001 {
		t.Errorf("zoom-in factor = %.3f, want 1.25", zoomedIn.SVGWidth/initial.SVGWidth)
	}
	if math.Abs(zoomedOut.SVGWidth-initial.SVGWidth) > 0.5 {
		t.Errorf("reciprocal zoom width = %.1f, want %.1f", zoomedOut.SVGWidth, initial.SVGWidth)
	}
	if math.Abs(zoomedOut.TransformX-initial.TransformX) > 0.5 ||
		math.Abs(zoomedOut.TransformY-initial.TransformY) > 0.5 {
		t.Errorf(
			"reciprocal zoom translation = (%.1f, %.1f), want (%.1f, %.1f)",
			zoomedOut.TransformX,
			zoomedOut.TransformY,
			initial.TransformX,
			initial.TransformY,
		)
	}
}

// This test prevents Space on a zoom button from advancing the walkthrough.
func TestPanZoom_ButtonSpaceActivatesZoomWithoutChangingStep(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/fits.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	initial := s.GetViewport()
	s.PressElement("#btn-zoom-in", " ")
	afterSpace := s.GetViewport()

	if afterSpace.CurrentStep != initial.CurrentStep {
		t.Errorf("step after zoom button Space = %d, want %d", afterSpace.CurrentStep, initial.CurrentStep)
	}
	if math.Abs(afterSpace.SVGWidth/initial.SVGWidth-1.25) > 0.001 {
		t.Errorf("Space zoom factor = %.3f, want 1.25", afterSpace.SVGWidth/initial.SVGWidth)
	}
}
