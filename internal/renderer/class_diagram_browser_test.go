package renderer_test

import (
	"math"
	"testing"
	"time"

	browsertest "github.com/scottrogowski/ariel/dev-tools/e2e-tests"
)

// This test prevents class mapping, visual emphasis, framing, and navigation from drifting apart.
func TestClassDiagramBrowserBehavior(t *testing.T) {
	htmlPath := generateHTML(t, "../../testdata/class-diagram.ariel.yaml")
	session := browsertest.Open(t, htmlPath)

	assertClassMappings(t, session)
	session.Next()
	if !session.WaitTrue(`getComputedStyle(nodeMap.Repository[0]).opacity === '0.4'`, time.Second) {
		t.Fatal("class emphasis transition did not finish")
	}
	assertClassVisualStates(t, session)
	assertClassFraming(t, session)
	assertClassNavigation(t, session)
}

// This test prevents sequence mapping from mixing SVG and screen coordinates.
func TestSequenceDiagramElementMapping(t *testing.T) {
	htmlPath := generateHTML(t, "../../examples/example-input/how-ariel-works.ariel.yaml")
	session := browsertest.Open(t, htmlPath)

	if got := session.Eval(`Object.values(nodeMap).every(elements => elements.length === 2).toString()`); got != "true" {
		t.Errorf("all sequence actors mapped twice = %s, want true", got)
	}
	wantEdges := "C-CLI,CLI-C,CLI-HP,CLI-HR,HP-C"
	if got := session.Eval(`Object.keys(edgeMap).sort().join(',')`); got != wantEdges {
		t.Errorf("sequence edges = %s, want %s", got, wantEdges)
	}
	session.Next()
	if got := session.Eval(`document.querySelectorAll('[data-ariel-edge-source].animated').length.toString()`); got != "0" {
		t.Errorf("animated sequence edges = %s, want 0", got)
	}
}

func assertClassMappings(t *testing.T, session *browsertest.Session) {
	t.Helper()
	got := session.Eval(`JSON.stringify({
    ids: Object.keys(nodeMap).sort().join(','),
    single: Object.values(nodeMap).every(elements => elements.length === 1),
    annotation: nodeMap.Repository[0].textContent.includes('interface')
  })`)
	want := `{"ids":"Cache,Handler,Metrics,Repository,SQLStore,Service","single":true,"annotation":true}`
	if got != want {
		t.Errorf("class mappings = %s, want %s", got, want)
	}
}

func assertClassVisualStates(t *testing.T, session *browsertest.Session) {
	t.Helper()
	got := session.Eval(`JSON.stringify({
    handler: nodeMap.Handler[0].classList.contains('highlighted'),
    handlerFill: getComputedStyle(nodeMap.Handler[0].querySelector('rect')).fill,
    service: nodeMap.Service[0].classList.contains('active'),
    serviceFill: getComputedStyle(nodeMap.Service[0].querySelector('rect')).fill,
    dimmed: ['Repository', 'SQLStore', 'Cache', 'Metrics'].every(id =>
      nodeMap[id][0].classList.contains('dimmed') && getComputedStyle(nodeMap[id][0]).opacity === '0.4'),
    edgeAnimated: edgeMap['Handler-Service'][0].classList.contains('animated')
  })`)
	want := `{"handler":true,"handlerFill":"rgb(30, 58, 110)","service":true,"serviceFill":"rgb(26, 74, 122)","dimmed":true,"edgeAnimated":true}`
	if got != want {
		t.Errorf("class visual states = %s, want %s", got, want)
	}
}

func assertClassFraming(t *testing.T, session *browsertest.Session) {
	t.Helper()
	viewport := session.GetViewport()
	if viewport.NaturalW <= viewport.ContainerW {
		t.Fatalf("class fixture natural width %.1f must exceed container width %.1f", viewport.NaturalW, viewport.ContainerW)
	}
	errorX, errorY := session.BBoxCenterError([]string{"Handler", "Service"})
	if math.Max(errorX, errorY) > centerTolerance {
		t.Errorf("class framing error = (%.1fpx, %.1fpx), want at most %.1fpx", errorX, errorY, centerTolerance)
	}
}

func assertClassNavigation(t *testing.T, session *browsertest.Session) {
	t.Helper()
	session.ClickElement(`[data-ariel-node-id="Service"]`)
	if step := session.GetViewport().CurrentStep; step != 2 {
		t.Errorf("Service click advanced to step %d, want 2", step)
	}
	session.ClickElement(`[data-ariel-node-id="Service"]`)
	if step := session.GetViewport().CurrentStep; step != 1 {
		t.Errorf("Service click cycled to step %d, want 1", step)
	}
}
