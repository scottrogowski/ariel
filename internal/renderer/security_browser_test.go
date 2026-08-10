package renderer_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	browsertest "github.com/scottrogowski/ariel/dev-tools/e2e-tests"
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/renderer"
	"github.com/scottrogowski/ariel/internal/theme"
)

// This test prevents walkthrough text from terminating the generated page's script element.
func TestGeneratedHTMLTreatsWalkthroughTextAsData(t *testing.T) {
	title := `</title><script>window.arielTitleInjected=true</script>`
	narration := `</script><script>window.arielNarrationInjected=true</script>`
	walkthrough := &dsl.Walkthrough{
		Title:          title,
		MermaidDiagram: "graph TD\n  A[Safe]",
		Steps: []dsl.Step{
			{Narration: "Overview"},
			{Narration: narration},
		},
	}
	generatedHTML, err := renderer.Generate(walkthrough, theme.ModeDark)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if strings.Contains(generatedHTML, "cdnjs.cloudflare.com") {
		t.Fatal("generated HTML depends on the Mermaid CDN")
	}
	path := filepath.Join(t.TempDir(), "security.html")
	if err := renderer.WriteFile(path, generatedHTML); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	session := browsertest.Open(t, path)
	session.Next()
	if !session.WaitTrue(`document.getElementById('narration').textContent.includes('arielNarrationInjected')`, time.Second) {
		t.Fatal("malicious narration did not render as text")
	}
	got := session.Eval(`JSON.stringify({
    injected: window.arielTitleInjected === true || window.arielNarrationInjected === true,
    title: document.querySelector('.page-title').firstChild.textContent,
    narration: document.getElementById('narration').textContent
  })`)
	want := `{"injected":false,"title":"` + title + `","narration":"` + narration + `"}`
	if got != want {
		t.Errorf("generated page state = %s, want %s", got, want)
	}
}
