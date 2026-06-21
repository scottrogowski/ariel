# Ariel — Specification

**Version:** 0.1 (pre-implementation)
**Working name:** ariel (final name TBD)
**Purpose:** A CLI tool that converts a YAML walkthrough file into an animated, narrated diagram presentation. Designed to be driven by an LLM (Claude Code) operating on the YAML file, with a live browser preview that updates as the file changes.

---

## About This Spec

This document is the source of truth for ariel's DSL, CLI contracts, and frontend behavior. It is a **living document** — every code change that affects user-visible behavior, the DSL schema, CLI flags, exit codes, or output format must be accompanied by a corresponding update to this spec. A spec that diverges from the code is worse than no spec. When in doubt: update the spec first, then write the code.

---

## Problem Statement

Code is being generated faster than engineers can understand it. Specs are written, PRs are opened, and reviewers approve without truly comprehending what the system does. Static diagrams help but are insufficient — they show structure without conveying flow, decision points, or what is non-obvious.

Ariel addresses this by turning a spec or system description into a guided, animated walkthrough. An LLM reads the spec, identifies what is important (decision points, non-obvious design choices, failure paths), and authors a YAML walkthrough file. Ariel renders that file as a step-by-step animated presentation in the browser.

---

## Design Principles

- **Single file artifacts.** The output of `generate` is a fully self-contained HTML file. No external dependencies at runtime except a pinned CDN import of Mermaid.
- **LLM-first authoring.** The YAML DSL is designed to be written by an LLM, not a human. Syntax is explicit and unambiguous. The `guide` subcommand exists specifically to load the DSL spec into LLM context.
- **Strong guardrails for agentic use.** `verify` is a full linter — both syntax and semantic errors — because agentic coding loops need fast, reliable feedback.
- **Single-threaded human attention.** The walkthrough presents one idea at a time: one narration sentence, one visual change. Animation and narration never compete.
- **Simplicity over features.** Build only what is needed. No auth, no persistence, no accounts.

---

## Language and Distribution

- **Implementation language:** Go
- **Output:** Single static binary, cross-compiled for macOS (arm64, amd64), Linux (amd64), Windows (amd64)
- **Build tooling:** GoReleaser + GitHub Actions for automated cross-platform releases
- **Distribution:**
  - GitHub Releases (pre-built binaries, curl-downloadable)
  - `go install github.com/[owner]/ariel@latest` (requires Go toolchain)
  - Homebrew tap (future, once adoption warrants it)
- **No runtime dependencies** for the binary itself

---

## CLI Interface

### Top-level help

`ariel --help` prints a terse summary of subcommands. It must not dump the full DSL — that lives in `guide`.

```
ariel — animated diagram walkthroughs

Usage:
  ariel <command> [arguments]

Commands:
  guide        Print a brief DSL reference and authoring tips (Agents: run this first)
  verify       Lint a walkthrough file for syntax and semantic errors
  generate     Render a walkthrough file to a self-contained HTML file
  watch        Serve a live-reloading browser preview of a walkthrough file

Run 'ariel <command> --help' for command-specific usage.
```

---

### `ariel guide`

**Purpose:** Print the complete DSL reference to stdout. Designed to be called by an LLM at the start of a session to load the spec into context. Also useful for humans who want the full reference without reading a README.

**Output includes:**
1. Full YAML schema with field definitions and types
2. Example snippets
3. Authoring tips for LLMs (how to identify what is important, how to sequence steps)
4. Common errors and how to avoid them
5. Node ID rules (how Mermaid node IDs map to DSL references)

**Flags:** None

**Exit codes:** 0 always

**Implementation Note**
Respect the context window of the calling LLM. Be as brief as possible

---

### `ariel verify <file>`

**Purpose:** Lint a `.ariel.yaml` file. Designed to be called in an agentic loop after every LLM edit. Fast feedback is critical — it should complete in milliseconds.

**Checks performed:**

*Syntax checks:*
- Valid YAML structure
- Required top-level fields present (`mermaid_diagram`, `steps`)
- Each step is a valid object
- No unknown fields at any level (strict mode — unknown fields are errors, not warnings)
- Mermaid syntax is valid

*Semantic checks:*
- All node IDs referenced in `highlight_nodes` or `active_nodes` exist in the `mermaid_diagram` block
- All edge references in `animate_edges` reference valid node ID pairs that exist as edges in the diagram
- Mermaid diagram block is itself valid Mermaid syntax (parse and validate)
- At least one step exists
- Steps with no narration, no label, AND no visual changes are flagged as warnings (not errors)
- Steps where `highlight_nodes` or `active_nodes` contain two nodes with no direct edge between them are flagged as warnings — this often indicates nodes that belong in separate steps

**Output format:**

On success (single diagram):
```
✓ ariel.yaml is valid (8 steps, 12 nodes, 9 edges)
```

On success (multiple sections):
```
✓ ariel.yaml is valid (2 sections, 14 steps, 19 nodes, 20 edges)
```

On failure (one line per issue, file:line format for editor compatibility):
```
ariel.yaml:14: error: highlight_nodes references unknown node ID "TG2"
ariel.yaml:22: error: animate_edges references edge "A-X" which does not exist in mermaid_diagram
ariel.yaml:31: warning: step 6 has no narration and no visual changes
```

**Exit codes:**
- `0` — valid (warnings allowed)
- `1` — one or more errors
- `2` — file not found or unreadable

**Implementation Notes — Mermaid Validation:**
verify must validate that the mermaid_diagram block is valid Mermaid syntax. Since Mermaid is a JavaScript library, this requires running JS from within the Go binary.
Approach: embed Mermaid in goja
goja is a pure Go JavaScript runtime. Bundle a pre-built Mermaid JS parse/validate script inside the binary and execute it via goja when verify runs.
The bundled script should expose a single function: take a Mermaid diagram string, attempt to parse it, return success or a structured error with line/message. Only the parser is needed — no rendering, no SVG output.
Tradeoffs:

No external runtime dependencies (Node not required)
Mermaid JS is ~2MB — adds to binary size and cold parse time
goja startup + Mermaid parse adds latency to verify — target under 1 second, acceptable for an agentic loop
Mermaid JS version in the validator should be pinned to match the CDN version used in the renderer (10.6.1)

---

### `ariel generate <file> [--output <path>]`

**Purpose:** Render a `.ariel.yaml` file to a self-contained HTML file.

**Flags:**
- `--output <path>` — output path (default: same directory as input, same filename with `.html` extension)

**Output:** A single `.html` file containing:
- All CSS inlined
- All JavaScript inlined
- The YAML step script baked in as a JS object (not loaded at runtime)
- Mermaid loaded from pinned CDN (see Frontend section)
- No websocket code
- No external dependencies beyond the Mermaid CDN import

The output file must be openable by double-clicking in any modern browser with no server required.

**Exit codes:**
- `0` — success
- `1` — verify failed (generate runs verify first and refuses to render an invalid file)
- `2` — file not found or unreadable
- `3` — output path not writable

---

### `ariel watch <file> [--port <n>]`

**Purpose:** Start a local HTTP server that serves the walkthrough and live-reloads when the file changes.

**Flags:**
- `--port <n>` — port to bind (default: 2313)

**Behavior:**
1. Runs `verify` on startup. Prints errors but does not refuse to start — the browser should show an error state when the file is invalid, not a blank page.
2. Starts an HTTP server on the specified port
3. Opens the browser automatically (`http://localhost:2313`)
4. Watches the input file for changes using filesystem events (not polling)
5. On change: re-runs verify, re-parses the YAML, broadcasts updated content over websocket
6. Browser receives the update, re-renders the diagram and resets to step 1

**Websocket behavior:**
- Server sends a JSON message on change: `{ "type": "update", "content": "<full rendered HTML page>" }`
- On verify error after a change: `{ "type": "error", "message": "<error text>" }`
- Browser displays error state non-destructively (overlay, does not lose current step position if the error is minor)

**The watch output HTML is identical to generate output except:**
- Includes a small websocket client snippet (~20 lines of JS)
- Connects to `ws://localhost:<port>/ws` on load
- On `update` message: replaces the entire page document with the new HTML (resets to step 1)
- On `error` message: shows error overlay with the message

**Exit codes:**
- `0` — clean shutdown (Ctrl+C)
- `1` — port already in use
- `2` — file not found

---

## DSL — The Walkthrough File Format

Files use the `.ariel.yaml` extension by convention (not enforced).

Two top-level formats are supported. The two formats cannot be combined in one file.

### Single-diagram format

```yaml
# Required. Title shown in the browser header.
title: "User Authentication Flow"

# Required. The Mermaid diagram definition.
# Embedded inline as a YAML block scalar.
# Must be valid Mermaid syntax.
mermaid_diagram: |
  graph TD
    U([User]) -->|submits credentials| LF[Login Form]
    ...

# Required. Ordered list of walkthrough steps.
# Must contain at least one step.
steps:
  - ...
```

### Multi-diagram format

```yaml
title: "My Walkthrough"

# Required. Each section has its own diagram and step list.
# Must contain at least one section; each section must have at least one step.
sections:
  - title: "Overview"           # optional section title; shown in progress UI
    mermaid_diagram: |
      graph LR
        A --> B
    steps:
      - narration: "Section one."

  - title: "Detail"
    mermaid_diagram: |
      graph TD
        ...
    steps:
      - narration: "Section two."
```

Clicking Next at the last step of a section transitions to the first step of the next section, re-rendering the diagram. The browser-level progress UI shows section dots (if more than one section) above step dots for the current section.

### Step structure

All fields in a step are optional except at least one of `narration`, `label`, `highlight_nodes`, `active_nodes`, or `animate_edges` must be present.

```yaml
steps:
  - # Optional. One or two sentences of plain English narration.
    # Shown as text beneath or beside the diagram.
    # Should describe what is happening and why it matters — not just restate the diagram.
    # Aim for one sentence per step. Two sentences maximum.
    narration: "The API is the decision-maker here — it will determine whether the user gets in or is rejected."

    # Optional. Label shown above the narration (e.g. "Entry point", "The decision").
    # Short — 2-4 words.
    label: "The decision"

    # Optional. List of node IDs to highlight.
    # Highlighted nodes are visually emphasized (filled, bordered) but not "active".
    # Use for nodes that are relevant context for this step.
    highlight_nodes:
      - API
      - PV

    # Optional. List of node IDs to mark as "active".
    # Active nodes are more strongly emphasized than highlighted — use for the primary actor in the step.
    active_nodes:
      - PV

    # Optional. List of edges to animate.
    # Format: "SOURCE_ID-TARGET_ID"
    # The animation shows data/control flowing along the edge (CSS stroke-dashoffset).
    animate_edges:
      - API-PV
      - LF-API
```

### Node ID rules

Node IDs are the identifiers used in the Mermaid diagram definition, not the display labels.

In the diagram:
```
API[Auth API]
```
The node ID is `API`. The display label is `Auth API`.

In steps, always reference the ID, never the label:
```yaml
highlight_nodes:
  - API   # correct
  - Auth API   # wrong — this is the label, not the ID
```

Edge references use the format `SOURCE_ID-TARGET_ID`:
```yaml
animate_edges:
  - API-PV   # edge from API to PV
```

Both the source and target must be valid node IDs, and an edge between them must exist in the diagram.

### Complete example

```yaml
mermaid_diagram: |
  graph TD
    U([User]) -->|submits credentials| LF[Login Form]
    LF -->|POST /auth/login| API[Auth API]
    API -->|lookup| DB[(User DB)]
    DB -->|user record| API
    API --> PV{Password Valid?}
    PV -->|yes| TG[Token Generator]
    PV -->|no| ER[Error Response]
    TG --> SE[Set Cookie]
    SE --> DA[Dashboard]
    ER -->|401| LF

steps:
  - label: "Overview"
    narration: "This is a user authentication flow. Before stepping through it, take a moment to see the full system — a login form, an API, a database, and a decision point that determines whether a user gets in or gets rejected."

  - label: "Entry point"
    narration: "It starts with the user submitting credentials into the login form. This is the only human-initiated action in the entire flow."
    highlight_nodes: [U, LF]
    animate_edges: [U-LF]

  - label: "API handoff"
    narration: "The form POSTs to the Auth API. The API is the brain from here — it will decide everything downstream."
    highlight_nodes: [API]
    active_nodes: [LF]
    animate_edges: [LF-API]

  - label: "Database lookup"
    narration: "The API looks up the user in the database. This is a blocking round-trip — if the DB is slow or down, the entire flow stalls here."
    highlight_nodes: [DB]
    active_nodes: [API]
    animate_edges: [API-DB, DB-API]

  - label: "The decision"
    narration: "Here is the critical fork. Everything downstream depends on this single check. What does 'valid' actually mean — bcrypt comparison, rate limiting, account lockout? This spec does not say."
    highlight_nodes: [PV]
    animate_edges: [API-PV]

  - label: "Failure path"
    narration: "On failure, a 401 is returned and the form is shown again. Notice this loop can repeat indefinitely — there is no brute-force protection visible in this spec."
    highlight_nodes: [ER, LF]
    animate_edges: [PV-ER, ER-LF]

  - label: "Success path"
    narration: "On success, a JWT is generated, stored in a cookie, and the user is redirected to the dashboard. The security properties of the JWT — expiry, algorithm, rotation — are unspecified."
    highlight_nodes: [TG, SE, DA]
    animate_edges: [PV-TG, TG-SE, SE-DA]

  - label: "What to scrutinize"
    narration: "Two things deserve attention in a review: the absence of rate limiting on the failure loop, and the unspecified JWT security properties."
    highlight_nodes: [PV, ER, TG]
```

---

## Frontend — Rendering

### Mermaid

- Version: `10.6.1` (pinned)
- CDN: `https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js`
- Theme: `dark`
- Mermaid renders the diagram as an inline SVG into a container div

### Node identification

After Mermaid renders, the frontend identifies nodes by parsing the `id` attribute of each `.node` SVG group element. Mermaid 10.6.1 gives every node group an ID of the form `flowchart-{nodeId}-{n}`. The node ID is extracted from this pattern. This approach is robust to duplicate display labels — two nodes can share the same label text without collision.

### Click-to-navigate

Nodes that appear in at least one step's `highlight_nodes` or `active_nodes` are navigable. Clicking a navigable node jumps to the next step (relative to the current position) that references that node, wrapping around. Nodes not referenced by any step in the current section are non-interactive and show no hover affordance.

Behavior:
- Cursor changes to `pointer` on hover for navigable nodes
- Subtle opacity reduction on hover to signal interactivity
- If the current step already references the node, the next click advances to the subsequent occurrence (cycling)
- Click navigation is scoped to the current section; cross-section navigation is not supported

### Node highlighting

After Mermaid renders, the frontend builds a node map by scanning the SVG for `.node` elements and matching their text content to node IDs in the step script.

Two highlight states:
- `.highlighted` — visually emphasized (distinct fill and border color)
- `.active` — more strongly emphasized (brighter border, glow effect)

CSS transitions handle the state changes (0.3–0.4s ease). All nodes not in `highlight_nodes` or `active_nodes` are in a dimmed default state.

### Edge animation

Animated edges use the CSS `stroke-dashoffset` technique:

```css
@keyframes flowEdge {
  from { stroke-dashoffset: 24; }
  to { stroke-dashoffset: 0; }
}

/* Mermaid 10.6.1 uses .flowchart-link for edge elements */
.flowchart-link.animated {
  stroke-dasharray: 8 4;
  animation: flowEdge 0.8s linear infinite;
}
```

This creates the appearance of data flowing along the edge.

All edges not in `animate_edges` for the current step are in their default (non-animated) state.

### Step player

- Previous / Next buttons
- Progress indicator (dot per step, current step pill-shaped); intro dot is circular and accent-colored
- Dots are clickable for direct navigation
- Keyboard navigation: ArrowRight / Space = next, ArrowLeft = previous
- Narration text fades out and in on step change (0.2s opacity transition)
- First step of each section is an **intro slide**: no step counter shown; label displays the section title (multi-diagram) or the step's label field (single-diagram)
- Numbered steps start at "1 of N" where N excludes the intro (e.g. "2 of 8 — The decision" is step index 2 of a 9-step section)
- On the last step of the last section, Next button changes to "Done" and is disabled

### Layout

- Two-pane layout: diagram on the left (or top on narrow screens), narration + controls on the right (or bottom)
- All CSS and JS inlined in the output HTML
- Background: dark theme (consistent with developer tooling context)
- No external fonts — system font stack

### Header

- Browser tab title: `<walkthrough title> | Ariel`
- Header shows the walkthrough title as a large centered `<h1>`
- "Ariel ↗" link in the top-right corner, linking to the ariel GitHub repository
- No "Walkthrough" badge or chip

### Self-contained output requirement

The `generate` output must:
- Contain no external resource references except the pinned Mermaid CDN
- Be openable by double-clicking in Chrome, Firefox, or Safari with no server
- Be shareable via email, Slack, or Git commit without any additional files
- Render correctly offline (except for the Mermaid CDN load)

---

## Repository Structure (suggested)

```
ariel/
├── main.go
├── cmd/
│   ├── guide.go
│   ├── verify.go
│   ├── generate.go
│   └── watch.go
├── dsl/
│   ├── parse.go        # YAML parsing and validation
│   ├── verify.go       # Semantic verification logic
│   └── schema.go       # DSL type definitions
├── renderer/
│   ├── generate.go     # HTML generation (static)
│   ├── watch.go        # HTTP server + websocket
│   └── template.go     # HTML/CSS/JS template
├── guide/
│   └── guide.go        # DSL reference text
├── .goreleaser.yaml
└── README.md
```

---

## Reference Artifact

A working prototype of the frontend renderer exists as `spec-walkthrough.html`. This file demonstrates:
- The target visual design (dark theme, two-pane layout)
- The CSS SVG highlighting and edge animation technique
- The step player behavior (narration fade, progress dots, keyboard navigation)
- The hardcoded step script format (which the YAML DSL is designed to produce)

Claude Code should use this file as the reference for the frontend output of `ariel generate`. The websocket snippet for `ariel watch` is an additive delta on top of this baseline.

---

## Authoring Tips for LLMs (content of `ariel guide`)

These tips should be included in the output of `ariel guide` to help an LLM produce high-quality walkthrough files.

**What to narrate:**
- Decision points — forks where the system chooses between paths
- Non-obvious design choices — places where the implementation is surprising or could easily have been done differently
- Failure modes and what is unspecified
- What changes between versions (for PR review use cases)
- The entry point and the primary happy path

**What not to narrate:**
- Every node and every edge — this is just reading the diagram aloud
- Implementation details that are obvious from the diagram
- More than two ideas per step

**Step sequencing:**
- Start with an overview step that shows the full diagram with no highlights
- Move from entry point to exit point following the primary happy path
- Cover the failure path after the happy path
- End with a "what to scrutinize" or "what to ask" step

**Narration style:**
- One sentence per step, two maximum
- Plain English — no jargon unless the jargon is the point
- Write from the system's perspective, not the diagram's: "The API decides" not "Node PV branches"
- The last step should identify what is worth scrutinizing in a review

**Node IDs:**
- Always check the `mermaid_diagram` block to find the correct node IDs before writing steps
- IDs are case-sensitive
- Run `ariel verify` after writing the file — it will catch any ID mismatches

---

## Out of Scope for V1

The following are explicitly deferred:

- Audio narration / text-to-speech
- Socratic interrogation mode (quiz the viewer on system comprehension)
- Multiple diagrams in one file
- Sub-diagrams or drill-down
- Collaboration features
- Persistence or cloud storage
- Authentication
- Branching walkthroughs (conditional paths)
- Custom node types or icons beyond standard Mermaid
- Replacing Mermaid with a custom renderer
