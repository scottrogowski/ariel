---
name: create-walkthrough
description: >-
  Create an animated Mermaid diagram walkthrough to explain a system, PR, code
  path, or concept with the `ariel` CLI. Use when the user asks to visualize,
  diagram, or walk through how something works. Renders step-by-step narrated diagrams from a YAML DSL.
---

# Create an ariel walkthrough

`ariel` renders step-by-step Mermaid diagram walkthroughs from a `.ariel.yaml` file.

1. Run `ariel guide` and read it fully — it is the authoritative DSL reference. Do this before authoring. It is short.
2. Author a `.ariel.yaml` file describing the diagram and the narrated steps.
3. Run `ariel verify <file>` and fix any reported issues.
4. Render or preview:
   - `ariel watch <file>` — live-reloading browser preview while iterating (best for working with the user).
   - `ariel generate <file>` — self-contained HTML file.
   - `ariel generate --format svg <file>` — interactive SVG for embedding in GitHub PRs and READMEs.

The `ariel` command is provided by this plugin and builds from source on first use, so Go must be installed.
