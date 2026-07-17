// Package logo embeds ariel's SVG mark so the HTML and SVG output templates
// can inline the same markup instead of each hand-drawing their own copy.
package logo

import (
	"encoding/base64"
	_ "embed"
	"strings"
)

//go:embed logo.svg
var SVG string

// faviconViewBox crops SVG down to just its first ("a") letter, replacing that letter's
// box outline with a solid app-background fill (transparent reads as too faint at
// favicon size) — logo.svg's outer <svg> tag, first box, and that box's coordinates are
// assumed stable; update all three if the layout changes.
const (
	fullSVGTag     = `viewBox="0 0 688 256" width="688" height="256"`
	faviconViewBox = `viewBox="8 80 96 96" width="96" height="96"`
	firstBoxRect   = `<rect x="8" y="80" width="96" height="96" rx="6" fill="none" stroke="#5b8dee" stroke-width="6"/>`
	faviconBgRect  = `<rect x="8" y="80" width="96" height="96" fill="#0f1117"/>`
)

// FaviconBase64 returns SVG cropped to its first letter on a solid app-background
// square (no box outline) and base64-encoded, for use as a data: URI favicon — derived
// from the same source rather than a second asset.
func FaviconBase64() string {
	cropped := strings.Replace(SVG, fullSVGTag, faviconViewBox, 1)
	cropped = strings.Replace(cropped, firstBoxRect, faviconBgRect, 1)
	return base64.StdEncoding.EncodeToString([]byte(cropped))
}
