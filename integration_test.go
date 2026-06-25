// Package main_test provides end-to-end CLI integration tests.
//
// These tests build the ariel binary and invoke it as a subprocess, exercising
// the full stack from CLI flag parsing through DSL verification and rendering.
//
// VISUAL OUTPUT TESTING LIMITATION: ariel generate produces HTML and MP4 files
// whose visual correctness (node highlighting, edge animation, layout, video
// playback) cannot be verified automatically. After any change to the renderer
// template, the MP4 capture pipeline, or CSS/JS, a human must:
//   - Open the generated HTML in a browser and step through it
//   - Play the MP4 and confirm distinct frames and correct highlighting
package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "ariel-inttest-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}
	binaryPath = filepath.Join(tmp, "ariel")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build binary:\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// run invokes the ariel binary with the given arguments and returns stdout,
// stderr, and the process exit code.
func run(args ...string) (stdout, stderr string, exitCode int) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return outBuf.String(), errBuf.String(), exit.ExitCode()
		}
		return outBuf.String(), errBuf.String(), -1
	}
	return outBuf.String(), errBuf.String(), 0
}

// TestCLI_VerifyKnownGoodFile confirms that the included testdata file is valid:
// verify must exit 0 and produce no error lines (warnings are acceptable).
func TestCLI_VerifyKnownGoodFile(t *testing.T) {
	stdout, _, exitCode := run("verify", "testdata/auth-flow.ariel.yaml")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}
	if strings.Contains(stdout, ": error:") {
		t.Errorf("expected no error lines in output, got: %q", stdout)
	}
}

// TestCLI_VerifyMissingFile confirms that a missing file produces exit code 2.
func TestCLI_VerifyMissingFile(t *testing.T) {
	_, stderr, exitCode := run("verify", "nonexistent.ariel.yaml")
	if exitCode != 2 {
		t.Fatalf("expected exit 2 for missing file, got %d", exitCode)
	}
	if !strings.Contains(stderr, "error") {
		t.Errorf("expected error message in stderr, got: %q", stderr)
	}
}

// TestCLI_VerifyBadNode confirms that an unknown node ID in highlight_nodes
// produces exit code 1 (semantic error).
func TestCLI_VerifyBadNode(t *testing.T) {
	yaml := `title: "Test"
mermaid_diagram: |
  graph TD
    A --> B
steps:
  - label: "Overview"
  - highlight_nodes: [BOGUS]
`
	f := writeTempYAML(t, yaml)
	_, _, exitCode := run("verify", f)
	if exitCode != 1 {
		t.Fatalf("expected exit 1 for unknown node, got %d", exitCode)
	}
}

// TestCLI_SingleDiagramExampleVerifies confirms that the built-in single-diagram
// example YAML is self-consistent: it must parse and verify without errors.
func TestCLI_SingleDiagramExampleVerifies(t *testing.T) {
	stdout, _, exitCode := run("single-diagram-example")
	if exitCode != 0 {
		t.Fatalf("single-diagram-example: exit %d", exitCode)
	}
	f := writeTempYAML(t, stdout)
	verifyOut, _, exitCode := run("verify", f)
	if exitCode != 0 {
		t.Fatalf("verify single-diagram-example: exit %d\noutput: %s", exitCode, verifyOut)
	}
	if !strings.Contains(verifyOut, "✓") {
		t.Errorf("expected ✓ in verify output, got: %q", verifyOut)
	}
}

// TestCLI_MultipleDiagramExampleVerifies confirms that the built-in multi-section
// example YAML is self-consistent: it must parse and verify without errors.
func TestCLI_MultipleDiagramExampleVerifies(t *testing.T) {
	stdout, _, exitCode := run("multiple-diagram-example")
	if exitCode != 0 {
		t.Fatalf("multiple-diagram-example: exit %d", exitCode)
	}
	f := writeTempYAML(t, stdout)
	verifyOut, _, exitCode := run("verify", f)
	if exitCode != 0 {
		t.Fatalf("verify multiple-diagram-example: exit %d\noutput: %s", exitCode, verifyOut)
	}
	if !strings.Contains(verifyOut, "✓") {
		t.Errorf("expected ✓ in verify output, got: %q", verifyOut)
	}
}

// TestCLI_GenerateHTML confirms that ariel generate produces a structurally
// correct HTML file. Visual correctness (layout, highlighting, animation)
// requires human review — see package-level comment.
func TestCLI_GenerateHTML(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.html")
	stdout, _, exitCode := run("generate", "--output", outPath, "testdata/auth-flow.ariel.yaml")
	if exitCode != 0 {
		t.Fatalf("generate: exit %d\noutput: %s", exitCode, stdout)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	html := string(data)

	for _, want := range []string{
		"<html",
		"mermaid.min.js",
		"User Authentication Flow", // title from testdata file
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated HTML missing %q", want)
		}
	}
	// Watch mode injects a WebSocket client; generate must not.
	if strings.Contains(html, "ws://localhost") {
		t.Error("generated HTML must not contain WebSocket code")
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.ariel.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
