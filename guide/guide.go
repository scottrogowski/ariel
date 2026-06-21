package guide

const Reference = `ariel DSL reference — read this before authoring a walkthrough file.

FILE STRUCTURE
  title: "My Walkthrough"          # required; shown in browser header
  mermaid_diagram: |               # required; valid Mermaid syntax
    graph TD
      A[Node A] --> B[Node B]
  steps:                           # required; at least one entry
    - label: "Step label"          # optional; 2–4 words shown above narration
      narration: "What happens."   # optional; 1–2 plain-English sentences
      highlight_nodes: [A, B]      # optional; node IDs to emphasize (context)
      active_nodes: [A]            # optional; node IDs for primary actor (stronger emphasis)
      animate_edges: [A-B]         # optional; edges to animate (SOURCE_ID-TARGET_ID)

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

  Unknown field in step
    → Remove it; the DSL does not support extra fields

  Missing title, mermaid_diagram, or steps
    → All three are required at the top level

  Steps with no content
    → Every step needs at least one of: narration, label, highlight_nodes,
       active_nodes, animate_edges

Run "ariel verify <file>" after writing — it catches all the above.
`
