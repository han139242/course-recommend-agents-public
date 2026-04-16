import { getCoursePalette } from './colorMap'

export const RELATION_CONFIG = {
  PREREQUISITE_OF: {
    label: '前置知识',
    stroke: '#CBD5E1',
    lineWidth: 2.1,
    lineDash: [],
    endArrow: true,
    opacity: 0.9,
    weight: 3.2,
  },
  SUBTOPIC_OF: {
    label: '属于',
    stroke: '#C7D2FE',
    lineWidth: 1.7,
    lineDash: [6, 3],
    endArrow: true,
    opacity: 0.78,
    weight: 2.4,
  },
  PART_OF: {
    label: '组成',
    stroke: '#BFDBFE',
    lineWidth: 1.7,
    lineDash: [8, 4],
    endArrow: true,
    opacity: 0.78,
    weight: 2.2,
  },
  RELATED_TO: {
    label: '相关',
    stroke: '#E2E8F0',
    lineWidth: 1.5,
    lineDash: [],
    endArrow: true,
    opacity: 0.82,
    weight: 1.6,
  },
  USES: {
    label: '使用',
    stroke: '#93C5FD',
    lineWidth: 1.45,
    lineDash: [4, 4],
    endArrow: true,
    opacity: 0.76,
    weight: 1.6,
  },
  APPLIES_TO: {
    label: '应用到',
    stroke: '#6EE7B7',
    lineWidth: 1.45,
    lineDash: [6, 4],
    endArrow: true,
    opacity: 0.76,
    weight: 1.6,
  },
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function sortText(left = '', right = '') {
  return String(left || '').localeCompare(String(right || ''), 'zh-CN')
}

function getChapterTitle(node) {
  return String(node?.chapterTitle || '其他').trim() || '其他'
}

export function getNodeDisplayName(node) {
  return String(node?.canonicalName || node?.name || node?.routeKnowledgeId || node?.id || '未命名节点').trim()
}

function ensureBucket(map, key, factory) {
  if (!map.has(key)) map.set(key, factory())
  return map.get(key)
}

function getNodeVisualState(progress = {}, isBookmarked = false) {
  const mastery = Number(progress?.masteryScore || 0)
  const answeredCount = Number(progress?.answeredCount || 0)
  const weak = mastery < 30 && answeredCount > 0
  const mastered = mastery >= 80
  const learning = mastery >= 40 && mastery < 80
  return { mastery, weak, mastered, learning, isBookmarked }
}

function buildGraphMetrics(nodes = [], edges = []) {
  const metrics = new Map(
    nodes.map((node) => [
      node.id,
      {
        degree: 0,
        weightedDegree: 0,
        inbound: 0,
        outbound: 0,
        crossCourse: 0,
      },
    ]),
  )

  const nodeChapterMap = new Map(nodes.map((node) => [node.id, getChapterTitle(node)]))

  edges.forEach((edge) => {
    const relation = RELATION_CONFIG[edge.relationType] || RELATION_CONFIG.RELATED_TO
    const sourceMetrics = metrics.get(edge.sourceId)
    const targetMetrics = metrics.get(edge.targetId)
    const crossCourse = nodeChapterMap.get(edge.sourceId) !== nodeChapterMap.get(edge.targetId)

    if (sourceMetrics) {
      sourceMetrics.degree += 1
      sourceMetrics.weightedDegree += relation.weight || 1
      sourceMetrics.outbound += 1
      if (crossCourse) sourceMetrics.crossCourse += 1
    }
    if (targetMetrics) {
      targetMetrics.degree += 1
      targetMetrics.weightedDegree += relation.weight || 1
      targetMetrics.inbound += 1
      if (crossCourse) targetMetrics.crossCourse += 1
    }
  })

  return metrics
}

function buildAdjacency(nodes = [], edges = []) {
  const adjacency = new Map(nodes.map((node) => [node.id, []]))
  edges.forEach((edge) => {
    const relation = RELATION_CONFIG[edge.relationType] || RELATION_CONFIG.RELATED_TO
    const edgeId = edge.relationId || `${edge.sourceId}-${edge.relationType || 'related'}-${edge.targetId}`
    const edgeMeta = {
      id: edgeId,
      sourceId: edge.sourceId,
      targetId: edge.targetId,
      relationType: edge.relationType,
      relation,
    }
    ensureBucket(adjacency, edge.sourceId, () => []).push({ ...edgeMeta, neighborId: edge.targetId, direction: 'out' })
    ensureBucket(adjacency, edge.targetId, () => []).push({ ...edgeMeta, neighborId: edge.sourceId, direction: 'in' })
  })
  adjacency.forEach((neighbors) => {
    neighbors.sort((left, right) => {
      if ((right.relation.weight || 1) !== (left.relation.weight || 1)) return (right.relation.weight || 1) - (left.relation.weight || 1)
      return sortText(left.neighborId, right.neighborId)
    })
  })
  return adjacency
}

function scoreNode(node, metrics) {
  const label = getNodeDisplayName(node)
  const meta = metrics.get(node.id) || { degree: 0, weightedDegree: 0, crossCourse: 0, outbound: 0 }
  const keywordBoost = /基础|导论|总览|原理|核心|框架|模型|方法/.test(label) ? 1.3 : 0
  return meta.weightedDegree * 2.2 + meta.degree * 1.2 + meta.outbound * 0.7 + meta.crossCourse * 1.1 + keywordBoost
}

export function buildSeedNodeIds(nodes = [], edges = [], desiredCount = 14) {
  if (!nodes.length) return []

  const metrics = buildGraphMetrics(nodes, edges)
  const grouped = new Map()
  nodes.forEach((node) => {
    const course = getChapterTitle(node)
    ensureBucket(grouped, course, () => []).push(node)
  })

  const seeds = []
  const used = new Set()
  const sortedGroups = [...grouped.entries()].sort((left, right) => right[1].length - left[1].length)

  sortedGroups.forEach(([, groupNodes]) => {
    if (seeds.length >= Math.ceil(desiredCount * 0.7)) return
    const best = [...groupNodes].sort((left, right) => scoreNode(right, metrics) - scoreNode(left, metrics))[0]
    if (best && !used.has(best.id)) {
      used.add(best.id)
      seeds.push(best.id)
    }
  })

  const ranked = [...nodes].sort((left, right) => {
    const scoreDiff = scoreNode(right, metrics) - scoreNode(left, metrics)
    if (scoreDiff !== 0) return scoreDiff
    return sortText(getNodeDisplayName(left), getNodeDisplayName(right))
  })

  ranked.forEach((node) => {
    if (seeds.length >= desiredCount) return
    if (used.has(node.id)) return
    used.add(node.id)
    seeds.push(node.id)
  })

  return seeds.slice(0, desiredCount)
}

function buildVisibleNodeContext(nodes = [], edges = [], options = {}) {
  const {
    seedNodeIds = buildSeedNodeIds(nodes, edges),
    expandedNodeIds = [],
    pinnedNodeIds = [],
    selectedNodeId = null,
  } = options

  const nodeMap = new Map(nodes.map((node) => [node.id, node]))
  const adjacency = buildAdjacency(nodes, edges)
  const rootIds = [...new Set(seedNodeIds)].filter((nodeId) => nodeMap.has(nodeId))
  const expandedIds = [...new Set(expandedNodeIds)].filter((nodeId) => nodeMap.has(nodeId))
  const pinnedIds = [...new Set(pinnedNodeIds)].filter((nodeId) => nodeMap.has(nodeId))
  const visible = new Set([...rootIds, ...expandedIds, ...pinnedIds])
  if (selectedNodeId && nodeMap.has(selectedNodeId)) visible.add(selectedNodeId)

  const anchorMap = new Map()
  rootIds.forEach((nodeId) => anchorMap.set(nodeId, { anchorId: null, depth: 0, rootId: nodeId, priority: -1 }))
  pinnedIds.forEach((nodeId) => {
    if (!anchorMap.has(nodeId)) anchorMap.set(nodeId, { anchorId: null, depth: 0, rootId: nodeId, priority: -0.5 })
  })
  if (selectedNodeId && nodeMap.has(selectedNodeId) && !anchorMap.has(selectedNodeId)) {
    anchorMap.set(selectedNodeId, { anchorId: null, depth: 0, rootId: selectedNodeId, priority: -0.25 })
  }

  expandedIds.forEach((nodeId, index) => {
    visible.add(nodeId)
    if (!anchorMap.has(nodeId)) {
      anchorMap.set(nodeId, { anchorId: null, depth: 0, rootId: nodeId, priority: index })
    }

    const parentContext = anchorMap.get(nodeId) || { rootId: nodeId, depth: 0 }
    ;(adjacency.get(nodeId) || []).forEach((neighbor) => {
      const neighborId = neighbor.neighborId
      visible.add(neighborId)
      const current = anchorMap.get(neighborId)
      if (!current || current.priority <= index) {
        anchorMap.set(neighborId, {
          anchorId: nodeId,
          depth: parentContext.depth + 1,
          rootId: parentContext.rootId || nodeId,
          priority: index + 1,
        })
      }
    })
  })

  ;[...visible].forEach((nodeId) => {
    if (!anchorMap.has(nodeId)) {
      anchorMap.set(nodeId, { anchorId: null, depth: 0, rootId: nodeId, priority: -2 })
    }
  })

  return {
    nodeMap,
    adjacency,
    rootIds,
    expandedIds,
    pinnedIds,
    visibleNodeIds: [...visible],
    anchorMap,
  }
}

export function transformGraphData(nodes = [], edges = [], options = {}) {
  const {
    progressMap = {},
    bookmarks = new Set(),
    expandedNodeIds = [],
    pinnedNodeIds = [],
    selectedNodeId = null,
    seedNodeIds = buildSeedNodeIds(nodes, edges),
  } = options

  const metrics = buildGraphMetrics(nodes, edges)
  const nodeChapterMap = new Map(nodes.map((node) => [node.id, getChapterTitle(node)]))
  const {
    nodeMap,
    adjacency,
    rootIds,
    expandedIds,
    pinnedIds,
    visibleNodeIds,
    anchorMap,
  } = buildVisibleNodeContext(nodes, edges, {
    seedNodeIds,
    expandedNodeIds,
    pinnedNodeIds,
    selectedNodeId,
  })

  const visibleNodeIdSet = new Set(visibleNodeIds)
  const visibleNodes = visibleNodeIds
    .map((nodeId) => nodeMap.get(nodeId))
    .filter(Boolean)
    .sort((left, right) => {
      const leftAnchor = anchorMap.get(left.id) || { depth: 0 }
      const rightAnchor = anchorMap.get(right.id) || { depth: 0 }
      if (leftAnchor.depth !== rightAnchor.depth) return leftAnchor.depth - rightAnchor.depth
      return sortText(getNodeDisplayName(left), getNodeDisplayName(right))
    })

  const graphNodes = visibleNodes.map((node) => {
    const title = getChapterTitle(node)
    const palette = getCoursePalette(title)
    const progress = progressMap[node.routeKnowledgeId] || {}
    const visual = getNodeVisualState(progress, bookmarks.has(node.routeKnowledgeId))
    const meta = metrics.get(node.id) || { degree: 0, weightedDegree: 0 }
    const context = anchorMap.get(node.id) || { anchorId: null, depth: 0, rootId: node.id }
    const isRoot = rootIds.includes(node.id)
    const isExpanded = expandedIds.includes(node.id)
    const isPinned = pinnedIds.includes(node.id)
    const isSelected = selectedNodeId === node.id
    const label = getNodeDisplayName(node)
    const shortLabel = label.length > 10 ? `${label.slice(0, 10)}…` : label
    const size = clamp(isRoot ? 82 : 58 + Math.sqrt(meta.degree || 0) * 4.6, isRoot ? 78 : 54, isRoot ? 94 : 72)

    let fill = 'rgba(241, 234, 255, 0.99)'
    let stroke = 'rgba(196, 181, 253, 1)'
    let shadowColor = palette.light
    let shadowBlur = isRoot ? 12 : 6
    let shadowOpacity = isRoot ? 0.2 : 0.1

    if (visual.learning) {
      fill = 'rgba(230, 236, 255, 0.99)'
      stroke = 'rgba(129, 140, 248, 0.98)'
      shadowOpacity = 0.18
    }
    if (visual.weak) {
      fill = 'rgba(255, 228, 196, 0.99)'
      stroke = 'rgba(251, 146, 60, 0.98)'
      shadowColor = '#FB923C'
      shadowOpacity = 0.22
    }
    if (visual.mastered) {
      fill = 'rgba(255, 246, 214, 0.99)'
      stroke = 'rgba(250, 204, 21, 0.98)'
      shadowColor = '#FACC15'
      shadowBlur = 18
      shadowOpacity = 0.34
    }
    if (isExpanded || isSelected) {
      stroke = 'rgba(103, 232, 249, 1)'
      shadowColor = '#67E8F9'
      shadowBlur = 20
      shadowOpacity = 0.36
    }

    return {
      id: node.id,
      data: {
        label,
        fullLabel: label,
        shortLabel,
        routeKnowledgeId: node.routeKnowledgeId,
        chapterTitle: title,
        entityType: node.entityType,
        description: node.description,
        masteryScore: visual.mastery,
        answeredCount: Number(progress?.answeredCount || 0),
        weak: visual.weak,
        mastered: visual.mastered,
        learning: visual.learning,
        bookmarked: visual.isBookmarked,
        relationCount: meta.degree,
        importanceScore: meta.weightedDegree,
        size,
        textMaxWidth: Math.max(46, size * 0.88),
        shape: 'ellipse',
        isRoot,
        isExpanded,
        isPinned,
        anchorId: context.anchorId,
        layoutDepth: context.depth,
        rootId: context.rootId,
      },
        style: {
          size,
          fill,
          stroke,
          lineWidth: isExpanded || isSelected ? 2.8 : visual.mastered ? 2.2 : 1.4,
          shadowColor,
          shadowBlur,
          shadowOpacity,
          labelText: label,
          labelFill: '#111827',
          labelFontSize: isRoot ? 12.4 : 10.8,
          cursor: 'pointer',
        },
      }
  })

  const graphEdges = edges
    .filter((edge) => visibleNodeIdSet.has(edge.sourceId) && visibleNodeIdSet.has(edge.targetId))
    .map((edge) => {
      const relation = RELATION_CONFIG[edge.relationType] || RELATION_CONFIG.RELATED_TO
      const sourceChapter = nodeChapterMap.get(edge.sourceId)
      const targetChapter = nodeChapterMap.get(edge.targetId)
      const edgeId = edge.relationId || `${edge.sourceId}-${edge.relationType || 'related'}-${edge.targetId}`
      const isCrossCourse = Boolean(sourceChapter && targetChapter && sourceChapter !== targetChapter)

      return {
        id: edgeId,
        source: edge.sourceId,
        target: edge.targetId,
        data: {
          relationType: edge.relationType,
          label: relation.label,
          crossCourse: isCrossCourse,
          sourceChapter: sourceChapter || '',
          targetChapter: targetChapter || '',
        },
        style: {
          stroke: relation.stroke,
          lineWidth: relation.lineWidth,
          lineDash: relation.lineDash,
          opacity: relation.opacity,
          endArrow: relation.endArrow,
        },
      }
    })

  return {
    nodes: graphNodes,
    edges: graphEdges,
    meta: {
      totalNodes: nodes.length,
      totalEdges: edges.length,
      visibleNodes: graphNodes.length,
      visibleEdges: graphEdges.length,
      rootCount: rootIds.length,
      expandedCount: expandedIds.length,
      pinnedCount: pinnedIds.length,
      adjacency,
      seedNodeIds: rootIds,
    },
  }
}
