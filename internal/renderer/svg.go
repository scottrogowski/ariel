package renderer

import (
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/svgformat"
)

// GenerateSVG delegates SVG generation to the internal svgformat package.
func GenerateSVG(w *dsl.Walkthrough, outPath string) error {
	return svgformat.Generate(w, outPath)
}
