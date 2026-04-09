/**
 * 将 Neo4j 子图 API 返回的 nodes/edges 布局为 viewBox="0 0 100 100" 的径向图（与 GraphView 一致）。
 */
export function layoutNeoSubgraph(data, focusId) {
  const nodes = data?.nodes || []
  if (!nodes.length) {
    return { positionedNodes: [], edges: [], nodeMap: {} }
  }

  const root =
    data?.root ||
    (focusId ? nodes.find((n) => n.id === focusId) : null) ||
    nodes[0]

  const neighbors = nodes.filter((n) => n.id !== root.id)

  const positionedNodes = [
    {
      ...root,
      label: root.canonicalName || root.name || root.id,
      group: root.entityType,
      x: 50,
      y: 50,
      level: 'focus',
    },
    ...neighbors.map((node, index) => {
      const angle = -Math.PI / 2 + (index * Math.PI * 2) / Math.max(neighbors.length, 1)
      const radius = 31
      return {
        ...node,
        label: node.canonicalName || node.name || node.id,
        group: node.entityType,
        x: 50 + radius * Math.cos(angle),
        y: 50 + radius * Math.sin(angle),
        level:
          node.entityType === 'application'
            ? 'project'
            : node.entityType === 'topic'
              ? 'core'
              : 'base',
      }
    }),
  ]

  const nodeMap = Object.fromEntries(positionedNodes.map((n) => [n.id, n]))
  const edges = (data.edges || []).filter((e) => nodeMap[e.sourceId] && nodeMap[e.targetId])

  return { positionedNodes, edges, nodeMap }
}

export function edgeSvgCoords(edge, nodeMap) {
  const from = nodeMap[edge.sourceId]
  const to = nodeMap[edge.targetId]
  return {
    x1: from?.x ?? 50,
    y1: from?.y ?? 50,
    x2: to?.x ?? 50,
    y2: to?.y ?? 50,
  }
}

export function resolveKnowledgeRouteId(node) {
  return node?.routeKnowledgeId || node?.knowledgeId || node?.slug || node?.id || ''
}
