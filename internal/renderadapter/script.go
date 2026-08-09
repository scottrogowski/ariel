package renderadapter

import "strings"

// InlineJavaScript removes line breaks when the adapter is embedded in generated artifacts.
func InlineJavaScript() string {
	return strings.ReplaceAll(strings.TrimSpace(JavaScript), "\n", " ")
}

// JavaScript maps Mermaid's rendered SVG elements to Ariel node and edge identifiers.
const JavaScript = `
function buildArielElementMap(svg, diagramKind, nodeLabels) {
  const nodeMap = {};
  const edgeMap = {};
  const addNode = (id, element) => arielAddNode(nodeMap, id, element);
  const addEdge = (source, target, element) => arielAddEdge(edgeMap, source, target, element);

  if (diagramKind === 'flowchart') arielMapFlowchartNodes(svg, addNode);
  if (diagramKind === 'sequence') arielMapSequenceNodes(svg, nodeLabels, addNode);
  if (diagramKind === 'class') arielMapClassNodes(svg, nodeLabels, addNode);
  arielMapRemainingNodesByLabel(svg, nodeLabels, nodeMap, addNode);

  if (diagramKind === 'sequence') arielFixSequenceZOrder(svg);
  if (diagramKind === 'flowchart') arielMapFlowchartEdges(svg, addEdge);
  if (diagramKind === 'sequence') arielMapSequenceEdges(svg, nodeMap, addEdge);
  if (diagramKind === 'class') arielMapClassEdges(svg, nodeMap, addEdge);
  return {nodeMap, edgeMap};
}

function arielAddNode(nodeMap, id, element) {
  if (!id || !element) return;
  if (!nodeMap[id]) nodeMap[id] = [];
  if (nodeMap[id].includes(element)) return;
  nodeMap[id].push(element);
  element.setAttribute('data-ariel-node-id', id);
}

function arielAddEdge(edgeMap, source, target, element) {
  if (!source || !target || !element || source === target) return;
  const key = source + '-' + target;
  if (!edgeMap[key]) edgeMap[key] = [];
  edgeMap[key].push(element);
  element.setAttribute('data-ariel-edge-source', source);
  element.setAttribute('data-ariel-edge-target', target);
}

function arielMapFlowchartNodes(svg, addNode) {
  svg.querySelectorAll('.node').forEach(group => {
    const match = group.id.match(/^flowchart-(\w+)-\d+$/);
    if (match) addNode(match[1], group);
  });
}

function arielMapClassNodes(svg, nodeLabels, addNode) {
  const knownIds = new Set(Object.keys(nodeLabels));
  svg.querySelectorAll('g[id^="classId-"]').forEach(group => {
    const id = group.id.slice('classId-'.length).replace(/-\d+$/, '');
    if (knownIds.has(id)) addNode(id, group);
  });
}

function arielMapSequenceNodes(svg, nodeLabels, addNode) {
  const labelToId = arielLabelToId(nodeLabels);
  svg.querySelectorAll('g.actor').forEach(group => {
    const id = labelToId[arielNormalizeText(group.textContent)];
    if (id) addNode(id, group);
  });
}

function arielLabelToId(nodeLabels) {
  const labelToId = {};
  for (const [id, label] of Object.entries(nodeLabels)) {
    const key = arielNormalizeText(label || id);
    if (!(key in labelToId)) labelToId[key] = id;
  }
  return labelToId;
}

function arielNormalizeText(text) {
  return text.replace(/\s+/g, ' ').trim();
}

function arielMapRemainingNodesByLabel(svg, nodeLabels, nodeMap, addNode) {
  if (!Object.keys(nodeLabels).some(id => !nodeMap[id])) return;
  const labelToId = arielLabelToId(nodeLabels);
  const mapped = new Set(Object.values(nodeMap).flat());
  svg.querySelectorAll('g').forEach(group => {
    if (mapped.has(group) || arielHasMappedAncestor(group, svg, mapped)) return;
    const id = labelToId[arielNormalizeText(group.textContent)];
    if (!id || nodeMap[id]) return;
    addNode(id, group);
    mapped.add(group);
  });
}

function arielHasMappedAncestor(element, svg, mapped) {
  let parent = element.parentElement;
  while (parent && parent !== svg) {
    if (mapped.has(parent)) return true;
    parent = parent.parentElement;
  }
  return false;
}

function arielFixSequenceZOrder(svg) {
  const children = Array.from(svg.children);
  const lifelineIndex = children.findIndex(element =>
    element.tagName && element.tagName.toLowerCase() === 'g' && (
      element.classList.contains('actor-line') ||
      element.querySelector('line:not(.messageLine0):not(.messageLine1)')
    )
  );
  if (lifelineIndex <= 0) return;
  children.slice(0, lifelineIndex)
    .filter(element => element.tagName && element.tagName.toLowerCase() === 'g')
    .filter(element => element.querySelector('rect.actor, text.actor'))
    .forEach(element => svg.appendChild(element));
}

function arielMapFlowchartEdges(svg, addEdge) {
  svg.querySelectorAll('.flowchart-link').forEach(element => {
    const classes = Array.from(element.classList);
    const sourceClass = classes.find(name => name.startsWith('LS-'));
    const targetClass = classes.find(name => name.startsWith('LE-'));
    if (!sourceClass || !targetClass) return;
    addEdge(sourceClass.slice(3), targetClass.slice(3), element);
  });
}

function arielMapSequenceEdges(svg, nodeMap, addEdge) {
  const actorCenters = arielNodeCenters(nodeMap);
  svg.querySelectorAll('.messageLine0, .messageLine1').forEach(element => {
    const endpoints = arielLineEndpointX(element);
    if (!endpoints) return;
    addEdge(
      arielClosestNode(actorCenters, endpoints[0], 0),
      arielClosestNode(actorCenters, endpoints[1], 0),
      element
    );
  });
}

function arielLineEndpointX(element) {
  const tag = element.tagName.toLowerCase();
  if (tag === 'line') return [parseFloat(element.getAttribute('x1')), parseFloat(element.getAttribute('x2'))];
  if (tag === 'polyline') {
    const points = (element.getAttribute('points') || '').trim().split(/[\s,]+/).map(Number);
    return points.length >= 4 ? [points[0], points[points.length - 2]] : null;
  }
  const numbers = (element.getAttribute('d') || '').match(/-?[\d.]+/g);
  return numbers && numbers.length >= 4 ? [parseFloat(numbers[0]), parseFloat(numbers[numbers.length - 2])] : null;
}

function arielMapClassEdges(svg, nodeMap, addEdge) {
  const centers = arielNodeCenters(nodeMap);
  svg.querySelectorAll('path.relation').forEach(element => {
    const start = arielPathScreenPoint(element, 0);
    const end = arielPathScreenPoint(element, element.getTotalLength());
    if (!start || !end) return;
    addEdge(
      arielClosestNode(centers, start.x, start.y),
      arielClosestNode(centers, end.x, end.y),
      element
    );
  });
}

function arielNodeCenters(nodeMap) {
  const centers = {};
  for (const [id, elements] of Object.entries(nodeMap)) {
    const rect = elements[0].getBoundingClientRect();
    centers[id] = {x: rect.left + rect.width / 2, y: rect.top + rect.height / 2};
  }
  return centers;
}

function arielPathScreenPoint(path, length) {
  const matrix = path.getScreenCTM();
  if (!matrix) return null;
  const point = path.getPointAtLength(length);
  return {x: matrix.a * point.x + matrix.c * point.y + matrix.e, y: matrix.b * point.x + matrix.d * point.y + matrix.f};
}

function arielClosestNode(centers, x, y) {
  let closest = '';
  let closestDistance = Infinity;
  for (const [id, center] of Object.entries(centers)) {
    const distance = Math.hypot(center.x - x, center.y - y);
    if (distance >= closestDistance) continue;
    closest = id;
    closestDistance = distance;
  }
  return closest;
}
`
