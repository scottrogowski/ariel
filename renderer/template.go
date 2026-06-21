package renderer

// htmlTemplate is the Go text/template for the generated HTML.
// Delimiters are [[ and ]] to avoid conflicts with CSS/JS braces.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>[[.Title]]</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js"></script>
<style>
  :root {
    --bg: #0f1117;
    --surface: #1a1d27;
    --border: #2a2d3a;
    --accent: #5b8dee;
    --accent-glow: rgba(91, 141, 238, 0.3);
    --success: #4ecdc4;
    --text: #e8eaf0;
    --muted: #6b7280;
    --narration-bg: #141720;
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    background: var(--bg);
    color: var(--text);
    font-family: 'Inter', system-ui, sans-serif;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  header {
    padding: 20px 32px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo {
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--accent);
  }

  .spec-title {
    font-size: 13px;
    color: var(--muted);
    margin-left: auto;
  }

  .main {
    flex: 1;
    display: grid;
    grid-template-columns: 1fr 340px;
  }

  .diagram-pane {
    padding: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-right: 1px solid var(--border);
  }

  #mermaid-container {
    width: 100%;
    max-width: 600px;
  }

  #mermaid-container svg {
    width: 100% !important;
    height: auto !important;
  }

  .side-pane {
    display: flex;
    flex-direction: column;
    background: var(--narration-bg);
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
    min-height: 120px;
    transition: opacity 0.25s ease;
  }

  .narration-text.fade { opacity: 0; }

  .progress-track {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .progress-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--border);
    transition: all 0.3s ease;
    cursor: pointer;
  }

  .progress-dot.active {
    background: var(--accent);
    box-shadow: 0 0 8px var(--accent-glow);
    width: 20px;
    border-radius: 3px;
  }

  .progress-dot.visited { background: var(--muted); }

  .controls {
    padding: 24px 32px;
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 12px;
  }

  button {
    padding: 10px 20px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: all 0.15s ease;
  }

  .btn-prev {
    background: transparent;
    color: var(--muted);
    border: 1px solid var(--border);
  }

  .btn-prev:hover:not(:disabled) {
    color: var(--text);
    border-color: var(--muted);
  }

  .btn-next {
    background: var(--accent);
    color: white;
    flex: 1;
  }

  .btn-next:hover:not(:disabled) {
    background: #4a7de0;
    box-shadow: 0 0 16px var(--accent-glow);
  }

  button:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  #mermaid-container .node rect,
  #mermaid-container .node circle,
  #mermaid-container .node polygon,
  #mermaid-container .node ellipse,
  #mermaid-container .node path {
    transition: fill 0.35s ease, stroke 0.35s ease, filter 0.35s ease;
  }

  /* Dimming: applied when any highlight or active nodes exist on the current step */
  #mermaid-container.has-highlights .node {
    opacity: 0.25;
    transition: opacity 0.35s ease;
  }

  #mermaid-container.has-highlights .node.highlighted,
  #mermaid-container.has-highlights .node.active {
    opacity: 1;
  }

  #mermaid-container .node.highlighted rect,
  #mermaid-container .node.highlighted circle,
  #mermaid-container .node.highlighted polygon,
  #mermaid-container .node.highlighted ellipse,
  #mermaid-container .node.highlighted path {
    fill: #1e3a6e !important;
    stroke: var(--accent) !important;
    stroke-width: 2px !important;
    filter: drop-shadow(0 0 8px var(--accent-glow));
  }

  #mermaid-container .node.active rect,
  #mermaid-container .node.active circle,
  #mermaid-container .node.active polygon,
  #mermaid-container .node.active ellipse,
  #mermaid-container .node.active path {
    fill: #1a4a7a !important;
    stroke: var(--success) !important;
    stroke-width: 2.5px !important;
    filter: drop-shadow(0 0 12px rgba(78, 205, 196, 0.4));
  }

  #mermaid-container .flowchart-link.animated {
    stroke: var(--accent) !important;
    stroke-width: 2.5px !important;
    stroke-dasharray: 8 4;
    animation: flowEdge 0.8s linear infinite;
  }

  @keyframes flowEdge {
    from { stroke-dashoffset: 24; }
    to   { stroke-dashoffset: 0;  }
  }

  .intro-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    background: rgba(91, 141, 238, 0.1);
    border: 1px solid rgba(91, 141, 238, 0.3);
    border-radius: 20px;
    font-size: 11px;
    color: var(--accent);
    font-weight: 500;
  }

  .pulse {
    width: 6px;
    height: 6px;
    background: var(--accent);
    border-radius: 50%;
    animation: pulse 1.5s ease infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50%       { opacity: 0.5; transform: scale(0.7); }
  }

  @media (max-width: 768px) {
    .main {
      grid-template-columns: 1fr;
      grid-template-rows: auto 1fr;
    }
    .diagram-pane {
      border-right: none;
      border-bottom: 1px solid var(--border);
      padding: 24px;
    }
  }
</style>
</head>
<body>

<header>
  <span class="logo">Walkthrough</span>
  <span class="spec-title">[[.Title]]</span>
</header>

<div class="main">
  <div class="diagram-pane">
    <div id="mermaid-container">
      <div class="mermaid">[[.MermaidDiagram]]</div>
    </div>
  </div>

  <div class="side-pane">
    <div class="narration-area">
      <div class="intro-badge"><span class="pulse"></span> Step-by-step walkthrough</div>
      <div class="step-label" id="step-label"></div>
      <div class="narration-text" id="narration"></div>
      <div class="progress-track" id="progress-track"></div>
    </div>
    <div class="controls">
      <button class="btn-prev" id="btn-prev" onclick="prevStep()" disabled>← Back</button>
      <button class="btn-next" id="btn-next" onclick="nextStep()"></button>
    </div>
  </div>
</div>

<script>
// nodeLabels maps Mermaid node IDs to their display labels.
// Used to find the correct SVG .node elements after render.
const nodeLabels = [[.NodeLabelsJSON]];
const steps = [[.StepsJSON]];

let nodeMap = {};
let edgeMap = {};
let currentStep = 0;
let initialized = false;

mermaid.initialize({
  startOnLoad: true,
  theme: 'dark',
  themeVariables: {
    primaryColor: '#1a2744',
    primaryTextColor: '#e8eaf0',
    primaryBorderColor: '#2a3a5a',
    lineColor: '#4a5568',
    secondaryColor: '#1a1d27',
    tertiaryColor: '#1a1d27',
    background: '#0f1117',
    mainBkg: '#1a2744',
    nodeBorder: '#2a3a5a',
    clusterBkg: '#1a1d27',
    titleColor: '#e8eaf0',
    edgeLabelBackground: '#1a1d27',
    fontFamily: 'Inter, system-ui, sans-serif'
  }
});

setTimeout(() => {
  buildNodeMap();
  buildProgressDots();
  renderStep();
  initialized = true;
}, 800);

function buildNodeMap() {
  const svg = document.querySelector('#mermaid-container svg');
  if (!svg) return;

  const labelToId = Object.fromEntries(
    Object.entries(nodeLabels).map(([id, label]) => [label, id])
  );

  svg.querySelectorAll('.node').forEach(group => {
    const textEl = group.querySelector('text, .label');
    if (!textEl) return;
    const id = labelToId[textEl.textContent.trim()];
    if (id) nodeMap[id] = group;
  });

  edgeMap.all = Array.from(svg.querySelectorAll('.flowchart-link'));
}

function clearAllHighlights() {
  const svg = document.querySelector('#mermaid-container svg');
  if (!svg) return;
  svg.querySelectorAll('.node').forEach(n => n.classList.remove('highlighted', 'active'));
  svg.querySelectorAll('.flowchart-link').forEach(e => e.classList.remove('animated'));
}

function applyStep(step) {
  clearAllHighlights();

  const hasHighlights = step.highlight_nodes.length > 0 || step.active_nodes.length > 0;
  document.getElementById('mermaid-container').classList.toggle('has-highlights', hasHighlights);

  step.highlight_nodes.forEach(id => { if (nodeMap[id]) nodeMap[id].classList.add('highlighted'); });
  step.active_nodes.forEach(id => { if (nodeMap[id]) nodeMap[id].classList.add('active'); });

  if (step.animate_edges.length > 0 && edgeMap.all) {
    edgeMap.all.forEach((edge, i) => {
      if (i < step.animate_edges.length) edge.classList.add('animated');
    });
  }
}

function buildProgressDots() {
  const track = document.getElementById('progress-track');
  steps.forEach((_, i) => {
    const dot = document.createElement('div');
    dot.className = 'progress-dot';
    dot.onclick = () => goToStep(i);
    track.appendChild(dot);
  });
}

function updateProgressDots() {
  document.querySelectorAll('.progress-dot').forEach((dot, i) => {
    dot.className = 'progress-dot';
    if (i < currentStep) dot.classList.add('visited');
    if (i === currentStep) dot.classList.add('active');
  });
}

function goToStep(index) {
  currentStep = index;
  renderStep();
}

function renderStep() {
  const step = steps[currentStep];
  const narrationEl = document.getElementById('narration');
  const labelEl = document.getElementById('step-label');

  narrationEl.classList.add('fade');
  setTimeout(() => {
    narrationEl.textContent = step.narration || '';
    labelEl.textContent = step.label
      ? (currentStep + 1) + ' of ' + steps.length + ' — ' + step.label
      : (currentStep + 1) + ' of ' + steps.length;
    narrationEl.classList.remove('fade');
  }, 200);

  if (initialized) applyStep(step);

  document.getElementById('btn-prev').disabled = currentStep === 0;
  const nextBtn = document.getElementById('btn-next');
  if (currentStep === steps.length - 1) {
    nextBtn.textContent = 'Done';
    nextBtn.disabled = true;
  } else {
    nextBtn.textContent = currentStep === 0 ? 'Begin walkthrough →' : 'Next →';
    nextBtn.disabled = false;
  }

  updateProgressDots();
}

function nextStep() {
  if (currentStep < steps.length - 1) { currentStep++; renderStep(); }
}

function prevStep() {
  if (currentStep > 0) { currentStep--; renderStep(); }
}

document.addEventListener('keydown', e => {
  if (e.key === 'ArrowRight' || e.key === ' ') { e.preventDefault(); nextStep(); }
  if (e.key === 'ArrowLeft') { e.preventDefault(); prevStep(); }
});
</script>
[[.WSSnippet]]</body>
</html>
`
