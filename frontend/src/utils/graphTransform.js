import { getCoursePalette } from './colorMap'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from './courseMeta'

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

function hexToRgb(color = '#CBD5E1') {
  const normalized = String(color || '').trim()
  if (!normalized.startsWith('#')) return null

  let hex = normalized.slice(1)
  if (hex.length === 3) {
    hex = hex
      .split('')
      .map((char) => char + char)
      .join('')
  }
  if (hex.length !== 6) return null

  return {
    red: Number.parseInt(hex.slice(0, 2), 16),
    green: Number.parseInt(hex.slice(2, 4), 16),
    blue: Number.parseInt(hex.slice(4, 6), 16),
  }
}

function mixCourseColor(fromColor, toColor, ratio, alpha = 1) {
  const left = hexToRgb(fromColor)
  const right = hexToRgb(toColor)
  const t = clamp(Number(ratio || 0), 0, 1)

  if (!left || !right) return String(toColor || fromColor || '#CBD5E1')

  const red = Math.round(left.red + (right.red - left.red) * t)
  const green = Math.round(left.green + (right.green - left.green) * t)
  const blue = Math.round(left.blue + (right.blue - left.blue) * t)
  return `rgba(${red}, ${green}, ${blue}, ${clamp(alpha, 0, 1)})`
}

function getChapterTitle(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}

export function getNodeDisplayName(node) {
  return String(node?.canonicalName || node?.name || node?.routeKnowledgeId || node?.id || '未命名节点').trim()
}

function splitLabelLines(label = '', lineLength = 5, maxLines = 3) {
  const cleaned = String(label || '').replace(/\s+/g, ' ').trim()
  if (!cleaned) return ['未命名']

  const lines = []
  for (let index = 0; index < cleaned.length; index += lineLength) {
    if (lines.length >= maxLines) break
    lines.push(cleaned.slice(index, index + lineLength))
  }

  if (cleaned.length > lineLength * maxLines && lines.length) {
    const last = lines[lines.length - 1]
    lines[lines.length - 1] = `${last.slice(0, Math.max(1, last.length - 1))}…`
  }

  return lines
}

function estimateCollisionRadius(label = '', radius = 32, options = {}) {
  const isRoot = Boolean(options.isRoot)
  const isExpanded = Boolean(options.isExpanded)
  const fontSize = isRoot ? 13.8 : 10.9
  const lines = splitLabelLines(label, isRoot ? 6 : 5, 3)
  const widest = Math.max(...lines.map((line) => String(line || '').length), 1)
  const textWidth = widest * fontSize * 1.04
  const textHeight = lines.length * (fontSize + 5)
  const labelRadius = Math.hypot(textWidth * 0.58, textHeight * 0.68)
  const labelSpreadBoost = Math.min(
    isRoot ? 32 : 24,
    Math.max(0, widest - 3) * (isRoot ? 2.8 : 2.3) + Math.max(0, lines.length - 1) * 5,
  )
  const basePadding = isRoot ? (isExpanded ? 32 : 28) : isExpanded ? 24 : 21
  return Math.ceil(
    Math.max(radius + basePadding, labelRadius + radius * (isRoot ? 0.46 : 0.4) + labelSpreadBoost),
  )
}

function resolveProgressState(progress = {}) {
  const mastery = Number(progress?.masteryScore || 0)
  const answeredCount = Number(progress?.answeredCount || 0)
  if (mastery >= 80) return 'mastered'
  if (mastery < 30 && answeredCount > 0) return 'weak'
  if (mastery >= 30) return 'learning'
  return 'default'
}

function buildNodeColors(courseTitle, progress = {}) {
  const palette = getCoursePalette(courseTitle)
  const masteryRatio = clamp(Number(progress?.masteryScore || 0), 0, 100) / 100
  const easedRatio = masteryRatio ** 0.88

  return {
    fill: mixCourseColor(palette.light, palette.primary, easedRatio, 0.99),
    stroke: mixCourseColor(palette.ring, palette.primary, 0.28 + easedRatio * 0.5, 0.98),
    shadow: mixCourseColor(palette.ring, palette.primary, 0.14 + easedRatio * 0.36, 0.2),
    accent: palette.primary,
    ring: palette.ring,
    courseLight: palette.light,
    masteryRatio,
  }
}

export function getRelationStroke(relationType, theme = 'dark') {
  const isLight = theme === 'light'
  switch (relationType) {
    case 'PREREQUISITE_OF':
      return isLight ? '#64748B' : '#CBD5E1'
    case 'SUBTOPIC_OF':
      return isLight ? '#6366F1' : '#C7D2FE'
    case 'PART_OF':
      return isLight ? '#3B82F6' : '#BFDBFE'
    case 'RELATED_TO':
      return isLight ? '#94A3B8' : '#E2E8F0'
    case 'USES':
      return isLight ? '#2563EB' : '#93C5FD'
    case 'APPLIES_TO':
      return isLight ? '#0D9488' : '#6EE7B7'
    default:
      return isLight ? '#94A3B8' : '#CBD5E1'
  }
}

function buildNodeStyle(node, options) {
  const progress = options.progressMap?.[node.routeKnowledgeId] || {}
  const colors = buildNodeColors(getChapterTitle(node), progress)
  const isRoot = Boolean(node.isRoot)
  const isExpanded = options.expandedSet.has(node.id)
  const isPinned = options.pinnedSet.has(node.id)
  const isSelected = options.selectedNodeId === node.id
  const radius = isRoot ? 42 : 30
  const shadowBlur = isSelected ? 24 : isExpanded ? 18 : isRoot ? 14 : 9
  const borderWidth = isSelected ? 4 : isExpanded ? 3 : isPinned ? 2.4 : 2
  const theme = options.theme || 'dark'
  const isLight = theme === 'light'

  return {
    radius,
    collisionRadius: estimateCollisionRadius(getNodeDisplayName(node), radius, {
      isRoot,
      isExpanded,
    }),
    fill: colors.fill,
    stroke: isSelected || isExpanded 
      ? (isLight ? 'rgba(79, 70, 229, 1)' : 'rgba(103, 232, 249, 1)') 
      : colors.stroke,
    borderWidth,
    shadowBlur,
    shadowColor: isSelected 
      ? (isLight ? 'rgba(79, 70, 229, 0.25)' : 'rgba(103, 232, 249, 0.42)') 
      : colors.shadow,
    badgeColor: colors.accent,
    ringColor: colors.ring,
    courseLight: colors.courseLight,
    masteryRatio: colors.masteryRatio,
    labelColor: isLight ? '#0F172A' : '#F8FAFC',
    labelStroke: isLight ? 'rgba(255, 255, 255, 0.95)' : 'rgba(15, 23, 42, 0.88)',
    opacity: 1,
    fontSize: isRoot ? 14 : 11,
  }
}

function normalizeGraphNode(node, options) {
  const label = getNodeDisplayName(node)
  const style = buildNodeStyle(node, options)
  return {
    id: node.id,
    label,
    x: Number.isFinite(node.x) ? node.x : 0,
    y: Number.isFinite(node.y) ? node.y : 0,
    fx: Number.isFinite(node.fx) ? node.fx : null,
    fy: Number.isFinite(node.fy) ? node.fy : null,
    style,
    data: {
      routeKnowledgeId: node.routeKnowledgeId,
      chapterTitle: getChapterTitle(node),
      entityType: node.entityType || 'concept',
      description: node.description || '',
      canonicalName: node.canonicalName || '',
      isRoot: Boolean(node.isRoot),
      isExpanded: options.expandedSet.has(node.id),
      isPinned: options.pinnedSet.has(node.id),
      rootId: node.rootId || node.id,
      layoutDepth: Number(node.depth || 0),
      expandMeta: node.expandMeta || null,
      layoutLock: node.layoutLock || null,
      synthetic: false,
    },
  }
}

function normalizeGraphEdge(edge, nodeMap, options = {}) {
  const theme = options.theme || 'dark'
  const relation = RELATION_CONFIG[edge.relationType] || RELATION_CONFIG.RELATED_TO
  const sourceNode = nodeMap.get(edge.sourceId)
  const targetNode = nodeMap.get(edge.targetId)
  const sourceCourse = getChapterTitle(sourceNode)
  const targetCourse = getChapterTitle(targetNode)
  const sourcePalette = getCoursePalette(sourceCourse)
  const targetPalette = getCoursePalette(targetCourse)
  return {
    id: edge.id,
    source: edge.sourceId,
    target: edge.targetId,
    data: {
      label: relation.label,
      relationType: edge.relationType,
      sourceName: edge.sourceName || '',
      targetName: edge.targetName || '',
      sourceCourse,
      targetCourse,
      crossCourse: sourceCourse !== targetCourse,
    },
    style: {
      stroke: getRelationStroke(edge.relationType, theme),
      lineWidth: relation.lineWidth,
      lineDash: relation.lineDash,
      endArrow: relation.endArrow,
      opacity: relation.opacity,
      sourceRing: sourcePalette.ring,
      targetRing: targetPalette.ring,
      sourceLight: sourcePalette.light,
      targetLight: targetPalette.light,
    },
  }
}

export function transformGraphData(nodes = [], edges = [], options = {}) {
  const expandedSet = new Set(options.expandedNodeIds || [])
  const pinnedSet = new Set(options.pinnedNodeIds || [])
  const nodeMap = new Map(nodes.map((node) => [node.id, node]))

  const visibleNodes = [...nodes]
    .sort((left, right) => {
      const rootDelta = Number(right.isRoot) - Number(left.isRoot)
      if (rootDelta !== 0) return rootDelta
      const depthDelta = Number(left.depth || 0) - Number(right.depth || 0)
      if (depthDelta !== 0) return depthDelta
      return getNodeDisplayName(left).localeCompare(getNodeDisplayName(right), 'zh-CN')
    })
    .map((node) =>
      normalizeGraphNode(node, {
        progressMap: options.progressMap || {},
        expandedSet,
        pinnedSet,
        selectedNodeId: options.selectedNodeId || null,
        theme: options.theme || 'dark',
      }),
    )

  const visibleEdges = edges
    .filter((edge) => nodeMap.has(edge.sourceId) && nodeMap.has(edge.targetId))
    .map((edge) => normalizeGraphEdge(edge, nodeMap, options))

  return {
    nodes: visibleNodes,
    edges: visibleEdges,
    meta: {
      totalNodes: Number(options.totalNodes || visibleNodes.length),
      visibleNodes: visibleNodes.length,
      visibleEdges: visibleEdges.length,
      rootCount: visibleNodes.filter((node) => node.data?.isRoot).length,
      expandedCount: expandedSet.size,
      rootNodeIds: [...(options.rootNodeIds || [])],
    },
  }
}

export { clamp }
