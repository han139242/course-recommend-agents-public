import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { DEFAULT_ZOOM, MAX_ZOOM, MIN_ZOOM, clamp } from '../utils/graphLayout'
import { useGraphEngine } from './useGraphEngine'
import { useGraphRenderer } from './useGraphRenderer'
import { useGraphInteraction } from './useGraphInteraction'

function getBounds(nodes = []) {
  if (!nodes.length) return null
  const extents = nodes.map((node) => {
    const radius = Number(
      node?.view?.style?.collisionRadius ||
      node?.collisionRadius ||
      node?.view?.style?.radius ||
      32,
    )
    return {
      minX: Number(node.x || 0) - radius,
      maxX: Number(node.x || 0) + radius,
      minY: Number(node.y || 0) - radius,
      maxY: Number(node.y || 0) + radius,
    }
  })
  const minX = Math.min(...extents.map((item) => item.minX))
  const maxX = Math.max(...extents.map((item) => item.maxX))
  const minY = Math.min(...extents.map((item) => item.minY))
  const maxY = Math.max(...extents.map((item) => item.maxY))
  return {
    minX,
    maxX,
    minY,
    maxY,
    width: Math.max(maxX - minX, 1),
    height: Math.max(maxY - minY, 1),
    centerX: (minX + maxX) / 2,
    centerY: (minY + maxY) / 2,
  }
}

const SAFE_VIEWPORT_PADDING = {
  left: 56,
  right: 56,
  top: 56,
  bottom: 56,
}

const CLUSTER_VIEWPORT_PADDING = {
  left: 108,
  right: 108,
  top: 92,
  bottom: 184,
}

const SCENE_VIEWPORT_PADDING = {
  left: 170,
  right: 170,
  top: 108,
  bottom: 234,
}

function resolveNodeIdList(nodeIds = []) {
  const items = Array.isArray(nodeIds) ? nodeIds : [nodeIds]
  return [...new Set(items.map((item) => String(item || '').trim()).filter(Boolean))]
}

function getNodesByIds(nodes = [], nodeIds = []) {
  const idSet = new Set(resolveNodeIdList(nodeIds))
  if (!idSet.size) return []
  return nodes.filter((node) => idSet.has(String(node?.id || '').trim()))
}

function inflateBounds(bounds, padding = 0) {
  if (!bounds) return null
  const pad = Math.max(0, Number(padding || 0))
  return {
    minX: bounds.minX - pad,
    maxX: bounds.maxX + pad,
    minY: bounds.minY - pad,
    maxY: bounds.maxY + pad,
    width: bounds.width + pad * 2,
    height: bounds.height + pad * 2,
    centerX: bounds.centerX,
    centerY: bounds.centerY,
  }
}

export function constrainTransformToViewport(nextTransform = {}, options = {}) {
  const normalized = {
    x: Number(nextTransform?.x || 0),
    y: Number(nextTransform?.y || 0),
    k: clamp(Number(nextTransform?.k || DEFAULT_ZOOM), MIN_ZOOM, MAX_ZOOM),
  }
  const nodes = Array.isArray(options.nodes) ? options.nodes : []
  const viewport = options.viewport || {}
  const width = Number(viewport.width || 0)
  const height = Number(viewport.height || 0)

  if (!nodes.length || width <= 0 || height <= 0) {
    return normalized
  }

  const bounds = getBounds(nodes)
  if (!bounds) {
    return normalized
  }

  const padding = {
    ...SAFE_VIEWPORT_PADDING,
    ...(options.padding || {}),
  }
  const availableWidth = Math.max(width - padding.left - padding.right, 1)
  const availableHeight = Math.max(height - padding.top - padding.bottom, 1)
  const contentWidth = bounds.width * normalized.k
  const contentHeight = bounds.height * normalized.k

  let nextX = normalized.x
  let nextY = normalized.y

  if (contentWidth <= availableWidth) {
    nextX = padding.left + (availableWidth - contentWidth) / 2 - bounds.minX * normalized.k
  } else {
    const minAllowedX = width - padding.right - bounds.maxX * normalized.k
    const maxAllowedX = padding.left - bounds.minX * normalized.k
    nextX = clamp(nextX, minAllowedX, maxAllowedX)
  }

  if (contentHeight <= availableHeight) {
    nextY = padding.top + (availableHeight - contentHeight) / 2 - bounds.minY * normalized.k
  } else {
    const minAllowedY = height - padding.bottom - bounds.maxY * normalized.k
    const maxAllowedY = padding.top - bounds.minY * normalized.k
    nextY = clamp(nextY, minAllowedY, maxAllowedY)
  }

  return {
    x: nextX,
    y: nextY,
    k: normalized.k,
  }
}

export function buildTransformToKeepBoundsVisible(baseTransform = {}, bounds, options = {}) {
  const normalized = {
    x: Number(baseTransform?.x || 0),
    y: Number(baseTransform?.y || 0),
    k: clamp(Number(baseTransform?.k || DEFAULT_ZOOM), MIN_ZOOM, MAX_ZOOM),
  }
  const viewport = options.viewport || {}
  const width = Number(viewport.width || 0)
  const height = Number(viewport.height || 0)

  if (!bounds || width <= 0 || height <= 0) {
    return normalized
  }

  const padding = {
    ...CLUSTER_VIEWPORT_PADDING,
    ...(options.padding || {}),
  }
  const visibleBounds = inflateBounds(bounds, options.boundsPadding ?? 28)
  if (!visibleBounds) {
    return normalized
  }

  const availableWidth = Math.max(width - padding.left - padding.right, 1)
  const availableHeight = Math.max(height - padding.top - padding.bottom, 1)

  let nextX = normalized.x
  let nextY = normalized.y
  let nextZoom = normalized.k

  if (options.allowZoomOut !== false) {
    const fitZoom = clamp(
      Math.min(
        availableWidth / Math.max(visibleBounds.width, 1),
        availableHeight / Math.max(visibleBounds.height, 1),
      ),
      MIN_ZOOM,
      MAX_ZOOM,
    )
    if (fitZoom < nextZoom) {
      const centerWorldX = (width / 2 - nextX) / nextZoom
      const centerWorldY = (height / 2 - nextY) / nextZoom
      nextZoom = fitZoom
      nextX = width / 2 - centerWorldX * nextZoom
      nextY = height / 2 - centerWorldY * nextZoom
    }
  }

  const minScreenX = visibleBounds.minX * nextZoom + nextX
  const maxScreenX = visibleBounds.maxX * nextZoom + nextX
  const minScreenY = visibleBounds.minY * nextZoom + nextY
  const maxScreenY = visibleBounds.maxY * nextZoom + nextY

  if (minScreenX < padding.left) {
    nextX += padding.left - minScreenX
  }
  if (maxScreenX > width - padding.right) {
    nextX -= maxScreenX - (width - padding.right)
  }
  if (minScreenY < padding.top) {
    nextY += padding.top - minScreenY
  }
  if (maxScreenY > height - padding.bottom) {
    nextY -= maxScreenY - (height - padding.bottom)
  }

  return {
    x: nextX,
    y: nextY,
    k: nextZoom,
  }
}

export function useGraph(options = {}) {
  const containerRef = ref(null)
  const graphMountRef = ref(null)
  const renderError = ref('')
  const zoom = ref(DEFAULT_ZOOM)

  const flashSet = new Set()
  let flashTimer = 0
  let pendingFocusTask = null
  let followFrame = 0
  let followLock = null
  let initialFitDone = false
  let settleFitTimers = []

  const renderer = useGraphRenderer()
  const engine = useGraphEngine({
    onTick: () => {
      zoom.value = renderer.getTransform().k
    },
  })
  const interaction = useGraphInteraction({
    renderer,
    engine,
    constrainTransform: (nextTransform) =>
      constrainTransformToViewport(nextTransform, {
        nodes: engine.getNodes(),
        viewport: renderer.getViewportSize(),
      }),
    onZoom: () => {
      zoom.value = renderer.getTransform().k
    },
    onNodeClick: options.onNodeClick,
    onNodeDoubleClick: options.onNodeDoubleClick,
    onHubClick: options.onHubClick,
    onHubDoubleClick: options.onHubDoubleClick,
    onCanvasClick: options.onCanvasClick,
    onNodeEnter: options.onNodeEnter,
    onNodeLeave: options.onNodeLeave,
    onNodeContextMenu: options.onNodeContextMenu,
  })

  function buildScene() {
    return {
      nodes: engine.getNodes(),
      edges: engine.getEdges(),
      stateMap: options.stateMap?.value || {},
      flashSet,
    }
  }

  function getViewportCenter() {
    const viewport = renderer.getViewportSize()
    return {
      x: viewport.width / 2,
      y: viewport.height / 2,
    }
  }

  function getViewportCenterWorld() {
    const center = getViewportCenter()
    const [x, y] = renderer.screenToWorld(center.x, center.y)
    return { x, y }
  }

  function constrainTransform(nextTransform, constraintOptions = {}) {
    return constrainTransformToViewport(nextTransform, {
      nodes: engine.getNodes(),
      viewport: renderer.getViewportSize(),
      padding: constraintOptions.padding,
    })
  }

  function buildViewportFocusTransform(nodeIds = [], options = {}) {
    const focusIds = resolveNodeIdList(nodeIds)
    const sceneNodes = getNodesByIds(engine.getNodes(), focusIds)
    if (!sceneNodes.length) return null

    const viewport = renderer.getViewportSize()
    const currentTransform = renderer.getTransform()
    const focusNode =
      engine.getNodeById(options.focusNodeId) ||
      sceneNodes[0]
    const nextZoom = clamp(
      Number.isFinite(options.zoomLevel)
        ? (options.panOnly ? Math.max(currentTransform.k, options.zoomLevel) : options.zoomLevel)
        : currentTransform.k,
      MIN_ZOOM,
      MAX_ZOOM,
    )
    const baseTransform =
      !options.panOnly && focusNode
        ? {
            x: viewport.width / 2 - focusNode.x * nextZoom,
            y: viewport.height / 2 - focusNode.y * nextZoom,
            k: nextZoom,
          }
        : {
            ...currentTransform,
            k: nextZoom,
          }

    return constrainTransform(
      buildTransformToKeepBoundsVisible(baseTransform, getBounds(sceneNodes), {
        viewport,
        padding: options.padding || CLUSTER_VIEWPORT_PADDING,
        allowZoomOut: options.allowZoomOut !== false,
        boundsPadding: options.boundsPadding ?? 28,
      }),
      {
        padding: options.padding || CLUSTER_VIEWPORT_PADDING,
      },
    )
  }

  function stopFollowLock() {
    followLock = null
    if (followFrame) {
      window.cancelAnimationFrame(followFrame)
      followFrame = 0
    }
  }

  function clearSettleFitTimers() {
    settleFitTimers.forEach((timer) => window.clearTimeout(timer))
    settleFitTimers = []
  }

  function cancelViewportAutomation(options = {}) {
    stopFollowLock()
    if (!options.preservePendingFocusTask) {
      pendingFocusTask = null
    }
    if (!options.preserveSettleFitTimers) {
      clearSettleFitTimers()
    }
  }

  function startFollowLock(nodeId, zoomLevel = null, duration = 0, options = {}) {
    stopFollowLock()
    if (!nodeId || duration <= 0) return

    followLock = {
      nodeId,
      until: performance.now() + duration,
      zoomLevel: Number.isFinite(zoomLevel) ? zoomLevel : null,
      relatedNodeIds: resolveNodeIdList(options.relatedNodeIds),
      padding: options.padding || null,
      mode: options.mode || 'center',
    }

    const step = () => {
      if (!followLock) return
      if (performance.now() >= followLock.until) {
        stopFollowLock()
        return
      }

      const node = engine.getNodeById(followLock.nodeId)
      if (!node) {
        stopFollowLock()
        return
      }

      const focusIds = resolveNodeIdList([followLock.nodeId, ...followLock.relatedNodeIds])
      if (followLock.mode === 'keep-visible' && focusIds.length > 1) {
        const nextTransform = buildViewportFocusTransform(focusIds, {
          focusNodeId: followLock.nodeId,
          zoomLevel: followLock.zoomLevel ?? renderer.getTransform().k,
          padding: followLock.padding,
          panOnly: true,
          allowZoomOut: true,
        })
        if (nextTransform) {
          interaction.setTransform(nextTransform)
          zoom.value = renderer.getTransform().k
          followFrame = window.requestAnimationFrame(step)
          return
        }
      }

      const viewport = renderer.getViewportSize()
      const currentTransform = renderer.getTransform()
      const nextZoom = clamp(
        Math.max(currentTransform.k, followLock.zoomLevel ?? currentTransform.k),
        MIN_ZOOM,
        MAX_ZOOM,
      )

      interaction.setTransform(constrainTransform({
        x: viewport.width / 2 - node.x * nextZoom,
        y: viewport.height / 2 - node.y * nextZoom,
        k: nextZoom,
      }))
      zoom.value = renderer.getTransform().k
      followFrame = window.requestAnimationFrame(step)
    }

    followFrame = window.requestAnimationFrame(step)
  }

  async function flushPendingFocusTask() {
    if (!pendingFocusTask?.id || !engine.getNodeById(pendingFocusTask.id)) return

    const task = pendingFocusTask
    pendingFocusTask = null
    const focusIds = resolveNodeIdList([task.id, ...(task.relatedNodeIds || [])])

    if (task.mode === 'keep-visible' || focusIds.length > 1) {
      const nextTransform = buildViewportFocusTransform(focusIds, {
        focusNodeId: task.id,
        zoomLevel: task.zoomLevel,
        padding: task.padding,
        allowZoomOut: true,
      })
      if (nextTransform) {
        stopFollowLock()
        await interaction.transformTo(nextTransform, task.duration)
        zoom.value = renderer.getTransform().k
      }
    } else {
      await centerElement(task.id, task.zoomLevel, task.duration)
    }

    startFollowLock(task.id, task.zoomLevel, task.lockDuration, {
      relatedNodeIds: task.relatedNodeIds,
      padding: task.padding,
      mode: task.mode,
    })
  }

  async function syncGraph(syncOptions = {}) {
    const syncMeta = engine.syncGraph(options.data?.value || { nodes: [], edges: [] })
    zoom.value = renderer.getTransform().k

    if (!initialFitDone && syncMeta?.totalNodes) {
      initialFitDone = true
      await fitView(0, { preserveAutomation: true })
      clearSettleFitTimers()
      ;[320, 980].forEach((delay, index) => {
        const timer = window.setTimeout(() => {
          fitView(index === 0 ? 180 : 220, { preserveAutomation: true }).catch(() => null)
          settleFitTimers = settleFitTimers.filter((item) => item !== timer)
        }, delay)
        settleFitTimers.push(timer)
      })
    }

    if (syncOptions.flushPendingFocus !== false) {
      await flushPendingFocusTask()
    }
  }

  function buildTransformForBounds(bounds, overrideZoom = null) {
    const viewport = renderer.getViewportSize()
    const paddingLeft = SCENE_VIEWPORT_PADDING.left
    const paddingRight = SCENE_VIEWPORT_PADDING.right
    const paddingTop = SCENE_VIEWPORT_PADDING.top
    const paddingBottom = SCENE_VIEWPORT_PADDING.bottom
    const availableWidth = Math.max(viewport.width - paddingLeft - paddingRight, 1)
    const availableHeight = Math.max(viewport.height - paddingTop - paddingBottom, 1)
    const contentWidth = bounds.width + 96
    const contentHeight = bounds.height + 168
    const scale = clamp(
      Math.min(availableWidth / contentWidth, availableHeight / contentHeight),
      MIN_ZOOM,
      MAX_ZOOM,
    )
    const nextZoom = clamp(overrideZoom ?? scale, MIN_ZOOM, MAX_ZOOM)
    const targetCenterY = paddingTop + availableHeight / 2
    return {
      x: viewport.width / 2 - bounds.centerX * nextZoom,
      y: targetCenterY - bounds.centerY * nextZoom,
      k: nextZoom,
    }
  }

  async function centerGraph(zoomLevel = DEFAULT_ZOOM, duration = 0, viewOptions = {}) {
    cancelViewportAutomation({
      preservePendingFocusTask: Boolean(viewOptions.preserveAutomation),
      preserveSettleFitTimers: Boolean(viewOptions.preserveAutomation),
    })
    const bounds = getBounds(engine.getNodes())
    const viewport = renderer.getViewportSize()
    const centerX = bounds?.centerX || 0
    const centerY = bounds?.centerY || 0
    await interaction.transformTo(
      constrainTransform({
        x: viewport.width / 2 - centerX * zoomLevel,
        y: viewport.height / 2 - centerY * zoomLevel,
        k: clamp(zoomLevel, MIN_ZOOM, MAX_ZOOM),
      }, {
        padding: SCENE_VIEWPORT_PADDING,
      }),
      duration,
      {
        padding: SCENE_VIEWPORT_PADDING,
      },
    )
    zoom.value = renderer.getTransform().k
  }

  async function fitView(duration = 260, viewOptions = {}) {
    cancelViewportAutomation({
      preservePendingFocusTask: Boolean(viewOptions.preserveAutomation),
      preserveSettleFitTimers: Boolean(viewOptions.preserveAutomation),
    })
    const bounds = getBounds(engine.getNodes())
    if (!bounds) return
    await interaction.transformTo(
      constrainTransform(buildTransformForBounds(bounds), {
        padding: SCENE_VIEWPORT_PADDING,
      }),
      duration,
      {
        padding: SCENE_VIEWPORT_PADDING,
      },
    )
    zoom.value = renderer.getTransform().k
  }

  async function centerElement(elementId, zoomLevel = 1.4, duration = 220) {
    stopFollowLock()
    const node = engine.getNodeById(elementId)
    if (!node) return
    const viewport = renderer.getViewportSize()
    const nextZoom = clamp(zoomLevel, MIN_ZOOM, MAX_ZOOM)
    await interaction.transformTo(
      constrainTransform({
        x: viewport.width / 2 - node.x * nextZoom,
        y: viewport.height / 2 - node.y * nextZoom,
        k: nextZoom,
      }),
      duration,
    )
    zoom.value = renderer.getTransform().k
  }

  async function focusElement(elementId, duration = 240) {
    const current = renderer.getTransform().k
    return centerElement(elementId, Math.max(current, 1.22), duration)
  }

  async function softFollowElement(elementId, options = {}) {
    stopFollowLock()
    const current = renderer.getTransform().k
    return centerElement(elementId, Math.max(current, options.zoomLevel || 1.08), options.duration || 180)
  }

  function queueViewportFocus(nodeId, options = {}) {
    pendingFocusTask = {
      id: nodeId,
      zoomLevel: Number(options.zoomLevel ?? 1.08),
      duration: Number(options.duration ?? 220),
      lockDuration: Number(options.lockDuration ?? 920),
      relatedNodeIds: resolveNodeIdList(options.relatedNodeIds),
      padding: options.padding || null,
      mode:
        options.mode ||
        (Array.isArray(options.relatedNodeIds) && options.relatedNodeIds.length ? 'keep-visible' : 'center'),
    }

    if (engine.getNodeById(nodeId)) {
      window.requestAnimationFrame(() => {
        flushPendingFocusTask().catch(() => null)
      })
    }
  }

  async function flashNode(nodeId) {
    if (!nodeId) return
    if (flashTimer) {
      window.clearTimeout(flashTimer)
      flashTimer = 0
    }
    flashSet.clear()
    flashSet.add(nodeId)
    flashTimer = window.setTimeout(() => {
      flashSet.clear()
      flashTimer = 0
    }, 900)
  }

  async function zoomIn() {
    stopFollowLock()
    await interaction.zoomByFactor(1.16, 120)
    zoom.value = renderer.getTransform().k
  }

  async function zoomOut() {
    stopFollowLock()
    await interaction.zoomByFactor(0.86, 120)
    zoom.value = renderer.getTransform().k
  }

  function getNodeSnapshot(nodeId) {
    return engine.getNodeSnapshot(nodeId)
  }

  function getNodeSnapshots() {
    return engine.getNodeSnapshots()
  }

  onMounted(async () => {
    try {
      await nextTick()
      renderer.init(graphMountRef.value, containerRef.value, buildScene)
      interaction.init(graphMountRef.value)
      await syncGraph()
      await fitView(0, { preserveAutomation: true })
    } catch (error) {
      renderError.value = error?.message || '图谱渲染初始化失败。'
    }
  })

  onBeforeUnmount(() => {
    if (flashTimer) {
      window.clearTimeout(flashTimer)
      flashTimer = 0
    }
    clearSettleFitTimers()
    stopFollowLock()
    interaction.destroy()
    renderer.destroy()
    engine.destroy()
  })

  watch(
    () => options.data?.value,
    () => {
      syncGraph().catch((error) => {
        renderError.value = error?.message || '图谱渲染同步失败。'
      })
    },
  )

  return {
    containerRef,
    graphMountRef,
    renderError,
    zoom,
    fitView,
    centerGraph,
    centerElement,
    queueViewportFocus,
    zoomIn,
    zoomOut,
    focusElement,
    softFollowElement,
    cancelViewportAutomation,
    flashNode,
    getNodeSnapshot,
    getNodeSnapshots,
    getViewportCenterWorld,
    syncNow: syncGraph,
  }
}

export const __graphViewTestUtils = {
  getBounds,
  constrainTransformToViewport,
  buildTransformToKeepBoundsVisible,
}
