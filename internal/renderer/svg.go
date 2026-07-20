package renderer

import (
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/svgformat"
	"github.com/scottrogowski/ariel/internal/theme"
)

// GenerateSVG delegates SVG generation to the internal svgformat package.
func GenerateSVG(w *dsl.Walkthrough, outPath string, mode theme.Mode) error {
	return svgformat.Generate(w, outPath, mode)
}
