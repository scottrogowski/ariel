package guide

import _ "embed"

//go:embed guide.txt
var Guide string

//go:embed single-diagram-example.ariel.yaml
var SingleDiagramExample string

//go:embed multiple-diagram-example.ariel.yaml
var MultipleDiagramExample string
