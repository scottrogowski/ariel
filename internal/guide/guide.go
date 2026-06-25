package guide

import _ "embed"

//go:embed reference.txt
var Reference string

//go:embed single-diagram-example.ariel.yaml
var SingleDiagramExample string

//go:embed multiple-diagram-example.ariel.yaml
var MultipleDiagramExample string
