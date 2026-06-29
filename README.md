# ariel

<p align="center"><img src="logo.png" width="128" alt="Ariel logo"></p>

![ariel-why walkthrough](examples/ariel-why-output.gif)

![ariel-what walkthrough](examples/ariel-what-output.gif)

Step-by-step Mermaid diagram walkthroughs. Each walkthrough pairs a diagram with a sequence of steps that highlight nodes, animate edges, and narrate what is happening. Outputs HTML (interactive), MP4, and GIF (for embedding in GitHub READMEs and PR descriptions).

Run `ariel guide` at the start of a session to load the full DSL into context.

## Why?

LLMs are generating the majority of our code and it is getting harder and harder to understand the systems they are creating. Humans are single-threaded. We process information best when presented in narrative format with ample visual aids.

## Install

**Go install**
```sh
go install github.com/scottmrogowski/ariel@latest
```

**Pre-built binaries** — download from [Releases](https://github.com/scottmrogowski/ariel/releases) and put the binary on your `PATH`.

MP4 output requires [`ffmpeg`](https://ffmpeg.org/download.html) on your `PATH`.

## Usage

```sh
# Load the DSL reference into LLM context (run this first when using an agent)
ariel guide

# Get a working example to start from
ariel single-diagram-example > walkthrough.ariel.yaml
ariel multiple-diagram-example > walkthrough.ariel.yaml

# Lint a walkthrough file
ariel verify my-system.ariel.yaml

# Live-reloading browser preview while editing
ariel watch my-system.ariel.yaml

# Render to a self-contained HTML file
ariel generate my-system.ariel.yaml

# Render to MP4 (for GitHub README embedding)
ariel generate --format mp4 my-system.ariel.yaml
```

### `ariel generate` flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `html` | Output format: `html`, `mp4`, or `gif` |
| `--output` | input filename with format extension | Output path |
| `--step-duration` | `2` | Seconds each step is held (mp4 and gif only) |

### `ariel watch` flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `2313` | Port to bind the preview server |

## Walkthrough file format

Files use the `.ariel.yaml` extension by convention.

```yaml
title: "User Authentication Flow"   # optional; default: "Ariel Walkthrough"

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
  # The first step of each section is the overview. It may only use
  # label and narration — no visual fields. The full diagram is shown
  # without dimming so the viewer can orient before the walkthrough begins.
  - label: "Overview"
    narration: "The full authentication flow — a login form, an API, and a decision point."

  - label: "The decision"
    narration: "Everything downstream depends on this single check."
    highlight_nodes: [API, PV]

  - label: "Failure path"
    narration: "On failure, a 401 is returned. Notice the loop has no rate limiting."
    highlight_nodes: [ER, LF]
```

**Node IDs** are the identifiers from the diagram (`API`, `PV`), not display labels (`Auth API`, `Password Valid?`). `ariel verify` catches all mismatches.

`highlight_nodes` and `focus_nodes` are supported for `flowchart`/`graph` and `sequenceDiagram` only. Using them with other Mermaid diagram types is a verify error.

For multi-diagram walkthroughs use `sections` instead of top-level `mermaid_diagram`/`steps`. See `ariel multiple-diagram-example` for a complete example.

Run `ariel guide` for the full DSL reference.

## Developing

AGENTS: Ariel follows spec-driven-development. Read spec/ariel-spec.md now. This is NOT optional. The spec MUST be kept in sync with all source changes.

## Testing

```sh
make test     # unit tests
make example  # generates HTML, MP4, and GIF outputs under examples/
```

Automated tests cover parsing, verification, and CLI behavior. Visual output correctness — node highlighting, edge animation, layout, and video/GIF playback — cannot be tested automatically. After any change to the renderer or template, open the generated HTML in a browser and inspect the GIF to verify.

## License

MIT
