import cytoscape from 'cytoscape'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function safeNumber(value, fallback = 0) {
  return Number.isFinite(value) ? value : fallback
}

function hashString(input = '') {
  let hash = 0
  const text = String(input || '')
  for (let index = 0; index < text.length; index += 1) {
    hash = (hash * 31 + text.charCodeAt(index)) % 2147483647
  }
  return hash
}

function getViewportSize(container) {
  if (!container) return null
  const width = container.clientWidth
  const height = container.clientHeight
  if (width < 320 || height < 240) return null
  return { width, height }
}

function normalizeLabel(label = '') {
  return String(label || '').replace(/\s*\n\s*/g, '\n').trim()
}

function hexToRgba(color, alpha = 1) {
  const normalized = String(color || '').trim().replace('#', '')
  if (!normalized) return `rgba(148, 163, 184, ${alpha})`
  if (normalized.startsWith('rgb')) {
    const match = normalized.match(/rgba?\(([^)]+)\)/i)
    if (!match) return normalized
    const [r, g, b] = match[1].split(',').map((part) => Number.parseFloat(part.trim()) || 0)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }

  let hex = normalized
  if (hex.length === 3 || hex.length === 4) {
    hex = hex.split('').map((item) => `${item}${item}`).join('')
  }
  if (hex.length === 8) hex = hex.slice(0, 6)
  if (!/^[0-9a-fA-F]{6}$/.test(hex)) return `rgba(148, 163, 184, ${alpha})`

  const r = Number.parseInt(hex.slice(0, 2), 16)
  const g = Number.parseInt(hex.slice(2, 4), 16)
  const b = Number.parseInt(hex.slice(4, 6), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

function polarToCartesian(center, radius, angle) {
  return {
    x: safeNumber(center.x + Math.cos(angle) * radius, center.x),
    y: safeNumber(center.y + Math.sin(angle) * radius, center.y),
  }
}

function clampPointToCircle(point, maxRadius) {
  const distance = Math.hypot(point.x, point.y)
  if (!distance || distance <= maxRadius) return point
  const ratio = maxRadius / distance
  return { x: point.x * ratio, y: point.y * ratio }
}

function getNodeCollisionRadius(node) {
  return Number(node?.data?.size || 58) / 2 + 20
}

function resolveCollisions(nodes = [], positions = new Map(), sceneRadius = 420, lockedIds = new Set(), maxIterations = 72) {
  const nodeList = [...nodes]

  for (let iteration = 0; iteration < maxIterations; iteration += 1) {
    let moved = false

    for (let index = 0; index < nodeList.length; index += 1) {
      const nodeA = nodeList[index]
      const pointA = positions.get(nodeA.id)
      if (!pointA) continue

      for (let otherIndex = index + 1; otherIndex < nodeList.length; otherIndex += 1) {
        const nodeB = nodeList[otherIndex]
        const pointB = positions.get(nodeB.id)
        if (!pointB) continue

        const dx = pointB.x - pointA.x
        const dy = pointB.y - pointA.y
        const distance = Math.hypot(dx, dy) || 0.0001
        const minDistance = getNodeCollisionRadius(nodeA) + getNodeCollisionRadius(nodeB)

        if (distance >= minDistance) continue

        const lockA = lockedIds.has(nodeA.id)
        const lockB = lockedIds.has(nodeB.id)
        if (lockA && lockB) continue

        moved = true
        const overlap = minDistance - distance
        const ux = dx / distance
        const uy = dy / distance

        if (lockA) {
          pointB.x += ux * overlap
          pointB.y += uy * overlap
        } else if (lockB) {
          pointA.x -= ux * overlap
          pointA.y -= uy * overlap
        } else {
          const halfOverlap = overlap / 2
          pointA.x -= ux * halfOverlap
          pointA.y -= uy * halfOverlap
          pointB.x += ux * halfOverlap
          pointB.y += uy * halfOverlap
        }
      }
    }

    nodeList.forEach((node) => {
      const point = positions.get(node.id)
      if (!point) return
      const limit = Math.max(40, sceneRadius - getNodeCollisionRadius(node))
      const clamped = clampPointToCircle(point, limit)
      point.x = clamped.x
      point.y = clamped.y
    })

    if (!moved) break
  }

  return positions
}

function getSceneRadius(nodeCount = 1) {
  return clamp(260 + Math.sqrt(nodeCount || 1) * 22, 320, 540)
}

function getNodePoint(position = null) {
  return {
    x: safeNumber(position?.x, 0),
    y: safeNumber(position?.y, 0),
  }
}

function buildLocalPrioritySet(nodes = [], selectedNodeId = '') {
  const normalizedSelectedId = String(selectedNodeId || '').trim()
  if (!normalizedSelectedId) return new Set()

  const nodeMap = new Map(nodes.map((node) => [node.id, node]))
  const selectedNode = nodeMap.get(normalizedSelectedId)
  if (!selectedNode) return new Set()

  const localIds = new Set([normalizedSelectedId])
  const selectedRootId = selectedNode?.data?.rootId || normalizedSelectedId

  let cursorId = normalizedSelectedId
  while (cursorId) {
    const cursorNode = nodeMap.get(cursorId)
    if (!cursorNode) break
    localIds.add(cursorId)
    cursorId = String(cursorNode?.data?.anchorId || '').trim()
  }

  nodes.forEach((node) => {
    const nodeRootId = node?.data?.rootId || node.id
    if (nodeRootId === selectedRootId) {
      localIds.add(node.id)
      return
    }
    if (localIds.has(String(node?.data?.anchorId || '').trim())) {
      localIds.add(node.id)
    }
  })

  return localIds
}

function pushPeripheralNodesOutward(nodes = [], positions = new Map(), localIds = new Set(), selectedNodeId = '', sceneRadius = 420) {
  const normalizedSelectedId = String(selectedNodeId || '').trim()
  if (!normalizedSelectedId || !positions.has(normalizedSelectedId)) return

  const focusPoint = positions.get(normalizedSelectedId)
  const localCount = Math.max(1, localIds.size)
  const minPeripheralDistance = clamp(210 + Math.sqrt(localCount) * 24, 240, 420)

  nodes.forEach((node, index) => {
    if (localIds.has(node.id)) return
    const point = positions.get(node.id)
    if (!point) return

    let dx = point.x - focusPoint.x
    let dy = point.y - focusPoint.y
    let distance = Math.hypot(dx, dy)

    if (distance < 0.001) {
      const angle = ((hashString(`${node.id}:${index}`) % 360) * Math.PI) / 180
      dx = Math.cos(angle)
      dy = Math.sin(angle)
      distance = 1
    }

    if (distance >= minPeripheralDistance) return

    const extraOffset = (hashString(node.id) % 48) + index * 2
    const targetDistance = minPeripheralDistance + extraOffset
    point.x = focusPoint.x + (dx / distance) * targetDistance
    point.y = focusPoint.y + (dy / distance) * targetDistance

    const clamped = clampPointToCircle(point, sceneRadius)
    point.x = clamped.x
    point.y = clamped.y
  })
}

function buildCircularPositions(nodes = [], persistedPositions = new Map(), options = {}) {
  const { selectedNodeId = '' } = options
  const positions = new Map()
  const lockedIds = new Set()
  const localPriorityIds = buildLocalPrioritySet(nodes, selectedNodeId)
  const roots = nodes.filter((node) => node?.data?.isRoot || !node?.data?.anchorId)
  const childrenByAnchor = new Map()

  nodes.forEach((node) => {
    const anchorId = node?.data?.anchorId
    if (!anchorId) return
    if (!childrenByAnchor.has(anchorId)) childrenByAnchor.set(anchorId, [])
    childrenByAnchor.get(anchorId).push(node)
  })

  const hasFocusedCluster = Boolean(localPriorityIds.size)
  const sceneRadius = getSceneRadius(nodes.length) + (hasFocusedCluster ? 90 : 0)
  const rootCount = roots.length || 1
  const innerRingCount = rootCount <= 10 ? rootCount : Math.ceil(rootCount * 0.5)
  const outerRingCount = Math.max(rootCount - innerRingCount, 0)
  const innerRadius = clamp(sceneRadius * 0.3, 110, 190)
  const outerRadius = clamp(sceneRadius * 0.5, 180, 270)

  roots.forEach((node, index) => {
    if (persistedPositions.has(node.id)) {
      positions.set(node.id, getNodePoint(persistedPositions.get(node.id)))
      if (!hasFocusedCluster || localPriorityIds.has(node.id)) lockedIds.add(node.id)
      return
    }

    const onInnerRing = index < innerRingCount
    const ringIndex = onInnerRing ? index : index - innerRingCount
    const ringCount = onInnerRing ? innerRingCount : outerRingCount
    const angle = -Math.PI / 2 + (Math.PI * 2 * ringIndex) / Math.max(ringCount, 1)
    const radius = onInnerRing ? innerRadius : outerRadius
    positions.set(node.id, polarToCartesian({ x: 0, y: 0 }, radius, angle))
  })

  const queue = [...roots.map((node) => node.id)]
  const visited = new Set(queue)
  while (queue.length) {
    const anchorId = queue.shift()
    const anchorPos = positions.get(anchorId) || { x: 0, y: 0 }
    const children = childrenByAnchor.get(anchorId) || []
    const count = children.length || 1
    const anchorAngle = Math.atan2(anchorPos.y, anchorPos.x) || ((hashString(anchorId) % 360) * Math.PI) / 180
    const arcSpan = Math.min(Math.PI * 1.2, Math.PI / 2 + count * 0.16)
    const baseChildRadius = clamp(80 + count * 6, 84, 128)

    children.forEach((node, index) => {
      if (persistedPositions.has(node.id)) {
        positions.set(node.id, getNodePoint(persistedPositions.get(node.id)))
        if (!hasFocusedCluster || localPriorityIds.has(node.id)) lockedIds.add(node.id)
        if (!visited.has(node.id)) {
          visited.add(node.id)
          queue.push(node.id)
        }
        return
      }

      const ringIndex = Math.floor(index / 8)
      const indexInRing = index % 8
      const ringCount = Math.min(8, count - ringIndex * 8) || 1
      const angleOffset = ringCount === 1 ? 0 : -arcSpan / 2 + (arcSpan * indexInRing) / Math.max(ringCount - 1, 1)
      const angle = anchorAngle + angleOffset
      const radius = baseChildRadius + ringIndex * 46
      const point = clampPointToCircle(polarToCartesian(anchorPos, radius, angle), sceneRadius)
      positions.set(node.id, point)

      if (!visited.has(node.id)) {
        visited.add(node.id)
        queue.push(node.id)
      }
    })
  }

  nodes.forEach((node, index) => {
    if (positions.has(node.id)) return
    if (persistedPositions.has(node.id)) {
      positions.set(node.id, getNodePoint(persistedPositions.get(node.id)))
      if (!hasFocusedCluster || localPriorityIds.has(node.id)) lockedIds.add(node.id)
      return
    }
    const angle = ((hashString(node.id) % 360) * Math.PI) / 180
    positions.set(node.id, clampPointToCircle(polarToCartesian({ x: 0, y: 0 }, 100 + index * 10, angle), sceneRadius))
  })

  if (hasFocusedCluster) {
    pushPeripheralNodesOutward(nodes, positions, localPriorityIds, selectedNodeId, sceneRadius)
  }

  return resolveCollisions(nodes, positions, sceneRadius, lockedIds)
}

function buildNodeVisual(node, states = [], flashedIds = new Set()) {
  const style = node?.style || {}
  const size = Number(style.size || node?.data?.size || 58)
  const isRoot = Boolean(node?.data?.isRoot)
  const isExpanded = Boolean(node?.data?.isExpanded)

  const visual = {
    color: style.fill || 'rgba(241, 234, 255, 0.99)',
    borderColor: style.stroke || 'rgba(196, 181, 253, 1)',
    borderWidth: Number(style.lineWidth || 1.4),
    size,
    opacity: 1,
    label: normalizeLabel(style.labelText || node?.data?.fullLabel || node?.data?.label || node.id),
    textColor: style.labelFill || '#111827',
    fontSize: Number(style.labelFontSize || (isRoot ? 12.4 : 10.8)),
    shadowColor: style.shadowColor || '#C4B5FD',
    shadowBlur: Number(style.shadowBlur || (isRoot ? 12 : 6)),
    shadowOpacity: Number(style.shadowOpacity || (isRoot ? 0.2 : 0.1)),
    zIndex: isRoot ? 30 : 18,
    shape: 'ellipse',
    textMaxWidth: Number(node?.data?.textMaxWidth || size * 0.88),
  }

  if (states.includes('dimmed')) {
    visual.opacity = node?.data?.isRoot ? 0.74 : 0.52
    visual.shadowOpacity *= 0.55
  }
  if (states.includes('active')) {
    visual.borderColor = 'rgba(226, 232, 240, 1)'
    visual.borderWidth += 0.7
    visual.shadowOpacity = Math.min(0.72, visual.shadowOpacity + 0.18)
    visual.shadowBlur += 8
    visual.zIndex += 10
  }
  if (isExpanded) {
    visual.borderColor = 'rgba(103, 232, 249, 1)'
    visual.borderWidth = Math.max(visual.borderWidth, 2.8)
    visual.shadowColor = '#67E8F9'
    visual.shadowOpacity = Math.max(visual.shadowOpacity, 0.34)
    visual.shadowBlur = Math.max(visual.shadowBlur, 18)
    visual.zIndex += 12
  }
  if (states.includes('selected')) {
    visual.borderColor = 'rgba(103, 232, 249, 1)'
    visual.borderWidth = Math.max(visual.borderWidth, 3.2)
    visual.shadowColor = '#67E8F9'
    visual.shadowOpacity = 0.44
    visual.shadowBlur = 22
    visual.zIndex += 18
  }
  if (states.includes('path')) {
    visual.borderColor = '#FBBF24'
    visual.shadowColor = '#FBBF24'
    visual.shadowOpacity = 0.42
    visual.shadowBlur = 18
    visual.zIndex += 14
  }
  if (flashedIds.has(node.id)) {
    visual.borderColor = '#A5F3FC'
    visual.shadowColor = '#A5F3FC'
    visual.shadowOpacity = 0.6
    visual.shadowBlur = 24
    visual.zIndex += 20
  }

  return visual
}

function buildEdgeVisual(edge, states = []) {
  const style = edge?.style || {}
  const isPath = states.includes('path')
  const isActive = states.includes('active')
  const isDimmed = states.includes('dimmed')
  const isCrossCourse = Boolean(edge?.data?.crossCourse)

  return {
    color: isPath
      ? 'rgba(251, 191, 36, 0.98)'
      : isActive
        ? 'rgba(255, 255, 255, 0.96)'
        : hexToRgba(style.stroke || '#E2E8F0', Number(style.opacity ?? 0.8)),
    width: Number(style.lineWidth || 1.5) * (isPath ? 1.4 : isActive ? 1.2 : 1),
    opacity: isDimmed && !isPath ? 0.2 : Number(style.opacity ?? 0.8),
    lineStyle: Array.isArray(style.lineDash) && style.lineDash.length ? 'dashed' : 'solid',
    targetArrow: style.endArrow ? 'triangle' : 'none',
    curveStyle: isCrossCourse ? 'bezier' : 'straight',
    zIndex: isPath ? 20 : isCrossCourse ? 14 : 10,
    label: String(edge?.data?.label || ''),
    labelColor: isPath ? '#FDE68A' : isActive ? '#F8FAFC' : 'rgba(226, 232, 240, 0.96)',
    labelBackground: isPath ? 'rgba(120, 53, 15, 0.92)' : 'rgba(15, 23, 42, 0.78)',
  }
}

function buildCyElements(data, stateLookup = {}, flashedIds = new Set(), persistedPositions = new Map(), options = {}) {
  const positions = buildCircularPositions(data?.nodes || [], persistedPositions, options)
  const elements = []

  ;(data?.nodes || []).forEach((node) => {
    const states = stateLookup?.[node.id] || []
    const visual = buildNodeVisual(node, states, flashedIds)
    const pos = positions.get(node.id) || { x: 0, y: 0 }
    const fullLabel = normalizeLabel(node?.data?.fullLabel || node?.data?.label || node.id)
    const shortLabel = normalizeLabel(node?.data?.shortLabel || fullLabel)

    elements.push({
      group: 'nodes',
      data: {
        id: node.id,
        color: visual.color,
        borderColor: visual.borderColor,
        borderWidth: visual.borderWidth,
        size: visual.size,
        opacity: visual.opacity,
        textColor: visual.textColor,
        fontSize: visual.fontSize,
        zIndex: visual.zIndex,
        shadowColor: visual.shadowColor,
        shadowBlur: visual.shadowBlur,
        shadowOpacity: visual.shadowOpacity,
        shape: visual.shape,
        textMaxWidth: visual.textMaxWidth,
        fullLabel,
        shortLabel,
        displayLabel: fullLabel,
        isRoot: Boolean(node?.data?.isRoot),
        isExpanded: Boolean(node?.data?.isExpanded),
        layoutDepth: Number(node?.data?.layoutDepth || 0),
        anchorId: node?.data?.anchorId || '',
        chapterTitle: node?.data?.chapterTitle || '',
        isSelected: states.includes('selected'),
        isActive: states.includes('active'),
        isPath: states.includes('path'),
      },
      position: { x: safeNumber(pos.x, 0), y: safeNumber(pos.y, 0) },
      classes: node?.data?.isRoot ? 'root' : 'leaf',
    })
  })

  ;(data?.edges || []).forEach((edge) => {
    const states = stateLookup?.[edge.id] || []
    const visual = buildEdgeVisual(edge, states)
    elements.push({
      group: 'edges',
      data: {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        color: visual.color,
        width: visual.width,
        opacity: visual.opacity,
        lineStyle: visual.lineStyle,
        targetArrow: visual.targetArrow,
        curveStyle: visual.curveStyle,
        zIndex: visual.zIndex,
        relationLabel: visual.label,
        displayLabel: visual.label,
        labelColor: visual.labelColor,
        labelBackground: visual.labelBackground,
        isActive: states.includes('active'),
        isPath: states.includes('path'),
      },
    })
  })

  return elements
}

function buildCyStyle() {
  return [
    {
      selector: 'node',
      style: {
        shape: 'data(shape)',
        width: 'data(size)',
        height: 'data(size)',
        'background-color': 'data(color)',
        'border-color': 'data(borderColor)',
        'border-width': 'data(borderWidth)',
        label: 'data(displayLabel)',
        color: 'data(textColor)',
        opacity: 'data(opacity)',
        'font-size': 'data(fontSize)',
        'font-weight': 600,
        'text-valign': 'center',
        'text-halign': 'center',
        'text-wrap': 'wrap',
        'text-max-width': 'data(textMaxWidth)',
        'text-margin-y': 0,
        'overlay-opacity': 0,
        'shadow-color': 'data(shadowColor)',
        'shadow-opacity': 'data(shadowOpacity)',
        'shadow-blur': 'data(shadowBlur)',
        'shadow-offset-x': 0,
        'shadow-offset-y': 0,
        'z-index': 'data(zIndex)',
        'transition-property': 'width height border-width background-color border-color opacity shadow-blur shadow-opacity',
        'transition-duration': '180ms',
        'transition-timing-function': 'ease-out',
      },
    },
    {
      selector: 'edge',
      style: {
        width: 'data(width)',
        'line-color': 'data(color)',
        'target-arrow-color': 'data(color)',
        'target-arrow-shape': 'data(targetArrow)',
        'line-style': 'data(lineStyle)',
        opacity: 'data(opacity)',
        'curve-style': 'data(curveStyle)',
        'overlay-opacity': 0,
        'z-index': 'data(zIndex)',
        'transition-property': 'line-color width opacity',
        'transition-duration': '160ms',
        'transition-timing-function': 'ease-out',
        label: 'data(displayLabel)',
        color: 'data(labelColor)',
        'font-size': 9,
        'font-weight': 600,
        'text-rotation': 'autorotate',
        'text-wrap': 'none',
        'text-background-color': 'data(labelBackground)',
        'text-background-opacity': 0.9,
        'text-background-padding': '2px',
        'text-background-shape': 'roundrectangle',
        'text-border-opacity': 0,
        'edge-text-rotation': 'autorotate',
      },
    },
  ]
}

function getStructureSignature(data = { nodes: [], edges: [] }) {
  const nodeIds = (data.nodes || []).map((item) => item.id).sort().join('|')
  const edgeIds = (data.edges || []).map((item) => item.id).sort().join('|')
  return `${nodeIds}__${edgeIds}`
}

export function useGraph({
  data,
  stateMap,
  selectedNodeId,
  onNodeClick,
  onNodeDoubleClick,
  onHubClick,
  onHubDoubleClick,
  onCanvasClick,
  onNodeEnter,
  onNodeLeave,
  onNodeContextMenu,
}) {
  const containerRef = ref(null)
  const graphMountRef = ref(null)
  const graphRef = ref(null)
  const renderError = ref('')
  const zoom = ref(1)

  let resizeObserver = null
  let renderFrame = null
  let renderInFlight = false
  let flashTimer = null
  let labelLod = ''
  let hasInitialFit = false
  let tapIntent = null
  let lastStructureSignature = ''
  let dragCollisionFrame = null
  let springMotionFrame = null
  let pendingDraggedNodeId = ''
  let pendingLockedDragIds = new Set()
  let dragContext = null
  let activeSpringMotion = null
  const flashedIds = new Set()
  const positionMemory = new Map()

  function clearTapIntent() {
    if (tapIntent?.timer) window.clearTimeout(tapIntent.timer)
    tapIntent = null
  }

  function destroyGraph() {
    clearTapIntent()
    if (dragCollisionFrame) {
      window.cancelAnimationFrame(dragCollisionFrame)
      dragCollisionFrame = null
    }
    if (springMotionFrame) {
      window.cancelAnimationFrame(springMotionFrame)
      springMotionFrame = null
    }
    pendingDraggedNodeId = ''
    pendingLockedDragIds = new Set()
    dragContext = null
    activeSpringMotion = null
    graphRef.value?.destroy?.()
    graphRef.value = null
    hasInitialFit = false
    labelLod = ''
    lastStructureSignature = ''
    positionMemory.clear()
  }

  function updateZoom() {
    zoom.value = safeNumber(graphRef.value?.zoom?.(), 1)
  }

  function toClientEvent(event, targetId) {
    const original = event?.originalEvent
    const bounds = containerRef.value?.getBoundingClientRect?.() || { left: 0, top: 0 }
    const rendered = event?.renderedPosition || { x: 0, y: 0 }
    const clientX = original?.clientX ?? bounds.left + rendered.x
    const clientY = original?.clientY ?? bounds.top + rendered.y

    return {
      target: { id: targetId },
      client: { x: clientX, y: clientY },
      clientX,
      clientY,
      preventDefault: () => original?.preventDefault?.(),
    }
  }

  function getLabelLod(nextZoom) {
    if (nextZoom < 0.42) return 'low'
    if (nextZoom < 0.8) return 'medium'
    return 'high'
  }

  function applyLabelLod(force = false) {
    const cy = graphRef.value
    if (!cy) return

    const currentZoom = safeNumber(cy.zoom(), 1)
    const totalNodes = cy.nodes().length
    const nextLod = getLabelLod(currentZoom)
    if (!force && nextLod === labelLod) return
    labelLod = nextLod

    cy.batch(() => {
      cy.nodes().forEach((node) => {
        const fullLabel = normalizeLabel(node.data('fullLabel') || '')
        const shortLabel = normalizeLabel(node.data('shortLabel') || fullLabel)
        const shouldForceShow = Boolean(node.data('isSelected') || node.data('isActive') || node.data('isPath') || node.data('isExpanded') || node.data('isRoot'))
        const smallScene = totalNodes <= 22

        let displayLabel = fullLabel
        if (nextLod === 'low') {
          displayLabel = shouldForceShow || smallScene ? shortLabel : ''
        } else if (nextLod === 'medium') {
          displayLabel = shouldForceShow ? fullLabel : smallScene ? fullLabel : shortLabel
        }
        node.data('displayLabel', displayLabel)
      })

      cy.edges().forEach((edge) => {
        const relationLabel = normalizeLabel(edge.data('relationLabel') || '')
        if (!relationLabel) {
          edge.data('displayLabel', '')
          return
        }

        const shouldForceShow = Boolean(edge.data('isActive') || edge.data('isPath'))
        let displayLabel = relationLabel
        if (nextLod === 'low') {
          displayLabel = ''
        } else if (nextLod === 'medium') {
          displayLabel = shouldForceShow || totalNodes <= 16 ? relationLabel : ''
        }
        edge.data('displayLabel', displayLabel)
      })
    })
  }

  function snapshotNodePositions(cy) {
    const positions = new Map()
    cy?.nodes?.().forEach((node) => {
      positions.set(node.id(), getNodePoint(node.position()))
    })
    return positions
  }

  function persistNodePositions(cy) {
    if (!cy) return
    const activeIds = new Set()
    cy.nodes().forEach((node) => {
      activeIds.add(node.id())
      positionMemory.set(node.id(), getNodePoint(node.position()))
    })

    Array.from(positionMemory.keys()).forEach((nodeId) => {
      if (!activeIds.has(nodeId)) positionMemory.delete(nodeId)
    })
  }

  function enforceGlobalNodeSeparation(cy, lockedIds = new Set(), maxIterations = 24) {
    if (!cy?.nodes?.().length) return

    const nodes = cy.nodes().map((node) => ({
      id: node.id(),
      data: { size: node.data('size') },
    }))
    const positions = snapshotNodePositions(cy)
    const resolvedPositions = resolveCollisions(
      nodes,
      positions,
      getSceneRadius(nodes.length) + 140,
      lockedIds,
      maxIterations,
    )

    cy.batch(() => {
      cy.nodes().forEach((node) => {
        if (lockedIds.has(node.id())) return
        const nextPoint = resolvedPositions.get(node.id())
        if (!nextPoint) return
        node.position(nextPoint)
      })
    })
  }

  function stopSpringMotion(persist = true) {
    if (springMotionFrame) {
      window.cancelAnimationFrame(springMotionFrame)
      springMotionFrame = null
    }
    if (persist && graphRef.value) {
      persistNodePositions(graphRef.value)
    }
    activeSpringMotion = null
  }

  function buildDragGroupState(cy, anchorId) {
    const normalizedAnchorId = String(anchorId || '').trim()
    if (!cy || !normalizedAnchorId) {
      return {
        dragIds: new Set(),
        depthMap: new Map(),
        basePositions: new Map(),
      }
    }

    const childrenByAnchor = new Map()
    cy.nodes().forEach((node) => {
      const parentId = String(node.data('anchorId') || '').trim()
      if (!parentId) return
      if (!childrenByAnchor.has(parentId)) childrenByAnchor.set(parentId, [])
      childrenByAnchor.get(parentId).push(node.id())
    })

    const dragIds = new Set([normalizedAnchorId])
    const queue = [normalizedAnchorId]
    const depthMap = new Map([[normalizedAnchorId, 0]])
    while (queue.length) {
      const currentId = queue.shift()
      const parentDepth = depthMap.get(currentId) || 0
      ;(childrenByAnchor.get(currentId) || []).forEach((childId) => {
        if (dragIds.has(childId)) return
        dragIds.add(childId)
        depthMap.set(childId, parentDepth + 1)
        queue.push(childId)
      })
    }

    const basePositions = new Map()
    dragIds.forEach((nodeId) => {
      const node = cy.getElementById(nodeId)
      if (!node?.length) return
      basePositions.set(nodeId, getNodePoint(node.position()))
    })

    return {
      dragIds,
      depthMap,
      basePositions,
    }
  }

  function createSpringMotion(cy, nodeId) {
    const groupState = buildDragGroupState(cy, nodeId)
    const anchorNode = cy.getElementById(nodeId)
    const anchorStartPosition = getNodePoint(anchorNode?.position?.())
    const relativeOffsets = new Map()
    const followerStates = new Map()

    groupState.dragIds.forEach((dragId) => {
      if (dragId === nodeId) return
      const basePoint = groupState.basePositions.get(dragId)
      if (!basePoint) return
      relativeOffsets.set(dragId, {
        x: basePoint.x - anchorStartPosition.x,
        y: basePoint.y - anchorStartPosition.y,
      })
      followerStates.set(dragId, { vx: 0, vy: 0 })
    })

    return {
      nodeId,
      dragIds: groupState.dragIds,
      depthMap: groupState.depthMap,
      basePositions: groupState.basePositions,
      relativeOffsets,
      followerStates,
      anchorStartPosition,
      pointerDown: true,
      anchorVelocity: { x: 0, y: 0 },
      lastDirection: { x: 1, y: 0 },
      phase: 0,
      lastTick: performance.now(),
    }
  }

  function stepSpringMotion(cy, motion, dt) {
    const anchorNode = cy.getElementById(motion.nodeId)
    if (!anchorNode?.length) return false

    const dtRatio = clamp(dt / 16, 0.65, 1.35)
    const anchorCurrent = getNodePoint(anchorNode.position())

    if (!motion.pointerDown) {
      const inertiaFriction = Math.pow(0.8, dtRatio)
      motion.anchorVelocity.x *= inertiaFriction
      motion.anchorVelocity.y *= inertiaFriction
      anchorNode.position({
        x: anchorCurrent.x + motion.anchorVelocity.x * dt,
        y: anchorCurrent.y + motion.anchorVelocity.y * dt,
      })
    }

    const anchorPosition = getNodePoint(anchorNode.position())
    const speed = Math.hypot(motion.anchorVelocity.x, motion.anchorVelocity.y)
    if (speed > 0.0001) {
      motion.lastDirection = {
        x: motion.anchorVelocity.x / speed,
        y: motion.anchorVelocity.y / speed,
      }
    }

    const direction = motion.lastDirection
    const tangent = { x: -direction.y, y: direction.x }
    const speedRatio = clamp(speed * 1.2, 0, 0.9)
    motion.phase += dt * (0.004 + speedRatio * 0.002)

    let residualEnergy = speed

    cy.batch(() => {
      cy.nodes().forEach((node) => {
        const nodeId = node.id()
        if (nodeId === motion.nodeId || !motion.dragIds.has(nodeId)) return

        const relative = motion.relativeOffsets.get(nodeId)
        const followerState = motion.followerStates.get(nodeId)
        if (!relative || !followerState) return

        const current = getNodePoint(node.position())
        const depth = motion.depthMap.get(nodeId) || 1
        const depthRatio = clamp(depth * 0.16, 0.12, 0.82)
        const hashPhase = ((hashString(nodeId) % 360) * Math.PI) / 180
        const stretch = (5 + depth * 3.5) * speedRatio * depthRatio
        const sway = (2.5 + depth * 1.6) * speedRatio * (0.45 + depthRatio * 0.4)
        const breathe = Math.sin(motion.phase + hashPhase)
        const swayOffset = sway * breathe

        const target = {
          x: anchorPosition.x + relative.x - direction.x * stretch + tangent.x * swayOffset,
          y: anchorPosition.y + relative.y - direction.y * stretch + tangent.y * swayOffset,
        }

        const stiffness = 0.06 + (1 - depthRatio) * 0.025
        const damping = 0.68 - depthRatio * 0.05

        followerState.vx = (followerState.vx + (target.x - current.x) * stiffness * dtRatio) * damping
        followerState.vy = (followerState.vy + (target.y - current.y) * stiffness * dtRatio) * damping

        const nextPoint = {
          x: current.x + followerState.vx * dtRatio,
          y: current.y + followerState.vy * dtRatio,
        }

        node.position(nextPoint)

        residualEnergy +=
          Math.abs(target.x - nextPoint.x) * 0.01 +
          Math.abs(target.y - nextPoint.y) * 0.01 +
          Math.hypot(followerState.vx, followerState.vy) * 0.08
      })
    })

    enforceGlobalNodeSeparation(cy, new Set([motion.nodeId]), 26)
    persistNodePositions(cy)

    const continueMotion = motion.pointerDown || residualEnergy > 0.14
    return continueMotion
  }

  function ensureSpringMotionLoop() {
    if (springMotionFrame || !activeSpringMotion) return

    const tick = (now) => {
      springMotionFrame = null
      const cy = graphRef.value
      const motion = activeSpringMotion
      if (!cy || !motion) return

      const dt = clamp(now - (motion.lastTick || now), 8, 28)
      motion.lastTick = now

      const shouldContinue = stepSpringMotion(cy, motion, dt)
      if (shouldContinue) {
        springMotionFrame = window.requestAnimationFrame(tick)
      } else {
        stopSpringMotion(true)
      }
    }

    springMotionFrame = window.requestAnimationFrame(tick)
  }

  function applyElasticDragGroupMotion(cy, context, anchorPosition, delta = { x: 0, y: 0 }, velocity = { x: 0, y: 0 }, speed = 0) {
    if (!cy || !context?.nodeId || !context?.dragIds?.size) return

    const dragDistance = Math.hypot(delta.x, delta.y)
    const speedRatio = clamp(speed / 2.8, 0, 0.75)
    const velocityLength = Math.hypot(velocity.x, velocity.y) || 1
    const direction =
      velocityLength > 0.001
        ? { x: velocity.x / velocityLength, y: velocity.y / velocityLength }
        : dragDistance > 0.001
          ? { x: delta.x / dragDistance, y: delta.y / dragDistance }
          : { x: 0, y: 0 }
    const tangent = { x: -direction.y, y: direction.x }

    cy.batch(() => {
      cy.nodes().forEach((node) => {
        const nodeId = node.id()
        if (nodeId === context.nodeId || !context.dragIds.has(nodeId)) return

        const basePoint = context.basePositions.get(nodeId)
        if (!basePoint) return

        const depth = context.depthMap.get(nodeId) || 1
        const depthRatio = clamp(depth * 0.18, 0.12, 0.72)
        const hashPhase = ((hashString(nodeId) % 200) / 200) * Math.PI * 2
        const swayStrength = (3 + depth * 2.4) * speedRatio
        const lagStrength = (2.5 + depth * 2.2) * speedRatio
        const carry = 1 - depthRatio * 0.02
        const wave = Math.sin(hashPhase + dragDistance * 0.018)

        node.position({
          x:
            basePoint.x +
            delta.x * carry -
            direction.x * lagStrength * depthRatio +
            tangent.x * swayStrength * wave,
          y:
            basePoint.y +
            delta.y * carry -
            direction.y * lagStrength * depthRatio +
            tangent.y * swayStrength * wave,
        })
      })
    })
  }

  function applyDynamicCollisionResponse(draggedNodeId, lockedIds = new Set()) {
    const cy = graphRef.value
    if (!cy || !draggedNodeId) return

    const draggedNode = cy.getElementById(draggedNodeId)
    if (!draggedNode?.length) return

    const nodes = cy.nodes().map((node) => ({
      id: node.id(),
      data: { size: node.data('size') },
    }))
    const positions = snapshotNodePositions(cy)
    const resolvedPositions = resolveCollisions(
      nodes,
      positions,
      getSceneRadius(nodes.length),
      lockedIds?.size ? lockedIds : new Set([draggedNodeId]),
      18,
    )

    cy.batch(() => {
      cy.nodes().forEach((node) => {
        if (lockedIds?.has(node.id()) || node.id() === draggedNodeId) return
        const nextPoint = resolvedPositions.get(node.id())
        if (!nextPoint) return
        const current = node.position()
        if (Math.hypot(nextPoint.x - current.x, nextPoint.y - current.y) < 0.35) return
        node.position(nextPoint)
      })
    })
  }

  function scheduleDynamicCollisionResponse(draggedNodeId, lockedIds = new Set([draggedNodeId])) {
    pendingDraggedNodeId = draggedNodeId
    pendingLockedDragIds = new Set(lockedIds)
    if (dragCollisionFrame) return

    dragCollisionFrame = window.requestAnimationFrame(() => {
      dragCollisionFrame = null
      const targetId = pendingDraggedNodeId
      const targetLockedIds = pendingLockedDragIds
      pendingDraggedNodeId = ''
      pendingLockedDragIds = new Set()
      applyDynamicCollisionResponse(targetId, targetLockedIds)
      persistNodePositions(graphRef.value)
    })
  }

  function registerEvents(cy) {
    cy.on('tap', 'node', (event) => {
      const nodeId = event.target.id()
      const now = Date.now()

      if (tapIntent && tapIntent.id === nodeId && now - tapIntent.time < 280) {
        clearTapIntent()
        onNodeDoubleClick?.({ target: { id: nodeId } })
        return
      }

      clearTapIntent()
      tapIntent = {
        id: nodeId,
        time: now,
        timer: window.setTimeout(() => {
          onNodeClick?.({ target: { id: nodeId } })
          tapIntent = null
        }, 210),
      }
    })

    cy.on('tap', (event) => {
      if (event.target !== cy) return
      clearTapIntent()
      onCanvasClick?.({ target: { id: null } })
    })

    cy.on('mouseover', 'node', (event) => {
      onNodeEnter?.(toClientEvent(event, event.target.id()))
    })

    cy.on('mouseout', 'node', (event) => {
      onNodeLeave?.(toClientEvent(event, event.target.id()))
    })

    cy.on('cxttap', 'node', (event) => {
      onNodeContextMenu?.(toClientEvent(event, event.target.id()))
    })

    cy.on('grab', 'node', (event) => {
      const nodeId = event.target.id()
      stopSpringMotion(false)
      const groupState = buildDragGroupState(cy, nodeId)
      activeSpringMotion = createSpringMotion(cy, nodeId)
      dragContext = {
        nodeId,
        dragIds: groupState.dragIds,
        depthMap: groupState.depthMap,
        basePositions: groupState.basePositions,
        anchorStartPosition: getNodePoint(event.target.position()),
        lastPosition: getNodePoint(event.target.position()),
        lastTimestamp: performance.now(),
        velocity: { x: 0, y: 0 },
      }
      ensureSpringMotionLoop()
    })

    cy.on('drag', 'node', (event) => {
      const nodeId = event.target.id()
      const currentPosition = getNodePoint(event.target.position())
      const now = performance.now()
      const activeContext =
        dragContext?.nodeId === nodeId
          ? dragContext
          : {
              nodeId,
              ...buildDragGroupState(cy, nodeId),
              anchorStartPosition: currentPosition,
              lastPosition: currentPosition,
              lastTimestamp: now,
              velocity: { x: 0, y: 0 },
            }

      const deltaStep = {
        x: currentPosition.x - activeContext.lastPosition.x,
        y: currentPosition.y - activeContext.lastPosition.y,
      }
      const deltaTotal = {
        x: currentPosition.x - activeContext.anchorStartPosition.x,
        y: currentPosition.y - activeContext.anchorStartPosition.y,
      }
      const dt = Math.max(8, now - (activeContext.lastTimestamp || now))
      const velocity = {
        x: deltaStep.x / dt,
        y: deltaStep.y / dt,
      }
      const speed = Math.hypot(velocity.x, velocity.y)

      if (Math.hypot(deltaStep.x, deltaStep.y) > 0.6) {
        clearTapIntent()
      }

      if (activeSpringMotion?.nodeId !== nodeId) {
        activeSpringMotion = createSpringMotion(cy, nodeId)
      }
      if (activeSpringMotion) {
        activeSpringMotion.pointerDown = true
        activeSpringMotion.anchorVelocity = velocity
        activeSpringMotion.lastTick = now
      }

      applyElasticDragGroupMotion(cy, activeContext, currentPosition, deltaTotal, velocity, speed)
      activeContext.lastPosition = currentPosition
      activeContext.lastTimestamp = now
      activeContext.velocity = velocity
      dragContext = activeContext
      scheduleDynamicCollisionResponse(nodeId, new Set([nodeId]))
      ensureSpringMotionLoop()
    })

    cy.on('free', 'node', (event) => {
      const nodeId = event.target.id()
      const lockedIds = new Set([nodeId])
      if (activeSpringMotion?.nodeId === nodeId) {
        activeSpringMotion.pointerDown = false
        activeSpringMotion.anchorVelocity = {
          x: (dragContext?.velocity?.x || 0) * 1.1,
          y: (dragContext?.velocity?.y || 0) * 1.1,
        }
        activeSpringMotion.lastTick = performance.now()
        ensureSpringMotionLoop()
      } else {
        applyDynamicCollisionResponse(nodeId, lockedIds)
        persistNodePositions(cy)
      }
      dragContext = null
    })

    cy.on('zoom', () => {
      updateZoom()
      applyLabelLod()
    })

    cy.on('pan', () => {
      updateZoom()
    })
  }

  function ensureGraph(elements) {
    if (graphRef.value) return graphRef.value

    graphRef.value = cytoscape({
      container: graphMountRef.value,
      elements,
      style: buildCyStyle(),
      layout: { name: 'preset', fit: false, animate: false, padding: 60 },
      minZoom: 0.18,
      maxZoom: 2,
      wheelSensitivity: 0.84,
      selectionType: 'single',
      boxSelectionEnabled: false,
      autounselectify: true,
      textureOnViewport: false,
      hideEdgesOnViewport: false,
      motionBlur: false,
      pixelRatio: 1,
    })

    registerEvents(graphRef.value)
    return graphRef.value
  }

  function syncElements(cy, elements) {
    const nextIds = new Set(elements.map((element) => element.data.id))

    cy.batch(() => {
      cy.elements().forEach((element) => {
        if (!nextIds.has(element.id())) element.remove()
      })

      elements.forEach((element) => {
        const existing = cy.getElementById(element.data.id)
        if (!existing?.length) {
          cy.add(element)
          return
        }
        existing.data({ ...element.data })
      })
    })
  }

  function applyPresetLayout(cy, elements, structureChanged) {
    if (!cy || !cy.nodes().length) return Promise.resolve()

    return new Promise((resolve) => {
      const animate = structureChanged && cy.nodes().length <= 96
      let remaining = 0

      cy.batch(() => {
        elements.forEach((element) => {
          if (element.group !== 'nodes' || !element.position) return
          const node = cy.getElementById(element.data.id)
          if (!node?.length) return

          if (!animate) {
            node.position({ ...element.position })
            return
          }

          remaining += 1
          node.animate(
            { position: { ...element.position } },
            {
              duration: 240,
              complete: () => {
                remaining -= 1
                if (remaining <= 0) resolve()
              },
            },
          )
        })
      })

      if (!animate || remaining <= 0) resolve()
    })
  }

  async function centerGraph(zoomLevel = 1, duration = 0) {
    const cy = graphRef.value
    if (!cy) return

    const nextZoom = clamp(zoomLevel, 0.18, 2)
    if (duration <= 0) {
      cy.zoom(nextZoom)
      cy.center(cy.nodes())
    } else {
      cy.animate(
        {
          zoom: nextZoom,
          center: { eles: cy.nodes() },
        },
        { duration },
      )
    }

    updateZoom()
    applyLabelLod(true)
  }

  async function fitView(duration = 260) {
    const cy = graphRef.value
    if (!cy) return
    const elements = cy.nodes()
    if (!elements.length) return

    if (duration <= 0) cy.fit(elements, 72)
    else cy.animate({ fit: { eles: elements, padding: 72 } }, { duration })

    updateZoom()
    applyLabelLod(true)
  }

  async function centerElement(elementId, zoomLevel = 1.4, duration = 260) {
    const cy = graphRef.value
    if (!cy || !elementId) return
    const element = cy.getElementById(elementId)
    if (!element?.length) return

    const nextZoom = clamp(zoomLevel, 0.18, 2)
    if (duration <= 0) {
      cy.zoom(nextZoom)
      cy.center(element)
    } else {
      cy.animate(
        {
          zoom: nextZoom,
          center: { eles: element },
        },
        { duration },
      )
    }

    updateZoom()
    applyLabelLod(true)
  }

  async function focusElement(elementId, duration = 280) {
    const cy = graphRef.value
    if (!cy || !elementId) return
    const element = cy.getElementById(elementId)
    if (!element?.length) return

    if (duration <= 0) cy.fit(element, 180)
    else cy.animate({ fit: { eles: element, padding: 180 } }, { duration })

    updateZoom()
    applyLabelLod(true)
  }

  async function flashNode(nodeId) {
    if (!nodeId) return
    if (flashTimer) {
      window.clearTimeout(flashTimer)
      flashTimer = null
    }

    await focusElement(nodeId, 220)
    flashedIds.add(nodeId)
    scheduleRenderGraph()
    flashTimer = window.setTimeout(() => {
      flashedIds.delete(nodeId)
      scheduleRenderGraph()
      flashTimer = null
    }, 900)
  }

  async function zoomIn() {
    const cy = graphRef.value
    if (!cy) return
    cy.zoom({
      level: clamp(cy.zoom() * 1.52, 0.18, 2),
      renderedPosition: { x: containerRef.value?.clientWidth / 2 || 0, y: containerRef.value?.clientHeight / 2 || 0 },
    })
    updateZoom()
    applyLabelLod(true)
  }

  async function zoomOut() {
    const cy = graphRef.value
    if (!cy) return
    cy.zoom({
      level: clamp(cy.zoom() / 1.52, 0.18, 2),
      renderedPosition: { x: containerRef.value?.clientWidth / 2 || 0, y: containerRef.value?.clientHeight / 2 || 0 },
    })
    updateZoom()
    applyLabelLod(true)
  }

  async function renderGraph() {
    if (renderInFlight) return
    renderInFlight = true

    try {
      renderError.value = ''
      await nextTick()
      if (!containerRef.value || !graphMountRef.value) return

      const viewport = getViewportSize(containerRef.value)
      if (!viewport) {
        window.setTimeout(() => scheduleRenderGraph(), 120)
        return
      }

      const nodeCount = data.value?.nodes?.length || 0
      const edgeCount = data.value?.edges?.length || 0
      if (nodeCount === 0 && edgeCount === 0) {
        destroyGraph()
        return
      }

      const signature = getStructureSignature(data.value || { nodes: [], edges: [] })
      const structureChanged = signature !== lastStructureSignature
      const persistedPositions = graphRef.value ? snapshotNodePositions(graphRef.value) : new Map(positionMemory)
      const elements = buildCyElements(
        data.value || { nodes: [], edges: [] },
        stateMap?.value || {},
        flashedIds,
        persistedPositions,
        { selectedNodeId: selectedNodeId?.value || '' },
      )
      const cy = ensureGraph(elements)
      syncElements(cy, elements)
      await applyPresetLayout(cy, elements, structureChanged)
      cy.resize()
      persistNodePositions(cy)
      lastStructureSignature = signature
      updateZoom()
      applyLabelLod(true)

      if (!hasInitialFit) {
        await centerGraph(1, 0)
        hasInitialFit = true
      }
    } catch (error) {
      console.error('[CytoscapeGraph] render failed:', error)
      renderError.value = error?.message || '知识图谱渲染失败，请刷新页面后重试。'
    } finally {
      renderInFlight = false
    }
  }

  function scheduleRenderGraph() {
    if (renderFrame) return
    renderFrame = window.requestAnimationFrame(() => {
      renderFrame = null
      renderGraph()
    })
  }

  watch(data, () => scheduleRenderGraph())
  watch(stateMap, () => scheduleRenderGraph())

  watch(containerRef, (element) => {
    if (!element) return
    resizeObserver?.disconnect()
    resizeObserver = new ResizeObserver(() => {
      graphRef.value?.resize?.()
      updateZoom()
      applyLabelLod(true)
    })
    resizeObserver.observe(element)
    scheduleRenderGraph()
  })

  onMounted(() => scheduleRenderGraph())

  onBeforeUnmount(() => {
    clearTapIntent()
    if (renderFrame) {
      window.cancelAnimationFrame(renderFrame)
      renderFrame = null
    }
    if (dragCollisionFrame) {
      window.cancelAnimationFrame(dragCollisionFrame)
      dragCollisionFrame = null
    }
    if (flashTimer) {
      window.clearTimeout(flashTimer)
      flashTimer = null
    }
    resizeObserver?.disconnect()
    destroyGraph()
  })

  return {
    containerRef,
    graphMountRef,
    graphRef,
    renderError,
    zoom,
    fitView,
    centerGraph,
    centerElement,
    zoomIn,
    zoomOut,
    focusElement,
    flashNode,
  }
}
