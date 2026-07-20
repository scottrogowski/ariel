package renderer_test

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/scottrogowski/ariel/internal/browsertest"
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/renderer"
	"github.com/scottrogowski/ariel/internal/theme"
)

// parseWalkthrough parses a fixture into a Walkthrough, failing on any error-severity issue.
func parseWalkthrough(t *testing.T, fixturePath string) *dsl.Walkthrough {
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
			t.Fatalf("fixture %s has error: %v", fixturePath, iss.Message)
		}
	}
	return w
}

// freePort reserves and releases an ephemeral TCP port, returning its number.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

// waitForServer blocks until the watch server accepts TCP connections on port.
func waitForServer(t *testing.T, port int) {
	t.Helper()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("watch server did not start on port %d", port)
}

// TestWatch_UpdateRerendersDiagram guards the live-reload bug where a file change
// re-rendered a blank diagram (resolved only by a manual refresh): the old page's
// top-level globals survived the reload and aborted the bootstrap script before
// mermaid rendered. Asserts a file change auto-renders the new diagram in-browser.
func TestWatch_UpdateRerendersDiagram(t *testing.T) {
	initial := parseWalkthrough(t, "../../testdata/fits.ariel.yaml")
	updated := parseWalkthrough(t, "../../testdata/overflows.ariel.yaml")

	port := freePort(t)
	initialHTML, err := renderer.RenderWatch(initial, port, theme.ModeDark)
	if err != nil {
		t.Fatalf("renderer.RenderWatch: %v", err)
	}

	srv := renderer.NewWatchServer("../../testdata/fits.ariel.yaml", port, initialHTML, theme.ModeDark)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	waitForServer(t, port)

	s := browsertest.OpenURL(t, fmt.Sprintf("http://127.0.0.1:%d", port))

	// Precondition: the initial (fits) diagram has node A but not node K.
	if s.Eval("(!!nodeMap['K']).toString()") == "true" {
		t.Fatal("precondition failed: fits fixture unexpectedly has node K")
	}

	// Simulate a file change on disk being picked up by the watcher.
	srv.UpdateContent(updated)

	// The updated (overflows) diagram must render automatically: SVG present,
	// node K mapped, and the page re-signalled readiness.
	rendered := s.WaitTrue(`(function() {
		var svg = document.querySelector('#mermaid-container svg');
		var ready = document.getElementById('ariel-ready');
		return !!svg
			&& typeof nodeMap !== 'undefined' && !!nodeMap['K']
			&& !!ready && ready.style.display === 'block';
	})()`, 15*time.Second)

	if !rendered {
		t.Fatal("diagram did not re-render after file change (blank-diagram bug)")
	}
}
