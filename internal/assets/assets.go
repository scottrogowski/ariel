package assets

import (
	_ "embed"
	"encoding/base64"
)

//go:embed ariel-logo.png
var arielLogoPNG []byte

// ArielLogoDataURI is the base64-encoded PNG data URI for the ariel logo.
var ArielLogoDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(arielLogoPNG)
