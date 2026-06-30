package renderer

import (
	"github.com/scottmrogowski/ariel/internal/dsl"
	"github.com/scottmrogowski/ariel/internal/svgformat"
)

// GenerateSVG delegates SVG generation to the internal svgformat package.
func GenerateSVG(w *dsl.Walkthrough, outPath string) error {
	return svgformat.Generate(w, outPath)
}
