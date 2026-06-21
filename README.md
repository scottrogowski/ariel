# ariel

Animated diagram walkthroughs. Give ariel a Mermaid diagram and a list of steps; it renders a self-contained HTML presentation with highlighted nodes, animated edges, and narration — one idea at a time.

Designed to be authored by an LLM. Run `ariel guide` at the start of a session to load the DSL into context.

## Install

**Homebrew** (coming soon)

**Go install**
```sh
go install github.com/scottmrogowski/ariel@latest
```

**Pre-built binaries** — download from [Releases](https://github.com/scottmrogowski/ariel/releases) and put the binary on your `PATH`.

## Usage

```sh
# 1. Load the DSL into LLM context
ariel guide

# 2. Author or generate a walkthrough file, then lint it
ariel verify my-system.ariel.yaml

# 3. Preview with live reload while editing
ariel watch my-system.ariel.yaml

# 4. Render to a shareable, self-contained HTML file
ariel generate my-system.ariel.yaml
```

### `ariel watch` options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `2313` | Port to bind the preview server |

### `ariel generate` options

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `<input>.html` | Output path for the generated HTML |

## Walkthrough file format

Files use the `.ariel.yaml` extension by convention.

```yaml
title: "User Authentication Flow"

mermaid_diagram: |
  graph TD
    U([User]) -->|submits credentials| LF[Login Form]
    LF -->|POST /auth/login| API[Auth API]
    API --> PV{Password Valid?}
    PV -->|yes| TG[Token Generator]
    PV -->|no| ER[Error Response]
    TG --> DA[Dashboard]
    ER -->|401| LF

steps:
  - label: "Overview"
    narration: "The full authentication flow — a login form, an API, and a decision point."

  - label: "The decision"
    narration: "Everything downstream depends on this single check."
    highlight_nodes: [PV]
    animate_edges: [API-PV]

  - label: "Failure path"
    narration: "On failure, a 401 is returned. Notice the loop has no rate limiting."
    highlight_nodes: [ER, LF]
    animate_edges: [PV-ER, ER-LF]
```

**Node IDs** are the identifiers from the diagram (`API`, `PV`), not the display labels (`Auth API`, `Password Valid?`). Run `ariel verify` to catch mismatches.

Run `ariel guide` for the full DSL reference including authoring tips for LLMs.

## Building from source

```sh
git clone https://github.com/scottmrogowski/ariel
cd ariel
make build
```

Requires Go 1.23+.

## License

MIT
