import { getRelationPriority } from './graphLayout'

const TEXT_WIDTH_CACHE_LIMIT = 12000
const textWidthCache = new Map()

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function measureTextWidthCached(ctx, text = '') {
  const key = `${String(ctx?.font || '')}::${String(text || '')}`
  if (textWidthCache.has(key)) {
    const cached = textWidthCache.get(key)
    textWidthCache.delete(key)
    textWidthCache.set(key, cached)
    return cached
  }

  const width = ctx.measureText(String(text || '')).width
  textWidthCache.set(key, width)
  if (textWidthCache.size > TEXT_WIDTH_CACHE_LIMIT) {
    const firstKey = textWidthCache.keys().next().value
    if (firstKey) textWidthCache.delete(firstKey)
  }
  return width
}

function clearTextWidthCache() {
  textWidthCache.clear()
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

function getStates(stateMap, id) {
  return new Set(stateMap?.[id] || [])
}

function resolveOpacity(baseOpacity, states) {
  if (states.has('dimmed')) return baseOpacity * 0.16
  if (states.has('path')) return Math.min(1, baseOpacity * 1.1)
  if (states.has('active')) return Math.min(1, baseOpacity * 1.04)
  return baseOpacity
}

function getViewportSize(ctx) {
  return {
    width: ctx.canvas?.clientWidth || ctx.canvas?.width || 0,
    height: ctx.canvas?.clientHeight || ctx.canvas?.height || 0,
  }
}

function projectToScreen(point, transform = { x: 0, y: 0, k: 1 }) {
  return {
    x: point.x * transform.k + transform.x,
    y: point.y * transform.k + transform.y,
  }
}

function projectRadiusToScreen(radius, transform = { k: 1 }) {
  return Number(radius || 0) * transform.k
}

function hexToRgba(color = '#CBD5E1', alpha = 1) {
  const normalized = String(color || '').trim()
  if (!normalized.startsWith('#')) return normalized

  let hex = normalized.slice(1)
  if (hex.length === 3) hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2]
  else if (hex.length === 4) hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2] + hex[3] + hex[3]
  else if (hex.length !== 6 && hex.length !== 8) return normalized

  const red = Number.parseInt(hex.slice(0, 2), 16)
  const green = Number.parseInt(hex.slice(2, 4), 16)
  const blue = Number.parseInt(hex.slice(4, 6), 16)
  return `rgba(${red}, ${green}, ${blue}, ${clamp(alpha, 0, 1)})`
}

function isScreenCircleVisible(circle, viewport, margin = 64) {
  return !(
    circle.x + circle.r < -margin ||
    circle.y + circle.r < -margin ||
    circle.x - circle.r > viewport.width + margin ||
    circle.y - circle.r > viewport.height + margin
  )
}

function isEdgeScreenVisible(edge, visibleNodeIdSet, states = new Set()) {
  if (states.has('selected') || states.has('path') || states.has('active')) return true
  const sourceId = edge.source?.id || edge.source
  const targetId = edge.target?.id || edge.target
  return visibleNodeIdSet.has(sourceId) || visibleNodeIdSet.has(targetId)
}

function boxesIntersect(a, b, padding = 0) {
  return !(
    a.x + a.width + padding < b.x ||
    b.x + b.width + padding < a.x ||
    a.y + a.height + padding < b.y ||
    b.y + b.height + padding < a.y
  )
}

function circleIntersectsBox(circle, box, padding = 0) {
  const nearestX = clamp(circle.x, box.x - padding, box.x + box.width + padding)
  const nearestY = clamp(circle.y, box.y - padding, box.y + box.height + padding)
  const dx = circle.x - nearestX
  const dy = circle.y - nearestY
  return dx * dx + dy * dy <= (circle.r + padding) ** 2
}

function isBoxInsideViewport(box, viewport, margin = 48) {
  return !(
    box.x + box.width < -margin ||
    box.y + box.height < -margin ||
    box.x > viewport.width + margin ||
    box.y > viewport.height + margin
  )
}

function resolveLabelPriority(node, states) {
  let priority = 0
  if (states.has('selected')) priority += 200
  if (states.has('path')) priority += 140
  if (node.view?.data?.isRoot) priority += 120
  if (node.view?.data?.isExpanded) priority += 70
  if (node.view?.data?.isPinned) priority += 36
  if (states.has('active')) priority += 28
  if (!states.has('dimmed')) priority += 18
  priority += Number(node.view?.style?.radius || 0)
  return priority
}

function resolveNodeDrawPriority(node, states) {
  let priority = 0
  if (states.has('dimmed')) priority -= 120
  if (node.view?.data?.isExpanded) priority += 24
  if (node.view?.data?.isRoot) priority += 36
  if (node.view?.data?.isPinned) priority += 20
  if (states.has('active')) priority += 34
  if (states.has('path')) priority += 56
  if (states.has('selected')) priority += 96
  return priority
}

function resolveEdgeLabelPriority(edge, states) {
  let priority = 0
  if (states.has('path')) priority += 90
  if (states.has('active')) priority += 38
  if (!states.has('dimmed')) priority += 14
  priority += Math.max(0, 28 - getRelationPriority(edge.view?.data?.relationType) * 4)
  return priority
}

function resolveEdgeDrawPriority(edge, states) {
  let priority = 0
  if (states.has('dimmed')) priority -= 120
  priority += Math.max(0, 20 - getRelationPriority(edge.view?.data?.relationType) * 3)
  if (states.has('active')) priority += 26
  if (states.has('path')) priority += 64
  return priority
}

function buildBoxFromCenter(centerX, centerY, width, height) {
  return {
    x: centerX - width / 2,
    y: centerY - height / 2,
    width,
    height,
  }
}

function buildRotatedBoxFromCenter(centerX, centerY, width, height, rotation = 0) {
  const cosine = Math.abs(Math.cos(rotation))
  const sine = Math.abs(Math.sin(rotation))
  return buildBoxFromCenter(
    centerX,
    centerY,
    width * cosine + height * sine,
    width * sine + height * cosine,
  )
}

function normalizeReadableAngle(angle = 0) {
  let next = angle
  while (next <= -Math.PI) next += Math.PI * 2
  while (next > Math.PI) next -= Math.PI * 2
  if (next > Math.PI / 2) next -= Math.PI
  if (next < -Math.PI / 2) next += Math.PI
  return next
}

function buildLabelCandidates(node, width, height, transform, states, zoomLevel) {
  const screenCenter = projectToScreen(node, transform)
  const nodeRadius = Number(node.view?.style?.radius || 28) * transform.k
  const gap = nodeRadius + (zoomLevel >= 1.2 ? 12 : zoomLevel >= 0.8 ? 14 : 16)
  const isHighPriority = states.has('selected') || states.has('path') || node.view?.data?.isRoot

  const placements = {
    center: {
      kind: 'center',
      centerX: screenCenter.x,
      centerY: screenCenter.y,
    },
    south: {
      kind: 'south',
      centerX: screenCenter.x,
      centerY: screenCenter.y + gap + height / 2,
    },
    north: {
      kind: 'north',
      centerX: screenCenter.x,
      centerY: screenCenter.y - gap - height / 2,
    },
    east: {
      kind: 'east',
      centerX: screenCenter.x + gap + width / 2,
      centerY: screenCenter.y,
    },
    west: {
      kind: 'west',
      centerX: screenCenter.x - gap - width / 2,
      centerY: screenCenter.y,
    },
    southEast: {
      kind: 'southEast',
      centerX: screenCenter.x + nodeRadius * 0.78 + width / 2,
      centerY: screenCenter.y + nodeRadius * 0.78 + height / 2,
    },
    southWest: {
      kind: 'southWest',
      centerX: screenCenter.x - nodeRadius * 0.78 - width / 2,
      centerY: screenCenter.y + nodeRadius * 0.78 + height / 2,
    },
  }

  const order = isHighPriority
    ? ['center', 'south', 'north', 'east', 'west', 'southEast', 'southWest']
    : zoomLevel >= 0.95
      ? ['south', 'north', 'east', 'west', 'southEast', 'southWest', 'center']
      : ['south', 'east', 'west', 'north', 'southEast', 'southWest', 'center']

  return order.map((key) => {
    const placement = placements[key]
    return {
      kind: placement.kind,
      centerX: placement.centerX,
      centerY: placement.centerY,
      box: buildBoxFromCenter(placement.centerX, placement.centerY, width, height),
    }
  })
}

function shouldShowNodeLabel(node, zoomLevel, states, qualityLevel = 'high') {
  if (qualityLevel === 'low') {
    return Boolean(node.view?.data?.isRoot || states.has('selected') || states.has('path'))
  }
  if (qualityLevel === 'medium') {
    return Boolean(
      zoomLevel >= 0.72 ||
      node.view?.data?.isRoot ||
      node.view?.data?.isExpanded ||
      states.has('selected') ||
      states.has('path'),
    )
  }
  if (zoomLevel >= 0.8) return true
  if (zoomLevel >= 0.6) return true
  if (zoomLevel >= 0.4) return Boolean(node.view?.data?.isRoot || node.view?.data?.isExpanded || states.has('selected') || states.has('path'))
  return Boolean(node.view?.data?.isRoot)
}

function shouldShowEdgeLabel(zoomLevel, visibleNodeCount = 0, states = new Set(), qualityLevel = 'high') {
  if (qualityLevel === 'low') return false
  if (qualityLevel === 'medium') {
    return (states.has('selected') || states.has('path') || states.has('active')) && zoomLevel >= 0.9
  }
  if (states.has('selected') || states.has('path') || states.has('active')) {
    return zoomLevel >= 0.74
  }
  if (zoomLevel < 0.98) return false
  if (visibleNodeCount > 72) return false
  if (visibleNodeCount > 42 && zoomLevel < 1.16) return false
  return true
}

function createPairKey(sourceId, targetId) {
  return [sourceId, targetId].sort().join('::')
}

function getCanonicalPairDirection(sourceId, targetId) {
  const left = String(sourceId || '')
  const right = String(targetId || '')
  return left.localeCompare(right) <= 0 ? `${left}->${right}` : `${right}->${left}`
}

function buildDeflectionSequence(size) {
  if (size <= 0) return []
  if (size === 1) return [0]

  const middleRelationshipIndex = (size - 1) / 2
  const defaultDeflectionStep = 30
  const maximumTotalDeflection = 150
  const numberOfSteps = size - 1
  const totalDeflection = defaultDeflectionStep * numberOfSteps
  const deflectionStep = totalDeflection > maximumTotalDeflection
    ? maximumTotalDeflection / numberOfSteps
    : defaultDeflectionStep

  return Array.from({ length: size }, (_, index) => deflectionStep * (index - middleRelationshipIndex))
}

function deflectionToCurvature(deflectionDegrees, distance) {
  if (!deflectionDegrees) return 0
  const distanceFactor = clamp(distance / 240, 0.82, 1.34)
  const radians = (Math.abs(deflectionDegrees) * Math.PI) / 180
  const normalized = Math.tan(radians * 0.56) / 2.55
  return clamp(normalized * distanceFactor * Math.sign(deflectionDegrees), -0.58, 0.58)
}

function compareEdgesForDraw(left, right, stateMap = {}) {
  const leftStates = getStates(stateMap, left.id)
  const rightStates = getStates(stateMap, right.id)
  const drawPriorityDelta = resolveEdgeDrawPriority(left, leftStates) - resolveEdgeDrawPriority(right, rightStates)
  if (drawPriorityDelta !== 0) return drawPriorityDelta
  const relationDelta = getRelationPriority(left.view?.data?.relationType) - getRelationPriority(right.view?.data?.relationType)
  if (relationDelta !== 0) return relationDelta
  const leftDirection = `${left.source?.id || left.source}->${left.target?.id || left.target}`
  const rightDirection = `${right.source?.id || right.source}->${right.target?.id || right.target}`
  const directionDelta = leftDirection.localeCompare(rightDirection)
  if (directionDelta !== 0) return directionDelta
  return String(left.id || '').localeCompare(String(right.id || ''))
}

function compareEdgesForSlotPriority(left, right, stateMap = {}) {
  const leftStates = getStates(stateMap, left.id)
  const rightStates = getStates(stateMap, right.id)
  const drawPriorityDelta = resolveEdgeDrawPriority(right, rightStates) - resolveEdgeDrawPriority(left, leftStates)
  if (drawPriorityDelta !== 0) return drawPriorityDelta
  const labelPriorityDelta = resolveEdgeLabelPriority(right, rightStates) - resolveEdgeLabelPriority(left, leftStates)
  if (labelPriorityDelta !== 0) return labelPriorityDelta
  const relationDelta = getRelationPriority(left.view?.data?.relationType) - getRelationPriority(right.view?.data?.relationType)
  if (relationDelta !== 0) return relationDelta
  return String(left.id || '').localeCompare(String(right.id || ''))
}

function assignPairCurvatures(group, source, target, stateMap = {}) {
  const distance = Math.hypot((target?.x || 0) - (source?.x || 0), (target?.y || 0) - (source?.y || 0))
  const canonicalDirection = getCanonicalPairDirection(source?.id, target?.id)
  const curvatureMap = new Map()
  const prioritizedGroup = [...group].sort((left, right) => compareEdgesForSlotPriority(left, right, stateMap))
  const deflections = buildDeflectionSequence(group.length).sort((left, right) => {
    const distanceDelta = Math.abs(left) - Math.abs(right)
    if (distanceDelta !== 0) return distanceDelta
    return left - right
  })

  prioritizedGroup.forEach((edge, index) => {
    const directionKey = `${edge.source?.id || edge.source}->${edge.target?.id || edge.target}`
    const directionMultiplier = directionKey === canonicalDirection ? 1 : -1
    const deflection = (deflections[index] || 0) * directionMultiplier
    curvatureMap.set(edge.id, deflectionToCurvature(deflection, distance))
  })

  return curvatureMap
}

function pointOnQuadratic(start, control, end, t) {
  const inv = 1 - t
  return {
    x: inv * inv * start.x + 2 * inv * t * control.x + t * t * end.x,
    y: inv * inv * start.y + 2 * inv * t * control.y + t * t * end.y,
  }
}

function tangentOnQuadratic(start, control, end, t) {
  return {
    x: 2 * (1 - t) * (control.x - start.x) + 2 * t * (end.x - control.x),
    y: 2 * (1 - t) * (control.y - start.y) + 2 * t * (end.y - control.y),
  }
}

function getEdgeTangent(geometry, t = 0.5) {
  const isStraight =
    Math.abs(geometry.control.x - ((geometry.start.x + geometry.end.x) / 2)) < 0.001 &&
    Math.abs(geometry.control.y - ((geometry.start.y + geometry.end.y) / 2)) < 0.001
  return isStraight
    ? { x: geometry.end.x - geometry.start.x, y: geometry.end.y - geometry.start.y }
    : tangentOnQuadratic(geometry.start, geometry.control, geometry.end, t)
}

function buildEdgeGeometry(source, target, curvature = 0) {
  const dx = target.x - source.x
  const dy = target.y - source.y
  const distance = Math.hypot(dx, dy) || 1
  const ux = dx / distance
  const uy = dy / distance
  const nx = -uy
  const ny = ux

  const sourceRadius = Number(source?.view?.style?.radius || 24)
  const targetRadius = Number(target?.view?.style?.radius || 24)
  const start = {
    x: source.x + ux * sourceRadius,
    y: source.y + uy * sourceRadius,
  }
  const end = {
    x: target.x - ux * targetRadius,
    y: target.y - uy * targetRadius,
  }
  const mid = {
    x: (start.x + end.x) / 2,
    y: (start.y + end.y) / 2,
  }
  const control = curvature
    ? {
        x: mid.x + nx * distance * curvature,
        y: mid.y + ny * distance * curvature,
      }
    : mid

  return { start, end, control, distance, curvature }
}

function canPlaceEdgeLabelOnShaft(geometry, labelWorldWidth = 0) {
  const shaftLength = geometry.distance * (1 + Math.min(Math.abs(geometry.curvature || 0) * 1.35, 0.34))
  const curvePenalty = clamp(Math.abs(geometry.curvature || 0) * 220, 0, 44)
  const usableLength = shaftLength - curvePenalty
  return usableLength >= Math.max(32, Number(labelWorldWidth || 0) + 18)
}

function drawArrowhead(ctx, point, tangent, color, opacity) {
  const length = 8
  const width = 4.2
  const magnitude = Math.hypot(tangent.x, tangent.y) || 1
  const ux = tangent.x / magnitude
  const uy = tangent.y / magnitude
  const nx = -uy
  const ny = ux

  ctx.save()
  ctx.fillStyle = color
  ctx.globalAlpha = opacity
  ctx.beginPath()
  ctx.moveTo(point.x, point.y)
  ctx.lineTo(point.x - ux * length + nx * width, point.y - uy * length + ny * width)
  ctx.lineTo(point.x - ux * length - nx * width, point.y - uy * length - ny * width)
  ctx.closePath()
  ctx.fill()
  ctx.restore()
}

function drawBridgeMarker(ctx, geometry, edge, opacity) {
  const midpoint = pointOnQuadratic(geometry.start, geometry.control, geometry.end, 0.5)
  const tangent = Math.abs(geometry.curvature) < 0.001
    ? { x: geometry.end.x - geometry.start.x, y: geometry.end.y - geometry.start.y }
    : tangentOnQuadratic(geometry.start, geometry.control, geometry.end, 0.5)
  const angle = Math.atan2(tangent.y, tangent.x)
  const sourceRing = edge.view?.style?.sourceRing || edge.view?.style?.stroke || '#CBD5E1'
  const targetRing = edge.view?.style?.targetRing || edge.view?.style?.stroke || '#CBD5E1'
  const size = 4

  ctx.save()
  ctx.translate(midpoint.x, midpoint.y)
  ctx.rotate(angle)
  const gradient = ctx.createLinearGradient(-size, 0, size, 0)
  gradient.addColorStop(0, sourceRing)
  gradient.addColorStop(1, targetRing)
  ctx.globalAlpha = clamp(opacity * 0.95, 0.18, 1)
  ctx.fillStyle = gradient
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.92)'
  ctx.lineWidth = 0.9
  ctx.beginPath()
  ctx.moveTo(0, -size)
  ctx.lineTo(size, 0)
  ctx.lineTo(0, size)
  ctx.lineTo(-size, 0)
  ctx.closePath()
  ctx.fill()
  ctx.stroke()
  ctx.restore()
}

function fitTextToWidth(ctx, text = '', maxWidth = 0) {
  const label = String(text || '').trim()
  if (!label) return ''
  if (!Number.isFinite(maxWidth) || maxWidth <= 0) return ''
  if (ctx.measureText(label).width <= maxWidth) return label

  let next = label
  while (next.length > 2) {
    next = `${next.slice(0, -2)}…`
    if (ctx.measureText(next).width <= maxWidth) {
      return next
    }
  }
  return ''
}

function fitTextToWidthCached(ctx, text = '', maxWidth = 0) {
  const label = String(text || '').trim()
  if (!label) return ''
  if (!Number.isFinite(maxWidth) || maxWidth <= 0) return ''
  if (measureTextWidthCached(ctx, label) <= maxWidth) return label

  const suffix = '...'
  if (measureTextWidthCached(ctx, suffix) > maxWidth) return ''

  let low = 1
  let high = label.length
  let best = ''
  while (low <= high) {
    const middle = Math.floor((low + high) / 2)
    const candidate = `${label.slice(0, middle)}${suffix}`
    if (measureTextWidthCached(ctx, candidate) <= maxWidth) {
      best = candidate
      low = middle + 1
    } else {
      high = middle - 1
    }
  }

  return best
}

function buildEdgeLabelLayout(ctx, edge, geometry, zoomLevel, transform, states) {
  const label = edge.view?.data?.label
  if (!label) return null
  const point = pointOnQuadratic(geometry.start, geometry.control, geometry.end, 0.5)
  const tangent = getEdgeTangent(geometry, 0.5)
  const magnitude = Math.hypot(tangent.x, tangent.y) || 1
  const tx = tangent.x / magnitude
  const ty = tangent.y / magnitude
  const nx = -ty
  const ny = tx
  const rotation = normalizeReadableAngle(Math.atan2(ty, tx))
  const screenPoint = projectToScreen(point, transform)
  const offset = clamp(14 + zoomLevel * 3 + Math.abs(geometry.curvature || 0) * 18, 14, 24)
  const tangentOffset = clamp(10 + zoomLevel * 2, 10, 16)
  const fontSize = Math.round(clamp(10 / zoomLevel, 8, 10))
  const shaftLength = geometry.distance * (1 + Math.min(Math.abs(geometry.curvature || 0) * 1.35, 0.34))
  const availableWorldWidth = clamp(
    shaftLength * (states.has('path') || states.has('active') ? 0.46 : 0.38),
    42,
    states.has('path') || states.has('active') ? 132 : 104,
  )

  ctx.save()
  ctx.font = `${fontSize}px "IBM Plex Sans SC", "Segoe UI", sans-serif`
  const displayLabel = fitTextToWidthCached(ctx, label, availableWorldWidth * transform.k)
  if (!displayLabel) {
    ctx.restore()
    return null
  }
  const textWidth = measureTextWidthCached(ctx, displayLabel)
  ctx.restore()

  const pillWidth = (textWidth + 14) * transform.k
  const pillHeight = 18 * transform.k
  const labelWorldWidth = (textWidth + 10) / Math.max(transform.k, 0.001)
  const prefersCenterPlacement = canPlaceEdgeLabelOnShaft(geometry, labelWorldWidth)
  const outerNormalSign = geometry.curvature < 0 ? -1 : 1
  const placementCandidates = prefersCenterPlacement
    ? [
        { kind: 'center', centerX: screenPoint.x, centerY: screenPoint.y },
        { kind: 'tangentForward', centerX: screenPoint.x + tx * tangentOffset, centerY: screenPoint.y + ty * tangentOffset },
        { kind: 'tangentBackward', centerX: screenPoint.x - tx * tangentOffset, centerY: screenPoint.y - ty * tangentOffset },
        { kind: 'outer', centerX: screenPoint.x + nx * offset * outerNormalSign, centerY: screenPoint.y + ny * offset * outerNormalSign },
        { kind: 'inner', centerX: screenPoint.x - nx * offset * outerNormalSign, centerY: screenPoint.y - ny * offset * outerNormalSign },
      ]
    : [
        { kind: 'outer', centerX: screenPoint.x + nx * offset * outerNormalSign, centerY: screenPoint.y + ny * offset * outerNormalSign },
        { kind: 'inner', centerX: screenPoint.x - nx * offset * outerNormalSign, centerY: screenPoint.y - ny * offset * outerNormalSign },
        { kind: 'tangentForward', centerX: screenPoint.x + tx * tangentOffset, centerY: screenPoint.y + ty * tangentOffset },
        { kind: 'tangentBackward', centerX: screenPoint.x - tx * tangentOffset, centerY: screenPoint.y - ty * tangentOffset },
        { kind: 'center', centerX: screenPoint.x, centerY: screenPoint.y },
      ]
  const placements = placementCandidates.map((placement) => ({
    ...placement,
    rotation,
    pillWidth,
    pillHeight,
    box: buildRotatedBoxFromCenter(placement.centerX, placement.centerY, pillWidth, pillHeight, rotation),
  }))

  return {
    edge,
    states,
    label: displayLabel,
    priority: resolveEdgeLabelPriority(edge, states),
    accentColor: edge.view?.style?.stroke || '#CBD5E1',
    prefersCenterPlacement,
    placements,
  }
}

function paintEdgeLabel(ctx, layout, placement, transform) {
  const worldX = (placement.centerX - transform.x) / transform.k
  const worldY = (placement.centerY - transform.y) / transform.k
  const worldWidth = placement.pillWidth / transform.k
  const worldHeight = placement.pillHeight / transform.k
  const integrated = placement.kind === 'center' && layout.prefersCenterPlacement
  const fill = integrated
    ? hexToRgba(layout.accentColor, layout.states.has('path') ? 0.3 : layout.states.has('active') ? 0.24 : 0.18)
    : layout.states.has('path')
      ? 'rgba(17, 94, 89, 0.82)'
      : layout.states.has('active')
        ? 'rgba(30, 41, 59, 0.82)'
        : 'rgba(15, 23, 42, 0.76)'

  ctx.save()
  ctx.globalAlpha = resolveOpacity(1, layout.states)
  ctx.font = `${Math.round(clamp(10 / transform.k, 8, 10))}px "IBM Plex Sans SC", "Segoe UI", sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.translate(worldX, worldY)
  ctx.rotate(placement.rotation || 0)
  ctx.fillStyle = fill
  ctx.beginPath()
  ctx.roundRect(
    -worldWidth / 2,
    -worldHeight / 2,
    worldWidth,
    worldHeight,
    8 / Math.max(transform.k, 0.001),
  )
  ctx.fill()
  if (integrated) {
    ctx.lineWidth = 1 / Math.max(transform.k, 0.001)
    ctx.strokeStyle = hexToRgba(layout.accentColor, layout.states.has('path') ? 0.88 : 0.72)
    ctx.stroke()
  }
  ctx.fillStyle = 'rgba(248, 250, 252, 0.96)'
  ctx.fillText(layout.label, 0, 1 / Math.max(transform.k, 0.001))
  ctx.restore()
}

function drawSingleEdge(ctx, edge, source, target, curvature, zoomLevel, stateMap, visibleNodeCount, transform, qualityLevel) {
  const states = getStates(stateMap, edge.id)
  const geometry = buildEdgeGeometry(source, target, curvature)
  const opacity = resolveOpacity(edge.view?.style?.opacity ?? 0.8, states)
  const isPath = states.has('path')
  const isActive = states.has('active')
  const isCrossCourse = states.has('cross-course') || Boolean(edge.view?.data?.crossCourse)
  const lineWidthBase = edge.view?.style?.lineWidth || 1.5
  const stroke = edge.view?.style?.stroke || '#CBD5E1'
  const sourceRing = edge.view?.style?.sourceRing || stroke
  const targetRing = edge.view?.style?.targetRing || stroke
  const useGradient = isCrossCourse && (isActive || isPath)
  const effectiveLineWidth = Math.max(
    lineWidthBase + (isPath ? 1.2 : isActive ? 0.45 : 0),
    useGradient ? 3 : isCrossCourse ? 1.8 : lineWidthBase,
  )
  const lineDash = states.has('cross-course-dimmed')
    ? [7, 8]
    : edge.view?.style?.lineDash || []

  ctx.save()
  ctx.beginPath()
  ctx.moveTo(geometry.start.x, geometry.start.y)
  if (Math.abs(curvature) < 0.001) {
    ctx.lineTo(geometry.end.x, geometry.end.y)
  } else {
    ctx.quadraticCurveTo(geometry.control.x, geometry.control.y, geometry.end.x, geometry.end.y)
  }
  ctx.strokeStyle = useGradient
    ? (() => {
        const gradient = ctx.createLinearGradient(geometry.start.x, geometry.start.y, geometry.end.x, geometry.end.y)
        gradient.addColorStop(0, sourceRing)
        gradient.addColorStop(1, targetRing)
        return gradient
      })()
    : stroke
  ctx.lineWidth = effectiveLineWidth
  ctx.globalAlpha = opacity
  ctx.setLineDash(lineDash)
  if (isPath) {
    ctx.shadowColor = 'rgba(45, 212, 191, 0.34)'
    ctx.shadowBlur = 10
  } else if (isActive) {
    ctx.shadowColor = 'rgba(148, 163, 184, 0.24)'
    ctx.shadowBlur = 6
  } else if (isCrossCourse && qualityLevel === 'high') {
    ctx.shadowColor = hexToRgba(sourceRing, 0.16)
    ctx.shadowBlur = 4
  }
  ctx.stroke()
  ctx.restore()

  if (edge.view?.style?.endArrow) {
    const tangent = Math.abs(curvature) < 0.001
      ? { x: geometry.end.x - geometry.start.x, y: geometry.end.y - geometry.start.y }
      : tangentOnQuadratic(geometry.start, geometry.control, geometry.end, 0.92)
    drawArrowhead(ctx, geometry.end, tangent, useGradient ? targetRing : stroke, opacity)
  }

  if (isCrossCourse && (isActive || isPath)) {
    drawBridgeMarker(ctx, geometry, edge, opacity)
  }

  if (shouldShowEdgeLabel(zoomLevel, visibleNodeCount, states, qualityLevel)) {
    return buildEdgeLabelLayout(ctx, edge, geometry, zoomLevel, transform, states)
  }

  return null
}

function drawEdges(ctx, edges = [], nodeMap = new Map(), zoomLevel = 1, stateMap = {}, transform = { x: 0, y: 0, k: 1 }, qualityLevel = 'high') {
  const visibleNodeCount = nodeMap.size
  const pairMap = new Map()
  const edgeLabelLayouts = []
  edges.forEach((edge) => {
    const sourceId = edge.source?.id || edge.source
    const targetId = edge.target?.id || edge.target
    if (!sourceId || !targetId) return
    const key = createPairKey(sourceId, targetId)
    if (!pairMap.has(key)) pairMap.set(key, [])
    pairMap.get(key).push(edge)
  })

  pairMap.forEach((group) => {
    const orderedGroup = [...group].sort((left, right) => compareEdgesForDraw(left, right, stateMap))
    const sampleEdge = orderedGroup[0]
    const sampleSourceId = sampleEdge?.source?.id || sampleEdge?.source
    const sampleTargetId = sampleEdge?.target?.id || sampleEdge?.target
    const sampleSource = nodeMap.get(sampleSourceId)
    const sampleTarget = nodeMap.get(sampleTargetId)
    const curvatureMap = assignPairCurvatures(orderedGroup, sampleSource, sampleTarget, stateMap)
    orderedGroup.forEach((edge, index) => {
      const sourceId = edge.source?.id || edge.source
      const targetId = edge.target?.id || edge.target
      const source = nodeMap.get(sourceId)
      const target = nodeMap.get(targetId)
      if (!source || !target) return
      const curvature = curvatureMap.get(edge.id) ?? 0
      const labelLayout = drawSingleEdge(ctx, edge, source, target, curvature, zoomLevel, stateMap, visibleNodeCount, transform, qualityLevel)
      if (labelLayout) edgeLabelLayouts.push(labelLayout)
    })
  })

  return edgeLabelLayouts
}

function drawNode(ctx, node, stateMap, flashSet, qualityLevel = 'high') {
  const states = getStates(stateMap, node.id)
  const style = node.view?.style || {}
  const opacity = resolveOpacity(style.opacity ?? 1, states)
  const radius = Number(style.radius || 28)
  const isSelected = states.has('selected')
  const isPath = states.has('path')
  const isFlashed = flashSet.has(node.id)
  const isExpanded = Boolean(node.view?.data?.isExpanded)
  const isRoot = Boolean(node.view?.data?.isRoot)
  const expandMeta = node.view?.data?.expandMeta
  const now = Date.now()

  ctx.save()
  ctx.globalAlpha = opacity

  if (qualityLevel !== 'low' && expandMeta?.expiresAt && expandMeta.expiresAt > now) {
    const startAt = Number(expandMeta.createdAt || (expandMeta.expiresAt - 1800))
    const progress = clamp((now - startAt) / Math.max(1, expandMeta.expiresAt - startAt), 0, 1)
    const pulseRadius = radius + 8 + progress * 18
    const pulseAlpha = (1 - progress) * 0.18
    ctx.save()
    ctx.globalAlpha = pulseAlpha
    ctx.beginPath()
    ctx.arc(node.x, node.y, pulseRadius, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(103, 232, 249, 0.9)'
    ctx.fill()
    ctx.restore()
  }

  if (isSelected || isPath || isRoot || isExpanded) {
    ctx.save()
    ctx.globalAlpha = isSelected ? 0.22 : isPath ? 0.14 : isExpanded ? 0.11 : 0.08
    ctx.beginPath()
    ctx.arc(node.x, node.y, radius + (isSelected ? 18 : isExpanded ? 14 : 12), 0, Math.PI * 2)
    ctx.fillStyle = isSelected ? 'rgba(103, 232, 249, 0.92)' : isPath ? 'rgba(45, 212, 191, 0.82)' : 'rgba(255, 255, 255, 0.78)'
    ctx.fill()
    ctx.restore()
  }

  ctx.shadowColor = style.shadowColor || 'rgba(99, 102, 241, 0.2)'
  ctx.shadowBlur = qualityLevel === 'low' ? 0 : (style.shadowBlur || 8) * (qualityLevel === 'medium' ? 0.5 : 1) + (isFlashed ? 10 : 0)
  ctx.beginPath()
  ctx.arc(node.x, node.y, radius + 4, 0, Math.PI * 2)
  ctx.fillStyle = style.ringColor || style.badgeColor || '#CBD5E1'
  ctx.fill()
  ctx.shadowBlur = 0
  ctx.beginPath()
  ctx.arc(node.x, node.y, radius, 0, Math.PI * 2)
  ctx.fillStyle = style.fill || '#E2E8F0'
  ctx.fill()

  if (isPath) {
    ctx.lineWidth = (style.borderWidth || 2) + 2
    ctx.strokeStyle = 'rgba(45, 212, 191, 0.92)'
    ctx.stroke()
  }

  ctx.shadowBlur = 0
  ctx.lineWidth = (style.borderWidth || 2) + (isSelected ? 1.4 : 0)
  ctx.strokeStyle = style.stroke || '#CBD5E1'
  ctx.stroke()

  if (isExpanded && !isSelected) {
    ctx.beginPath()
    ctx.arc(node.x, node.y, radius + 9, 0, Math.PI * 2)
    ctx.setLineDash([4, 5])
    ctx.lineWidth = 1.2
    ctx.strokeStyle = 'rgba(103, 232, 249, 0.5)'
    ctx.stroke()
    ctx.setLineDash([])
  }

  if (isRoot) {
    ctx.beginPath()
    ctx.arc(node.x, node.y, radius + 6, 0, Math.PI * 2)
    ctx.lineWidth = 1.4
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.28)'
    ctx.stroke()
  }
  ctx.restore()
}

function buildNodeScreenCircle(node, transform) {
  const style = node.view?.style || {}
  const radius = Number(style.radius || 28) * transform.k
  const center = projectToScreen(node, transform)
  return {
    id: node.id,
    x: center.x,
    y: center.y,
    r: radius,
  }
}

function buildNodeLabelLayout(ctx, node, zoomLevel, transform, states) {
  const style = node.view?.style || {}
  const fontSize = clamp((style.fontSize || 11) / Math.max(zoomLevel, 0.65), 8, node.view?.data?.isRoot ? 14 : 11)
  const lines = splitLabelLines(node.view?.label || node.view?.data?.canonicalName || node.id, node.view?.data?.isRoot ? 6 : 5, 3)
  const lineHeight = fontSize + 4

  ctx.save()
  ctx.font = `${Math.round(fontSize)}px "IBM Plex Sans SC", "Segoe UI", sans-serif`
  const textWidth = Math.max(...lines.map((line) => measureTextWidthCached(ctx, line)), 0)
  ctx.restore()

  const width = (textWidth + 12) * transform.k
  const height = (lines.length * lineHeight + 6) * transform.k
  return {
    node,
    states,
    lines,
    fontSize,
    lineHeight,
    priority: resolveLabelPriority(node, states),
    placements: buildLabelCandidates(node, width, height, transform, states, zoomLevel),
  }
}

function drawNodeLabels(ctx, nodes = [], zoomLevel = 1, stateMap = {}, transform = { x: 0, y: 0, k: 1 }, screenCircles = [], qualityLevel = 'high') {
  const viewport = getViewportSize(ctx)
  const occupiedBoxes = []

  const candidates = nodes
    .map((node) => ({ node, states: getStates(stateMap, node.id) }))
    .filter(({ node, states }) => shouldShowNodeLabel(node, zoomLevel, states, qualityLevel))
    .map(({ node, states }) => buildNodeLabelLayout(ctx, node, zoomLevel, transform, states))
    .sort((left, right) => right.priority - left.priority)

  candidates.forEach((candidate) => {
    const labelPadding = zoomLevel >= 1.2 ? 7 : zoomLevel >= 0.8 ? 9 : 12
    const isHighPriority = candidate.priority >= 150
    const selectedPlacement = candidate.placements.reduce((best, placement) => {
      let score = 0
      if (!isBoxInsideViewport(placement.box, viewport, 54)) {
        score += 1000
      }

      occupiedBoxes.forEach((box) => {
        if (boxesIntersect(placement.box, box, labelPadding)) score += 220
      })

      screenCircles.forEach((circle) => {
        if (circle.id === candidate.node.id) return
        if (circleIntersectsBox(circle, placement.box, 4)) {
          score += placement.kind === 'center' ? 140 : 80
        }
      })

      if (placement.kind === 'center') score += isHighPriority ? 8 : 26
      if (placement.kind === 'north' || placement.kind === 'south') score += 4
      if (placement.kind === 'east' || placement.kind === 'west') score += 10
      if (placement.kind === 'southEast' || placement.kind === 'southWest') score += 12

      if (!best || score < best.score) {
        return { ...placement, score }
      }
      return best
    }, null)

    if (!selectedPlacement) return
    if (!isHighPriority && selectedPlacement.score >= 80) return
    if (isHighPriority && selectedPlacement.score >= 1200) return

    const { node, states, lines, fontSize, lineHeight } = candidate
    const style = node.view?.style || {}
    const worldX = (selectedPlacement.centerX - transform.x) / transform.k
    const worldY = (selectedPlacement.centerY - transform.y) / transform.k
    const drawBackground = selectedPlacement.kind !== 'center' || isHighPriority
    const drawLeader = selectedPlacement.kind !== 'center'

    ctx.save()
    ctx.globalAlpha = resolveOpacity(1, states)
    ctx.font = `${Math.round(fontSize)}px "IBM Plex Sans SC", "Segoe UI", sans-serif`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.strokeStyle = style.labelStroke || 'rgba(15, 23, 42, 0.88)'
    ctx.fillStyle = style.labelColor || '#F8FAFC'
    ctx.lineWidth = node.view?.data?.isRoot ? 4.8 : 4

    if (drawLeader) {
      const dx = worldX - node.x
      const dy = worldY - node.y
      const distance = Math.hypot(dx, dy) || 1
      const ux = dx / distance
      const uy = dy / distance
      const radius = Number(style.radius || 28)
      const labelHalfWidth = (selectedPlacement.box.width / transform.k) / 2
      const labelHalfHeight = (selectedPlacement.box.height / transform.k) / 2
      const endOffset = Math.min(labelHalfWidth, labelHalfHeight) * 0.42
      const lineStartX = node.x + ux * (radius + 4)
      const lineStartY = node.y + uy * (radius + 4)
      const lineEndX = worldX - ux * endOffset
      const lineEndY = worldY - uy * endOffset

      ctx.save()
      ctx.globalAlpha = resolveOpacity(0.56, states)
      ctx.beginPath()
      ctx.moveTo(lineStartX, lineStartY)
      ctx.lineTo(lineEndX, lineEndY)
      ctx.strokeStyle = states.has('selected')
        ? 'rgba(103, 232, 249, 0.82)'
        : states.has('path')
          ? 'rgba(45, 212, 191, 0.72)'
          : style.labelLeaderColor || 'rgba(148, 163, 184, 0.62)'
      ctx.lineWidth = node.view?.data?.isRoot ? 1.4 : 1.1
      ctx.stroke()
      ctx.restore()
    }

    if (drawBackground) {
      const backgroundWorldWidth = selectedPlacement.box.width / transform.k
      const backgroundWorldHeight = selectedPlacement.box.height / transform.k
      ctx.fillStyle = states.has('path')
        ? 'rgba(17, 94, 89, 0.8)'
        : style.labelBackground || (states.has('selected')
          ? 'rgba(8, 47, 73, 0.82)'
          : node.view?.data?.isRoot
            ? 'rgba(30, 41, 59, 0.74)'
            : 'rgba(15, 23, 42, 0.62)')
      ctx.beginPath()
      ctx.roundRect(
        worldX - backgroundWorldWidth / 2,
        worldY - backgroundWorldHeight / 2,
        backgroundWorldWidth,
        backgroundWorldHeight,
        8 / Math.max(transform.k, 0.001),
      )
      ctx.fill()
      if (style.labelBorder) {
        ctx.lineWidth = 1 / Math.max(transform.k, 0.001)
        ctx.strokeStyle = style.labelBorder
        ctx.stroke()
      }
      ctx.fillStyle = style.labelColor || '#F8FAFC'
    }

    const startY = worldY - ((lines.length - 1) * lineHeight) / 2
    lines.forEach((line, index) => {
      const y = startY + index * lineHeight
      ctx.strokeText(line, worldX, y)
      ctx.fillText(line, worldX, y)
    })
    ctx.restore()

    occupiedBoxes.push(selectedPlacement.box)
  })

  return occupiedBoxes
}

function drawEdgeLabels(ctx, edgeLabelLayouts = [], occupiedBoxes = [], screenCircles = [], transform = { x: 0, y: 0, k: 1 }) {
  if (!edgeLabelLayouts.length) return

  const viewport = getViewportSize(ctx)
  const placedBoxes = [...occupiedBoxes]
  const candidates = [...edgeLabelLayouts].sort((left, right) => right.priority - left.priority)

  candidates.forEach((candidate) => {
    const selectedPlacement = candidate.placements.reduce((best, placement) => {
      let score = 0
      if (!isBoxInsideViewport(placement.box, viewport, 42)) {
        score += 1000
      }

      placedBoxes.forEach((box) => {
        if (boxesIntersect(placement.box, box, 8)) score += 240
      })

      screenCircles.forEach((circle) => {
        if (circleIntersectsBox(circle, placement.box, 6)) score += 180
      })

      if (placement.kind === 'center') {
        score += candidate.prefersCenterPlacement ? -18 : 24
      }
      if (placement.kind === 'outer') {
        score += candidate.prefersCenterPlacement ? 12 : 0
      }
      if (placement.kind === 'inner') {
        score += candidate.prefersCenterPlacement ? 16 : 4
      }
      if (placement.kind === 'tangentForward' || placement.kind === 'tangentBackward') {
        score += candidate.prefersCenterPlacement ? 10 : 8
      }

      if (!best || score < best.score) {
        return { ...placement, score }
      }
      return best
    }, null)

    if (!selectedPlacement || selectedPlacement.score >= 120) return
    paintEdgeLabel(ctx, candidate, selectedPlacement, transform)
    placedBoxes.push(selectedPlacement.box)
  })
}

function drawCourseNebula(ctx, nodes = [], zoomLevel = 1, qualityLevel = 'high') {
  if (qualityLevel === 'low' || zoomLevel < 0.5) return

  const groups = new Map()
  nodes.forEach((node) => {
    const course = String(node.view?.data?.chapterTitle || '').trim()
    if (!course) return
    if (!groups.has(course)) groups.set(course, [])
    groups.get(course).push(node)
  })

  const rankedGroups = [...groups.values()]
    .filter((group) => group.length >= 3)
    .sort((left, right) => right.length - left.length)
    .slice(0, 8)

  rankedGroups.forEach((group) => {
    const centerX = group.reduce((sum, node) => sum + node.x, 0) / group.length
    const centerY = group.reduce((sum, node) => sum + node.y, 0) / group.length
    const radius = Math.max(
      ...group.map((node) => Math.hypot(node.x - centerX, node.y - centerY) + Number(node.view?.style?.radius || 28) + 54),
      140,
    )
    const color = group[0]?.view?.style?.courseLight || 'rgba(148, 163, 184, 0.06)'
    const gradient = ctx.createRadialGradient(centerX, centerY, radius * 0.1, centerX, centerY, radius)
    gradient.addColorStop(0, hexToRgba(color, qualityLevel === 'medium' ? 0.042 : 0.06))
    gradient.addColorStop(1, hexToRgba(color, 0))
    ctx.save()
    ctx.fillStyle = gradient
    ctx.beginPath()
    ctx.arc(centerX, centerY, radius, 0, Math.PI * 2)
    ctx.fill()
    ctx.restore()
  })
}

function drawPathParticles(ctx, edges = [], nodeMap = new Map(), stateMap = {}, timestamp = Date.now()) {
  const pathEdges = edges.filter((edge) => getStates(stateMap, edge.id).has('path'))
  if (!pathEdges.length) return

  pathEdges.forEach((edge, edgeIndex) => {
    const source = nodeMap.get(edge.source?.id || edge.source)
    const target = nodeMap.get(edge.target?.id || edge.target)
    if (!source || !target) return

    const geometry = buildEdgeGeometry(source, target, 0)
    const particleCount = 2
    for (let particleIndex = 0; particleIndex < particleCount; particleIndex += 1) {
      const phase = ((timestamp / 1400) + edgeIndex * 0.11 + particleIndex * 0.5) % 1
      const point = pointOnQuadratic(geometry.start, geometry.control, geometry.end, phase)
      ctx.save()
      ctx.globalAlpha = 0.42 + (1 - phase) * 0.26
      ctx.shadowColor = 'rgba(45, 212, 191, 0.55)'
      ctx.shadowBlur = 8
      ctx.fillStyle = 'rgba(45, 212, 191, 0.96)'
      ctx.beginPath()
      ctx.arc(point.x, point.y, 2, 0, Math.PI * 2)
      ctx.fill()
      ctx.restore()
    }
  })
}

export function drawScene(ctx, scene = {}, transform = { x: 0, y: 0, k: 1 }) {
  const { width, height } = getViewportSize(ctx)
  const nodes = scene.nodes || []
  const edges = scene.edges || []
  const qualityLevel = scene.qualityLevel || 'high'
  const nodeMap = new Map(nodes.map((node) => [node.id, node]))
  const flashSet = scene.flashSet || new Set()
  const stateMap = scene.stateMap || {}
  const viewport = { width, height }

  ctx.save()
  ctx.clearRect(0, 0, width, height)
  ctx.translate(transform.x, transform.y)
  ctx.scale(transform.k, transform.k)

  const screenCircles = nodes.map((node) => buildNodeScreenCircle(node, transform))
  const visibleScreenCircles = screenCircles.filter((circle) => isScreenCircleVisible(circle, viewport, 96))
  const visibleNodeIdSet = new Set(
    visibleScreenCircles
      .map((circle) => circle.id)
      .concat(
        nodes
          .filter((node) => {
            const states = getStates(stateMap, node.id)
            return states.has('selected') || states.has('path') || states.has('active')
          })
          .map((node) => node.id),
      ),
  )
  const visibleNodes = nodes.filter((node) => visibleNodeIdSet.has(node.id))
  const visibleEdges = edges.filter((edge) => isEdgeScreenVisible(edge, visibleNodeIdSet, getStates(stateMap, edge.id)))
  const edgeLabelLayouts = drawEdges(ctx, visibleEdges, nodeMap, transform.k, stateMap, transform, qualityLevel)
  const orderedNodes = [...visibleNodes].sort((left, right) => {
      const leftStates = getStates(stateMap, left.id)
      const rightStates = getStates(stateMap, right.id)
      const priorityDelta = resolveNodeDrawPriority(left, leftStates) - resolveNodeDrawPriority(right, rightStates)
      if (priorityDelta !== 0) return priorityDelta
      return String(left.id || '').localeCompare(String(right.id || ''))
    })

  drawCourseNebula(ctx, orderedNodes, transform.k, qualityLevel)
  if (qualityLevel !== 'low') {
    drawPathParticles(ctx, visibleEdges, nodeMap, stateMap, Date.now())
  }
  orderedNodes.forEach((node) => drawNode(ctx, node, stateMap, flashSet, qualityLevel))
  const occupiedBoxes = drawNodeLabels(ctx, orderedNodes, transform.k, stateMap, transform, visibleScreenCircles, qualityLevel)
  if (qualityLevel !== 'low') {
    drawEdgeLabels(ctx, edgeLabelLayouts, occupiedBoxes, visibleScreenCircles, transform)
  }

  ctx.restore()
}

export const __graphDrawTestUtils = {
  assignPairCurvatures,
  buildDeflectionSequence,
  canPlaceEdgeLabelOnShaft,
  clearTextWidthCache,
  measureTextWidthCached,
  shouldShowEdgeLabel,
  shouldShowNodeLabel,
}
