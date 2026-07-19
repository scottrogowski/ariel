package cmd

const RootShort = "Step-by-step Mermaid diagram walkthroughs from a YAML DSL"
const RootLong = `ariel generates annotated walkthroughs from a YAML file paired with a Mermaid diagram.
Each walkthrough defines a sequence of steps that highlight nodes, animate edges,
and display narration text — rendered as self-contained HTML (interactive, best experience),
SVG (for embedding in GitHub READMEs and PR summaries), or MP4 (non-interactive video).
The 'watch' command is useful as a way for your agent to visually explain things to you in real-time.
`

const guideShort = "Print the DSL reference and authoring tips (Agents: run this first)"

const singleDiagramExampleShort = "Print a complete single-diagram walkthrough example"

const multipleDiagramExampleShort = "Print a complete multi-section walkthrough example"

const verifyShort = "Lint a walkthrough file for syntax and semantic errors"

const generateShort = "Render a walkthrough file to HTML, SVG, or MP4"
const generateLong = `Render a walkthrough file to HTML, SVG, or MP4.

Output formats:
  html  Highly interactive diagram. Best experience.
  svg   Interactive image. Embeddable in READMEs and PR summaries.
  mp4   Non-interactive video.`
const generateFlagOutputHelp = "output path (default: input path with format extension)"
const generateFlagFormatHelp = "output format: html, svg, or mp4 (default: html)"
const generateFlagStepDurationHelp = "seconds per step (mp4 only)"

const watchShort = "Serve a live-reloading browser preview of a walkthrough file"
const watchFlagPortHelp = "port to bind"

const versionShort = "Print the installed ariel version"
