package svgformat

import (
	"bytes"
	"encoding/json"
	"html"
	"text/template"

	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/renderadapter"
	"github.com/scottrogowski/ariel/internal/theme"
)

var extractionTmpl = template.Must(
	template.New("svg-extract").Delims("[[", "]]").Parse(extractionHTMLTemplate),
)

func renderExtractionHTML(p theme.Palette, mermaidDiagram string, analysis dsl.DiagramAnalysis) string {
	labelsJSON, _ := json.Marshal(analysis.Nodes)
	diagramKindJSON, _ := json.Marshal(analysis.Kind)
	var buf bytes.Buffer
	if err := extractionTmpl.Execute(&buf, struct {
		MermaidDiagram  string
		DiagramKindJSON string
		NodeLabelsJSON  string
		AnimateEdges    bool
		MermaidInit     string
		DiagramColorsJS string
		RenderAdapterJS string
		BodyBg          string
	}{
		MermaidDiagram:  html.EscapeString(mermaidDiagram),
		DiagramKindJSON: string(diagramKindJSON),
		NodeLabelsJSON:  string(labelsJSON),
		AnimateEdges:    analysis.AnimateEdges,
		MermaidInit:     p.MermaidInit(),
		DiagramColorsJS: p.DiagramColorsJS(),
		RenderAdapterJS: renderadapter.InlineJavaScript(),
		BodyBg:          p.Bg,
	}); err != nil {
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
  body { background: [[.BodyBg]]; }
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
[[.MermaidInit]]
[[.DiagramColorsJS]]
[[.RenderAdapterJS]]

let nodeMap = {}, edgeMap = {};
const diagramKind = [[.DiagramKindJSON]];
const nodeLabels = [[.NodeLabelsJSON]];
const animateEdges = [[.AnimateEdges]];

async function init() {
  await mermaid.run({ nodes: [document.querySelector('.mermaid')] });
  const svg = document.querySelector('#mermaid-container svg');
  // Force SVG to render at its natural Mermaid width so all getBoundingClientRect()
  // calls return coordinates in the natural pixel space.
  const naturalW = parseFloat(svg.style.maxWidth) || Math.ceil(svg.getBoundingClientRect().width);
  svg.style.width = naturalW + 'px';
  svg.style.maxWidth = 'none';
  const elementMap = buildArielElementMap(svg, diagramKind, nodeLabels);
  nodeMap = elementMap.nodeMap;
  edgeMap = elementMap.edgeMap;
  svg.querySelectorAll('marker path, marker polygon').forEach(element => {
    element.style.setProperty('fill', ARIEL_COLORS.arrowHead, 'important');
    element.style.setProperty('stroke', ARIEL_COLORS.arrowHead, 'important');
  });
  document.getElementById('ready').style.display = 'block';
}

// applyStep sets visual state as inline styles so the extracted SVG is
// self-contained. When highlightNodes and focusNodes are both empty (step 0),
// the diagram is left as Mermaid rendered it.
function applyStep(highlightNodes, focusNodes) {
  const hasHighlights = highlightNodes.length > 0 || focusNodes.length > 0;
  if (!hasHighlights) return;

  const activeSet = new Set([...highlightNodes, ...focusNodes]);
  const focusSet = new Set(focusNodes);

  // Apply emphasis to mapped nodes for every supported diagram type.
  Object.entries(nodeMap).forEach(([id, els]) => {
    els.forEach(group => {
      if (focusSet.has(id)) {
        group.style.opacity = '1';
        group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
          el.style.setProperty('fill', ARIEL_COLORS.focusFill, 'important');
          el.style.setProperty('stroke', ARIEL_COLORS.focusStroke, 'important');
          el.style.setProperty('stroke-width', '2.5px', 'important');
        });
      } else if (activeSet.has(id)) {
        group.style.opacity = '1';
        group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
          el.style.setProperty('fill', ARIEL_COLORS.highlightFill, 'important');
          el.style.setProperty('stroke', ARIEL_COLORS.highlightStroke, 'important');
          el.style.setProperty('stroke-width', '2px', 'important');
        });
      } else {
        // Top actor box groups (contain rect.actor but no lifeline <line>) must stay
        // opaque so the lifeline behind them doesn't show through. Dim via fill/text
        // instead of group opacity. Other groups (lifeline+bottom, flowchart nodes) dim
        // normally with opacity.
        const isTopActorBox = !!group.querySelector('rect.actor') && !group.querySelector('line');
        if (isTopActorBox) {
          group.style.opacity = '1';
          group.querySelectorAll('rect.actor').forEach(el => {
            el.style.setProperty('fill', ARIEL_COLORS.dimFill, 'important');
            el.style.setProperty('stroke-opacity', ARIEL_COLORS.dimBorderOpacity, 'important');
          });
          group.querySelectorAll('text.actor').forEach(el => {
            el.style.setProperty('opacity', ARIEL_COLORS.dimTextOpacity, 'important');
          });
        } else {
          group.style.opacity = ARIEL_COLORS.dimOpacity;
        }
      }
    });
  });

  if (!animateEdges) return;
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
            target.style.setProperty('stroke', ARIEL_COLORS.edgeStroke, 'important');
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

function getNodeBBoxes(nodeIds) {
  const svgEl = document.querySelector('#mermaid-container svg');
  const svgRect = svgEl.getBoundingClientRect();
  const result = {};
  for (const id of nodeIds) {
    const groups = nodeMap[id];
    if (!groups || groups.length === 0) continue;
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
    for (const g of groups) {
      const r = g.getBoundingClientRect();
      x0 = Math.min(x0, r.left - svgRect.left);
      y0 = Math.min(y0, r.top - svgRect.top);
      x1 = Math.max(x1, r.right - svgRect.left);
      y1 = Math.max(y1, r.bottom - svgRect.top);
    }
    if (x0 < Infinity) result[id] = {x: x0, y: y0, w: x1 - x0, h: y1 - y0};
  }
  return JSON.stringify(result);
}

function getDimensions() {
  const svg = document.querySelector('#mermaid-container svg');
  const rect = svg.getBoundingClientRect();
  const w = Math.ceil(rect.width);
  return JSON.stringify({w, h: Math.ceil(rect.height), nw: w});
}

init().catch(error => {
  window.arielInitError = error && (error.str || error.message) ? (error.str || error.message) : String(error);
  document.getElementById('ready').style.display = 'block';
});
</script>
</body>
</html>
`
