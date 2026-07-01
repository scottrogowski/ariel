package svgformat

import (
	"bytes"
	"encoding/json"
	"text/template"
)

var extractionTmpl = template.Must(
	template.New("svg-extract").Delims("[[", "]]").Parse(extractionHTMLTemplate),
)

func renderExtractionHTML(mermaidDiagram string, nodeLabels map[string]string) string {
	labelsJSON, _ := json.Marshal(nodeLabels)
	var buf bytes.Buffer
	if err := extractionTmpl.Execute(&buf, struct {
		MermaidDiagram string
		NodeLabelsJSON string
	}{mermaidDiagram, string(labelsJSON)}); err != nil {
		panic("svgformat: extraction template: " + err.Error())
	}
	return buf.String()
}

// extractionHTMLTemplate renders a minimal headless page used to capture
// per-step SVG strings. The mermaid container is exactly outputWidth wide
// so extracted SVGs need no rescaling. Visual state (highlighting, dimming)
// is applied as inline styles so each extracted SVG is self-contained.
const extractionHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<script src="https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.6.1/mermaid.min.js"></script>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { background: #0f1117; }
  #mermaid-container { width: 900px; }
  /* Let the SVG render at its natural Mermaid size so getDimensions() returns the
     true natural width and height. The output SVG scales it up to 2× in CSS. */
  #mermaid-container svg { display: block; }
</style>
</head>
<body>
<div id="mermaid-container">
  <div class="mermaid">[[.MermaidDiagram]]</div>
</div>
<div id="ready" style="display:none"></div>
<script>
mermaid.initialize({
  startOnLoad: false,
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

let nodeMap = {}, edgeMap = {};
const nodeLabels = [[.NodeLabelsJSON]];

async function init() {
  await mermaid.run({ nodes: [document.querySelector('.mermaid')] });
  const svg = document.querySelector('#mermaid-container svg');
  buildNodeMap(svg);
  buildEdgeMap(svg);
  document.getElementById('ready').style.display = 'block';
}

function buildNodeMap(svg) {
  // Invert nodeLabels: display label → node ID.
  const labelToId = {};
  for (const [id, label] of Object.entries(nodeLabels)) {
    const key = (label && label.trim()) ? label.trim() : id;
    if (!(key in labelToId)) labelToId[key] = id;
  }
  function normalize(text) { return text.replace(/\s+/g, ' ').trim(); }
  function addGroup(id, group) {
    if (!nodeMap[id]) nodeMap[id] = [];
    nodeMap[id].push(group);
  }
  // Strategy 1: flowchart nodes.
  svg.querySelectorAll('.node').forEach(group => {
    const m = group.id.match(/^flowchart-(\w+)-\d+$/);
    if (m) addGroup(m[1], group);
  });
  // Strategy 2: sequence diagram actors (top and bottom mirrors).
  svg.querySelectorAll('g.actor').forEach(group => {
    const label = normalize(group.textContent);
    const id = labelToId[label];
    if (id) addGroup(id, group);
  });
  // Strategy 3: generic fallback for any remaining unmapped IDs.
  if (Object.keys(nodeLabels).some(id => !nodeMap[id])) {
    const alreadyMapped = new Set(Object.values(nodeMap).flat());
    svg.querySelectorAll('g').forEach(group => {
      if (alreadyMapped.has(group)) return;
      const label = normalize(group.textContent);
      const id = labelToId[label];
      if (id) { addGroup(id, group); alreadyMapped.add(group); }
    });
  }
  // Sequence diagram z-order fix: Mermaid renders top actor groups before lifelines in DOM
  // order, causing lifelines to paint over actor boxes. Moving top actors to SVG end fixes
  // stacking without changing their visual position (SVG uses coordinates, not flow).
  const firstActorLine = svg.querySelector('.actor-line');
  if (firstActorLine) {
    const toMove = [];
    let sibling = svg.firstElementChild;
    while (sibling && sibling !== firstActorLine) {
      if (sibling.classList && sibling.classList.contains('actor')) toMove.push(sibling);
      sibling = sibling.nextElementSibling;
    }
    toMove.forEach(el => svg.appendChild(el));
  }
}

function buildEdgeMap(svg) {
  // Flowchart edges — LS-{src}/LE-{dst} classes.
  svg.querySelectorAll('.flowchart-link').forEach(el => {
    const cls = Array.from(el.classList);
    const srcCls = cls.find(c => c.startsWith('LS-'));
    const dstCls = cls.find(c => c.startsWith('LE-'));
    if (!srcCls || !dstCls) return;
    const key = srcCls.slice(3) + '-' + dstCls.slice(3);
    if (!edgeMap[key]) edgeMap[key] = [];
    edgeMap[key].push(el);
  });
  // Sequence diagram edges — infer source/target from line endpoint x vs actor x-center.
  const actorX = {};
  Object.entries(nodeMap).forEach(([id, els]) => {
    try { const b = els[0].getBBox(); actorX[id] = b.x + b.width / 2; } catch (_) {}
  });
  const actorEntries = Object.entries(actorX);
  if (actorEntries.length > 0) {
    function closestActor(x) {
      let best = actorEntries[0][0], bestDist = Infinity;
      for (const [id, cx] of actorEntries) { const d = Math.abs(cx - x); if (d < bestDist) { bestDist = d; best = id; } }
      return best;
    }
    function lineEndpointX(el) {
      const tag = el.tagName.toLowerCase();
      if (tag === 'line') return [parseFloat(el.getAttribute('x1')), parseFloat(el.getAttribute('x2'))];
      if (tag === 'polyline') {
        const pts = (el.getAttribute('points') || '').trim().split(/[\s,]+/).map(Number);
        return pts.length >= 4 ? [pts[0], pts[pts.length - 2]] : null;
      }
      const nums = (el.getAttribute('d') || '').match(/-?[\d.]+/g);
      return nums && nums.length >= 4 ? [parseFloat(nums[0]), parseFloat(nums[nums.length - 2])] : null;
    }
    svg.querySelectorAll('.messageLine0, .messageLine1').forEach(el => {
      const xs = lineEndpointX(el);
      if (!xs) return;
      const src = closestActor(xs[0]), dst = closestActor(xs[1]);
      if (src === dst) return;
      const key = src + '-' + dst;
      if (!edgeMap[key]) edgeMap[key] = [];
      edgeMap[key].push(el);
    });
  }
}

// applyStep sets visual state as inline styles so the extracted SVG is
// self-contained. When highlightNodes and focusNodes are both empty (step 0),
// the diagram is left as Mermaid rendered it.
function applyStep(highlightNodes, focusNodes) {
  const hasHighlights = highlightNodes.length > 0 || focusNodes.length > 0;
  if (!hasHighlights) return;

  const activeSet = new Set([...highlightNodes, ...focusNodes]);
  const focusSet = new Set(focusNodes);

  // Apply highlight/dim to all known nodes via nodeMap (covers flowchart + sequence).
  Object.entries(nodeMap).forEach(([id, els]) => {
    els.forEach(group => {
      if (focusSet.has(id)) {
        group.style.opacity = '1';
        group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
          el.style.setProperty('fill', '#1a4a7a', 'important');
          el.style.setProperty('stroke', '#4ecdc4', 'important');
          el.style.setProperty('stroke-width', '2.5px', 'important');
        });
      } else if (activeSet.has(id)) {
        group.style.opacity = '1';
        group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
          el.style.setProperty('fill', '#1e3a6e', 'important');
          el.style.setProperty('stroke', '#5b8dee', 'important');
          el.style.setProperty('stroke-width', '2px', 'important');
        });
      } else {
        group.style.opacity = '0.4';
      }
    });
  });

  const allActive = [...activeSet];
  for (let i = 0; i < allActive.length; i++) {
    for (let j = 0; j < allActive.length; j++) {
      if (i !== j) {
        (edgeMap[allActive[i] + '-' + allActive[j]] || []).forEach(el => {
          // flowchart-link is on the <path> itself in Mermaid 10.6.1, not a wrapping <g>.
          // Sequence messageLine0/messageLine1 are line/polyline elements.
          const targets = el.tagName.toLowerCase() === 'path'
            ? [el]
            : (el.tagName.toLowerCase() === 'g' ? Array.from(el.querySelectorAll('path')) : [el]);
          targets.forEach(target => {
            target.style.setProperty('stroke', '#5b8dee', 'important');
            target.style.setProperty('stroke-width', '2.5px', 'important');
            target.style.setProperty('stroke-dasharray', '10 5', 'important');
            // SMIL animate survives GitHub's SVG sanitizer; CSS @keyframes do not.
            const anim = document.createElementNS('http://www.w3.org/2000/svg', 'animate');
            anim.setAttribute('attributeName', 'stroke-dashoffset');
            anim.setAttribute('from', '0');
            anim.setAttribute('to', '-15');
            anim.setAttribute('dur', '0.5s');
            anim.setAttribute('repeatCount', 'indefinite');
            target.appendChild(anim);
          });
        });
      }
    }
  }
}

function getSVG() {
  return document.querySelector('#mermaid-container svg').outerHTML;
}

function getDimensions() {
  const svg = document.querySelector('#mermaid-container svg');
  const rect = svg.getBoundingClientRect();
  // naturalW: Mermaid's own max-width (before any CSS override) — used to cap scale-up.
  const naturalW = parseFloat(svg.style.maxWidth) || Math.ceil(rect.width);
  return JSON.stringify({w: Math.ceil(rect.width), h: Math.ceil(rect.height), nw: Math.ceil(naturalW)});
}

init();
</script>
</body>
</html>
`
