package guide

const Reference = `ariel DSL reference — read this before authoring a walkthrough file.

FILE STRUCTURE — single diagram
  title: "My Walkthrough"          # optional; default: "Ariel Walkthrough"
  mermaid_diagram: |               # required; valid Mermaid syntax
    graph TD
      A[Node A] --> B[Node B]
  steps:                           # required; at least one entry
    - label: "Step label"          # optional; 2–4 words shown above narration
      narration: "What happens."   # optional; 1–2 plain-English sentences
      # NOTE: the first step of each section is the overview. It may only use
      # "label" and "narration" — highlight_nodes, active_nodes, and animate_edges
      # are not allowed. All subsequent steps may use any combination.
    - label: "Step 2"
      highlight_nodes: [A, B]      # optional; node IDs to emphasize (context)
      active_nodes: [A]            # optional; node IDs for primary actor (stronger emphasis)
      animate_edges: [A-B]         # optional; edges to animate (SOURCE_ID-TARGET_ID)

FILE STRUCTURE — multiple diagrams
  title: "My Walkthrough"
  sections:
    - title: "Overview"            # optional section title
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

  Use "sections" for multi-diagram walkthroughs. Use "mermaid_diagram"+"steps"
  for single-diagram. The two formats cannot be combined in one file.

  Each step must have at least one of: narration, label, highlight_nodes,
  active_nodes, or animate_edges. Unknown fields at any level are errors.

NODE IDs
  From "A[Display Label]", the node ID is "A". Always reference the ID, never
  the label. IDs are case-sensitive. Check the mermaid_diagram block for exact
  IDs before writing steps.

  Supported node shapes (all extract the same way):
    A[text]       rectangle
    A{text}       diamond
    A([text])     rounded
    A[(text)]     cylinder
    A((text))     circle

EDGE FORMAT
  animate_edges entries use "SOURCE_ID-TARGET_ID". Both must be valid node IDs
  with a direct edge between them in mermaid_diagram. Example: "API-PV".

AUTHORING TIPS
  Step sequencing:
    1. Overview step — no highlights, lets the viewer see the full diagram
    2. Entry point — first human-initiated action
    3. Happy path — follow the primary flow from start to finish
    4. Failure path — cover error branches after the happy path
    5. "What to scrutinize" — final step identifying what deserves review

  For multi-diagram files, use sections to progress from simple to detailed:
    - Section 1: high-level overview (few nodes, major components only)
    - Section 2+: drill into subsystems or failure paths

  What to narrate:
    - Decision points (forks where the system chooses between paths)
    - Non-obvious design choices (surprising or could easily be done differently)
    - Failure modes and missing specifications
    - The entry point and primary happy path

  What NOT to narrate:
    - Every node and every edge (this just reads the diagram aloud)
    - Implementation details obvious from the diagram
    - More than two ideas per step

  Narration style:
    - One sentence per step, two maximum
    - Write from the system's perspective: "The API decides" not "Node PV branches"
    - Plain English — no jargon unless the jargon is the point
    - The last step should identify what is worth scrutinizing in a review

COMMON ERRORS
  highlight_nodes/active_nodes reference unknown ID
    → Check mermaid_diagram for the exact node ID (case-sensitive)

  animate_edges references a non-existent edge
    → Both node IDs must exist AND there must be a direct edge between them

  Unknown field in step or section
    → Remove it; the DSL does not support extra fields

  Missing mermaid_diagram or steps
    → Both are required (in either the flat or sections format)

  Mixing "sections" with "mermaid_diagram"/"steps"
    → Use one format per file, not both

  Steps with no content
    → Every step needs at least one of: narration, label, highlight_nodes,
       active_nodes, animate_edges

  Warning: not all highlighted components are connected
    → The nodes referenced across highlight_nodes, active_nodes, and
       animate_edges for this step don't form a single connected subgraph.
       Often means you're combining unrelated diagram regions in one step —
       consider splitting into two steps, or add animate_edges that bridge them

  First step has highlight_nodes, active_nodes, or animate_edges
    → The first step of each section is the overview. Remove visual fields
       from step 1 — it may only use "label" and "narration"

Run "ariel verify <file>" after writing — it catches all the above.
`
