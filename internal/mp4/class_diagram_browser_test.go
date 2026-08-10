package mp4

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/theme"
)

// This test prevents MP4 capture pages from losing class mapping or visual emphasis.
func TestClassDiagramCapturePage(t *testing.T) {
	walkthrough, issues, err := dsl.ParseFile("../../testdata/class-diagram.ariel.yaml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("fixture issues = %+v, want none", issues)
	}

	pagePath := filepath.Join(t.TempDir(), "class.html")
	html := buildSectionHTML(theme.ModeDark.Baked(), walkthrough.Title, walkthrough.ToSections()[0])
	if strings.Contains(html, "cdnjs.cloudflare.com") {
		t.Fatal("MP4 capture page depends on the Mermaid CDN")
	}
	if err := os.WriteFile(pagePath, []byte(html), 0644); err != nil {
		t.Fatalf("write capture page: %v", err)
	}

	ctx, cancel := newBrowserCtx()
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+pagePath),
		chromedp.WaitVisible("#ready", chromedp.ByID),
		chromedp.Evaluate(`applyStep(['Handler'], ['Service'], 'Request entry', 'Delegates work.')`, nil),
	); err != nil {
		t.Fatalf("render capture page: %v", err)
	}

	var raw string
	js := `JSON.stringify({
    mapped: Object.keys(nodeMap).length,
    handler: nodeMap.Handler[0].classList.contains('highlighted'),
    service: nodeMap.Service[0].classList.contains('active'),
    repository: getComputedStyle(nodeMap.Repository[0]).opacity === '0.4',
    annotation: nodeMap.Repository[0].textContent.includes('interface'),
    edgeAnimated: edgeMap[arielEdgeKey('Handler', 'Service')][0].classList.contains('animated'),
    label: document.getElementById('step-label').textContent,
    narration: document.getElementById('narration').textContent
  })`
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &raw)); err != nil {
		t.Fatalf("read class state: %v", err)
	}
	want := `{"mapped":6,"handler":true,"service":true,"repository":true,"annotation":true,"edgeAnimated":true,"label":"Request entry","narration":"Delegates work."}`
	if raw != want {
		t.Errorf("capture class state = %s, want %s", raw, want)
	}

	// This check prevents missing rendered nodes from producing incomplete frames.
	err = chromedp.Run(ctx, chromedp.Evaluate(
		`applyStep(['Missing'], [], 'Invalid', 'Invalid')`, nil,
	))
	if err == nil || !strings.Contains(err.Error(), "Ariel could not map rendered Mermaid nodes: Missing") {
		t.Errorf("missing node error = %v, want rendered-node mapping error", err)
	}
}

// This test prevents walkthrough titles from executing code during MP4 capture.
func TestCapturePageTreatsTitleAsText(t *testing.T) {
	title := `</title><script>window.arielTitleInjected=true</script>`
	section := dsl.Section{MermaidDiagram: "graph TD\n  A[Safe]"}
	pagePath := filepath.Join(t.TempDir(), "security.html")
	if err := os.WriteFile(pagePath, []byte(buildSectionHTML(theme.ModeDark.Baked(), title, section)), 0644); err != nil {
		t.Fatalf("write capture page: %v", err)
	}

	ctx, cancel := newBrowserCtx()
	defer cancel()
	var state string
	if err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+pagePath),
		chromedp.WaitVisible("#ready", chromedp.ByID),
		chromedp.Evaluate(`JSON.stringify({
      injected: window.arielTitleInjected === true,
      title: document.querySelector('.page-title').textContent
    })`, &state),
	); err != nil {
		t.Fatalf("render capture page: %v", err)
	}
	want := `{"injected":false,"title":"` + title + `"}`
	if state != want {
		t.Errorf("capture page state = %s, want %s", state, want)
	}
}
