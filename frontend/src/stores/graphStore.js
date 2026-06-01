import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'

import {
  fetchFrontierBriefing,
  fetchFullGraph,
  fetchGraphCatalog,
  fetchGraphNeighbors,
  fetchGraphSeeds,
  fetchKnowledgeDetail,
} from '../api/graph'
import { fetchRecommendedPath, fetchUnifiedPath } from '../api/recommend'
import {
  NEIGHBOR_LIMIT,
  MAX_EXPAND_DEPTH,
  MAX_VISIBLE_NODES,
  createCircularPlacement,
  createDetachedRootOrigin,
  createExpansionPlacement,
  getRelationPriority,
} from '../utils/graphLayout'
import { createSearchIndex, searchInIndex } from '../utils/searchIndex'
import { transformGraphData } from '../utils/graphTransform'
import { getCourseOptions } from '../utils/colorMap'
import {
  ALL_COURSE_TITLE,
  COURSE_TITLE_ORDER,
  DEFAULT_COURSE_TITLE,
  normalizeCourseTitle,
  resolveNodeCourseTitle,
} from '../utils/courseMeta'
import { useUserStore } from './userStore'
import { useThemeStore } from './themeStore'

function getNormalizedCourseTitle(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}

function normalizeCatalogNode(node = {}) {
  const id = String(node.id || node.routeKnowledgeId || '').trim()
  const sourcePath = String(node.sourcePath || '').trim()
  const curatedSource = String(node.curatedSource || '').trim()
  return {
    id,
    routeKnowledgeId: String(node.routeKnowledgeId || id).trim() || id,
    name: String(node.name || node.canonicalName || id).trim() || id,
    canonicalName: String(node.canonicalName || node.name || id).trim() || id,
    chapterTitle: getNormalizedCourseTitle(node),
    rawChapterTitle: String(node.chapterTitle || '').trim(),
    entityType: String(node.entityType || 'concept').trim() || 'concept',
    description: String(node.description || '').trim(),
    sourcePath,
    curatedSource,
  }
}

function normalizeKnowledgeDetail(detail = {}) {
  const normalizedNode = normalizeCatalogNode(detail)
  return {
    ...detail,
    ...normalizedNode,
    summary: String(detail.summary || '').trim(),
    recommendReason: String(detail.recommendReason || '').trim(),
    prerequisites: Array.isArray(detail.prerequisites) ? detail.prerequisites.filter(Boolean) : [],
    related: Array.isArray(detail.related) ? detail.related.filter(Boolean) : [],
    scenarios: Array.isArray(detail.scenarios) ? detail.scenarios.filter(Boolean) : [],
    parentTopics: Array.isArray(detail.parentTopics) ? detail.parentTopics.filter(Boolean) : [],
  }
}

function normalizeVisibleNode(node = {}, overrides = {}) {
  const base = normalizeCatalogNode(node)
  return {
    ...base,
    x: Number.isFinite(overrides.x) ? overrides.x : Number.isFinite(node.x) ? node.x : 0,
    y: Number.isFinite(overrides.y) ? overrides.y : Number.isFinite(node.y) ? node.y : 0,
    fx: Number.isFinite(overrides.fx) ? overrides.fx : Number.isFinite(node.fx) ? node.fx : null,
    fy: Number.isFinite(overrides.fy) ? overrides.fy : Number.isFinite(node.fy) ? node.fy : null,
    depth: Number.isFinite(overrides.depth) ? overrides.depth : Number(node.depth || 0) || 0,
    rootId: String(overrides.rootId || node.rootId || base.id).trim() || base.id,
    isRoot: Boolean(overrides.isRoot ?? node.isRoot),
    expandMeta: overrides.expandMeta ?? node.expandMeta ?? null,
    layoutLock: overrides.layoutLock ?? node.layoutLock ?? null,
  }
}

function normalizeVisibleEdge(edge = {}) {
  const sourceId = String(edge.sourceId || edge.source || '').trim()
  const targetId = String(edge.targetId || edge.target || '').trim()
  const relationType = String(edge.relationType || 'RELATED_TO').trim() || 'RELATED_TO'
  const relationId = String(edge.relationId || edge.id || `${sourceId}:${relationType}:${targetId}`).trim()
  return {
    id: relationId,
    relationId,
    relationType,
    sourceId,
    targetId,
    sourceName: String(edge.sourceName || '').trim(),
    targetName: String(edge.targetName || '').trim(),
  }
}

function cloneMapOfSets(map) {
  return new Map([...map.entries()].map(([key, value]) => [key, new Set(value)]))
}

function cloneSet(value) {
  return new Set(value || [])
}

function addOwner(ownerMap, key, ownerKey) {
  const next = new Map(ownerMap)
  const owners = new Set(next.get(key) || [])
  owners.add(ownerKey)
  next.set(key, owners)
  return next
}

function removeOwner(ownerMap, key, ownerKey) {
  const next = new Map(ownerMap)
  const owners = new Set(next.get(key) || [])
  owners.delete(ownerKey)
  if (owners.size) next.set(key, owners)
  else next.delete(key)
  return next
}

function isExpansionOwner(ownerKey = '') {
  return String(ownerKey || '').startsWith('expand:')
}

function mergeVisibleNode(existing, incoming) {
  if (!existing) return incoming
  return {
    ...existing,
    ...incoming,
    x: Number.isFinite(existing.x) ? existing.x : incoming.x,
    y: Number.isFinite(existing.y) ? existing.y : incoming.y,
    fx: Number.isFinite(existing.fx) ? existing.fx : incoming.fx,
    fy: Number.isFinite(existing.fy) ? existing.fy : incoming.fy,
    depth: Math.min(Number(existing.depth || 0), Number(incoming.depth || 0)),
    isRoot: Boolean(existing.isRoot || incoming.isRoot),
    rootId: existing.isRoot ? existing.rootId : incoming.rootId,
    expandMeta: incoming.expandMeta ?? existing.expandMeta ?? null,
    layoutLock: incoming.layoutLock ?? existing.layoutLock ?? null,
  }
}

function buildEdgeId(edge) {
  return String(edge.id || edge.relationId || `${edge.sourceId}:${edge.relationType}:${edge.targetId}`).trim()
}

function buildLayoutLock(node, expiresAt) {
  if (!node) return null
  return {
    x: Number(node.x || 0),
    y: Number(node.y || 0),
    expiresAt,
  }
}

export const useGraphStore = defineStore('graph', () => {
  const userStore = useUserStore()
  const themeStore = useThemeStore()

  const catalogNodes = ref([])
  const prerequisiteIndex = ref([])

  const loading = ref(false)
  const error = ref('')

  const selectedNodeId = ref(null)
  const focusedNodeId = ref(null)
  const highlightFilter = ref(null)
  const highlightPath = ref([])
  const activeCourseFilter = ref(ALL_COURSE_TITLE)

  const detailCache = ref({})
  const frontierCache = ref({})
  const recommendCache = ref({})

  const nodesMap = shallowRef(new Map())
  const edgesMap = shallowRef(new Map())
  const nodeOwners = shallowRef(new Map())
  const edgeOwners = shallowRef(new Map())
  const rootNodeIds = shallowRef(new Set())
  const expandedNodeIds = shallowRef(new Set())
  const expandedBy = shallowRef(new Map())
  const pinnedNodeIds = shallowRef(new Set())
  const expansionRecords = shallowRef(new Map())
  const expansionChildrenMap = ref({})

  const rawNodes = computed(() => catalogNodes.value)
  const rawEdges = computed(() => prerequisiteIndex.value)

  const routeNodeMap = computed(() => {
    const map = new Map()
    catalogNodes.value.forEach((node) => {
      const normalized = normalizeCatalogNode(node)
      ;[normalized.routeKnowledgeId, normalized.name, normalized.canonicalName].forEach((key) => {
        const value = String(key || '').trim()
        if (value) map.set(value, normalized)
      })
    })
    return map
  })

  const searchIndex = computed(() => createSearchIndex(catalogNodes.value))
  const courseOptions = computed(() => getCourseOptions(catalogNodes.value))
  const courseGroups = computed(() => {
    const groups = {}
    catalogNodes.value.forEach((node) => {
      const chapterTitle = getNormalizedCourseTitle(node)
      const progress = userStore.getProgressMeta(node.routeKnowledgeId)
      if (!groups[chapterTitle]) {
        groups[chapterTitle] = {
          title: chapterTitle,
          total: 0,
          mastered: 0,
          openDepth: 0,
          maxDepth: 1,
          nodes: [],
        }
      }
      groups[chapterTitle].total += 1
      if (Number(progress.masteryScore || 0) >= 80) groups[chapterTitle].mastered += 1
      groups[chapterTitle].nodes.push(node)
    })
    const orderIndex = new Map(COURSE_TITLE_ORDER.map((title, index) => [title, index]))
    return Object.values(groups).sort((left, right) => {
      const leftRank = orderIndex.get(left.title) ?? Number.MAX_SAFE_INTEGER
      const rightRank = orderIndex.get(right.title) ?? Number.MAX_SAFE_INTEGER
      if (leftRank !== rightRank) return leftRank - rightRank
      if (right.total !== left.total) return right.total - left.total
      return String(left.title || '').localeCompare(String(right.title || ''), 'zh-CN')
    })
  })

  const prerequisiteParentMap = computed(() => {
    const map = new Map()
    prerequisiteIndex.value.forEach((edge) => {
      const source = String(edge.sourceRouteKnowledgeId || '').trim()
      const target = String(edge.targetRouteKnowledgeId || '').trim()
      if (!source || !target) return
      if (!map.has(target)) map.set(target, new Set())
      map.get(target).add(source)
    })
    return map
  })

  const prerequisiteNextMap = computed(() => {
    const map = new Map()
    prerequisiteIndex.value.forEach((edge) => {
      const source = String(edge.sourceRouteKnowledgeId || '').trim()
      const target = String(edge.targetRouteKnowledgeId || '').trim()
      if (!source || !target) return
      if (!map.has(source)) map.set(source, new Set())
      map.get(source).add(target)
    })
    return map
  })

  const graphData = computed(() =>
    transformGraphData([...nodesMap.value.values()], [...edgesMap.value.values()], {
      progressMap: userStore.progressMap,
      bookmarks: new Set(userStore.bookmarks),
      expandedNodeIds: [...expandedNodeIds.value],
      pinnedNodeIds: [...pinnedNodeIds.value],
      selectedNodeId: selectedNodeId.value,
      rootNodeIds: [...rootNodeIds.value],
      totalNodes: catalogNodes.value.length,
      theme: themeStore.theme,
    }),
  )

  const nodeMap = computed(() => new Map(graphData.value.nodes.map((node) => [node.id, node])))
  const selectedNode = computed(() => {
    const key = String(selectedNodeId.value || '').trim()
    return key ? nodesMap.value.get(key) || null : null
  })
  const selectedRouteKnowledgeId = computed(() => selectedNode.value?.routeKnowledgeId || null)
  const selectedDetail = computed(() => detailCache.value[selectedRouteKnowledgeId.value] || null)
  const visibleNodeIds = computed(() => graphData.value.nodes.map((node) => node.id))
  const visibleNodeIdSet = computed(() => new Set(visibleNodeIds.value))
  const adjacentMap = computed(() => {
    const map = new Map()
    graphData.value.nodes.forEach((node) => {
      map.set(node.id, {
        nodes: new Set([node.id]),
        edges: new Set(),
      })
    })
    graphData.value.edges.forEach((edge) => {
      if (!map.has(edge.source) || !map.has(edge.target)) return
      map.get(edge.source).nodes.add(edge.target)
      map.get(edge.target).nodes.add(edge.source)
      map.get(edge.source).edges.add(edge.id)
      map.get(edge.target).edges.add(edge.id)
    })
    return map
  })
  const seedNodeIds = computed(() => [...rootNodeIds.value])

  function rebuildExpansionChildrenMap(nextExpansionRecords = expansionRecords.value) {
    const next = {}
    nextExpansionRecords.forEach((record, nodeId) => {
      next[nodeId] = [...record.childIds]
    })
    expansionChildrenMap.value = next
  }

  function setCourseFilter(courseTitle = ALL_COURSE_TITLE) {
    activeCourseFilter.value = normalizeCourseTitle(courseTitle, ALL_COURSE_TITLE)
  }

  function setFocusedNode(nodeId) {
    focusedNodeId.value = String(nodeId || '').trim() || null
    if (focusedNodeId.value) pinNode(focusedNodeId.value)
  }

  function pinNode(nodeId) {
    const key = String(nodeId || '').trim()
    if (!key) return
    const next = cloneSet(pinnedNodeIds.value)
    next.add(key)
    pinnedNodeIds.value = next
  }

  function unpinNode(nodeId) {
    const key = String(nodeId || '').trim()
    if (!key) return
    const next = cloneSet(pinnedNodeIds.value)
    next.delete(key)
    pinnedNodeIds.value = next
  }

  function clearCanvasGraph() {
    nodesMap.value = new Map()
    edgesMap.value = new Map()
    nodeOwners.value = new Map()
    edgeOwners.value = new Map()
    rootNodeIds.value = new Set()
    expandedNodeIds.value = new Set()
    expandedBy.value = new Map()
    pinnedNodeIds.value = new Set()
    expansionRecords.value = new Map()
    rebuildExpansionChildrenMap(new Map())
    selectedNodeId.value = null
    focusedNodeId.value = null
    highlightPath.value = []
  }

  function getVisibleNode(nodeId) {
    return nodesMap.value.get(String(nodeId || '').trim()) || null
  }

  function getDetachedRevealOrigin(nodeId, anchorPosition = null) {
    const rootCount = rootNodeIds.value.size
    const point = Number.isFinite(anchorPosition?.x) && Number.isFinite(anchorPosition?.y)
      ? (() => {
          const hashAngle = ((String(nodeId || '').length * 47 + rootCount * 29) % 360) * (Math.PI / 180)
          const radius = 82 + (rootCount % 6) * 18
          return {
            x: anchorPosition.x + Math.cos(hashAngle) * radius,
            y: anchorPosition.y + Math.sin(hashAngle) * radius * 0.72,
          }
        })()
      : createDetachedRootOrigin(rootCount)
    const jitter = (String(nodeId || '').length % 7) * 18
    return {
      x: point.x + jitter,
      y: point.y - jitter * 0.6,
    }
  }

  function ensureVisibleCapacity(incomingNodes = []) {
    const uniqueIncoming = incomingNodes.filter((node) => !nodesMap.value.has(node.id))
    if (nodesMap.value.size + uniqueIncoming.length > MAX_VISIBLE_NODES) {
      error.value = '当前图谱节点过多，请先收起部分子图，再继续展开。'
      return false
    }
    return true
  }

  function getNeighborRequestLimit(overrideLimit = null) {
    if (Number.isFinite(overrideLimit) && overrideLimit > 0) {
      return overrideLimit
    }
    return nodesMap.value.size > 180 ? 12 : NEIGHBOR_LIMIT
  }

  function applySeedGraph(payload) {
    const nextNodes = new Map()
    const nextEdges = new Map()
    const nextNodeOwners = new Map()
    const nextEdgeOwners = new Map()
    const nextRoots = new Set()

    const normalizedNodes = (payload?.nodes || []).map((node) => normalizeVisibleNode(node, { isRoot: true, depth: 0 }))
    const placements = createCircularPlacement(normalizedNodes.length, {
      spacing: 132,
      yScale: 0.88,
    })
    normalizedNodes.forEach((node, index) => {
      const placement = placements[index] || { x: 0, y: 0 }
      const positioned = {
        ...node,
        x: placement.x,
        y: placement.y,
        rootId: node.id,
        isRoot: true,
      }
      nextNodes.set(positioned.id, positioned)
      nextNodeOwners.set(positioned.id, new Set([`root:${positioned.id}`]))
      nextRoots.add(positioned.id)
    })

    ;(payload?.edges || []).map((edge) => normalizeVisibleEdge(edge)).forEach((edge) => {
      if (!nextNodes.has(edge.sourceId) || !nextNodes.has(edge.targetId)) return
      nextEdges.set(edge.id, edge)
      nextEdgeOwners.set(edge.id, new Set(['seed']))
    })

    nodesMap.value = nextNodes
    edgesMap.value = nextEdges
    nodeOwners.value = nextNodeOwners
    edgeOwners.value = nextEdgeOwners
    rootNodeIds.value = nextRoots
    expandedNodeIds.value = new Set()
    expandedBy.value = new Map()
    expansionRecords.value = new Map()
    rebuildExpansionChildrenMap(new Map())
  }

  function applyNeighborGraph(targetId, payload, options = {}) {
    const ownerKey = `expand:${targetId}`
    const asDetachedRoot = Boolean(options.asDetachedRoot)
    const anchorPosition = options.anchorPosition || null
    const occupiedNodes = Array.isArray(options.occupiedNodes) ? options.occupiedNodes : []
    const viewportCenterWorld = options.viewportCenterWorld || null
    const expansionStartedAt = Date.now()
    const normalizedNodes = (payload?.nodes || []).map((node) => normalizeVisibleNode(node))
    const normalizedEdges = (payload?.edges || []).map((edge) => normalizeVisibleEdge(edge))

    if (!ensureVisibleCapacity(normalizedNodes)) {
      return []
    }

    const currentNodes = new Map(nodesMap.value)
    const currentEdges = new Map(edgesMap.value)
    let nextNodeOwners = cloneMapOfSets(nodeOwners.value)
    let nextEdgeOwners = cloneMapOfSets(edgeOwners.value)
    let nextRoots = cloneSet(rootNodeIds.value)
    let nextExpanded = cloneSet(expandedNodeIds.value)
    let nextExpandedBy = cloneMapOfSets(expandedBy.value)
    const nextExpansionRecords = new Map(expansionRecords.value)

    const existingTarget = currentNodes.get(targetId)
    const catalogTarget = routeNodeMap.value.get(targetId)
    const baseTarget = normalizeVisibleNode(normalizedNodes.find((node) => node.id === targetId) || catalogTarget || { id: targetId }, {
      isRoot: asDetachedRoot || Boolean(existingTarget?.isRoot),
      depth: asDetachedRoot ? 0 : Number(existingTarget?.depth || 0),
      rootId: asDetachedRoot ? targetId : existingTarget?.rootId || targetId,
      x: Number.isFinite(anchorPosition?.x) ? anchorPosition.x : existingTarget?.x,
      y: Number.isFinite(anchorPosition?.y) ? anchorPosition.y : existingTarget?.y,
      fx: existingTarget?.fx,
      fy: existingTarget?.fy,
    })

    if (!existingTarget) {
      const origin = getDetachedRevealOrigin(targetId, anchorPosition)
      currentNodes.set(
        targetId,
        normalizeVisibleNode(baseTarget, {
          isRoot: true,
          depth: 0,
          rootId: targetId,
          x: origin.x,
          y: origin.y,
        }),
      )
      nextRoots.add(targetId)
      nextNodeOwners = addOwner(nextNodeOwners, targetId, `root:${targetId}`)
    } else if (asDetachedRoot && !existingTarget.isRoot) {
      currentNodes.set(
        targetId,
        mergeVisibleNode(existingTarget, {
          ...baseTarget,
          isRoot: true,
          depth: 0,
          rootId: targetId,
        }),
      )
      nextRoots.add(targetId)
      nextNodeOwners = addOwner(nextNodeOwners, targetId, `root:${targetId}`)
    }

    if (currentNodes.has(targetId) && Number.isFinite(anchorPosition?.x) && Number.isFinite(anchorPosition?.y)) {
      const currentTarget = currentNodes.get(targetId)
      currentNodes.set(targetId, {
        ...currentTarget,
        x: anchorPosition.x,
        y: anchorPosition.y,
        layoutLock: {
          x: anchorPosition.x,
          y: anchorPosition.y,
          expiresAt: expansionStartedAt + (asDetachedRoot ? 1100 : 900),
        },
      })
    }

    const anchorNode = currentNodes.get(targetId) || baseTarget
    const relationPriorityByNodeId = new Map()
    normalizedEdges.forEach((edge) => {
      if (edge.sourceId !== targetId && edge.targetId !== targetId) return
      const neighborId = edge.sourceId === targetId ? edge.targetId : edge.sourceId
      if (!neighborId) return
      const currentPriority = relationPriorityByNodeId.get(neighborId)
      const nextPriority = getRelationPriority(edge.relationType)
      if (currentPriority == null || nextPriority < currentPriority) {
        relationPriorityByNodeId.set(neighborId, nextPriority)
      }
    })

    const childNodes = normalizedNodes
      .filter((node) => node.id && node.id !== targetId)
      .sort((left, right) => {
        const priorityDelta = (relationPriorityByNodeId.get(left.id) ?? 99) - (relationPriorityByNodeId.get(right.id) ?? 99)
        if (priorityDelta !== 0) return priorityDelta
        return String(left.canonicalName || left.name || left.id).localeCompare(String(right.canonicalName || right.name || right.id), 'zh-CN')
      })
    const newChildNodes = childNodes.filter((node) => !currentNodes.has(node.id))
    const placement = createExpansionPlacement(anchorNode, newChildNodes.length, currentNodes.size, {
      occupiedNodes,
      viewportCenterWorld,
    })
    const childIds = new Set()
    const newChildIds = []
    const edgeIds = new Set()
    const layoutLockExpiresAt = expansionStartedAt + 720
    let placementIndex = 0

    childNodes.forEach((node) => {
      const existing = currentNodes.get(node.id)
      const slot = existing ? null : placement[placementIndex++]
      const relationPriority = relationPriorityByNodeId.get(node.id) ?? 99
      const radialOffset = slot ? Math.min(Math.max(relationPriority, 0), 5) * 14 : 0
      const positionedX = slot ? anchorNode.x + Math.cos(slot.angle) * (slot.radius + radialOffset) : existing?.x ?? anchorNode.x
      const positionedY = slot ? anchorNode.y + Math.sin(slot.angle) * (slot.radius + radialOffset) : existing?.y ?? anchorNode.y
      const expandMeta =
        existing || !slot
          ? existing?.expandMeta || null
          : {
              anchorId: targetId,
              x: anchorNode.x,
              y: anchorNode.y,
              targetX: positionedX,
              targetY: positionedY,
              radius: slot.radius + radialOffset,
              createdAt: expansionStartedAt,
              expiresAt: expansionStartedAt + 1800,
            }
      const positioned = normalizeVisibleNode(node, {
        x: existing?.x ?? positionedX,
        y: existing?.y ?? positionedY,
        fx: existing?.fx,
        fy: existing?.fy,
        depth: Math.min(Number(existing?.depth ?? MAX_EXPAND_DEPTH), Number(anchorNode.depth || 0) + 1),
        rootId: existing?.rootId || anchorNode.rootId || targetId,
        isRoot: Boolean(existing?.isRoot),
        expandMeta,
        layoutLock:
          existing && !existing.isRoot
            ? {
                x: existing.x,
                y: existing.y,
                expiresAt: layoutLockExpiresAt,
              }
            : existing?.layoutLock ?? null,
      })
      currentNodes.set(node.id, mergeVisibleNode(existing, positioned))
      nextNodeOwners = addOwner(nextNodeOwners, node.id, ownerKey)
      const currentParents = new Set(nextExpandedBy.get(node.id) || [])
      currentParents.add(targetId)
      nextExpandedBy.set(node.id, currentParents)
      childIds.add(node.id)
      if (!existing) newChildIds.push(node.id)
    })

    normalizedEdges.forEach((edge) => {
      const edgeId = buildEdgeId(edge)
      const normalized = { ...edge, id: edgeId }
      if (!currentNodes.has(normalized.sourceId) || !currentNodes.has(normalized.targetId)) return
      currentEdges.set(edgeId, normalized)
      nextEdgeOwners = addOwner(nextEdgeOwners, edgeId, ownerKey)
      edgeIds.add(edgeId)
    })

    nextExpanded.add(targetId)
    nextExpansionRecords.set(targetId, {
      childIds,
      edgeIds,
    })

    nodesMap.value = currentNodes
    edgesMap.value = currentEdges
    nodeOwners.value = nextNodeOwners
    edgeOwners.value = nextEdgeOwners
    rootNodeIds.value = nextRoots
    expandedNodeIds.value = nextExpanded
    expandedBy.value = nextExpandedBy
    expansionRecords.value = nextExpansionRecords
    rebuildExpansionChildrenMap(nextExpansionRecords)

    return newChildIds
  }

  async function loadCatalog(options = {}) {
    const silent = Boolean(options.silent)
    if (!silent) {
      loading.value = true
      error.value = ''
    }

    try {
      const payload = await fetchGraphCatalog()
      catalogNodes.value = (payload?.nodes || []).map((node) => normalizeCatalogNode(node)).filter((node) => node.id)
      prerequisiteIndex.value = (payload?.prerequisiteEdges || [])
        .map((edge) => ({
          sourceRouteKnowledgeId: String(edge.sourceRouteKnowledgeId || '').trim(),
          targetRouteKnowledgeId: String(edge.targetRouteKnowledgeId || '').trim(),
        }))
        .filter((edge) => edge.sourceRouteKnowledgeId && edge.targetRouteKnowledgeId)
      return payload
    } catch (loadError) {
      error.value = loadError?.message || '图谱目录加载失败，请检查后端服务。'
      throw loadError
    } finally {
      if (!silent) loading.value = false
    }
  }

  async function loadSeeds(count = 14, options = {}) {
    const silent = Boolean(options.silent)
    if (!silent) {
      loading.value = true
      error.value = ''
    }

    try {
      const payload = await fetchGraphSeeds(count)
      clearCanvasGraph()
      applySeedGraph(payload)
      return payload
    } catch (loadError) {
      error.value = loadError?.message || '知识图谱种子加载失败，请检查后端服务。'
      throw loadError
    } finally {
      if (!silent) loading.value = false
    }
  }

  async function initGraph() {
    loading.value = true
    error.value = ''
    try {
      await Promise.all([loadCatalog({ silent: true }), loadSeeds(14, { silent: true })])
    } catch (initError) {
      error.value = initError?.message || '知识图谱初始化失败，请检查后端服务是否运行。'
      throw initError
    } finally {
      loading.value = false
    }
  }

  async function loadFullGraph(limit = 1600) {
    loading.value = true
    error.value = ''
    try {
      const payload = await fetchFullGraph(limit)
      const nodes = (payload?.nodes || []).map((node) => normalizeCatalogNode(node))
      const edges = (payload?.edges || []).map((edge) => normalizeVisibleEdge(edge))
      catalogNodes.value = nodes
      prerequisiteIndex.value = edges
        .filter((edge) => edge.relationType === 'PREREQUISITE_OF')
        .map((edge) => ({
          sourceRouteKnowledgeId: edge.sourceId,
          targetRouteKnowledgeId: edge.targetId,
        }))
      clearCanvasGraph()
      applySeedGraph({
        nodes,
        edges,
      })
      return payload
    } catch (loadError) {
      error.value = loadError?.message || '知识图谱加载失败，请检查后端服务是否运行。'
      throw loadError
    } finally {
      loading.value = false
    }
  }

  function isNodeExpanded(nodeId) {
    return expandedNodeIds.value.has(String(nodeId || '').trim())
  }

  async function expandNode(nodeId, options = {}) {
    const key = String(nodeId || '').trim()
    if (!key) return []
    const current = getVisibleNode(key)
    if (!current) return []
    if (Number(current.depth || 0) >= MAX_EXPAND_DEPTH) return []
    if (isNodeExpanded(key)) return []

    error.value = ''
    const payload = await fetchGraphNeighbors(key, {
      limit: getNeighborRequestLimit(options.limit),
      depth: options.depth ?? 1,
      direction: options.direction || 'both',
    })
    return applyNeighborGraph(key, payload, {
      asDetachedRoot: false,
      anchorPosition: options.anchorPosition,
      occupiedNodes: options.occupiedNodes,
      viewportCenterWorld: options.viewportCenterWorld,
    })
  }

  async function collapseNode(nodeId) {
    const key = String(nodeId || '').trim()
    if (!key || !expandedNodeIds.value.has(key)) return

    const record = expansionRecords.value.get(key)
    if (!record) return
    const ownerKey = `expand:${key}`
    const collapseStartedAt = Date.now()
    const collapseLockUntil = collapseStartedAt + 760

    for (const childId of record.childIds) {
      const ownersAfterRemoval = new Set(nodeOwners.value.get(childId) || [])
      ownersAfterRemoval.delete(ownerKey)
      if (expandedNodeIds.value.has(childId) && ownersAfterRemoval.size === 0) {
        await collapseNode(childId)
      }
    }

    const nextNodes = new Map(nodesMap.value)
    const nextEdges = new Map(edgesMap.value)
    let nextNodeOwners = cloneMapOfSets(nodeOwners.value)
    let nextEdgeOwners = cloneMapOfSets(edgeOwners.value)
    const nextExpanded = cloneSet(expandedNodeIds.value)
    const nextExpandedBy = cloneMapOfSets(expandedBy.value)
    const nextExpansionRecords = new Map(expansionRecords.value)

    record.edgeIds.forEach((edgeId) => {
      nextEdgeOwners = removeOwner(nextEdgeOwners, edgeId, ownerKey)
      if (!nextEdgeOwners.has(edgeId)) {
        nextEdges.delete(edgeId)
      }
    })

    record.childIds.forEach((childId) => {
      nextNodeOwners = removeOwner(nextNodeOwners, childId, ownerKey)
      const parentRefs = new Set(nextExpandedBy.get(childId) || [])
      parentRefs.delete(key)
      if (parentRefs.size) nextExpandedBy.set(childId, parentRefs)
      else nextExpandedBy.delete(childId)

      if (!nextNodeOwners.has(childId)) {
        nextNodes.delete(childId)
        if (selectedNodeId.value === childId) {
          selectedNodeId.value = key
          focusedNodeId.value = key
        }
      }
    })

    const survivorLockIds = new Set([key])
    nextEdges.forEach((edge) => {
      const touchesCollapsedCluster =
        edge.sourceId === key ||
        edge.targetId === key ||
        record.childIds.has(edge.sourceId) ||
        record.childIds.has(edge.targetId)
      if (!touchesCollapsedCluster) return

      if (nextNodes.has(edge.sourceId)) survivorLockIds.add(edge.sourceId)
      if (nextNodes.has(edge.targetId)) survivorLockIds.add(edge.targetId)
    })

    survivorLockIds.forEach((survivorId) => {
      const survivor = nextNodes.get(survivorId)
      if (!survivor) return
      nextNodes.set(survivorId, {
        ...survivor,
        layoutLock: buildLayoutLock(survivor, collapseLockUntil),
      })
    })

    nextExpanded.delete(key)
    nextExpansionRecords.delete(key)

    nodesMap.value = nextNodes
    edgesMap.value = nextEdges
    nodeOwners.value = nextNodeOwners
    edgeOwners.value = nextEdgeOwners
    expandedNodeIds.value = nextExpanded
    expandedBy.value = nextExpandedBy
    expansionRecords.value = nextExpansionRecords
    rebuildExpansionChildrenMap(nextExpansionRecords)
  }

  async function toggleNodeExpansion(nodeId, options = {}) {
    const key = String(nodeId || '').trim()
    if (!key) return []
    if (expandedNodeIds.value.has(key)) {
      await collapseNode(key)
      return []
    }
    return expandNode(key, options)
  }

  async function collapseAllExpansions() {
    const candidates = [...expandedNodeIds.value].sort((left, right) => {
      const leftDepth = Number(nodesMap.value.get(left)?.depth || 0)
      const rightDepth = Number(nodesMap.value.get(right)?.depth || 0)
      return rightDepth - leftDepth
    })
    for (const nodeId of candidates) {
      await collapseNode(nodeId)
    }
    pinnedNodeIds.value = new Set()
  }

  async function collapseAllCourses() {
    await collapseAllExpansions()
  }

  async function ensureNodeDetail(routeKnowledgeId) {
    const key = String(routeKnowledgeId || '').trim()
    if (!key) return null
    if (detailCache.value[key]) return detailCache.value[key]

    const detail = normalizeKnowledgeDetail(await fetchKnowledgeDetail(key))
    detailCache.value = {
      ...detailCache.value,
      [key]: detail,
    }
    return detail
  }

  async function ensureFrontier(knowledgeId) {
    const key = String(knowledgeId || '').trim()
    if (!key) return null
    if (frontierCache.value[key]) return frontierCache.value[key]

    try {
      const catalogNode = routeNodeMap.value.get(key)
      const briefing = await fetchFrontierBriefing({
        knowledgeId: key,
        knowledge: catalogNode?.canonicalName || catalogNode?.name || key,
      })
      frontierCache.value = {
        ...frontierCache.value,
        [key]: briefing,
      }
      return briefing
    } catch {
      frontierCache.value = {
        ...frontierCache.value,
        [key]: null,
      }
      return null
    }
  }

  async function ensureRecommendedPath(target, knowledgeId, scene = 'detail') {
    const cacheKey = `${target || ''}|${knowledgeId || ''}|${scene}`
    if (recommendCache.value[cacheKey]) return recommendCache.value[cacheKey]

    try {
      const payload = await fetchUnifiedPath({ target, knowledgeId, scene, limit: 10 })
      recommendCache.value = {
        ...recommendCache.value,
        [cacheKey]: payload,
      }
      return payload
    } catch {
      recommendCache.value = {
        ...recommendCache.value,
        [cacheKey]: { path: [] },
      }
      return { path: [] }
    }
  }

  async function selectNode(nodeId) {
    const key = String(nodeId || '').trim()
    if (!key) return
    const node = getVisibleNode(key)
    if (!node) return

    selectedNodeId.value = key
    focusedNodeId.value = key
    pinNode(key)

    if (node.routeKnowledgeId) {
      await ensureNodeDetail(node.routeKnowledgeId)
    }
  }

  function clearFocus() {
    focusedNodeId.value = null
    highlightFilter.value = null
    highlightPath.value = []
  }

  function clearSelection() {
    selectedNodeId.value = null
    clearFocus()
  }

  async function revealNodeByRouteKnowledgeId(routeKnowledgeId, options = {}) {
    const key = String(routeKnowledgeId || '').trim()
    if (!key) return null

    const catalogNode = routeNodeMap.value.get(key)
    if (!catalogNode?.id) return null

    if (!nodesMap.value.has(catalogNode.id)) {
      const payload = await fetchGraphNeighbors(catalogNode.id, {
        limit: getNeighborRequestLimit(),
        depth: 1,
        direction: 'both',
      })
      applyNeighborGraph(catalogNode.id, payload, {
        asDetachedRoot: true,
        anchorPosition: options.anchorPosition,
        occupiedNodes: options.occupiedNodes,
        viewportCenterWorld: options.viewportCenterWorld,
      })
    }

    pinNode(catalogNode.id)
    await selectNode(catalogNode.id)
    return getVisibleNode(catalogNode.id)
  }

  function highlightByStatus(status) {
    highlightFilter.value = highlightFilter.value === status ? null : status
  }

  function highlightPathNodes(nodeIds = []) {
    const normalized = [...nodeIds].filter(Boolean)
    highlightPath.value = normalized
    normalized.forEach((routeKnowledgeId) => {
      const node = routeNodeMap.value.get(String(routeKnowledgeId || '').trim())
      if (node?.id && nodesMap.value.has(node.id)) {
        pinNode(node.id)
      }
    })
  }

  function getPrerequisiteRouteChain(routeKnowledgeId) {
    const start = String(routeKnowledgeId || '').trim()
    if (!start) return []

    const visited = new Set([start])
    const ordered = [start]
    const queue = [start]

    while (queue.length) {
      const current = queue.shift()
      const parents = [...(prerequisiteParentMap.value.get(current) || [])]
      parents.forEach((parent) => {
        if (visited.has(parent)) return
        visited.add(parent)
        ordered.unshift(parent)
        queue.push(parent)
      })
    }

    return ordered
  }

  function getNextLearningOptions(routeKnowledgeId) {
    const start = String(routeKnowledgeId || '').trim()
    if (!start) return []
    return [...(prerequisiteNextMap.value.get(start) || [])]
      .map((item) => routeNodeMap.value.get(item)?.routeKnowledgeId || item)
      .filter(Boolean)
  }

  function searchNodes(keyword) {
    return searchInIndex(searchIndex.value, keyword)
  }

  async function focusCourseRepresentative(courseTitle, options = {}) {
    const title = normalizeCourseTitle(courseTitle, '')
    if (!title) return null

    const visibleNode =
      [...nodesMap.value.values()].find((node) => node.isRoot && getNormalizedCourseTitle(node) === title) ||
      [...nodesMap.value.values()].find((node) => getNormalizedCourseTitle(node) === title)

    if (visibleNode) {
      pinNode(visibleNode.id)
      return visibleNode
    }

    const catalogNode = catalogNodes.value.find((node) => getNormalizedCourseTitle(node) === title)
    if (!catalogNode?.routeKnowledgeId) return null

    return revealNodeByRouteKnowledgeId(catalogNode.routeKnowledgeId, options)
  }

  function getCourseDepth() {
    return 0
  }

  function getCourseMaxDepth() {
    return 1
  }

  function setCourseDepth() {
    return 0
  }

  function toggleCourse() {
    return 0
  }

  function expandCourse() {
    return 0
  }

  function deepenCourse() {
    return 0
  }

  async function deepenCourseByNode(nodeId) {
    await toggleNodeExpansion(nodeId)
    return expandedNodeIds.value.size
  }

  function collapseCourse() {
    return 0
  }

  return {
    rawNodes,
    rawEdges,
    catalogNodes,
    prerequisiteIndex,
    nodesMap,
    edgesMap,
    rootNodeIds,
    expandedNodeIds,
    expandedBy,
    pinnedNodeIds,
    expansionChildrenMap,
    loading,
    error,
    selectedNodeId,
    focusedNodeId,
    highlightFilter,
    highlightPath,
    activeCourseFilter,
    detailCache,
    frontierCache,
    recommendCache,
    nodeMap,
    routeNodeMap,
    searchIndex,
    courseOptions,
    courseGroups,
    seedNodeIds,
    selectedNode,
    selectedRouteKnowledgeId,
    selectedDetail,
    graphData,
    visibleNodeIds,
    visibleNodeIdSet,
    adjacentMap,
    prerequisiteParentMap,
    prerequisiteNextMap,
    initGraph,
    loadCatalog,
    loadSeeds,
    loadFullGraph,
    isNodeExpanded,
    pinNode,
    unpinNode,
    expandNode,
    collapseNode,
    toggleNodeExpansion,
    collapseAllExpansions,
    collapseAllCourses,
    ensureNodeDetail,
    ensureFrontier,
    ensureRecommendedPath,
    setCourseFilter,
    setFocusedNode,
    selectNode,
    clearFocus,
    clearSelection,
    revealNodeByRouteKnowledgeId,
    highlightByStatus,
    highlightPathNodes,
    getPrerequisiteRouteChain,
    getNextLearningOptions,
    searchNodes,
    focusCourseRepresentative,
    getCourseDepth,
    getCourseMaxDepth,
    setCourseDepth,
    toggleCourse,
    expandCourse,
    deepenCourse,
    deepenCourseByNode,
    collapseCourse,
  }
})
