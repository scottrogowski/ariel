# Ariel — Specification

**Version:** 0.4
**Purpose:** A CLI tool that converts a YAML walkthrough file into an animated, narrated diagram presentation. Designed to be authored by an LLM (Claude Code), with live browser preview, HTML export (interactive, best experience), SVG export (embeddable in GitHub PRs and READMEs), and MP4 export (non-interactive video).

---

## About This Spec

This document is the source of truth for ariel's DSL, CLI contracts, and frontend behavior. It is a **living document** — every code change that affects user-visible behavior, the DSL schema, CLI flags, exit codes, or output format must be accompanied by a corresponding update to this spec. A spec that diverges from the code is worse than no spec. When in doubt: update the spec first, then write the code.

---

## Problem Statement

Code is being generated faster than engineers can understand it. Specs are written, PRs are opened, and reviewers approve without truly comprehending what the system does. Static diagrams help but are insufficient — they show structure without conveying flow, decision points, or what is non-obvious.

Ariel addresses this by turning a system description into a guided, animated walkthrough. An LLM reads the spec, identifies what is important (decision points, non-obvious design choices, failure paths), and authors a YAML walkthrough file. Ariel renders that file as a step-by-step animated presentation — in the browser, as an interactive SVG for embedding directly in GitHub PRs and READMEs, or as an MP4 for video sharing.

---

## Design Principles

- **LLM-first authoring.** The YAML DSL is written by an LLM. Syntax is explicit and unambiguous. The `guide` subcommand loads the full DSL reference into LLM context.
- **Strong guardrails for agentic use.** `verify` is a full linter — syntax, semantic, and Mermaid validity — because agentic loops need fast, reliable feedback.
- **Single-threaded human attention.** Each step presents one idea: one narration sentence, one visual change. Animation and narration never compete.
- **Single file artifacts.** The HTML output is fully self-contained. The MP4 output is a standard H.264 video. The SVG output is a single XML file. None require a server.
- **Simplicity over features.** Build only what is needed.

---

## Language and Distribution

- **Implementation language:** Go
- **Output:** Single static binary, cross-compiled for macOS (arm64, amd64), Linux (amd64), Windows (amd64)
- **Build tooling:** GoReleaser + GitHub Actions
- **Distribution:** GitHub Releases (pre-built binaries), `go install github.com/scottmrogowski/ariel@latest`
- **Runtime dependencies:** None for the binary itself. `ffmpeg` must be on PATH when using `--format mp4`. Chromium (managed by chromedp) is used for `--format mp4` and `--format svg`.

---

## CLI Interface

### Top-level

`ariel --help` description:

```
ariel generates annotated walkthroughs from a YAML file paired with a Mermaid diagram.
Each walkthrough defines a sequence of steps that highlight nodes, animate edges,
and display narration text — rendered as self-contained HTML (interactive, best experience),
SVG (for embedding in GitHub READMEs and PR summaries), or MP4 (non-interactive video).
```

Subcommands:

| Command | Purpose |
|---|---|
| `guide` | Print the DSL reference and authoring tips |
| `single-diagram-example` | Print a complete single-diagram walkthrough YAML example |
| `multiple-diagram-example` | Print a complete multi-section walkthrough YAML example |
| `verify` | Lint a walkthrough file for syntax and semantic errors |
| `generate` | Render a walkthrough file to HTML, SVG, or MP4 |
| `watch` | Serve a live-reloading browser preview |

---

### `ariel guide`

Print the complete DSL reference to stdout. Designed to be called by an LLM at the start of a session.

**Output includes:** full YAML schema, field definitions, node ID rules, edge format, authoring tips, common errors.

**Flags:** None. **Exit codes:** 0 always.

---

### `ariel single-diagram-example` / `ariel multiple-diagram-example`

Print a complete, valid `.ariel.yaml` example to stdout. Pipe to a file to use as a starting point.

The examples are meta — they explain the cognitive science behind why walkthroughs aid comprehension (working memory limits, Dual Coding Theory, Progressive Disclosure) while also demonstrating most DSL features.

**Flags:** None. **Exit codes:** 0 always.

---

### `ariel verify <file.ariel.yaml>`

Lint a walkthrough file. Runs automatically as part of `generate` and `watch`.

**Checks performed:**

*Syntax:*
- Valid YAML structure
- Required top-level fields present (`mermaid_diagram`, `steps` — or `sections`)
- No unknown fields at any level (strict mode — unknown fields are errors)
- Valid Mermaid syntax (via embedded goja + Mermaid 10.6.1 parser)

*Semantic:*
- All node IDs in `highlight_nodes`, `focus_nodes` exist in the diagram
- At least one step per section
- The first step of each section may only use `label` and `narration` — `highlight_nodes` and `focus_nodes` on step 1 are errors (see DSL section)
- Steps with no content are warnings

*Warnings (non-blocking):*
- The nodes referenced by a step's `highlight_nodes` and `focus_nodes` do not form a single connected component when traversing direct diagram edges — often signals unrelated nodes grouped in one step

**Output format:**

On success:
```
✓ ariel.yaml is valid (8 steps, 12 nodes, 9 edges)
✓ ariel.yaml is valid (2 sections, 14 steps, 19 nodes, 20 edges)
```

On failure:
```
ariel.yaml:14: error: highlight_nodes references unknown node ID "TG2"
ariel.yaml:31: warning: step 6 has no narration and no visual changes
```

**Exit codes:** `0` valid (warnings OK), `1` one or more errors, `2` file not found.

---

### `ariel generate <file.ariel.yaml> [flags]`

Render a walkthrough to HTML, SVG, or MP4.

**Flags:**
- `--output <path>` — output path (default: input filename with format extension)
- `--format <html|svg|mp4>` — output format (default: `html`)
- `--step-duration <n>` — seconds each step is held in MP4 output (default: `2`, mp4 only)

**HTML output:** Highly interactive diagram. Best experience. A single `.html` file with all CSS and JS inlined, Mermaid loaded from pinned CDN, no server required. Openable by double-clicking in any modern browser.

**SVG output:** Interactive image. Embeddable in READMEs and PR summaries. An interactive `.svg` file for embedding in GitHub PRs and READMEs. Supports both single- and multi-section walkthroughs; sections are flattened into a single step sequence. See SVG Architecture below.

**MP4 output:** Non-interactive video. A standard H.264 `.mp4` file at 25fps (CFR). Requires `ffmpeg` on PATH.

**Exit codes:** `0` success, `1` verify failed or render error, `2` file not found, `3` output path not writable.

#### MP4 Architecture

MP4 generation uses headless Chrome (via chromedp) to screenshot each step, then assembles the frames with ffmpeg.

Per-section static HTML rendering:
1. For each section, ariel generates a minimal static HTML page containing the section's Mermaid diagram. CSS transitions and animations are disabled (`transition: none !important`). The page exposes a synchronous `applyStep(highlightNodes, focusNodes, label, narration)` function and signals readiness via a `#ready` element once Mermaid finishes rendering.
2. Chrome navigates to the section HTML (`file://` URL), then waits for `#ready` to be visible.
3. For each step in the section, ariel calls `applyStep()` via CDP and immediately captures a screenshot. No sleeps or polling — the state change is synchronous because transitions are disabled.
4. Frames are named `frame0000.png`, `frame0001.png`, … in a temporary directory.

Assembly:
- ffmpeg is invoked with `-framerate 1/<step-duration> -i frame%04d.png -r 25` to produce a CFR H.264 video.
- Encoding flags: `-crf 26 -preset slow -pix_fmt yuv420p -movflags +faststart`.
- The temporary directory is removed after assembly.

Diagram scaling in screenshots:
- The screenshot HTML uses `100vh`-based layout so it adapts to the actual Chrome viewport.
- `#mermaid-container` uses `align-self: stretch; height: 100%`, and the SVG uses `width: 100%; height: 100%`. Mermaid's default `preserveAspectRatio: xMidYMid meet` scales any diagram to fit the container without clipping, regardless of aspect ratio.

---

#### SVG Architecture

SVG generation uses headless Chrome (via chromedp) to render each step and extract self-contained SVG strings, then assembles them into a single interactive SVG file using `foreignObject` + CSS `:checked`.

**Interactivity model:**
- When embedded as `<img>` in GitHub markdown the SVG renders statically. Clicking opens GitHub's SVG file viewer where full interactivity is available. One click from the PR body gives the complete walkthrough.
- Navigation uses N radio inputs (one per step) as state — no JavaScript. CSS `#sN:checked ~` selectors drive visibility of diagram, CTA bar, nav buttons, and step dots.
- Hover effects work in the SVG file viewer, not in the `<img>` embed.

**Per-step extraction:**
1. Each step is rendered in a minimal headless page (`browserWidth: 920px`). Node highlighting and edge animation are applied as inline `style.setProperty(..., 'important')` calls so each extracted SVG is visually self-contained.
2. Node highlighting mirrors the HTML renderer: dimmed nodes at 40% opacity, highlighted nodes with blue tint, focus nodes with teal border.
3. Animated edges use SMIL `<animate attributeName="stroke-dashoffset">` elements appended to each edge path. CSS `@keyframes` are used in the HTML renderer but would be stripped by GitHub's SVG sanitizer, so SMIL is the only viable option here.
4. Bare `<br>` tags from Mermaid's HTML output inside `foreignObject` are replaced with `<br/>` so the output file is valid XML.

**Output SVG structure:**
```
<?xml ...?><svg width="1200" height="≤850">
  <foreignObject width="1200" height="≤850">
    <div xmlns="http://www.w3.org/1999/xhtml" style="display:flex;flex-direction:column">
      <style>  (navigation CSS — :checked rules for N steps)  </style>
      <input type="radio" id="s0" checked/> ... <input type="radio" id="sN"/>
      <div class="page-header">  (full-width walkthrough title)  </div>
      <div class="content">  (flex row)
        <div class="diagram-col">  (900px — pre-rendered Mermaid SVGs)  </div>
        <div class="narrations">  (300px — step headers and narration text)  </div>
      </div>
      <label class="cta-overlay" for="s1">  (shown only on step 0)  </label>
      <div class="bottom">  (nav controls — prev/dots/next)  </div>
    </div>
  </foreignObject>
</svg>
```

**Sizing:** Max 1200×850. `narrationWidth = 300px` (fixed). `diagramColumnWidth = 900px` (fixed). Diagrams are scaled up to `1.5×` their natural Mermaid width, capped at 810px (90% of column after 10% horizontal padding), then scaled down if the resulting height exceeds 686px (850 minus fixed chrome). The output SVG height is `164 + maxEffectiveH` (always ≤ 850).

**Multi-section support:** Sections are flattened into a single global step sequence. Each step is rendered from its section's Mermaid diagram. The section title is prepended to step labels in the narration panel header (e.g. "Section Title — step label") so the reader knows which section they are in. The diagram area is sized to the tallest diagram across all sections.

**Why HTML and SVG renderers are not abstracted into shared code:** The two renderers share layout concepts (diagram + narration + navigation) but not implementation. The HTML renderer drives everything with JavaScript — live DOM manipulation, CSS transitions, WebSocket hot-reload, click-to-navigate via event listeners. The SVG renderer uses CSS `:checked` + radio inputs with no JavaScript; visual state is baked into each pre-rendered step SVG as inline styles by chromedp. A shared abstraction would need to accommodate both paradigms, adding indirection without reducing meaningful duplication.

---

### `ariel watch <file.ariel.yaml> [--port <n>]`

Start a local HTTP server that serves the walkthrough and live-reloads when the file changes.

**Flags:** `--port <n>` (default: `2313`)

**Behavior:**
1. Runs `verify` on startup. Prints errors but does not refuse to start.
2. Opens the browser automatically at `http://localhost:<port>`.
3. Watches the file with filesystem events (not polling). On change: re-parses the YAML and broadcasts updated HTML over WebSocket.
4. Browser replaces the page with the new HTML on `update` message, or shows an error overlay on `error` message.

The watch HTML is identical to the generate HTML except it includes a ~20-line WebSocket client snippet that connects to `ws://localhost:<port>/ws`.

**Exit codes:** `0` clean shutdown (Ctrl+C), `1` port in use, `2` file not found.

---

## DSL — The Walkthrough File Format

The authoritative DSL guide is `internal/guide/guide.txt`. Run `ariel guide` to print it. What follows covers only the structural constraints needed to understand the rest of this spec.

Files use the `.ariel.yaml` extension by convention (not enforced). Two top-level formats are supported and cannot be combined: single-diagram (`mermaid_diagram` + `steps`) and multi-diagram (`sections`). See `internal/guide/guide.txt` for full field definitions, node ID rules, edge format, authoring tips, and common errors.

Clicking Next at the last step of a section advances to the first step of the next section and re-renders the diagram.

**The first step of each section is the overview.** It may only use `label` and `narration`. Using `highlight_nodes` or `focus_nodes` on step 1 is an error.

Complete examples: `internal/guide/single-diagram-example.ariel.yaml` and `internal/guide/multiple-diagram-example.ariel.yaml` (also printed by `ariel single-diagram-example` / `ariel multiple-diagram-example`).

---

## Frontend — HTML Rendering

### Mermaid

- Version: `10.6.1` (pinned), loaded from `https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js`
- Theme: `dark`
- Renders as inline SVG

### Node identification

After rendering, the frontend builds a `nodeMap` using three strategies in priority order:

1. **Flowchart:** `.node` groups with ID `flowchart-{nodeId}-{n}` — Mermaid 10.6.1 standard pattern.
2. **Sequence diagram:** `g.actor` groups matched by normalized text content against inverted `node_labels`.
3. **Generic fallback:** any `<g>` whose normalized text content matches an unmapped `node_labels` entry.

`node_labels` is derived from `dsl.ExtractGraph` (static DSL parse) and maps node ID → display label. The map is inverted at runtime (display label → node ID) to enable text-based matching in strategies 2 and 3.

### Node highlighting

When a step has any `highlight_nodes` or `focus_nodes`, all unreferenced nodes are dimmed to 40% opacity. Referenced nodes are restored to full opacity with visual emphasis:

- `.highlighted` — distinct fill and border color (blue tint); applied to `highlight_nodes` not also in `focus_nodes`
- `.active` — stronger emphasis (teal border, glow); applied to all `focus_nodes`; takes precedence if a node appears in both lists

CSS transitions (`0.35s ease`) handle state changes in the interactive HTML. In MP4 screenshots, all transitions are disabled.

### Edge animation

Edges between any two nodes in the combined set of `highlight_nodes` and `focus_nodes` are animated automatically — no manual specification. The frontend checks `edgeMap[src + '-' + dst]` for every ordered pair and adds `.animated` to matching SVG paths.

```css
@keyframes flowEdge { from { stroke-dashoffset: 24; } to { stroke-dashoffset: 0; } }
.flowchart-link.animated { stroke-dasharray: 8 4; animation: flowEdge 0.8s linear infinite; }
```

In MP4 screenshots, animations are disabled — animated edges appear as static dashed lines.

### Step player

- Previous / Next buttons; keyboard: ArrowRight/Space = next, ArrowLeft = previous
- Progress dots (one per step, current step pill-shaped); section dots above if multiple sections
- Narration fades out/in on step change (0.2s)
- Step 1 is the overview: no step counter; label shows section title (multi-diagram) or step label
- Numbered steps: "2 of N — label" where N excludes the overview step
- Final step of final section: Next becomes "Done" (disabled)

### Click-to-navigate

Nodes appearing in any step's `highlight_nodes` or `focus_nodes` are navigable. Clicking advances to the next step referencing that node (cycling). Scoped to the current section.

### Layout

Two-pane: diagram left, narration + controls right. Responsive (stacks vertically on narrow screens). Dark theme. All CSS and JS inlined in the output HTML.

Header: walkthrough title centered; "Ariel ↗" link top-right.

---

## Repository Structure

```
ariel/
├── main.go
├── internal/
│   ├── cmd/
│   │   ├── root.go
│   │   ├── guide.go          # guide, single-diagram-example, multiple-diagram-example
│   │   ├── verify.go
│   │   ├── generate.go
│   │   └── watch.go
│   ├── dsl/
│   │   ├── parse.go          # YAML parsing and validation
│   │   ├── verify.go         # semantic verification
│   │   ├── mermaid.go        # node/edge extraction
│   │   └── schema.go         # DSL type definitions
│   ├── renderer/
│   │   ├── generate.go       # HTML generation
│   │   ├── mp4.go            # MP4 shim (delegates to internal/mp4)
│   │   ├── svg.go            # SVG shim (delegates to internal/svgformat)
│   │   ├── watch.go          # HTTP server + WebSocket
│   │   └── template.go       # HTML/CSS/JS template
│   ├── mp4/
│   │   ├── mp4.go            # chromedp capture + ffmpeg assembly
│   │   └── template.go       # per-section static screenshot HTML
│   ├── svgformat/
│   │   ├── generate.go       # chromedp extraction + output SVG assembly
│   │   └── template.go       # extraction HTML + nav CSS generation
│   ├── mermaidjs/
│   │   └── ...               # embedded Mermaid 10.6.1 + goja validator
│   └── guide/
│       ├── guide.go          # DSL reference text
│       └── examples.go       # single/multiple diagram example YAML
├── testdata/
│   └── auth-flow.ariel.yaml
├── spec/
│   └── ariel-spec.md
└── .goreleaser.yaml
```

---

## Out of Scope

The following are explicitly deferred:

- Audio narration / text-to-speech
- Socratic interrogation mode (quiz the viewer on system comprehension)
- Sub-diagrams or drill-down within a section
- Collaboration features
- Persistence or cloud storage
- Authentication
- Branching walkthroughs (conditional paths)
- Custom node types or icons beyond standard Mermaid shapes
- Replacing Mermaid with a custom renderer
- Clickable links within MP4 output (not supported by the MP4 container format or GitHub's `<video>` embed)
