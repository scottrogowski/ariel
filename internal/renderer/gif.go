package renderer

import (
	"github.com/scottmrogowski/ariel/internal/dsl"
	internlmp4 "github.com/scottmrogowski/ariel/internal/mp4"
)

// GenerateGIF delegates GIF generation to the internal mp4 package.
func GenerateGIF(w *dsl.Walkthrough, outPath string, stepDuration int) error {
	return internlmp4.GenerateGIF(w, outPath, stepDuration)
}
