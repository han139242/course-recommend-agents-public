import { drag } from 'd3-drag'
import { select } from 'd3-selection'
import { zoom, zoomIdentity } from 'd3-zoom'

import { DEFAULT_ZOOM, MAX_ZOOM, MIN_ZOOM, clamp } from '../utils/graphLayout'
import { findNodeAtPoint } from '../utils/graphHitTest'

const DRAG_REHEAT_TOLERANCE = 25

function buildClientEvent(node, nativeEvent, fallbackPoint = { x: 0, y: 0 }) {
  const clientX = nativeEvent?.clientX ?? fallbackPoint.x
  const clientY = nativeEvent?.clientY ?? fallbackPoint.y
  return {
    target: { id: node?.id || null },
    clientX,
    clientY,
    client: { x: clientX, y: clientY },
    preventDefault: () => nativeEvent?.preventDefault?.(),
  }
}

function hasExceededDragTolerance(startPoint, nextPoint, tolerance = DRAG_REHEAT_TOLERANCE) {
  const startX = Number(startPoint?.x || 0)
  const startY = Number(startPoint?.y || 0)
  const nextX = Number(nextPoint?.x || 0)
  const nextY = Number(nextPoint?.y || 0)
  const dx = nextX - startX
  const dy = nextY - startY
  return dx * dx + dy * dy > tolerance * tolerance
}

export function useGraphInteraction(options = {}) {
  const renderer = options.renderer
  const engine = options.engine

  let canvas = null
  let selection = null
  let zoomBehavior = null
  let dragBehavior = null
  let hoverNodeId = null
  let clickTimer = 0
  let suppressClickUntil = 0
  let dragStartPoint = null
  let dragDidMove = false

  function applyConstraint(nextTransform, constraintOptions = {}) {
    if (typeof options.constrainTransform !== 'function') {
      return nextTransform
    }
    return options.constrainTransform(nextTransform, constraintOptions)
  }

  function getViewportCenter() {
    const viewport = renderer.getViewportSize()
    return {
      x: viewport.width / 2,
      y: viewport.height / 2,
    }
  }

  function localToClient(x, y) {
    const bounds = canvas?.getBoundingClientRect()
    return {
      x: (bounds?.left || 0) + x,
      y: (bounds?.top || 0) + y,
    }
  }

  function findNodeFromClient(clientX, clientY) {
    const [worldX, worldY] = renderer.screenToWorld(clientX, clientY)
    return findNodeAtPoint(engine.getNodes(), worldX, worldY, 64)
  }

  function dispatchNodeEvent(kind, node, nativeEvent) {
    if (!node) return
    const viewNode = node.view
    const event = buildClientEvent(node, nativeEvent)
    if (viewNode?.data?.isHub) {
      if (kind === 'click') options.onHubClick?.(event)
      if (kind === 'doubleClick') options.onHubDoubleClick?.(event)
      return
    }
    if (kind === 'click') options.onNodeClick?.(event)
    if (kind === 'doubleClick') options.onNodeDoubleClick?.(event)
  }

  function updateHover(node, nativeEvent) {
    const nextHoverId = node?.id || null
    if (hoverNodeId === nextHoverId) return

    if (hoverNodeId) {
      const previous = engine.getNodeById(hoverNodeId)
      if (previous) {
        options.onNodeLeave?.(buildClientEvent(previous, nativeEvent))
      }
    }

    hoverNodeId = nextHoverId
    canvas.style.cursor = node ? 'pointer' : 'default'

    if (node) {
      options.onNodeEnter?.(buildClientEvent(node, nativeEvent))
    }
  }

  function syncZoomTransform(nextTransform, constraintOptions = {}) {
    if (!selection || !zoomBehavior) return
    const constrained = applyConstraint(nextTransform, constraintOptions)
    selection.call(
      zoomBehavior.transform,
      zoomIdentity.translate(constrained.x, constrained.y).scale(constrained.k),
    )
  }

  async function transformTo(nextTransform, duration = 0, constraintOptions = {}) {
    const constrained = applyConstraint(nextTransform, constraintOptions)
    await renderer.animateTo(constrained, duration)
    syncZoomTransform(constrained, constraintOptions)
  }

  function setTransform(nextTransform, constraintOptions = {}) {
    const normalized = {
      x: Number(nextTransform?.x || 0),
      y: Number(nextTransform?.y || 0),
      k: clamp(Number(nextTransform?.k || DEFAULT_ZOOM), MIN_ZOOM, MAX_ZOOM),
    }
    const constrained = applyConstraint(normalized, constraintOptions)

    if (selection && zoomBehavior) {
      selection.call(
        zoomBehavior.transform,
        zoomIdentity.translate(constrained.x, constrained.y).scale(constrained.k),
      )
      return
    }

    renderer.setTransform(constrained)
  }

  async function zoomByFactor(factor = 1, duration = 120) {
    const current = renderer.getTransform()
    const center = getViewportCenter()
    const [worldX, worldY] = renderer.screenToWorld(center.x, center.y)
    const nextK = clamp(current.k * factor, MIN_ZOOM, MAX_ZOOM)
    await transformTo(
      {
        x: center.x - worldX * nextK,
        y: center.y - worldY * nextK,
        k: nextK,
      },
      duration,
    )
  }

  function handleClick(nativeEvent) {
    if (Date.now() < suppressClickUntil) return
    const node = findNodeFromClient(nativeEvent.clientX, nativeEvent.clientY)
    if (!node) {
      if (clickTimer) {
        window.clearTimeout(clickTimer)
        clickTimer = 0
      }
      options.onCanvasClick?.(buildClientEvent(null, nativeEvent))
      return
    }

    if (clickTimer) {
      window.clearTimeout(clickTimer)
      clickTimer = 0
    }
    clickTimer = window.setTimeout(() => {
      clickTimer = 0
      dispatchNodeEvent('click', node, nativeEvent)
    }, 180)
  }

  function handleDoubleClick(nativeEvent) {
    const node = findNodeFromClient(nativeEvent.clientX, nativeEvent.clientY)
    if (!node) return
    if (clickTimer) {
      window.clearTimeout(clickTimer)
      clickTimer = 0
    }
    dispatchNodeEvent('doubleClick', node, nativeEvent)
  }

  function handleMouseMove(nativeEvent) {
    const node = findNodeFromClient(nativeEvent.clientX, nativeEvent.clientY)
    updateHover(node, nativeEvent)
  }

  function handleMouseLeave(nativeEvent) {
    updateHover(null, nativeEvent)
  }

  function handleContextMenu(nativeEvent) {
    nativeEvent.preventDefault?.()
    const node = findNodeFromClient(nativeEvent.clientX, nativeEvent.clientY)
    if (!node) return
    options.onNodeContextMenu?.(buildClientEvent(node, nativeEvent))
  }

  function init(nextCanvas) {
    canvas = nextCanvas
    selection = select(canvas)

    zoomBehavior = zoom()
      .scaleExtent([MIN_ZOOM, MAX_ZOOM])
      .constrain((nextTransform) => {
        const constrained = applyConstraint(nextTransform)
        return zoomIdentity.translate(constrained.x, constrained.y).scale(constrained.k)
      })
      .filter((event) => {
        if (event.type === 'dblclick') return false
        if (event.type === 'mousedown' && event.button === 2) return false
        return true
      })
      .on('zoom', (event) => {
        renderer.setTransform(applyConstraint(event.transform))
      })

    selection.call(zoomBehavior).on('dblclick.zoom', null)
    syncZoomTransform({ x: 0, y: 0, k: DEFAULT_ZOOM })

    dragBehavior = drag()
      .container(canvas)
      .subject((event) => {
        const point = localToClient(event.x, event.y)
        return findNodeFromClient(point.x, point.y)
      })
      .on('start', (event) => {
        if (!event.subject) return
        dragStartPoint = { x: event.x, y: event.y }
        dragDidMove = false
        event.subject.isUserPinned = true
        event.subject.fx = event.subject.x
        event.subject.fy = event.subject.y
      })
      .on('drag', (event) => {
        if (!event.subject) return
        if (!dragDidMove && hasExceededDragTolerance(dragStartPoint, event)) {
          suppressClickUntil = Date.now() + 220
          dragDidMove = true
          engine.reheat({
            alphaTarget: 0.09,
            alpha: 0.8,
            duration: 1400,
          })
        }
        const point = localToClient(event.x, event.y)
        const [worldX, worldY] = renderer.screenToWorld(point.x, point.y)
        event.subject.fx = worldX
        event.subject.fy = worldY
        event.subject.x = worldX
        event.subject.y = worldY
      })
      .on('end', (event) => {
        if (!event.subject) return
        if (dragDidMove) {
          suppressClickUntil = Date.now() + 220
        }
        event.subject.isUserPinned = true
        event.subject.fx = event.subject.x
        event.subject.fy = event.subject.y
        dragStartPoint = null
        dragDidMove = false
      })

    selection.call(dragBehavior)

    canvas.addEventListener('click', handleClick)
    canvas.addEventListener('dblclick', handleDoubleClick)
    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('mouseleave', handleMouseLeave)
    canvas.addEventListener('contextmenu', handleContextMenu)
  }

  function destroy() {
    if (clickTimer) {
      window.clearTimeout(clickTimer)
      clickTimer = 0
    }
    if (!canvas) return
    canvas.removeEventListener('click', handleClick)
    canvas.removeEventListener('dblclick', handleDoubleClick)
    canvas.removeEventListener('mousemove', handleMouseMove)
    canvas.removeEventListener('mouseleave', handleMouseLeave)
    canvas.removeEventListener('contextmenu', handleContextMenu)
    selection?.on('.zoom', null)
    selection?.on('.drag', null)
    selection = null
    canvas = null
  }

  return {
    init,
    destroy,
    transformTo,
    setTransform,
    zoomByFactor,
    getTransform: () => renderer.getTransform(),
  }
}

export const __graphInteractionTestUtils = {
  hasExceededDragTolerance,
}
