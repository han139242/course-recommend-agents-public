import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const graphApi = vi.hoisted(() => ({
  fetchFrontierBriefing: vi.fn(),
  fetchFullGraph: vi.fn(),
  fetchGraphCatalog: vi.fn(),
  fetchGraphNeighbors: vi.fn(),
  fetchGraphSeeds: vi.fn(),
  fetchKnowledgeDetail: vi.fn(),
}))

const recommendApi = vi.hoisted(() => ({
  fetchRecommendedPath: vi.fn(),
}))

vi.mock('../../api/graph', () => graphApi)
vi.mock('../../api/recommend', () => recommendApi)

import { useGraphStore } from '../graphStore'

function makeCatalogNode(id, chapterTitle, overrides = {}) {
  return {
    id,
    routeKnowledgeId: id,
    name: overrides.name || id,
    canonicalName: overrides.canonicalName || overrides.name || id,
    chapterTitle,
    entityType: 'concept',
    ...overrides,
  }
}

function makeEdge(sourceId, targetId, relationType = 'RELATED_TO') {
  return {
    id: `${sourceId}:${relationType}:${targetId}`,
    relationId: `${sourceId}:${relationType}:${targetId}`,
    sourceId,
    targetId,
    sourceName: sourceId,
    targetName: targetId,
    relationType,
  }
}

function clonePayload(payload) {
  return {
    nodes: (payload.nodes || []).map((node) => ({ ...node })),
    edges: (payload.edges || []).map((edge) => ({ ...edge })),
  }
}

const catalogNodes = [
  makeCatalogNode('A', '线性代数', { name: '矩阵', canonicalName: '矩阵' }),
  makeCatalogNode('B', '深度学习', { name: '神经网络', canonicalName: '神经网络' }),
  makeCatalogNode('C', '机器学习', { name: '梯度下降', canonicalName: '梯度下降' }),
  makeCatalogNode('D', '机器学习', { name: '线性回归', canonicalName: '线性回归' }),
  makeCatalogNode('S', '机器学习', { name: '共享桥节点', canonicalName: '共享桥节点' }),
  makeCatalogNode('Z', '自然语言处理', { name: 'Transformer', canonicalName: 'Transformer' }),
]

const prerequisiteEdges = [
  { sourceRouteKnowledgeId: 'A', targetRouteKnowledgeId: 'B' },
  { sourceRouteKnowledgeId: 'B', targetRouteKnowledgeId: 'C' },
]

const seedPayload = {
  nodes: [catalogNodes[0], catalogNodes[3]],
  edges: [],
}

const neighborPayloads = {
  A: {
    nodes: [catalogNodes[0], catalogNodes[1], catalogNodes[4]],
    edges: [
      makeEdge('A', 'B', 'PREREQUISITE_OF'),
      makeEdge('A', 'S', 'RELATED_TO'),
    ],
  },
  B: {
    nodes: [catalogNodes[1], catalogNodes[2]],
    edges: [makeEdge('B', 'C', 'PREREQUISITE_OF')],
  },
  D: {
    nodes: [catalogNodes[3], catalogNodes[4]],
    edges: [makeEdge('D', 'S', 'RELATED_TO')],
  },
  Z: {
    nodes: [catalogNodes[5], catalogNodes[2]],
    edges: [makeEdge('Z', 'C', 'RELATED_TO')],
  },
}

describe('graphStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()

    graphApi.fetchGraphCatalog.mockResolvedValue({
      nodes: catalogNodes,
      prerequisiteEdges,
    })
    graphApi.fetchGraphSeeds.mockImplementation(async () => clonePayload(seedPayload))
    graphApi.fetchGraphNeighbors.mockImplementation(async (nodeId) => clonePayload(neighborPayloads[nodeId] || { nodes: [], edges: [] }))
    graphApi.fetchKnowledgeDetail.mockImplementation(async (knowledgeId) => ({
      routeKnowledgeId: knowledgeId,
      canonicalName: `detail-${knowledgeId}`,
      prerequisites: [],
      related: [],
    }))
    graphApi.fetchFrontierBriefing.mockResolvedValue(null)
    graphApi.fetchFullGraph.mockResolvedValue({ nodes: [], edges: [] })
    recommendApi.fetchRecommendedPath.mockResolvedValue({ path: [] })
  })

  it('initializes catalog and keeps catalog-driven search and prerequisite chains available', async () => {
    const store = useGraphStore()

    await store.initGraph()

    expect(graphApi.fetchGraphCatalog).toHaveBeenCalledTimes(1)
    expect(graphApi.fetchGraphSeeds).toHaveBeenCalledWith(14)
    expect(store.catalogNodes).toHaveLength(6)
    expect(store.nodesMap.size).toBe(2)
    expect(store.routeNodeMap.get('Transformer')?.routeKnowledgeId).toBe('Z')
    expect(store.searchNodes('梯度下降').map((item) => item.routeKnowledgeId)).toContain('C')
    expect(store.getPrerequisiteRouteChain('C')).toEqual(['A', 'B', 'C'])
    expect(store.getNextLearningOptions('A')).toEqual(['B'])
    expect(store.courseOptions).toEqual(expect.arrayContaining(['全部', '线性代数', '深度学习', '自然语言处理']))
  })

  it('keeps expansion idempotent and preserves shared children until all parents collapse', async () => {
    const store = useGraphStore()
    await store.initGraph()

    const firstExpand = await store.expandNode('A')
    const secondExpand = await store.expandNode('A')
    await store.expandNode('D')

    expect(firstExpand).toEqual(expect.arrayContaining(['B', 'S']))
    expect(secondExpand).toEqual([])
    expect(graphApi.fetchGraphNeighbors).toHaveBeenCalledWith('A', expect.any(Object))
    expect(graphApi.fetchGraphNeighbors).toHaveBeenCalledWith('D', expect.any(Object))
    expect(store.nodesMap.has('S')).toBe(true)

    await store.collapseNode('A')
    expect(store.nodesMap.has('B')).toBe(false)
    expect(store.nodesMap.has('S')).toBe(true)

    await store.collapseNode('D')
    expect(store.nodesMap.has('S')).toBe(false)
  })

  it('recursively collapses descendants when a parent subtree is closed', async () => {
    const store = useGraphStore()
    await store.initGraph()

    await store.expandNode('A')
    await store.expandNode('B')
    expect(store.nodesMap.has('C')).toBe(true)

    await store.collapseNode('A')

    expect(store.nodesMap.has('B')).toBe(false)
    expect(store.nodesMap.has('C')).toBe(false)
    expect(store.expandedNodeIds.size).toBe(0)
    expect(store.expansionChildrenMap.A).toBeUndefined()
  })

  it('reveals a non-visible catalog node as a detached root cluster and selects it', async () => {
    const store = useGraphStore()
    await store.initGraph()

    const node = await store.revealNodeByRouteKnowledgeId('Z', {
      anchorPosition: { x: 140, y: 60 },
      occupiedNodes: [],
    })

    expect(node?.id).toBe('Z')
    expect(store.nodesMap.has('Z')).toBe(true)
    expect(store.rootNodeIds.has('Z')).toBe(true)
    expect(store.selectedNodeId).toBe('Z')
    expect(graphApi.fetchGraphNeighbors).toHaveBeenCalledWith('Z', expect.objectContaining({ depth: 1, direction: 'both' }))
    expect(graphApi.fetchKnowledgeDetail).toHaveBeenCalledWith('Z')
  })

  it('restores the seed scene when seeds are loaded again after detached reveals and expansions', async () => {
    const store = useGraphStore()
    await store.initGraph()

    await store.expandNode('A')
    await store.revealNodeByRouteKnowledgeId('Z', {
      anchorPosition: { x: 120, y: 48 },
      occupiedNodes: [],
    })

    expect(store.nodesMap.has('Z')).toBe(true)
    expect(store.expandedNodeIds.has('A')).toBe(true)

    await store.loadSeeds(14, { silent: true })

    expect([...store.rootNodeIds]).toEqual(['A', 'D'])
    expect(store.nodesMap.size).toBe(2)
    expect(store.nodesMap.has('Z')).toBe(false)
    expect(store.expandedNodeIds.size).toBe(0)
    expect(store.selectedNodeId).toBe(null)
    expect(store.highlightPath).toEqual([])
  })

  it('loads frontier briefings by stable knowledge id with catalog name fallback', async () => {
    const store = useGraphStore()
    graphApi.fetchFrontierBriefing.mockResolvedValueOnce({
      status: 'ready',
      digestSummary: 'frontier',
      topics: [],
      papers: [],
      generatedAt: '2026-04-01T00:00:00Z',
    })

    await store.initGraph()
    const briefing = await store.ensureFrontier('A')

    expect(briefing?.digestSummary).toBe('frontier')
    expect(graphApi.fetchFrontierBriefing).toHaveBeenCalledWith({
      knowledgeId: 'A',
      knowledge: '矩阵',
    })
  })

  it('respects the maximum expand depth before requesting neighbors', async () => {
    const store = useGraphStore()
    await store.initGraph()

    store.nodesMap = new Map(store.nodesMap)
    store.nodesMap.set('deep', {
      ...makeCatalogNode('deep', '测试课程', { name: '深层节点' }),
      x: 0,
      y: 0,
      depth: 4,
      rootId: 'A',
      isRoot: false,
    })

    const result = await store.expandNode('deep')

    expect(result).toEqual([])
    expect(graphApi.fetchGraphNeighbors).not.toHaveBeenCalledWith('deep', expect.anything())
  })

  it('normalizes detail course titles from curated metadata before caching', async () => {
    const store = useGraphStore()
    graphApi.fetchKnowledgeDetail.mockResolvedValueOnce({
      routeKnowledgeId: 'A',
      canonicalName: '矩阵',
      chapterTitle: '人工智能基础知识骨架',
      curatedSource: 'course_ai_intro',
      prerequisites: [],
      related: [],
    })

    await store.initGraph()
    const detail = await store.ensureNodeDetail('A')

    expect(detail?.chapterTitle).toBe('人工智能导论')
    expect(store.detailCache.A?.chapterTitle).toBe('人工智能导论')
    expect(store.detailCache.A?.rawChapterTitle).toBe('人工智能基础知识骨架')
  })

  it('blocks expansions that would exceed the visible-node ceiling', async () => {
    const store = useGraphStore()
    await store.initGraph()

    const crowded = new Map()
    crowded.set('A', {
      ...makeCatalogNode('A', '线性代数', { name: '矩阵' }),
      x: 0,
      y: 0,
      depth: 0,
      rootId: 'A',
      isRoot: true,
    })
    for (let index = 0; index < 199; index += 1) {
      const id = `N${index}`
      crowded.set(id, {
        ...makeCatalogNode(id, '测试课程', { name: id }),
        x: index,
        y: index,
        depth: 1,
        rootId: 'A',
        isRoot: false,
      })
    }

    store.nodesMap = crowded
    store.rootNodeIds = new Set(['A'])
    graphApi.fetchGraphNeighbors.mockResolvedValueOnce(clonePayload({
      nodes: [
        catalogNodes[0],
        makeCatalogNode('OVERFLOW', '测试课程', { name: 'Overflow' }),
      ],
      edges: [makeEdge('A', 'OVERFLOW')],
    }))

    const result = await store.expandNode('A')

    expect(result).toEqual([])
    expect(store.nodesMap.size).toBe(200)
    expect(store.error).toBeTruthy()
    expect(store.nodesMap.has('OVERFLOW')).toBe(false)
  })
})
