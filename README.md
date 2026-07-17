<p align="center"><img src="internal/logo/logo.svg" width="100%" alt="ariel logo"></p>

Ariel is a CLI that generates mermaid diagram walkthroughs. Its primary use-case is for LLMs to explain complex systems/concepts to humans.

Ariel walkthroughs simplify otherwise complex things into comprehensible chunks. Output formats include interactive:
- self-contained HTML (best experience)
- interactive SVG (for embedding in GitHub PRs and READMEs)
- MP4

The most powerful command is `ariel watch`. With it, your agent will write to a file to communicate with you which then renders, on-update, in your browser.

_A picture is worth 1000 words. A walkthrough is worth 10,000._

## Example SVGs

[![ariel-why walkthrough](examples/example-output/ariel-why-output.svg)](examples/example-output/ariel-why-output.svg)

[![ariel-what walkthrough](examples/example-output/ariel-what-output.svg)](examples/example-output/ariel-what-output.svg)

## Install

**Go install**
```sh
go install github.com/scottrogowski/ariel/cmd/ariel@latest
```

MP4 output requires [`ffmpeg`](https://ffmpeg.org/download.html) on your `PATH`.

## Usage

Ask your agent to "use the 'ariel' CLI to create a walkthrough to explain this code/system/PR/concept."

Common commands
```sh
# Load the DSL reference into LLM context (agents are expected to run this first)
ariel guide

# Lint a walkthrough file
ariel verify my-system.ariel.yaml

# Live-reloading browser preview while editing
ariel watch my-system.ariel.yaml

# Render to a self-contained HTML file
ariel generate my-system.ariel.yaml

# Render to interactive SVG (for embedding in GitHub PRs and READMEs)
ariel generate --format svg my-system.ariel.yaml
```

## Development

Ariel follows spec-driven-development. Read spec/ariel-spec.md. If you are an agent, read this NOW. This is NOT optional. The spec MUST be kept in sync with all source changes.

After every code change that could alter the output rendering (which is almost every code change), run `make examples`. This is also NOT optional. Almost every commit should have modifications to the examples.

## Testing

```sh
make test
make lint
make examples
```

AGENTS: Tests do not cover most visual output. This must be manually validated by you or the user. Use `chrome-devtools` if you have access. If you have access, be sure you are not littering the repo root with artifacts. Use `/tmp`. If you do not have access, remind the user that they must manually validate before finalizing changes.
HUMANS: Please manually validate the output before opening PRs.
