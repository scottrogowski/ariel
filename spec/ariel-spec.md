# Ariel — Specification

**Version:** 0.4
**Purpose:** A CLI tool that converts a YAML walkthrough file into an animated, narrated diagram presentation. Designed to be authored by an LLM (Claude Code), with live browser preview, HTML export (interactive, best experience), SVG export (embeddable in GitHub PRs and READMEs), and MP4 export (non-interactive video).

---

## About This Spec

This document is the source of truth for ariel's DSL, CLI contracts, and frontend behavior. It is a **living document** — every code change that affects user-visible behavior, the DSL schema, CLI flags, exit codes, or output format must be accompanied by a corresponding update to this spec. A spec that diverges from the code is worse than no spec. When in doubt: update the spec first, then write the code.

Spec entries should be true regardless of current implementation state. If a sentence would need editing the next time a bug is fixed or a limitation is lifted, it belongs in a commit message or comment, not here.

---

## Problem Statement

Code is being generated faster than engineers can understand it. Specs are written, PRs are opened, and reviewers approve without truly comprehending what the system does. Static diagrams help but are insufficient — they show structure without conveying flow, decision points, or what is non-obvious.

Ariel addresses this by turning a system description into a guided, animated walkthrough. An LLM reads the spec, identifies what is important (decision points, non-obvious design choices, failure paths), and authors a YAML walkthrough file. Ariel renders that file as a step-by-step animated presentation — in the browser, as an interactive SVG for embedding directly in GitHub PRs and READMEs, or as an MP4 for video sharing.

---

## Design Principles

- **LLM-first authoring.** The YAML DSL is written by an LLM. Syntax is explicit and unambiguous. The `guide` subcommand loads the full DSL reference into LLM context.
- **Strong guardrails for agentic use.** `verify` is a full linter — syntax, semantic, and Mermaid validity — because agentic loops need fast, reliable feedback.
- **Single-threaded human attention.** Each step presents one idea: one narration sentence, one visual change. Animation and narration never compete.
- **Single file artifacts.** Each output format is a single self-contained file. None require a server.
- **Simplicity over features.** Build only what is needed.

---

## Language and Distribution

- **Implementation language:** Go
- **Output:** Single static binary, cross-compiled for macOS (arm64, amd64), Linux (amd64), Windows (amd64)
- **Build tooling:** GoReleaser + GitHub Actions
- **Distribution:** GitHub Releases (pre-built binaries), `go install github.com/scottrogowski/ariel@latest`
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
- Valid Mermaid syntax (via embedded Mermaid parser)

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

**HTML output:** Highly interactive diagram. Best experience. Single self-contained file, no server required.

**SVG output:** Interactive image. Embeddable in READMEs and PR summaries. Renders statically when embedded as `<img>`; clicking opens GitHub's SVG viewer where full interactivity is available. Supports both single- and multi-section walkthroughs; sections are flattened into a single step sequence.

**MP4 output:** Non-interactive video. Each step is held for `--step-duration` seconds. Requires `ffmpeg` on PATH.

**Exit codes:** `0` success, `1` verify failed or render error, `2` file not found, `3` output path not writable.

**Parity goal:** HTML and SVG outputs should look and behave identically. They should differ only where the SVG format imposes hard constraints.

---

### `ariel watch <file.ariel.yaml> [--port <n>]`

Start a local HTTP server that serves the walkthrough and live-reloads when the file changes.

**Flags:** `--port <n>` (default: `2313`)

**Behavior:**
1. Runs `verify` on startup. Prints errors but does not refuse to start.
2. Opens the browser automatically at `http://localhost:<port>`.
3. On file change: re-parses the YAML and pushes the updated walkthrough to the browser.
4. Browser replaces the page with the new content, or shows an error overlay on parse failure.

The watch output is identical to the HTML generate output.

**Exit codes:** `0` clean shutdown (Ctrl+C), `1` port in use, `2` file not found.

---

## DSL — The Walkthrough File Format

The authoritative DSL guide is `internal/guide/guide.txt`. Run `ariel guide` to print it. What follows covers only the structural constraints needed to understand the rest of this spec.

Files use the `.ariel.yaml` extension by convention (not enforced). Two top-level formats are supported and cannot be combined: single-diagram (`mermaid_diagram` + `steps`) and multi-diagram (`sections`). See `internal/guide/guide.txt` for full field definitions, node ID rules, edge format, authoring tips, and common errors.

Clicking Next at the last step of a section advances to the first step of the next section and re-renders the diagram.

**The first step of each section is the overview.** It may only use `label` and `narration`. Using `highlight_nodes` or `focus_nodes` on step 1 is an error.

Complete examples: `internal/guide/single-diagram-example.ariel.yaml` and `internal/guide/multiple-diagram-example.ariel.yaml` (also printed by `ariel single-diagram-example` / `ariel multiple-diagram-example`).

---

## Frontend — Rendering

### Mermaid

Mermaid is used as the diagram renderer. A single pinned version is used across all output formats to ensure consistent rendering. Theme: dark.

### Node identification

Nodes are identified by their DSL node ID after rendering. For flowchart diagrams, IDs are read directly from the rendered SVG. For sequence diagrams and other types, nodes are matched by their display label using the `node_labels` mapping derived from the DSL. A generic text-content fallback handles any remaining types.

### Node highlighting

When a step has any `highlight_nodes` or `focus_nodes`, all unreferenced nodes are dimmed. Referenced nodes are visually emphasized:

- **highlighted** — blue tint; applied to `highlight_nodes`
- **active** — stronger emphasis with teal border; applied to `focus_nodes`; takes precedence if a node appears in both lists

### Edge animation

Edges between any two nodes in the combined set of `highlight_nodes` and `focus_nodes` are animated automatically — no manual specification.

### Click-for-walkthrough CTA

When a walkthrough has more than one step, the initial view shows a "Click for walkthrough" overlay covering the full output. Clicking the overlay advances to step 1 (the section overview) and the overlay disappears permanently. The CTA is a **one-way entry point**: it is not reachable via Back, step dots, or section dots. Back navigation starts from step 2 (the first step that has a predecessor). Section and step dot navigation targets the first real step (step 1), never the CTA state.

### Step player

- Previous / Next buttons; keyboard navigation (HTML only)
- Progress dots (one per step); section dots above if multiple sections
- Narration updates on step change
- Step 1 is the overview: no step counter
- Numbered steps: "N of M — label" where M excludes the overview step
- Final step: Next becomes "Done" (disabled)

### Click-to-navigate

Nodes appearing in any step's `highlight_nodes` or `focus_nodes` are navigable (HTML only). Clicking advances to the next step referencing that node (cycling). Scoped to the current section.

### Layout

Two-pane: diagram left, narration + controls right. Dark theme.

Header: walkthrough title centered; logo top-right.

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
- Clickable links within MP4 output
