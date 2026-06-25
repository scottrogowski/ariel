package renderer

import (
	"github.com/scottmrogowski/ariel/internal/dsl"
	internlmp4 "github.com/scottmrogowski/ariel/internal/mp4"
)

const DefaultStepDuration = internlmp4.DefaultStepDuration

// GenerateMP4 delegates MP4 generation to the internal mp4 package.
func GenerateMP4(w *dsl.Walkthrough, outPath string, stepDuration int) error {
	return internlmp4.Generate(w, outPath, stepDuration)
}
