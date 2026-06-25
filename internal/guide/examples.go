package guide

// SingleDiagramExample is a complete single-section walkthrough YAML.
// It demonstrates most DSL features while explaining why walkthroughs work.
const SingleDiagramExample = `# Save as walkthrough.ariel.yaml, then run: ariel generate walkthrough.ariel.yaml
title: How Walkthroughs Aid Comprehension

mermaid_diagram: |
  flowchart TD
    Reader[Reader] --> WM[Working Memory]
    WM --> Overload[Cognitive Overload]
    WM --> Model[Mental Model]
    Walkthrough[Walkthrough] --> WM

steps:
  - label: The challenge of understanding complex systems
    narration: This walkthrough is itself a demonstration of the technique it describes. Before stepping through it, take a moment to see the full diagram — a reader, working memory, two possible outcomes, and one intervention.

  - label: The bottleneck
    highlight_nodes: [Reader, WM]
    animate_edges: [Reader-WM]
    narration: Working memory holds roughly four chunks at a time (Cowan, 2001). Every new node, edge, and label a reader encounters competes for those same limited slots.

  - label: Overload
    highlight_nodes: [WM, Overload]
    animate_edges: [WM-Overload]
    narration: When a complex system is presented all at once, working memory fills and the reader loses the thread. The response is confusion or abandonment — not understanding.

  - label: The guided path
    highlight_nodes: [WM, Model]
    animate_edges: [WM-Model]
    narration: A walkthrough sequences the same information into steps. Each step adds one idea. The reader builds a mental model incrementally, never hitting the capacity limit at any single moment. This is Progressive Disclosure.

  - label: Dual coding
    active_nodes: [Walkthrough]
    highlight_nodes: [WM]
    animate_edges: [Walkthrough-WM]
    narration: When narration and a matching visual are delivered together, they engage two separate cognitive channels — verbal and pictorial — which the brain integrates into a richer representation. Paivio's Dual Coding Theory (1986) explains why this produces deeper and more durable understanding than either channel alone.
`

// MultipleDiagramExample is a complete multi-section walkthrough YAML.
// It demonstrates sections, cross-diagram narration, and most DSL features.
const MultipleDiagramExample = `# Save as walkthrough.ariel.yaml, then run: ariel generate walkthrough.ariel.yaml
title: Why Walkthroughs Work

sections:
  - title: The Problem
    mermaid_diagram: |
      flowchart TD
        System[Complex System] --> Dump[Information Dump]
        Dump --> WM[Working Memory]
        WM --> Overload[Cognitive Overload]
        Overload --> Lost[Reader Disengages]
    steps:
      - label: Why complex systems are hard to teach
        narration: Every complex system has more moving parts than a person can hold in mind at once. The instinct is to present everything upfront — a full architecture diagram, every edge case, every dependency. For the reader, this is the worst possible approach.

      - label: The information dump
        highlight_nodes: [System, Dump]
        animate_edges: [System-Dump]
        narration: Documentation is often written from the author's perspective — someone who already understands the system. The result is an undifferentiated wall of information with no clear entry point.

      - label: Working memory is the bottleneck
        highlight_nodes: [Dump, WM]
        animate_edges: [Dump-WM]
        narration: Humans have a single cognitive thread. Working memory holds roughly four chunks at a time (Cowan, 2001). When new information arrives faster than it can be encoded into a mental model, the chunks simply fall out.

      - label: The outcome
        active_nodes: [Overload, Lost]
        animate_edges: [WM-Overload, Overload-Lost]
        narration: Cognitive overload does not announce itself as confusion — it feels like the material is too advanced, too dense, or not worth the effort. The reader disengages. The knowledge transfer fails.

  - title: The Solution
    mermaid_diagram: |
      flowchart TD
        Author[Author] --> YAML[ariel YAML]
        YAML --> Generate[ariel generate]
        Generate --> Output[HTML or MP4]
        Output --> Reader[Reader]
    steps:
      - label: How ariel addresses cognitive load
        narration: ariel inverts the information-dump model. Instead of presenting a full diagram and leaving the reader to find the thread, it sequences the diagram through a series of narrated steps — each one adding exactly one idea.

      - label: The author's job
        highlight_nodes: [Author, YAML]
        animate_edges: [Author-YAML]
        narration: The author writes a YAML file pairing a Mermaid diagram with a sequence of steps. Each step specifies which nodes to highlight, which edges to animate, and one or two sentences of narration. The diagram does not change — only what is foregrounded does.

      - label: ariel generates the output
        highlight_nodes: [YAML, Generate]
        animate_edges: [YAML-Generate]
        narration: ariel verify catches unknown node IDs, missing edges, and structural errors before generation. ariel generate produces either an interactive HTML page (keyboard navigable, with progress tracking) or an MP4 suitable for embedding directly in a GitHub README.

      - label: The reader's experience
        active_nodes: [Output, Reader]
        animate_edges: [Output-Reader]
        narration: The reader follows a single narrative thread through the diagram. Each step occupies one slot in working memory. By the end, they have a mental model built incrementally — never overloaded at any single moment. This is the same reason a good technical talk is more effective than a dense slide deck.
`
