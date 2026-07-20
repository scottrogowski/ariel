package renderer

// htmlTemplate is the Go text/template for the generated HTML.
// Delimiters are [[ and ]] to avoid conflicts with CSS/JS braces.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>[[.Title]] | Ariel</title>
<link rel="icon" type="image/svg+xml" href="data:image/svg+xml;base64,[[.FaviconBase64]]">
<script src="https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js"></script>
<style>
  [[.ThemeCSS]]

  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    background: var(--bg);
    color: var(--text);
    font-family: 'Inter', system-ui, sans-serif;
    height: 100vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  header {
    padding: 20px 32px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }

  .page-title {
    font-size: 22px;
    font-weight: 600;
    color: var(--text);
    text-align: center;
  }

  .page-title-sep {
    margin: 0 10px;
    color: var(--muted);
    font-weight: 300;
  }

  .page-section-title {
    font-size: 22px;
    font-weight: 400;
    color: var(--text);
  }

  .ariel-link {
    position: absolute;
    right: 32px;
    opacity: 0.7;
    transition: opacity 0.15s ease;
    text-decoration: none;
    color: var(--muted);
  }

  .ariel-link:hover { opacity: 1; }

  .ariel-logo {
    display: block;
    width: 160px;
    height: auto;
  }

  .ariel-logo svg {
    display: block;
    width: 160px;
    height: auto;
  }

  .main {
    flex: 1;
    display: grid;
    grid-template-columns: 1fr 340px;
  }

  .diagram-pane {
    position: relative;
    overflow: hidden;
    border-right: 1px solid var(--border);
  }

  #mermaid-container {
    position: absolute;
    inset: 0;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: opacity 0.2s ease;
  }

  #mermaid-container svg {
    display: block;
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
    overflow-y: auto;
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

  .narration-text a {
    color: var(--accent);
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  .narration-text a:hover { color: var(--link-hover); }

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

  /* Intro dot (first step of each section) stays circular, uses accent color */
  .progress-dot.intro-dot {
    background: var(--accent);
    opacity: 0.3;
  }

  .progress-dot.intro-dot.active {
    opacity: 1;
    width: 6px;
    border-radius: 50%;
  }

  .progress-dot.intro-dot.visited {
    background: var(--muted);
    opacity: 1;
  }

  .section-track {
    display: flex;
    gap: 8px;
    align-items: center;
    padding-bottom: 8px;
  }

  .section-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--border);
    transition: all 0.3s ease;
    cursor: pointer;
  }

  .section-dot.active {
    background: var(--success);
    box-shadow: 0 0 8px var(--success-glow);
    width: 24px;
    border-radius: 3px;
  }

  .section-dot.visited { background: var(--muted); }

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
    color: var(--on-accent);
    flex: 1;
  }

  .btn-next:hover:not(:disabled) {
    background: var(--accent-hover);
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
  #mermaid-container .node path,
  #mermaid-container g.actor rect,
  #mermaid-container g.actor path {
    transition: fill 0.35s ease, stroke 0.35s ease, filter 0.35s ease;
  }

  #mermaid-container .node.navigable { cursor: pointer; }

  #mermaid-container .node.navigable:hover rect,
  #mermaid-container .node.navigable:hover circle,
  #mermaid-container .node.navigable:hover polygon,
  #mermaid-container .node.navigable:hover ellipse,
  #mermaid-container .node.navigable:hover path {
    opacity: 0.75;
  }

  /* Dimming: applied per-element when any highlighted/active nodes exist on the current step */
  #mermaid-container .dimmed {
    opacity: 0.4;
    transition: opacity 0.35s ease;
  }

  /* Top actor box groups must stay opaque so the lifeline behind them doesn't show through.
     Use dimmed-actor instead of dimmed: keeps group opacity at 1 but darkens fill/text. */
  #mermaid-container .dimmed-actor rect.actor {
    fill: var(--dim-fill) !important;
    stroke-opacity: 0.2 !important;
    transition: fill 0.35s ease;
  }
  #mermaid-container .dimmed-actor text.actor {
    opacity: 0.15;
    transition: opacity 0.35s ease;
  }

  #mermaid-container .highlighted rect,
  #mermaid-container .highlighted circle,
  #mermaid-container .highlighted polygon,
  #mermaid-container .highlighted ellipse,
  #mermaid-container .highlighted path {
    fill: var(--highlight-fill) !important;
    stroke: var(--accent) !important;
    stroke-width: 2px !important;
    filter: drop-shadow(0 0 8px var(--accent-glow));
  }

  #mermaid-container .active rect,
  #mermaid-container .active circle,
  #mermaid-container .active polygon,
  #mermaid-container .active ellipse,
  #mermaid-container .active path {
    fill: var(--focus-fill) !important;
    stroke: var(--success) !important;
    stroke-width: 2.5px !important;
    filter: drop-shadow(0 0 12px var(--focus-glow));
  }

  #mermaid-container .flowchart-link.animated,
  #mermaid-container .messageLine0.animated,
  #mermaid-container .messageLine1.animated {
    stroke: var(--accent) !important;
    stroke-width: 2px !important;
    stroke-dasharray: 8 4;
    animation: flowEdge 0.8s linear infinite;
  }

  @keyframes flowEdge {
    from { stroke-dashoffset: 24; }
    to   { stroke-dashoffset: 0;  }
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
  <h1 class="page-title">[[.Title]]<span id="section-title-sep" class="page-title-sep" style="display:none">|</span><span id="section-title" class="page-section-title"></span></h1>
  <a class="ariel-link" href="[[.GitHubURL]]" target="_blank" rel="noopener">
    <span class="ariel-logo">[[.LogoSVG]]</span>
  </a>
</header>

<div class="main">
  <div class="diagram-pane">
    <div id="mermaid-container"></div>
  </div>

  <div class="side-pane">
    <div class="narration-area">
      <div class="step-label" id="step-label"></div>
      <div class="narration-text" id="narration"></div>
      <div class="section-track" id="section-track"></div>
      <div class="progress-track" id="progress-track"></div>
    </div>
    <div class="controls">
      <button class="btn-prev" id="btn-prev" onclick="prevStep()" disabled>← Back</button>
      <button class="btn-next" id="btn-next" onclick="nextStep()"></button>
    </div>
  </div>
</div>

<div id="ariel-ready" style="display:none"></div>
<script>
const sections = [[.SectionsJSON]];

let nodeMap = {};   // id → [SVGElement, ...] — all SVG groups for this node (seq has top+bottom)
let edgeMap = {};
let nodeSteps = {}; // node ID → sorted list of step indices that reference it
let currentSection = 0;
let currentStep = 0;
let initialized = false;
let diagramNaturalW = 0; // natural pixel width of the current diagram SVG; captured before any transform

[[.MermaidConfigJS]]
mermaid.initialize(arielMermaidConfig());

// reapplyTheme re-initializes Mermaid with the current OS color scheme and
// re-renders the active section+step. Wired to a prefers-color-scheme listener
// only in auto mode; a no-op reference otherwise.
async function reapplyTheme() {
  mermaid.initialize(arielMermaidConfig());
  await initSection(currentSection);
  renderStep();
}

function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }


async function initSection(idx) {
  const sec = sections[idx];
  const container = document.getElementById('mermaid-container');
  container.innerHTML = '<div class="mermaid">' + sec.mermaid_diagram + '</div>';
  await mermaid.run({ nodes: [container.querySelector('.mermaid')] });
  // viewBox.baseVal.width is the true natural coordinate width mermaid always sets.
  // style.maxWidth may be "100%" (→ parseFloat gives 100, not the pixel width), so we
  // avoid it. getBoundingClientRect reflects CSS layout, not the intrinsic SVG size.
  const freshSvg = container.querySelector('svg');
  diagramNaturalW = freshSvg ? (freshSvg.viewBox.baseVal.width || 0) : 0;
  nodeMap = {};
  edgeMap = {};
  nodeSteps = {};
  buildNodeMap();
  buildNodeSteps();
}

function sectionIndexFromHash() {
  const slug = window.location.hash.slice(1);
  if (!slug) return 0;
  for (let i = 0; i < sections.length; i++) {
    if (sectionSlug(i) === slug) return i;
  }
  return 0;
}

const startSection = sectionIndexFromHash();
currentSection = startSection;
initSection(startSection).then(() => {
  buildSectionDots();
  buildProgressDots();
  updateHeaderSectionTitle();
  renderStep();
  updateHash(startSection);
  initialized = true;
  document.getElementById('ariel-ready').style.display = 'block';
});

[[.ThemeListener]]

function buildNodeMap() {
  const svg = document.querySelector('#mermaid-container svg');
  if (!svg) return;

  const labels = sections[currentSection].node_labels || {};
  // Invert: display label → node ID. First occurrence wins (labels should be unique).
  const labelToId = {};
  for (const [id, label] of Object.entries(labels)) {
    const key = (label && label.trim()) ? label.trim() : id;
    if (!(key in labelToId)) labelToId[key] = id;
  }

  function addGroup(id, group) {
    if (!nodeMap[id]) nodeMap[id] = [];
    nodeMap[id].push(group);
  }

  // normalize collapses runs of whitespace (handles multi-tspan SVG text and foreignObject HTML).
  function normalize(text) { return text.replace(/\s+/g, ' ').trim(); }

  // Strategy 1: flowchart — Mermaid 10.6.1 sets id="flowchart-{nodeId}-{n}" on .node groups.
  svg.querySelectorAll('.node').forEach(group => {
    const m = group.id.match(/^flowchart-(\w+)-\d+$/);
    if (m) addGroup(m[1], group);
  });

  // Strategy 2: sequence diagram — match all g.actor groups (top AND bottom mirrors) by text.
  // Uses textContent to cover both SVG <text> elements and HTML inside <foreignObject>.
  svg.querySelectorAll('g.actor').forEach(group => {
    const label = normalize(group.textContent);
    const id = labelToId[label];
    if (id) addGroup(id, group);
  });

  // Strategy 3: generic fallback — for any diagram type not covered above.
  // Only runs if some expected IDs remain unmapped after strategies 1 and 2.
  if (Object.keys(labels).some(id => !nodeMap[id])) {
    const alreadyMapped = new Set(Object.values(nodeMap).flat());
    // Skip groups nested inside an already-mapped group: sequence diagram outer
    // lifeline+actor groups and their inner actor box sub-groups share the same
    // textContent; mapping both causes opacity double-multiplication (0.4 × 0.4).
    function isDescendantOfMapped(el) {
      let p = el.parentElement;
      while (p && p !== svg) {
        if (alreadyMapped.has(p)) return true;
        p = p.parentElement;
      }
      return false;
    }
    svg.querySelectorAll('g').forEach(group => {
      if (alreadyMapped.has(group) || isDescendantOfMapped(group)) return;
      const label = normalize(group.textContent);
      const id = labelToId[label];
      if (id) { addGroup(id, group); alreadyMapped.add(group); }
    });
  }

  // Sequence diagram z-order fix: Mermaid renders top actor box groups before lifelines in
  // DOM order, causing lifelines to paint over actor boxes. Moving top actor boxes to the
  // SVG end fixes stacking. In some Mermaid versions class="actor" is on <rect>/<text>
  // children, not on the <g> wrapper, so we detect by presence of rect.actor without <line>.
  {
    const svgKids = Array.from(svg.children);
    const firstLifelineIdx = svgKids.findIndex(el =>
      el.tagName && el.tagName.toLowerCase() === 'g' && (
        el.classList.contains('actor-line') ||
        el.querySelector('line:not(.messageLine0):not(.messageLine1)')
      )
    );
    if (firstLifelineIdx > 0) {
      const toMove = svgKids.slice(0, firstLifelineIdx).filter(el =>
        el.tagName && el.tagName.toLowerCase() === 'g' &&
        el.querySelector('rect.actor, text.actor')
      );
      toMove.forEach(el => svg.appendChild(el));
    }
  }

  // Flowchart edge map — Mermaid labels edges with LS-{src} and LE-{dst} classes.
  svg.querySelectorAll('.flowchart-link').forEach(el => {
    const cls = Array.from(el.classList);
    const srcCls = cls.find(c => c.startsWith('LS-'));
    const dstCls = cls.find(c => c.startsWith('LE-'));
    if (!srcCls || !dstCls) return;
    const key = srcCls.slice(3) + '-' + dstCls.slice(3);
    if (!edgeMap[key]) edgeMap[key] = [];
    edgeMap[key].push(el);
  });

  // Sequence edge map — message lines have no participant-based selectors, so infer
  // source/target by matching each line's endpoint x-coordinates to actor x-centers.
  const actorX = {};
  Object.entries(nodeMap).forEach(([id, els]) => {
    try {
      const b = els[0].getBBox();
      actorX[id] = b.x + b.width / 2;
    } catch (_) {}
  });
  const actorEntries = Object.entries(actorX);
  if (actorEntries.length > 0) {
    function closestActor(x) {
      let best = actorEntries[0][0], bestDist = Infinity;
      for (const [id, cx] of actorEntries) {
        const d = Math.abs(cx - x);
        if (d < bestDist) { bestDist = d; best = id; }
      }
      return best;
    }
    function lineEndpointX(el) {
      const tag = el.tagName.toLowerCase();
      if (tag === 'line') {
        return [parseFloat(el.getAttribute('x1')), parseFloat(el.getAttribute('x2'))];
      }
      if (tag === 'polyline') {
        const pts = (el.getAttribute('points') || '').trim().split(/[\s,]+/).map(Number);
        return pts.length >= 4 ? [pts[0], pts[pts.length - 2]] : null;
      }
      // path — extract all numbers from d, take first x and last x (second-to-last number).
      const nums = (el.getAttribute('d') || '').match(/-?[\d.]+/g);
      return nums && nums.length >= 4 ? [parseFloat(nums[0]), parseFloat(nums[nums.length - 2])] : null;
    }
    svg.querySelectorAll('.messageLine0, .messageLine1').forEach(el => {
      const xs = lineEndpointX(el);
      if (!xs) return;
      const src = closestActor(xs[0]);
      const dst = closestActor(xs[1]);
      if (src === dst) return; // skip self-messages
      const key = src + '-' + dst;
      if (!edgeMap[key]) edgeMap[key] = [];
      edgeMap[key].push(el);
    });
  }
}

function buildNodeSteps() {
  const steps = sections[currentSection].steps;
  steps.forEach((step, i) => {
    [...step.highlight_nodes, ...step.focus_nodes].forEach(id => {
      if (!nodeSteps[id]) nodeSteps[id] = [];
      if (!nodeSteps[id].includes(i)) nodeSteps[id].push(i);
    });
  });
  Object.keys(nodeMap).forEach(id => {
    if (nodeSteps[id]) {
      // Add navigable only to the first (top) group to avoid duplicate click areas.
      const el = nodeMap[id][0];
      el.classList.add('navigable');
      el.onclick = () => handleNodeClick(id);
    }
  });
}

function handleNodeClick(id) {
  const stepList = nodeSteps[id];
  if (!stepList) return;
  const next = stepList.find(i => i > currentStep);
  goToStep(next !== undefined ? next : stepList[0]);
}

function clearAllHighlights() {
  const svg = document.querySelector('#mermaid-container svg');
  if (!svg) return;
  svg.querySelectorAll('.highlighted, .active, .dimmed, .dimmed-actor').forEach(el => {
    el.classList.remove('highlighted', 'active', 'dimmed', 'dimmed-actor');
  });
  svg.querySelectorAll('.flowchart-link, .messageLine0, .messageLine1').forEach(e => e.classList.remove('animated'));
}

function applyStep(step) {
  clearAllHighlights();
  const activeSet = new Set([...step.highlight_nodes, ...step.focus_nodes]);
  if (activeSet.size === 0) return;

  const focusSet = new Set(step.focus_nodes);

  // Apply dimmed/highlighted/active to every SVG group for each node.
  // Top actor box groups (rect.actor present, no lifeline <line>) get dimmed-actor instead
  // of dimmed so their background rect stays opaque and blocks the lifeline behind them.
  Object.entries(nodeMap).forEach(([id, els]) => {
    const activeCls = focusSet.has(id) ? 'active' : 'highlighted';
    els.forEach(el => {
      if (!activeSet.has(id)) {
        const isTopActorBox = el.querySelector('rect.actor') && !el.querySelector('line');
        el.classList.add(isTopActorBox ? 'dimmed-actor' : 'dimmed');
      } else {
        el.classList.add(activeCls);
      }
    });
  });

  const allNodes = [...activeSet];
  for (let i = 0; i < allNodes.length; i++) {
    for (let j = 0; j < allNodes.length; j++) {
      if (i !== j) (edgeMap[allNodes[i] + '-' + allNodes[j]] || []).forEach(el => el.classList.add('animated'));
    }
  }
}

function buildProgressDots() {
  const track = document.getElementById('progress-track');
  track.innerHTML = '';
  sections[currentSection].steps.forEach((_, i) => {
    const dot = document.createElement('div');
    dot.className = i === 0 ? 'progress-dot intro-dot' : 'progress-dot';
    dot.onclick = () => goToStep(i);
    track.appendChild(dot);
  });
}

function updateProgressDots() {
  document.querySelectorAll('.progress-dot').forEach((dot, i) => {
    dot.className = i === 0 ? 'progress-dot intro-dot' : 'progress-dot';
    if (i < currentStep) dot.classList.add('visited');
    if (i === currentStep) dot.classList.add('active');
  });
}

function buildSectionDots() {
  const track = document.getElementById('section-track');
  if (!track) return;
  track.innerHTML = '';
  if (sections.length <= 1) { track.style.display = 'none'; return; }
  track.style.display = 'flex';
  sections.forEach((sec, i) => {
    const dot = document.createElement('div');
    dot.className = 'section-dot';
    dot.title = sec.title || ('Section ' + (i + 1));
    dot.onclick = () => goToSection(i);
    track.appendChild(dot);
  });
}

function updateSectionDots() {
  document.querySelectorAll('.section-dot').forEach((dot, i) => {
    dot.className = 'section-dot';
    if (i < currentSection) dot.classList.add('visited');
    if (i === currentSection) dot.classList.add('active');
  });
}

function sectionSlug(idx) {
  const title = sections[idx].title;
  if (title) return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  return 'section-' + idx;
}

function updateHash(idx) {
  const hash = '#' + sectionSlug(idx);
  history.replaceState(null, '', hash);
}

function goToStep(index) {
  currentStep = index;
  renderStep();
}

function updateHeaderSectionTitle() {
  const sep = document.getElementById('section-title-sep');
  const el = document.getElementById('section-title');
  if (sections.length <= 1) { sep.style.display = 'none'; el.textContent = ''; return; }
  const title = sections[currentSection].title || ('Section ' + (currentSection + 1));
  sep.style.display = '';
  el.textContent = title;
}

async function goToSection(idx, stepIdx) {
  const container = document.getElementById('mermaid-container');
  container.style.opacity = '0';
  document.getElementById('narration').classList.add('fade');
  await sleep(200);
  currentSection = idx;
  currentStep = stepIdx !== undefined ? stepIdx : 0;
  await initSection(idx);
  buildProgressDots();
  updateSectionDots();
  updateHeaderSectionTitle();
  renderStep();
  updateHash(idx);
  container.style.opacity = '1';
}

function renderStep() {
  const sec = sections[currentSection];
  const step = sec.steps[currentStep];
  const narrationEl = document.getElementById('narration');
  const labelEl = document.getElementById('step-label');

  narrationEl.classList.add('fade');
  setTimeout(() => {
    narrationEl.innerHTML = step.narration || '';
    let label;
    if (currentStep === 0) {
      label = (sections.length > 1 && sec.title) ? sec.title : (step.label || '');
    } else {
      label = currentStep + ' of ' + (sec.steps.length - 1);
      if (step.label) label += ' — ' + step.label;
      if (sections.length > 1 && sec.title) label = sec.title + ' · ' + label;
    }
    labelEl.textContent = label;
    narrationEl.classList.remove('fade');
  }, 200);

  if (initialized) applyStep(step);
  applyPanZoom(step);

  const isFirst = currentSection === 0 && currentStep === 0;
  const isLastStep = currentStep === sec.steps.length - 1;
  const isLastSection = currentSection === sections.length - 1;

  document.getElementById('btn-prev').disabled = isFirst;

  const nextBtn = document.getElementById('btn-next');
  if (isLastStep && isLastSection) {
    nextBtn.textContent = 'Done';
    nextBtn.disabled = true;
  } else {
    nextBtn.textContent = 'Next →';
    nextBtn.disabled = false;
  }

  updateProgressDots();
  updateSectionDots();
}

function nextStep() {
  const sec = sections[currentSection];
  if (currentStep < sec.steps.length - 1) {
    currentStep++;
    renderStep();
  } else if (currentSection < sections.length - 1) {
    goToSection(currentSection + 1);
  }
}

function prevStep() {
  if (currentStep > 0) {
    currentStep--;
    renderStep();
  } else if (currentSection > 0) {
    const prevIdx = currentSection - 1;
    goToSection(prevIdx, sections[prevIdx].steps.length - 1);
  }
}

document.addEventListener('keydown', e => {
  if (e.key === 'ArrowRight' || e.key === ' ') { e.preventDefault(); nextStep(); }
  if (e.key === 'ArrowLeft') { e.preventDefault(); prevStep(); }
});

function applyPanZoom(step) {
  const svg = document.querySelector('#mermaid-container svg');
  if (!svg || diagramNaturalW === 0) return;

  const container = document.getElementById('mermaid-container');
  const availW = container.clientWidth;
  const availH = container.clientHeight;
  if (availW === 0 || availH === 0) return;

  const vb = svg.viewBox.baseVal;
  if (!vb || vb.width === 0) return;

  // naturalH: diagram's natural pixel height, derived from viewBox aspect ratio.
  const naturalH = diagramNaturalW * vb.height / vb.width;
  const fits = diagramNaturalW <= availW && naturalH <= availH;
  const activeNodes = [...step.highlight_nodes, ...step.focus_nodes];

  if (fits) {
    // Diagram fits at natural scale: show at natural size, centered. No change between steps.
    svg.style.cssText = 'display:block;width:' + diagramNaturalW.toFixed(1) + 'px;height:' + naturalH.toFixed(1) + 'px;max-width:none;';
    return;
  }

  if (activeNodes.length === 0) {
    // Overflow diagram, overview step: scale to fit container with 10% padding per side.
    const scale = Math.min(availW * 0.8 / diagramNaturalW, availH * 0.8 / naturalH);
    const w = diagramNaturalW * scale;
    const h = naturalH * scale;
    svg.style.cssText = 'display:block;width:' + w.toFixed(1) + 'px;height:' + h.toFixed(1) + 'px;max-width:none;';
    return;
  }

  // Overflow diagram, highlight step: pan and zoom toward the highlighted nodes.
  // getCTM() returns coordinates in SVG viewport space, which is scaled by the current
  // viewBox-to-viewport ratio. Divide by currentScale to recover viewBox units.
  const currentRenderedW = svg.getBoundingClientRect().width;
  if (currentRenderedW === 0) return;
  const currentScale = currentRenderedW / vb.width;

  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  for (const id of activeNodes) {
    const groups = nodeMap[id];
    if (!groups) continue;
    for (const g of groups) {
      try {
        const lb = g.getBBox();
        const m = g.getCTM();
        if (!m) continue;
        const corners = [
          [lb.x, lb.y], [lb.x + lb.width, lb.y],
          [lb.x, lb.y + lb.height], [lb.x + lb.width, lb.y + lb.height]
        ];
        for (const [lx, ly] of corners) {
          // Transform local coords to SVG viewport coords via CTM.
          const vpx = m.a * lx + m.c * ly + m.e;
          const vpy = m.b * lx + m.d * ly + m.f;
          // Convert SVG viewport coords to viewBox units.
          const vbx = vpx / currentScale + vb.x;
          const vby = vpy / currentScale + vb.y;
          x0 = Math.min(x0, vbx); y0 = Math.min(y0, vby);
          x1 = Math.max(x1, vbx); y1 = Math.max(y1, vby);
        }
      } catch (_) {}
    }
  }

  if (x0 === Infinity) {
    // No bboxes found: fall back to overview scale-to-fit.
    const scale = Math.min(availW / diagramNaturalW, availH / naturalH);
    const w = diagramNaturalW * scale;
    const h = naturalH * scale;
    svg.style.cssText = 'display:block;width:' + w.toFixed(1) + 'px;height:' + h.toFixed(1) + 'px;max-width:none;';
    return;
  }

  const margin = 0.15;
  const paddedW = (x1 - x0) * (1 + margin);
  const paddedH = (y1 - y0) * (1 + margin);
  const cx = (x0 + x1) / 2;
  const cy = (y0 + y1) / 2;

  // CSS pixels per viewBox unit. Cap at natural scale (1.0) per spec.
  const naturalPxPerUnit = diagramNaturalW / vb.width;
  const pxPerUnit = Math.min(availW / paddedW, availH / paddedH, naturalPxPerUnit);

  const svgW = vb.width * pxPerUnit;
  const svgH = vb.height * pxPerUnit;

  // Map viewBox bbox center to CSS pixels within the SVG element.
  const cxPx = (cx - vb.x) / vb.width * svgW;
  const cyPx = (cy - vb.y) / vb.height * svgH;

  // Container centers SVG via flexbox. Translate so bbox center aligns with container center.
  // Flex places SVG top-left at ((availW-svgW)/2, (availH-svgH)/2). After translate(tx, ty),
  // bbox center lands at: (availW-svgW)/2 + tx + cxPx = availW/2  →  tx = svgW/2 - cxPx.
  const tx = svgW / 2 - cxPx;
  const ty = svgH / 2 - cyPx;

  svg.style.cssText = 'display:block;width:' + svgW.toFixed(1) + 'px;height:' + svgH.toFixed(1) + 'px;max-width:none;transform:translate(' + tx.toFixed(1) + 'px,' + ty.toFixed(1) + 'px);';
}
</script>
[[.WSSnippet]]</body>
</html>
`
