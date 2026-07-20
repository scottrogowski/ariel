package renderer_test

import (
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

// TestPanZoom_FitsDiagram_NeverMoves asserts that a diagram whose natural size
// fits within the container is shown at natural size with no transform on every
// step, including highlight steps.
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

// TestPanZoom_OverflowDiagram_Step2Centered asserts that step 2 (highlight G, H —
// right side of the chain) centers the union bbox within the container.
func TestPanZoom_OverflowDiagram_Step2Centered(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/overflows.ariel.yaml")
	s := browsertest.Open(t, htmlPath)

	st := s.GetViewport()
	if st.NaturalW <= st.ContainerW {
		t.Skipf("overflows fixture is not large enough: naturalW=%.0fpx containerW=%.0fpx", st.NaturalW, st.ContainerW)
	}

	s.Next() // → step 1
	s.Next() // → step 2: highlight G, H

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

// TestPanZoom_FitsDiagram_NoMoveOnHighlightSteps guards regression 2: pan/zoom
// must be completely disabled for diagrams that fit at natural scale, even on
// steps with highlight_nodes or focus_nodes. The spec states: "If the diagram
// fits at natural scale, it is shown at natural size and centered for every
// step — no panning or zooming, even on highlight steps."
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

	s.Next() // → step 2: highlight G, H (right)
	st2 := s.GetViewport()

	t.Logf("step 1 tx=%.1f  step 2 tx=%.1f  diff=%.1f", st1.TransformX, st2.TransformX, math.Abs(st1.TransformX-st2.TransformX))

	// A and B are on the far left; G and H on the far right. The translate X
	// values must differ by a substantial margin (20px minimum).
	if math.Abs(st1.TransformX-st2.TransformX) < 20 {
		t.Errorf("steps 1 and 2 have nearly identical X transforms (%.1f vs %.1f); expected significant difference for left vs right nodes", st1.TransformX, st2.TransformX)
	}
}
