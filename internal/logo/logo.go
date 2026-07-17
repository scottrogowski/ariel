// Package logo embeds ariel's SVG mark so the HTML and SVG output templates
// can inline the same markup instead of each hand-drawing their own copy.
package logo

import _ "embed"

//go:embed logo.svg
var SVG string
