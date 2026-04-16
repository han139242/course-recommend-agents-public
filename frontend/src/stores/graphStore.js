import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { fetchFrontierBriefing, fetchFullGraph, fetchKnowledgeDetail } from '../api/graph'
import { fetchRecommendedPath } from '../api/recommend'
import { createSearchIndex, searchInIndex } from '../utils/searchIndex'
import { buildSeedNodeIds, transformGraphData } from '../utils/graphTransform'
import { getCourseOptions } from '../utils/colorMap'
import { useUserStore } from './userStore'

function getChapterTitle(node) {
  return String(node?.chapterTitle || '其他').trim() || '其他'
}

function findNodeByRoute(nodes, routeKnowledgeId) {
  const routeId = String(routeKnowledgeId || '').trim()
  return nodes.find((node) => [node.routeKnowledgeId, node.name, node.canonicalName].includes(routeId)) || null
}

function toggleInList(list = [], value) {
  const normalized = String(value || '').trim()
  if (!normalized) return [...list]
  return list.includes(normalized) ? list.filter((item) => item !== normalized) : [...list, normalized]
}

export const useGraphStore = defineStore('graph', () => {
  const userStore = useUserStore()

  const rawNodes = ref([])
  const rawEdges = ref([])
  const loading = ref(false)
  const error = ref('')

  const selectedNodeId = ref(null)
  const focusedNodeId = ref(null)
  const expandedNodeIds = ref([])
  const pinnedNodeIds = ref([])
  const highlightFilter = ref(null)
  const highlightPath = ref([])
  const activeCourseFilter = ref('全部')

  const detailCache = ref({})
  const frontierCache = ref({})
  const recommendCache = ref({})

  const nodeMap = computed(() => {
    const map = new Map()
    rawNodes.value.forEach((node) => map.set(node.id, node))
    return map
  })

  const routeNodeMap = computed(() => {
    const map = new Map()
    rawNodes.value.forEach((node) => {
      ;[node.routeKnowledgeId, node.name, node.canonicalName].forEach((key) => {
        const value = String(key || '').trim()
        if (value) map.set(value, node)
      })
    })
    return map
  })

  const searchIndex = computed(() => createSearchIndex(rawNodes.value))
  const courseOptions = computed(() => getCourseOptions(rawNodes.value))
  const seedNodeIds = computed(() => buildSeedNodeIds(rawNodes.value, rawEdges.value, 14))

  const courseGroups = computed(() => {
    const groups = {}
    rawNodes.value.forEach((node) => {
      const chapterTitle = getChapterTitle(node)
      const progress = userStore.getProgressMeta(node.routeKnowledgeId)
      if (!groups[chapterTitle]) {
        groups[chapterTitle] = {
          title: chapterTitle,
          total: 0,
          mastered: 0,
          openDepth: 0,
          maxDepth: 0,
          nodes: [],
        }
      }
      groups[chapterTitle].total += 1
      if (Number(progress.masteryScore || 0) >= 80) groups[chapterTitle].mastered += 1
      groups[chapterTitle].nodes.push(node)
    })

    return Object.values(groups).sort((left, right) => {
      if (right.total !== left.total) return right.total - left.total
      return String(left.title || '').localeCompare(String(right.title || ''), 'zh-CN')
    })
  })

  const selectedNode = computed(() => (selectedNodeId.value ? nodeMap.value.get(selectedNodeId.value) || null : null))
  const selectedRouteKnowledgeId = computed(() => selectedNode.value?.routeKnowledgeId || null)
  const selectedDetail = computed(() => detailCache.value[selectedRouteKnowledgeId.value] || null)

  const graphData = computed(() =>
    transformGraphData(rawNodes.value, rawEdges.value, {
      progressMap: userStore.progressMap,
      bookmarks: new Set(userStore.bookmarks),
      expandedNodeIds: expandedNodeIds.value,
      pinnedNodeIds: pinnedNodeIds.value,
      selectedNodeId: selectedNodeId.value,
      seedNodeIds: seedNodeIds.value,
    }),
  )

  const visibleNodeIds = computed(() => graphData.value.nodes.map((node) => node.id))
  const visibleNodeIdSet = computed(() => new Set(visibleNodeIds.value))

  const adjacentMap = computed(() => {
    const map = new Map()
    rawNodes.value.forEach((node) => {
      map.set(node.id, {
        nodes: new Set([node.id]),
        edges: new Set(),
      })
    })

    rawEdges.value.forEach((edge) => {
      if (!map.has(edge.sourceId) || !map.has(edge.targetId)) return
      const edgeId = edge.relationId || `${edge.sourceId}-${edge.relationType || 'related'}-${edge.targetId}`
      map.get(edge.sourceId).nodes.add(edge.targetId)
      map.get(edge.targetId).nodes.add(edge.sourceId)
      map.get(edge.sourceId).edges.add(edgeId)
      map.get(edge.targetId).edges.add(edgeId)
    })

    return map
  })

  const prerequisiteParentMap = computed(() => {
    const map = new Map()
    rawEdges.value.forEach((edge) => {
      if (edge.relationType !== 'PREREQUISITE_OF') return
      const targetNode = nodeMap.value.get(edge.targetId)
      const sourceNode = nodeMap.value.get(edge.sourceId)
      if (!targetNode?.routeKnowledgeId || !sourceNode?.routeKnowledgeId) return
      const targetKey = targetNode.routeKnowledgeId
      const sourceKey = sourceNode.routeKnowledgeId
      if (!map.has(targetKey)) map.set(targetKey, new Set())
      map.get(targetKey).add(sourceKey)
    })
    return map
  })

  const prerequisiteNextMap = computed(() => {
    const map = new Map()
    rawEdges.value.forEach((edge) => {
      if (edge.relationType !== 'PREREQUISITE_OF') return
      const sourceNode = nodeMap.value.get(edge.sourceId)
      const targetNode = nodeMap.value.get(edge.targetId)
      if (!sourceNode?.routeKnowledgeId || !targetNode?.routeKnowledgeId) return
      const sourceKey = sourceNode.routeKnowledgeId
      const targetKey = targetNode.routeKnowledgeId
      if (!map.has(sourceKey)) map.set(sourceKey, new Set())
      map.get(sourceKey).add(targetKey)
    })
    return map
  })

  function isNodeExpanded(nodeId) {
    return expandedNodeIds.value.includes(String(nodeId || '').trim())
  }

  function pinNode(nodeId) {
    const normalized = String(nodeId || '').trim()
    if (!normalized) return
    if (seedNodeIds.value.includes(normalized)) return
    if (pinnedNodeIds.value.includes(normalized)) return
    pinnedNodeIds.value = [...pinnedNodeIds.value, normalized]
  }

  function unpinNode(nodeId) {
    const normalized = String(nodeId || '').trim()
    if (!normalized) return
    pinnedNodeIds.value = pinnedNodeIds.value.filter((item) => item !== normalized)
  }

  function expandNode(nodeId) {
    const normalized = String(nodeId || '').trim()
    if (!normalized || expandedNodeIds.value.includes(normalized)) return
    expandedNodeIds.value = [...expandedNodeIds.value, normalized]
    pinNode(normalized)
  }

  function collapseNode(nodeId) {
    const normalized = String(nodeId || '').trim()
    if (!normalized) return
    expandedNodeIds.value = expandedNodeIds.value.filter((item) => item !== normalized)
    if (!seedNodeIds.value.includes(normalized) && selectedNodeId.value !== normalized) {
      unpinNode(normalized)
    }
  }

  function toggleNodeExpansion(nodeId) {
    const normalized = String(nodeId || '').trim()
    if (!normalized) return
    if (isNodeExpanded(normalized)) {
      collapseNode(normalized)
      return
    }
    expandNode(normalized)
  }

  function collapseAllExpansions() {
    expandedNodeIds.value = []
    pinnedNodeIds.value = []
  }

  function collapseAllCourses() {
    collapseAllExpansions()
  }

  function setCourseFilter(courseTitle = '全部') {
    activeCourseFilter.value = courseTitle || '全部'
  }

  function setFocusedNode(nodeId) {
    focusedNodeId.value = nodeId
    if (nodeId) pinNode(nodeId)
  }

  async function loadFullGraph(limit = 1600) {
    loading.value = true
    error.value = ''
    try {
      const payload = await fetchFullGraph(limit)
      const nodes = payload?.nodes || []
      const edges = payload?.edges || []
      console.info(`[GraphStore] loaded ${nodes.length} nodes, ${edges.length} edges`)
      if (!nodes.length) {
        error.value = '知识图谱数据为空，请确认后端服务和 Neo4j 数据库已启动。'
      }
      rawNodes.value = nodes
      rawEdges.value = edges
    } catch (loadError) {
      console.error('[GraphStore] loadFullGraph failed:', loadError)
      error.value = loadError?.message || '知识图谱加载失败，请检查后端服务是否运行。'
      throw loadError
    } finally {
      loading.value = false
    }
  }

  async function ensureNodeDetail(routeKnowledgeId) {
    const key = String(routeKnowledgeId || '').trim()
    if (!key) return null
    if (detailCache.value[key]) return detailCache.value[key]

    const detail = await fetchKnowledgeDetail(key)
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
      const briefing = await fetchFrontierBriefing(key)
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
      const payload = await fetchRecommendedPath({ target, knowledgeId, scene, limit: 10 })
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
    const node = nodeMap.value.get(key)
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

  async function revealNodeByRouteKnowledgeId(routeKnowledgeId) {
    const node = routeNodeMap.value.get(String(routeKnowledgeId || '').trim()) || findNodeByRoute(rawNodes.value, routeKnowledgeId)
    if (!node) return null

    pinNode(node.id)
    expandNode(node.id)
    await selectNode(node.id)
    return node
  }

  function highlightByStatus(status) {
    highlightFilter.value = highlightFilter.value === status ? null : status
  }

  function highlightPathNodes(nodeIds = []) {
    const normalized = [...nodeIds].filter(Boolean)
    highlightPath.value = normalized
    normalized.forEach((routeKnowledgeId) => {
      const node = routeNodeMap.value.get(String(routeKnowledgeId || '').trim())
      if (!node?.id) return
      pinNode(node.id)
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

  function focusCourseRepresentative(courseTitle) {
    const title = String(courseTitle || '').trim()
    if (!title) return null
    const courseNodes = rawNodes.value.filter((node) => getChapterTitle(node) === title)
    if (!courseNodes.length) return null
    const preferred = courseNodes.find((node) => seedNodeIds.value.includes(node.id)) || courseNodes[0]
    pinNode(preferred.id)
    return preferred
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

  function deepenCourseByNode(nodeId) {
    toggleNodeExpansion(nodeId)
    return expandedNodeIds.value.length
  }

  function collapseCourse() {
    return 0
  }

  return {
    rawNodes,
    rawEdges,
    loading,
    error,
    selectedNodeId,
    focusedNodeId,
    expandedNodeIds,
    pinnedNodeIds,
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
    isNodeExpanded,
    pinNode,
    unpinNode,
    expandNode,
    collapseNode,
    toggleNodeExpansion,
    collapseAllExpansions,
    collapseAllCourses,
    loadFullGraph,
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
