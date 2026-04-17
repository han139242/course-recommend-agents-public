import {
  forceCenter,
  forceCollide,
  forceLink,
  forceManyBody,
  forceSimulation,
} from 'd3-force'

export const MAX_EXPAND_DEPTH = 4
export const MAX_VISIBLE_NODES = 200
export const NEIGHBOR_LIMIT = 20
export const MIN_ZOOM = 0.4
export const MAX_ZOOM = 2
export const DEFAULT_ZOOM = 1
const DEFAULT_RELATION_PRIORITY = 99
const DEFAULT_NODE_VISUAL_RADIUS = 48
const MAX_NODE_VISUAL_RADIUS = 164
const FULL_CIRCLE = Math.PI * 2
export const RELATION_PRIORITY = {
  PREREQUISITE_OF: 0,
  SUBTOPIC_OF: 1,
  PART_OF: 2,
  RELATED_TO: 3,
  USES: 4,
  APPLIES_TO: 5,
}

const TYPE_DISTANCE = {
  PREREQUISITE_OF: 62,
  SUBTOPIC_OF: 36,
  PART_OF: 38,
  RELATED_TO: 84,
  USES: 70,
  APPLIES_TO: 70,
}

const TYPE_STRENGTH = {
  PREREQUISITE_OF: 0.74,
  SUBTOPIC_OF: 0.82,
  PART_OF: 0.8,
  RELATED_TO: 0.34,
  USES: 0.42,
  APPLIES_TO: 0.42,
}

export function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function stableHash(value = '') {
  return [...String(value || '')].reduce((hash, char) => (hash * 33 + char.charCodeAt(0)) % 2147483647, 7)
}

export function getNodeVisualRadius(node) {
  const radius = Number(
    node?.view?.style?.collisionRadius ||
    node?.collisionRadius ||
    node?.view?.style?.radius ||
    node?.radius ||
    DEFAULT_NODE_VISUAL_RADIUS,
  )
  return clamp(radius, 28, MAX_NODE_VISUAL_RADIUS)
}

export function getLinkDistance(link) {
  const relationType = link?.view?.data?.relationType || link?.relationType
  const gap = TYPE_DISTANCE[relationType] || 72
  const sourceRadius = getNodeVisualRadius(link?.source)
  const targetRadius = getNodeVisualRadius(link?.target)
  return clamp(sourceRadius + targetRadius + gap, gap + 56, 420)
}

export function getLinkStrength(link) {
  return TYPE_STRENGTH[link?.view?.data?.relationType || link?.relationType] || 0.4
}

export function getNodeCharge(node) {
  const isRoot = Boolean(node?.view?.data?.isRoot)
  const isExpanded = Boolean(node?.view?.data?.isExpanded)
  const depth = Number(node?.view?.data?.layoutDepth || 0)
  const visualRadius = getNodeVisualRadius(node)
  let charge = isRoot ? -460 : -240
  if (isExpanded) charge -= 32
  if (depth >= 2) charge -= Math.min(depth * 10, 36)
  charge -= Math.max(0, visualRadius - DEFAULT_NODE_VISUAL_RADIUS) * (isRoot ? 1.2 : 1.45)
  return Math.round(charge)
}

export function getNodeCollisionRadius(node) {
  const radius = Number(
    node?.view?.style?.collisionRadius ||
    node?.collisionRadius ||
    node?.view?.style?.radius ||
    node?.radius ||
    DEFAULT_NODE_VISUAL_RADIUS,
  )
  return clamp(radius + 14, 24, 164)
}

function createExpansionForce() {
  let nodes = []
  let nodeMap = new Map()

  function force(alpha) {
    const now = Date.now()
    nodes.forEach((node) => {
      const anchor = node.expandAnchor
      if (!anchor) return
      if (anchor.expiresAt <= now) {
        node.expandAnchor = null
        return
      }
      const centerNode = anchor.anchorId ? nodeMap.get(anchor.anchorId) : null
      const centerX = Number.isFinite(centerNode?.x) ? centerNode.x : anchor.x
      const centerY = Number.isFinite(centerNode?.y) ? centerNode.y : anchor.y
      const dx = node.x - centerX
      const dy = node.y - centerY
      const distance = Math.hypot(dx, dy) || 1
      const ux = dx / distance
      const uy = dy / distance
      const targetRadius =
        Number.isFinite(anchor.radius) && anchor.radius > 0
          ? anchor.radius
          : Math.hypot((anchor.targetX ?? node.x) - centerX, (anchor.targetY ?? node.y) - centerY)

      const radialError = targetRadius - distance
      const radialStrength = (anchor.radialStrength || 0.046) * alpha
      node.vx += ux * radialError * radialStrength
      node.vy += uy * radialError * radialStrength

      const targetStrength = (anchor.targetStrength || 0.064) * alpha
      if (Number.isFinite(anchor.targetX) && Number.isFinite(anchor.targetY)) {
        node.vx += (anchor.targetX - node.x) * targetStrength
        node.vy += (anchor.targetY - node.y) * targetStrength
      }
    })
  }

  force.initialize = (nextNodes) => {
    nodes = nextNodes || []
    nodeMap = new Map(nodes.map((node) => [node.id, node]))
  }

  return force
}

export function createSimulation(nodes = [], links = [], onTick = () => {}) {
  const simulation = forceSimulation(nodes)
    .force(
      'link',
      forceLink(links)
        .id((node) => node.id)
        .distance((link) => getLinkDistance(link))
        .strength((link) => getLinkStrength(link)),
    )
    .force(
      'charge',
      forceManyBody()
        .strength((node) => getNodeCharge(node))
        .distanceMax(520),
    )
    .force(
      'collide',
      forceCollide()
        .radius((node) => getNodeCollisionRadius(node))
        .iterations(4)
        .strength(1),
    )
    .force('center', forceCenter(0, 0))
    .force('expand', createExpansionForce())
    .velocityDecay(0.34)
    .alphaDecay(0.032)
    .alphaMin(0.001)

  simulation.on('tick', onTick)
  return simulation
}

export function createCircularPlacement(count = 0, options = {}) {
  const total = Math.max(0, Number(count || 0))
  if (!total) return []

  const centerX = Number.isFinite(options.center?.x) ? options.center.x : 0
  const centerY = Number.isFinite(options.center?.y) ? options.center.y : 0
  const spacing = clamp(Number(options.spacing || 110), 88, 168)
  const radius = Number.isFinite(options.radius)
    ? Math.max(0, options.radius)
    : clamp((total * spacing) / (Math.PI * 2), spacing * 0.72, 540)
  const startAngle = Number.isFinite(options.startAngle) ? options.startAngle : -Math.PI / 2
  const yScale = clamp(Number(options.yScale || 0.92), 0.62, 1.2)

  return Array.from({ length: total }, (_, index) => {
    const angle = startAngle + (Math.PI * 2 * index) / total
    return {
      x: centerX + Math.cos(angle) * radius,
      y: centerY + Math.sin(angle) * radius * yScale,
      angle,
      radius,
    }
  })
}

export function createDetachedRootOrigin(rootCount = 0) {
  if (rootCount <= 0) {
    return { x: 0, y: 0 }
  }
  const angle = (stableHash(`root:${rootCount}`) % 360) * (Math.PI / 180)
  const radius = clamp(260 + rootCount * 18, 260, 520)
  return {
    x: Math.cos(angle) * radius,
    y: Math.sin(angle) * radius * 0.72,
  }
}

function normalizeAngle(angle = 0) {
  let next = angle % FULL_CIRCLE
  if (next < 0) next += FULL_CIRCLE
  return next
}

function angleDistance(left = 0, right = 0) {
  const delta = Math.abs(normalizeAngle(left) - normalizeAngle(right))
  return Math.min(delta, FULL_CIRCLE - delta)
}

function angleInsideGap(angle, start, end) {
  let adjusted = normalizeAngle(angle)
  while (adjusted < start) {
    adjusted += FULL_CIRCLE
  }
  return adjusted >= start && adjusted <= end
}

function clampAngleToGap(angle, gap, margin = 0) {
  const safeMargin = Math.max(0, Math.min(margin, Math.max(gap.gap / 2 - 0.01, 0)))
  let adjusted = normalizeAngle(angle)
  while (adjusted < gap.start) {
    adjusted += FULL_CIRCLE
  }
  return normalizeAngle(clamp(adjusted, gap.start + safeMargin, gap.end - safeMargin))
}

function resolvePreferredAngle(anchor, options = {}) {
  if (Number.isFinite(options.preferredAngle)) {
    return normalizeAngle(options.preferredAngle)
  }
  const preferredPoint = options.preferredPoint || options.viewportCenterWorld
  if (!Number.isFinite(preferredPoint?.x) || !Number.isFinite(preferredPoint?.y)) {
    return null
  }
  return normalizeAngle(Math.atan2(preferredPoint.y - anchor.y, preferredPoint.x - anchor.x))
}

function buildOccupiedAngularIntervals(anchor, occupiedNodes = []) {
  const anchorRadius = getNodeVisualRadius(anchor)

  const intervals = occupiedNodes
    .filter((node) => node && node.id !== anchor.id && Number.isFinite(node.x) && Number.isFinite(node.y))
    .map((node) => {
      const dx = node.x - anchor.x
      const dy = node.y - anchor.y
      const distance = Math.hypot(dx, dy)
      if (distance <= 24) {
        return {
          start: 0,
          end: FULL_CIRCLE,
          distance,
        }
      }

      const occupiedRadius =
        getNodeVisualRadius(node) +
        18 +
        Math.min(anchorRadius * 0.16, 18)
      const projectedRadius = Math.min(occupiedRadius, distance * 0.98)
      const halfSpan = clamp(
        Math.asin(clamp(projectedRadius / Math.max(distance, 1), 0, 0.98)) + 0.04,
        0.05,
        Math.PI * 0.92,
      )
      const angle = normalizeAngle(Math.atan2(dy, dx))
      return {
        start: normalizeAngle(angle - halfSpan),
        end: normalizeAngle(angle + halfSpan),
        distance,
      }
    })

  if (!intervals.length) {
    return []
  }

  const expanded = intervals.flatMap((interval) => {
    if (interval.end < interval.start) {
      return [
        { start: interval.start, end: FULL_CIRCLE },
        { start: 0, end: interval.end },
      ]
    }
    return [interval]
  })

  const sorted = expanded
    .map((interval) => ({
      start: clamp(interval.start, 0, FULL_CIRCLE),
      end: clamp(interval.end, 0, FULL_CIRCLE),
    }))
    .sort((left, right) => left.start - right.start)

  return sorted.reduce((merged, interval) => {
    if (!merged.length) {
      merged.push(interval)
      return merged
    }

    const previous = merged[merged.length - 1]
    if (interval.start <= previous.end + 0.02) {
      previous.end = Math.max(previous.end, interval.end)
      return merged
    }

    merged.push(interval)
    return merged
  }, [])
}

function pickExpansionSector(anchor, occupiedNodes = [], options = {}) {
  const fallbackAngle = Number.isFinite(options.fallbackAngle) ? options.fallbackAngle : 0
  const preferredAngle = resolvePreferredAngle(anchor, options)
  const occupiedIntervals = buildOccupiedAngularIntervals(anchor, occupiedNodes)

  if (!occupiedIntervals.length) {
    return {
      center: preferredAngle ?? normalizeAngle(fallbackAngle),
      gap: FULL_CIRCLE,
    }
  }

  const gaps = []

  for (let index = 0; index < occupiedIntervals.length; index += 1) {
    const current = occupiedIntervals[index]
    const next = occupiedIntervals[(index + 1) % occupiedIntervals.length]
    const start = current.end
    const end = index === occupiedIntervals.length - 1 ? next.start + FULL_CIRCLE : next.start
    const gap = end - start
    if (gap <= 0.01) continue
    gaps.push({
      start,
      end,
      gap,
      center: normalizeAngle(start + gap / 2),
    })
  }

  if (!gaps.length) {
    return {
      center: preferredAngle ?? normalizeAngle(fallbackAngle),
      gap: Math.PI * 0.18,
    }
  }

  if (preferredAngle != null) {
    const matchingGap = gaps
      .filter((item) => angleInsideGap(preferredAngle, item.start, item.end))
      .sort((left, right) => right.gap - left.gap)[0]

    if (matchingGap && matchingGap.gap >= Math.PI * 0.34) {
      const preferredMargin = Math.min(matchingGap.gap * 0.18, 0.32)
      return {
        center: clampAngleToGap(preferredAngle, matchingGap, preferredMargin),
        gap: matchingGap.gap,
      }
    }
  }

  const ranked = gaps
    .map((item) => ({
      ...item,
      score:
        item.gap * 1.45 -
        (preferredAngle == null ? angleDistance(item.center, fallbackAngle) * 0.08 : angleDistance(item.center, preferredAngle) * 0.85),
    }))
    .sort((left, right) => right.score - left.score)

  if (ranked.length) {
    return {
      center: ranked[0].center,
      gap: ranked[0].gap,
    }
  }

  return {
    center: preferredAngle ?? normalizeAngle(fallbackAngle),
    gap: FULL_CIRCLE,
  }
}

export function createExpansionPlacement(anchorNode, childCount = 0, existingCount = 0, options = {}) {
  const count = Math.max(0, Number(childCount || 0))
  if (!count) return []

  const anchor = anchorNode || { x: 0, y: 0, collisionRadius: 48, id: 'anchor' }
  const anchorRadius = getNodeVisualRadius(anchor)
  const baseRadius = clamp(
    anchorRadius + 118 + Math.max(0, existingCount - 1) * 7,
    176,
    420,
  )
  const angleOffset = ((stableHash(anchor.id) % 360) * Math.PI) / 180
  const nearbyOccupiedNodes = (options.occupiedNodes || []).filter((node) => {
    if (!node || node.id === anchor.id || !Number.isFinite(node.x) || !Number.isFinite(node.y)) return false
    const distance = Math.hypot(node.x - anchor.x, node.y - anchor.y)
    return distance <= baseRadius + getNodeVisualRadius(node) + 220
  })
  const sector = pickExpansionSector(anchor, nearbyOccupiedNodes, {
    fallbackAngle: angleOffset,
    preferredAngle: options.preferredAngle,
    preferredPoint: options.preferredPoint,
    viewportCenterWorld: options.viewportCenterWorld,
  })
  const desiredSpan = Math.PI * (0.56 + Math.min(count, 12) * 0.038)
  const span = clamp(Math.min(desiredSpan, Math.max(sector.gap * 0.84, Math.PI * 0.54)), Math.PI * 0.54, Math.PI * 1.18)
  const centerAngle = sector.center
  const desiredSpacing = clamp(anchorRadius * 0.92 + 34, 82, 118)
  const ringSpacing = clamp(desiredSpacing * 0.94, 68, 102)

  const rings = []
  let remaining = count
  let ringIndex = 0

  while (remaining > 0) {
    const radius = baseRadius + ringIndex * ringSpacing
    const usableSpan = Math.max(span - 0.14, Math.PI * 0.32)
    const capacity = clamp(Math.floor((usableSpan * radius) / desiredSpacing), 4, 9)
    const size = Math.min(remaining, capacity)
    rings.push({ ringIndex, radius, size })
    remaining -= size
    ringIndex += 1
  }

  return rings.flatMap(({ ringIndex: currentRing, radius, size }) => {
    const ringSpan = clamp(span - currentRing * 0.06, Math.PI * 0.38, span)
    const anglePadding = Math.min(ringSpan * 0.12, 0.22)
    const localStart = centerAngle - ringSpan / 2 + anglePadding
    const localSpan = Math.max(ringSpan - anglePadding * 2, Math.PI * 0.16)
    const ringOffset = currentRing % 2 === 0 ? 0 : localSpan / Math.max(size * 2, 3)

    return Array.from({ length: size }, (_, index) => {
      const ratio = size === 1 ? 0.5 : index / (size - 1)
      const angle = localStart + localSpan * ratio + ringOffset
      return {
        x: anchor.x + Math.cos(angle) * radius,
        y: anchor.y + Math.sin(angle) * radius,
        angle,
        radius,
      }
    })
  })
}

export function attachExpansionBias(engineNodes = [], anchorPoint = { x: 0, y: 0 }, nodeIds = [], duration = 1800) {
  const expiresAt = Date.now() + duration
  const idSet = new Set(nodeIds)
  engineNodes.forEach((node) => {
    if (!idSet.has(node.id)) return
    node.expandAnchor = {
      x: anchorPoint.x,
      y: anchorPoint.y,
      strength: 0.05,
      expiresAt,
    }
  })
}

export function getRelationPriority(relationType = '') {
  return RELATION_PRIORITY[relationType] ?? DEFAULT_RELATION_PRIORITY
}
