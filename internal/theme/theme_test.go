package theme

import (
	"strings"
	"testing"
)

// Catches a wrong channel order or malformed alpha in the hex→rgba conversion
// that feeds every derived glow/overlay color.
func TestRGBADerivation(t *testing.T) {
	got := rgba("#5b8dee", "0.3")
	want := "rgba(91, 141, 238, 0.3)"
	if got != want {
		t.Fatalf("rgba(#5b8dee, 0.3) = %q, want %q", got, want)
	}
}

// Catches a Palette field silently dropping out of the :root block (which would
// leave a var(--x) reference unresolved and render an element with no color).
func TestRootBlockCoversEveryVar(t *testing.T) {
	block := Dark.RootBlock()
	for _, kv := range Dark.cssVars() {
		if !strings.Contains(block, kv[0]+": "+kv[1]) {
			t.Errorf("RootBlock missing %s: %s", kv[0], kv[1])
		}
	}
}
