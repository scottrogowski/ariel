// Package skillsync copies the `ariel guide` reference into the create-walkthrough
// SKILL.md so an agent reads it at skill-load time without shelling out to
// `ariel guide`.
package skillsync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottrogowski/ariel/internal/guide"
)

const (
	beginMarker  = "<!-- BEGIN GENERATED: ariel guide — regenerate with `make sync-skill`; do not edit by hand -->"
	endMarker    = "<!-- END GENERATED: ariel guide -->"
	skillRelPath = "skills/create-walkthrough/SKILL.md"
)

// renderBlock is the canonical generated region: markers wrapping the guide text
// in a fenced code block so its ASCII layout survives Markdown rendering. Both
// Sync and Check derive from this single definition.
func renderBlock() string {
	return beginMarker + "\n\n```\n" + strings.TrimRight(guide.Guide, "\n") + "\n```\n\n" + endMarker
}

// skillPath returns the absolute path to the managed SKILL.md.
func skillPath() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, skillRelPath), nil
}

// Sync overwrites the generated region of SKILL.md with the current guide text
// and returns the path it wrote.
func Sync() (string, error) {
	path, err := skillPath()
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	updated, err := replaceBlock(string(content))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Check reports an error if SKILL.md's generated region does not match the
// current guide text. The error names the fix so a failing `make test` is
// self-explanatory.
func Check() error {
	path, err := skillPath()
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	got, err := extractBlock(string(content))
	if err != nil {
		return err
	}
	if got != renderBlock() {
		return fmt.Errorf("%s is out of sync with `ariel guide`; run `make sync-skill`", skillRelPath)
	}
	return nil
}

// extractBlock returns the marker-delimited region (inclusive) from content.
func extractBlock(content string) (string, error) {
	start := strings.Index(content, beginMarker)
	if start < 0 {
		return "", fmt.Errorf("begin marker not found in SKILL.md: %q", beginMarker)
	}
	end := strings.Index(content, endMarker)
	if end < 0 {
		return "", fmt.Errorf("end marker not found in SKILL.md: %q", endMarker)
	}
	if end < start {
		return "", fmt.Errorf("end marker precedes begin marker in SKILL.md")
	}
	return content[start : end+len(endMarker)], nil
}

// replaceBlock returns content with its marker-delimited region replaced by the
// freshly rendered block.
func replaceBlock(content string) (string, error) {
	existing, err := extractBlock(content)
	if err != nil {
		return "", err
	}
	return strings.Replace(content, existing, renderBlock(), 1), nil
}

// repoRoot walks up from the working directory to the module root (the directory
// holding go.mod) so callers resolve SKILL.md identically whether invoked from
// the repo root (`go run`) or a package directory (`go test`).
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}
