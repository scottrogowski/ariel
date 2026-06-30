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
	"encoding/xml"
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

// TestCLI_GenerateHTML_MarkdownLinks confirms that markdown-style links in narration
// are rendered as <a> tags in the generated HTML and not emitted as raw syntax.
func TestCLI_GenerateHTML_MarkdownLinks(t *testing.T) {
	yaml := `title: "Link Test"
mermaid_diagram: |
  graph TD
    A --> B
steps:
  - label: "Overview"
    narration: "Plain text."
  - label: "Step"
    highlight_nodes: [A, B]
    narration: "[See the paper](https://example.com/paper) for details."
`
	f := writeTempYAML(t, yaml)
	outPath := filepath.Join(t.TempDir(), "out.html")
	_, _, exitCode := run("generate", "--output", outPath, f)
	if exitCode != 0 {
		t.Fatalf("generate: exit %d", exitCode)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	html := string(data)

	// Inside the JSON blob, attribute quotes are backslash-escaped.
	if !strings.Contains(html, `href=\"https://example.com/paper\"`) {
		t.Error("generated HTML missing expected href")
	}
	if !strings.Contains(html, `>See the paper<`) {
		t.Error("generated HTML missing expected link text")
	}
	if strings.Contains(html, "[See the paper]") {
		t.Error("generated HTML contains raw markdown link syntax")
	}
}

// TestCLI_GenerateSVG confirms that ariel generate --format svg produces a
// structurally valid SVG file. Visual correctness requires human review.
func TestCLI_GenerateSVG(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.svg")
	stdout, stderr, exitCode := run("generate", "--format", "svg", "--output", outPath, "testdata/auth-flow.ariel.yaml")
	if exitCode != 0 {
		t.Fatalf("generate svg: exit %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	svg := string(data)

	for _, want := range []string{
		`<?xml`,
		`<svg`,
		`<foreignObject`,
		`type="radio"`,
		`class="cta-overlay"`,
		`class="col-title"`,
		`class="diagrams"`,
		`class="narrations"`,
		`class="nav-controls"`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("generated SVG missing %q", want)
		}
	}

	if strings.Count(svg, `<input type="radio"`) < 2 {
		t.Error("expected at least 2 radio inputs (one per step)")
	}

	// The SVG must be valid XML. Mermaid generates HTML void elements (e.g. <br>)
	// inside foreignObject; if they aren't made self-closing the file is broken.
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("generated SVG is not valid XML: %v", err)
	}
}

// TestCLI_GenerateSVG_MultiSection confirms that multi-section walkthroughs
// produce a valid SVG with steps from all sections.
func TestCLI_GenerateSVG_MultiSection(t *testing.T) {
	yaml := `title: "Multi"
sections:
  - title: "Section A"
    mermaid_diagram: |
      graph TD
        A --> B
    steps:
      - label: "Overview"
      - label: "Step 1"
        highlight_nodes: [A]
  - title: "Section B"
    mermaid_diagram: |
      graph TD
        C --> D
    steps:
      - label: "Overview"
      - label: "Step 1"
        highlight_nodes: [C]
`
	f := writeTempYAML(t, yaml)
	outPath := filepath.Join(t.TempDir(), "out.svg")
	stdout, stderr, exitCode := run("generate", "--format", "svg", "--output", outPath, f)
	if exitCode != 0 {
		t.Fatalf("generate svg multi-section: exit %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	svg := string(data)

	// 4 steps = 4 radio inputs (count opening tags, not CSS selectors)
	if strings.Count(svg, `<input type="radio"`) != 4 {
		t.Errorf("expected 4 radio inputs for 4 steps, got %d", strings.Count(svg, `<input type="radio"`))
	}
	// Section title should appear in the narration panel headers
	if !strings.Contains(svg, "Section A") {
		t.Error("expected Section A title in output")
	}
	if !strings.Contains(svg, "Section B") {
		t.Error("expected Section B title in output")
	}
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("generated SVG is not valid XML: %v", err)
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
