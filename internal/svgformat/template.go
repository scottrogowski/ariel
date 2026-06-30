package svgformat

import (
	"bytes"
	"text/template"
)

var extractionTmpl = template.Must(
	template.New("svg-extract").Delims("[[", "]]").Parse(extractionHTMLTemplate),
)

func renderExtractionHTML(mermaidDiagram string) string {
	var buf bytes.Buffer
	if err := extractionTmpl.Execute(&buf, struct{ MermaidDiagram string }{mermaidDiagram}); err != nil {
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

// applyStep sets visual state as inline styles so the extracted SVG is
// self-contained. When highlightNodes and focusNodes are both empty (step 0),
// the diagram is left as Mermaid rendered it.
function applyStep(highlightNodes, focusNodes) {
  const hasHighlights = highlightNodes.length > 0 || focusNodes.length > 0;
  if (!hasHighlights) return;

  const highlightSet = new Set(highlightNodes);
  const focusSet = new Set(focusNodes);
  const svg = document.querySelector('#mermaid-container svg');

  svg.querySelectorAll('.node').forEach(group => {
    const m = group.id.match(/^flowchart-(\w+)-\d+$/);
    const nodeId = m ? m[1] : null;

    if (focusSet.has(nodeId)) {
      group.style.opacity = '1';
      group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
        el.style.setProperty('fill', '#1a4a7a', 'important');
        el.style.setProperty('stroke', '#4ecdc4', 'important');
        el.style.setProperty('stroke-width', '2.5px', 'important');
      });
    } else if (highlightSet.has(nodeId)) {
      group.style.opacity = '1';
      group.querySelectorAll('rect,circle,polygon,ellipse,path').forEach(el => {
        el.style.setProperty('fill', '#1e3a6e', 'important');
        el.style.setProperty('stroke', '#5b8dee', 'important');
        el.style.setProperty('stroke-width', '2px', 'important');
      });
    } else {
      group.style.opacity = '0.25';
    }
  });

  const allActive = [...new Set([...highlightNodes, ...focusNodes])];
  for (let i = 0; i < allActive.length; i++) {
    for (let j = 0; j < allActive.length; j++) {
      if (i !== j) {
        (edgeMap[allActive[i] + '-' + allActive[j]] || []).forEach(g => {
          // Target path elements directly; animation must be on the element with
          // the stroke-dashoffset property, not the parent group.
          g.querySelectorAll('path').forEach(path => {
            path.style.setProperty('stroke', '#5b8dee', 'important');
            path.style.setProperty('stroke-width', '2.5px', 'important');
            path.style.setProperty('stroke-dasharray', '8 4', 'important');
            path.style.setProperty('animation', 'ariel-flow 0.6s linear infinite', 'important');
          });
        });
      }
    }
  }
}

function getSVG() {
  const svg = document.querySelector('#mermaid-container svg');
  // Inject @keyframes into the SVG so animated edges are self-contained when
  // the SVG string is embedded in the output file's foreignObject.
  const s = document.createElementNS('http://www.w3.org/2000/svg', 'style');
  s.textContent = '@keyframes ariel-flow{from{stroke-dashoffset:24}to{stroke-dashoffset:0}}';
  svg.insertBefore(s, svg.firstChild);
  return svg.outerHTML;
}

function getDimensions() {
  const svg = document.querySelector('#mermaid-container svg');
  const rect = svg.getBoundingClientRect();
  return JSON.stringify({w: Math.ceil(rect.width), h: Math.ceil(rect.height)});
}

init();
</script>
</body>
</html>
`
