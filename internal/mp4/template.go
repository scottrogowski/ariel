package mp4

// sectionHTMLTemplate renders a static, no-animation page for one walkthrough section.
// The page exposes applyStep() so the Go capture loop can set each step's state synchronously.
const sectionHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>[[.Title]]</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js"></script>
<style>
  *, *::before, *::after {
    transition: none !important;
    animation: none !important;
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  [[.ThemeCSS]]

  html, body {
    width: 100%;
    height: 100vh;
    overflow: hidden;
    background: var(--bg);
    color: var(--text);
    font-family: 'Inter', system-ui, sans-serif;
  }

  header {
    height: 64px;
    padding: 0 32px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .page-title {
    font-size: 22px;
    font-weight: 600;
    color: var(--text);
  }

  .main {
    display: grid;
    grid-template-columns: 1fr 340px;
    height: calc(100vh - 64px);
  }

  .diagram-pane {
    padding: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-right: 1px solid var(--border);
    overflow: hidden;
  }

  #mermaid-container {
    width: 100%;
    max-width: 600px;
    /* Stretch to fill the diagram pane height so the SVG viewBox has a sized
       container. preserveAspectRatio (xMidYMid meet) then scales the diagram
       to fit both axes without clipping, for any diagram aspect ratio. */
    align-self: stretch;
  }

  #mermaid-container .mermaid,
  #mermaid-container svg {
    width: 100% !important;
    height: 100% !important;
    display: block;
  }

  .side-pane {
    display: flex;
    flex-direction: column;
    background: var(--narration-bg);
    overflow: hidden;
  }

  .narration-area {
    flex: 1;
    padding: 40px 32px 32px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .step-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--accent);
  }

  .narration-text {
    font-size: 17px;
    line-height: 1.65;
    color: var(--text);
  }

  #mermaid-container.has-highlights .node { opacity: 0.25; }

  #mermaid-container.has-highlights .node.highlighted,
  #mermaid-container.has-highlights .node.active { opacity: 1; }

  #mermaid-container .node.highlighted rect,
  #mermaid-container .node.highlighted circle,
  #mermaid-container .node.highlighted polygon,
  #mermaid-container .node.highlighted ellipse,
  #mermaid-container .node.highlighted path {
    fill: var(--highlight-fill) !important;
    stroke: var(--accent) !important;
    stroke-width: 2px !important;
    filter: drop-shadow(0 0 8px var(--accent-glow));
  }

  #mermaid-container .node.active rect,
  #mermaid-container .node.active circle,
  #mermaid-container .node.active polygon,
  #mermaid-container .node.active ellipse,
  #mermaid-container .node.active path {
    fill: var(--focus-fill) !important;
    stroke: var(--success) !important;
    stroke-width: 2.5px !important;
    filter: drop-shadow(0 0 12px var(--focus-glow));
  }

  #mermaid-container .flowchart-link.animated {
    stroke: var(--accent) !important;
    stroke-width: 2.5px !important;
    stroke-dasharray: 8 4;
  }
</style>
</head>
<body>
<header><h1 class="page-title">[[.Title]]</h1></header>
<div class="main">
  <div class="diagram-pane">
    <div id="mermaid-container">
      <div class="mermaid">[[.MermaidDiagram]]</div>
    </div>
  </div>
  <div class="side-pane">
    <div class="narration-area">
      <div class="step-label" id="step-label"></div>
      <div class="narration-text" id="narration"></div>
    </div>
  </div>
</div>
<div id="ready" style="display:none"></div>
<script>
[[.MermaidInit]]

let nodeMap = {}, edgeMap = {};

async function init() {
  await mermaid.run({ nodes: [document.querySelector('.mermaid')] });
  const svg = document.querySelector('#mermaid-container svg');
  svg.querySelectorAll('.node').forEach(group => {
    const m = group.id.match(/^flowchart-(\w+)-\d+$/);
    if (m) nodeMap[m[1]] = group;
  });
  svg.querySelectorAll('.flowchart-link').forEach(el => {
    const cls = Array.from(el.classList);
    const srcCls = cls.find(c => c.startsWith('LS-'));
    const dstCls = cls.find(c => c.startsWith('LE-'));
    if (!srcCls || !dstCls) return;
    const key = srcCls.slice(3) + '-' + dstCls.slice(3);
    if (!edgeMap[key]) edgeMap[key] = [];
    edgeMap[key].push(el);
  });
  document.getElementById('ready').style.display = 'block';
}

function applyStep(highlightNodes, focusNodes, label, narration) {
  const svg = document.querySelector('#mermaid-container svg');
  svg.querySelectorAll('.node').forEach(n => n.classList.remove('highlighted', 'active'));
  svg.querySelectorAll('.flowchart-link').forEach(e => e.classList.remove('animated'));
  const container = document.getElementById('mermaid-container');
  const hasHighlights = highlightNodes.length > 0 || focusNodes.length > 0;
  container.classList.toggle('has-highlights', hasHighlights);
  const focusSet = new Set(focusNodes);
  highlightNodes.forEach(id => { if (nodeMap[id] && !focusSet.has(id)) nodeMap[id].classList.add('highlighted'); });
  focusNodes.forEach(id => { if (nodeMap[id]) nodeMap[id].classList.add('active'); });
  const allNodes = [...new Set([...highlightNodes, ...focusNodes])];
  for (let i = 0; i < allNodes.length; i++) {
    for (let j = 0; j < allNodes.length; j++) {
      if (i !== j) (edgeMap[allNodes[i] + '-' + allNodes[j]] || []).forEach(el => el.classList.add('animated'));
    }
  }
  document.getElementById('step-label').textContent = label;
  document.getElementById('narration').textContent = narration;
}

init();
</script>
</body>
</html>
`
