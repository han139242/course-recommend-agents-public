import { createSimulation } from '../utils/graphLayout'

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function fallbackPosition(index = 0) {
  const angle = (index % 12) * (Math.PI / 6)
  const radius = 140 + Math.floor(index / 12) * 36
  return {
    x: Math.cos(angle) * radius,
    y: Math.sin(angle) * radius,
  }
}

export function useGraphEngine(options = {}) {
  const onTick = typeof options.onTick === 'function' ? options.onTick : () => {}
  let simulation = null
  let nodeIndex = new Map()
  let edgeIndex = new Map()
  let alphaTimer = 0

  function releaseExpiredNodeState() {
    const now = Date.now()
    nodeIndex.forEach((node) => {
      if (node.layoutLockExpiresAt && node.layoutLockExpiresAt <= now) {
        node.layoutLockExpiresAt = 0
        if (!node.isUserPinned) {
          node.fx = null
          node.fy = null
        }
      }
      if (node.expandAnchor?.expiresAt && node.expandAnchor.expiresAt <= now) {
        node.expandAnchor = null
      }
    })
  }

  function getNodes() {
    return [...nodeIndex.values()]
  }

  function getEdges() {
    return [...edgeIndex.values()]
  }

  function getNodeById(nodeId) {
    return nodeIndex.get(String(nodeId || '').trim()) || null
  }

  function ensureSimulation(nodes, edges) {
    if (!simulation) {
      simulation = createSimulation(nodes, edges, () => {
        releaseExpiredNodeState()
        onTick()
      })
      return simulation
    }
    simulation.nodes(nodes)
    simulation.force('link').links(edges)
    return simulation
  }

  function syncGraph(viewGraph = { nodes: [], edges: [] }) {
    releaseExpiredNodeState()
    const nextNodeIndex = new Map()
    const newNodeIds = []
    const now = Date.now()
    const previousNodeCount = nodeIndex.size
    const wasEmpty = previousNodeCount === 0

    ;(viewGraph.nodes || []).forEach((viewNode, index) => {
      const key = String(viewNode.id || '').trim()
      if (!key) return

      const existing = nodeIndex.get(key)
      const position = existing
        ? {
            x: existing.x,
            y: existing.y,
            fx: existing.fx,
            fy: existing.fy,
            vx: existing.vx,
            vy: existing.vy,
          }
        : {
            ...fallbackPosition(index),
            fx: null,
            fy: null,
            vx: 0,
            vy: 0,
          }

      const engineNode = existing || { id: key }
      const expandMeta = viewNode?.data?.expandMeta
      const layoutLock = viewNode?.data?.layoutLock
      const hasActiveExpandMeta = expandMeta?.expiresAt && expandMeta.expiresAt > now
      const hasLayoutLock = layoutLock?.expiresAt && layoutLock.expiresAt > now

      engineNode.view = viewNode
      engineNode.isUserPinned = Boolean(existing?.isUserPinned)
      engineNode.x = Number.isFinite(position.x) ? position.x : Number(viewNode.x || 0)
      engineNode.y = Number.isFinite(position.y) ? position.y : Number(viewNode.y || 0)
      engineNode.vx = Number.isFinite(position.vx) ? position.vx : 0
      engineNode.vy = Number.isFinite(position.vy) ? position.vy : 0
      engineNode.fx = Number.isFinite(position.fx) ? position.fx : Number.isFinite(viewNode.fx) ? viewNode.fx : null
      engineNode.fy = Number.isFinite(position.fy) ? position.fy : Number.isFinite(viewNode.fy) ? viewNode.fy : null
      engineNode.collisionRadius = Number(viewNode.style?.collisionRadius || 48)

      if (!existing) {
        engineNode.x = hasActiveExpandMeta && Number.isFinite(expandMeta.targetX) ? expandMeta.targetX : Number.isFinite(viewNode.x) ? viewNode.x : engineNode.x
        engineNode.y = hasActiveExpandMeta && Number.isFinite(expandMeta.targetY) ? expandMeta.targetY : Number.isFinite(viewNode.y) ? viewNode.y : engineNode.y
        newNodeIds.push(key)
      }

      if (hasActiveExpandMeta) {
        engineNode.expandAnchor = {
          anchorId: String(expandMeta.anchorId || '').trim() || null,
          x: Number.isFinite(expandMeta.x) ? expandMeta.x : engineNode.x,
          y: Number.isFinite(expandMeta.y) ? expandMeta.y : engineNode.y,
          targetX: Number.isFinite(expandMeta.targetX) ? expandMeta.targetX : engineNode.x,
          targetY: Number.isFinite(expandMeta.targetY) ? expandMeta.targetY : engineNode.y,
          radius: Number.isFinite(expandMeta.radius) ? expandMeta.radius : null,
          expiresAt: expandMeta.expiresAt,
        }
      }

      if (hasLayoutLock) {
        engineNode.layoutLockExpiresAt = layoutLock.expiresAt
        if (!engineNode.isUserPinned) {
          engineNode.fx = Number.isFinite(layoutLock.x) ? layoutLock.x : engineNode.x
          engineNode.fy = Number.isFinite(layoutLock.y) ? layoutLock.y : engineNode.y
        }
      } else if (!engineNode.isUserPinned && existing?.layoutLockExpiresAt && existing.layoutLockExpiresAt <= now) {
        engineNode.layoutLockExpiresAt = 0
        engineNode.fx = null
        engineNode.fy = null
      }

      nextNodeIndex.set(key, engineNode)
    })

    const nextEdgeIndex = new Map()
    ;(viewGraph.edges || []).forEach((viewEdge) => {
      const key = String(viewEdge.id || '').trim()
      if (!key) return
      const sourceId = String(viewEdge.source?.id || viewEdge.source || '').trim()
      const targetId = String(viewEdge.target?.id || viewEdge.target || '').trim()
      if (!nextNodeIndex.has(sourceId) || !nextNodeIndex.has(targetId)) return

      nextEdgeIndex.set(key, {
        id: key,
        source: nextNodeIndex.get(sourceId),
        target: nextNodeIndex.get(targetId),
        view: viewEdge,
      })
    })

    nodeIndex = nextNodeIndex
    edgeIndex = nextEdgeIndex

    const nodes = getNodes()
    const edges = getEdges()
    const activeSimulation = ensureSimulation(nodes, edges)
    if (nodes.length > 150) {
      activeSimulation.alphaDecay(0.06)
      activeSimulation.velocityDecay(0.5)
    } else {
      activeSimulation.alphaDecay(0.032)
      activeSimulation.velocityDecay(0.34)
    }
    if (newNodeIds.length) {
      const prewarmTicks = wasEmpty
        ? Math.min(40, Math.max(14, Math.round(nodes.length * 1.6)))
        : Math.min(20, Math.max(6, Math.round(newNodeIds.length * 1.5)))
      activeSimulation.alpha(Math.max(activeSimulation.alpha(), wasEmpty ? 0.92 : 0.42))
      activeSimulation.tick(prewarmTicks)
    }
    activeSimulation.alphaTarget(newNodeIds.length ? 0.3 : 0.18).restart()

    if (alphaTimer) {
      window.clearTimeout(alphaTimer)
      alphaTimer = 0
    }
    alphaTimer = window.setTimeout(() => {
      simulation?.alphaTarget(0)
      alphaTimer = 0
    }, newNodeIds.length ? 1800 : 500)

    return {
      totalNodes: nodes.length,
      newNodeCount: newNodeIds.length,
      wasEmpty,
    }
  }

  function setNodeFixed(nodeId, position = {}, persist = true) {
    const node = getNodeById(nodeId)
    if (!node) return
    node.isUserPinned = Boolean(persist)
    if (persist) {
      node.fx = Number.isFinite(position.x) ? position.x : node.x
      node.fy = Number.isFinite(position.y) ? position.y : node.y
    } else {
      node.x = Number.isFinite(position.x) ? position.x : node.x
      node.y = Number.isFinite(position.y) ? position.y : node.y
      node.fx = null
      node.fy = null
    }
  }

  function getNodeSnapshot(nodeId) {
    const node = getNodeById(nodeId)
    if (!node) return null
    return {
      id: node.id,
      x: Number(node.x || 0),
      y: Number(node.y || 0),
      fx: Number.isFinite(node.fx) ? node.fx : null,
      fy: Number.isFinite(node.fy) ? node.fy : null,
      collisionRadius: Number(node.collisionRadius || node.view?.style?.collisionRadius || 48),
    }
  }

  function getNodeSnapshots() {
    return getNodes().map((node) => ({
      id: node.id,
      x: Number(node.x || 0),
      y: Number(node.y || 0),
      fx: Number.isFinite(node.fx) ? node.fx : null,
      fy: Number.isFinite(node.fy) ? node.fy : null,
      collisionRadius: Number(node.collisionRadius || node.view?.style?.collisionRadius || 48),
    }))
  }

  function reheat(options = {}) {
    if (!simulation) return
    const config =
      typeof options === 'number'
        ? { alphaTarget: options }
        : (options || {})
    const alphaTarget = clamp(Number(config.alphaTarget ?? 0.3), 0, 1)
    const duration = Math.max(0, Number(config.duration ?? 1200))
    const alpha = Number.isFinite(config.alpha) ? clamp(Number(config.alpha), 0, 1) : null

    simulation.alphaTarget(alphaTarget)
    if (alpha != null) {
      simulation.alpha(Math.max(simulation.alpha(), alpha))
    }
    simulation.restart()

    if (alphaTimer) {
      window.clearTimeout(alphaTimer)
    }
    if (!duration) {
      simulation.alphaTarget(0)
      alphaTimer = 0
      return
    }
    alphaTimer = window.setTimeout(() => {
      simulation?.alphaTarget(0)
      alphaTimer = 0
    }, duration)
  }

  function destroy() {
    if (alphaTimer) {
      window.clearTimeout(alphaTimer)
      alphaTimer = 0
    }
    simulation?.stop()
    simulation = null
    nodeIndex = new Map()
    edgeIndex = new Map()
  }

  return {
    syncGraph,
    getNodes,
    getEdges,
    getNodeById,
    getNodeSnapshot,
    getNodeSnapshots,
    setNodeFixed,
    reheat,
    destroy,
    get simulation() {
      return simulation
    },
  }
}
