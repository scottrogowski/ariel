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
	assertClassEdgeAnimation(t, session)
	assertClassNavigation(t, session)
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
      nodeMap[id][0].classList.contains('dimmed') && getComputedStyle(nodeMap[id][0]).opacity === '0.4')
  })`)
	want := `{"handler":true,"handlerFill":"rgb(30, 58, 110)","service":true,"serviceFill":"rgb(26, 74, 122)","dimmed":true}`
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

func assertClassEdgeAnimation(t *testing.T, session *browsertest.Session) {
	t.Helper()
	animated := session.Eval(`JSON.stringify({
    mapped: (edgeMap['Handler-Service'] || []).length,
    animated: (edgeMap['Handler-Service'] || []).filter(element => element.classList.contains('animated')).length
  })`)
	if animated != `{"mapped":1,"animated":1}` {
		t.Errorf("Handler-Service edge state = %s, want one mapped animated edge", animated)
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
