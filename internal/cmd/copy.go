package cmd

const RootShort = "Step-by-step Mermaid diagram walkthroughs from a YAML DSL"
const RootLong = `ariel generates annotated walkthroughs from a YAML file paired with a Mermaid diagram.
Each walkthrough defines a sequence of steps that highlight nodes, animate edges,
and display narration text — rendered as self-contained HTML (interactive) or MP4 (for embedding in GitHub READMEs and PR summaries).`

const guideShort = "Print the DSL reference and authoring tips (Agents: run this first)"

const singleDiagramExampleShort = "Print a complete single-diagram walkthrough example"

const multipleDiagramExampleShort = "Print a complete multi-section walkthrough example"

const verifyShort = "Lint a walkthrough file for syntax and semantic errors"

const generateShort = "Render a walkthrough file to HTML or MP4"
const generateFlagOutputHelp = "output path (default: input path with format extension)"
const generateFlagFormatHelp = "output format: html or mp4"
const generateFlagStepDurationHelp = "seconds per step (mp4 only)"

const watchShort = "Serve a live-reloading browser preview of a walkthrough file"
const watchFlagPortHelp = "port to bind"
