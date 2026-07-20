package renderer

import (
	"github.com/scottrogowski/ariel/internal/dsl"
	internlmp4 "github.com/scottrogowski/ariel/internal/mp4"
	"github.com/scottrogowski/ariel/internal/theme"
)

const DefaultStepDuration = internlmp4.DefaultStepDuration

// GenerateMP4 delegates MP4 generation to the internal mp4 package.
func GenerateMP4(w *dsl.Walkthrough, outPath string, stepDuration int, mode theme.Mode) error {
	return internlmp4.Generate(w, outPath, stepDuration, mode)
}
