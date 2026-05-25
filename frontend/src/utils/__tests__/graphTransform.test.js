import { transformGraphData } from '../graphTransform'
import { getCoursePalette } from '../colorMap'

describe('graphTransform', () => {
  it('keeps total catalog size separate from visible graph stats and marks cross-course edges', () => {
    const nodes = [
      {
        id: 'A',
        routeKnowledgeId: 'A',
        name: '矩阵',
        canonicalName: '矩阵',
        chapterTitle: '线性代数',
        entityType: 'concept',
        x: 0,
        y: 0,
        depth: 0,
        rootId: 'A',
        isRoot: true,
      },
      {
        id: 'B',
        routeKnowledgeId: 'B',
        name: '神经网络',
        canonicalName: '神经网络',
        chapterTitle: '深度学习',
        entityType: 'concept',
        x: 180,
        y: 40,
        depth: 1,
        rootId: 'A',
        isRoot: false,
      },
    ]
    const edges = [
      {
        id: 'A:RELATED_TO:B',
        relationId: 'A:RELATED_TO:B',
        sourceId: 'A',
        targetId: 'B',
        relationType: 'RELATED_TO',
      },
    ]

    const result = transformGraphData(nodes, edges, {
      expandedNodeIds: ['A'],
      pinnedNodeIds: ['B'],
      selectedNodeId: 'A',
      rootNodeIds: ['A'],
      totalNodes: 896,
    })
    const compactResult = transformGraphData(nodes, edges, {
      expandedNodeIds: [],
      pinnedNodeIds: ['B'],
      selectedNodeId: null,
      rootNodeIds: ['A'],
      totalNodes: 896,
    })

    const rootNode = result.nodes.find((node) => node.id === 'A')
    const childNode = result.nodes.find((node) => node.id === 'B')
    const edge = result.edges[0]
    const compactRootNode = compactResult.nodes.find((node) => node.id === 'A')

    expect(rootNode.style.ringColor).toBeTruthy()
    expect(rootNode.style.collisionRadius).toBeGreaterThan(rootNode.style.radius)
    expect(rootNode.style.collisionRadius).toBeGreaterThan(compactRootNode.style.collisionRadius)
    expect(rootNode.data.isExpanded).toBe(true)
    expect(childNode.data.isPinned).toBe(true)
    expect(edge.data.crossCourse).toBe(true)
    expect(edge.style.sourceRing).not.toBe(edge.style.targetRing)
    expect(edge.style.lineWidth).toBeGreaterThan(1)
    expect(result.meta.totalNodes).toBe(896)
    expect(result.meta.visibleNodes).toBe(2)
    expect(result.meta.visibleEdges).toBe(1)
    expect(result.meta.rootCount).toBe(1)
    expect(result.meta.expandedCount).toBe(1)
  })

  it('normalizes course titles from source metadata before rendering node and edge data', () => {
    const nodes = [
      {
        id: 'intro',
        routeKnowledgeId: 'intro',
        name: '知识表示',
        canonicalName: '知识表示',
        chapterTitle: '人工智能基础知识骨架',
        curatedSource: 'curated_ai_professional_ontology',
        entityType: 'topic',
        x: 0,
        y: 0,
      },
      {
        id: 'llm',
        routeKnowledgeId: 'llm',
        name: 'Transformer',
        canonicalName: 'Transformer',
        chapterTitle: '人工智能课程知识图谱',
        sourcePath: 'curated/batch_29_transformer_llm_full_route',
        entityType: 'topic',
        x: 120,
        y: 30,
      },
    ]
    const edges = [
      {
        id: 'intro:RELATED_TO:llm',
        relationId: 'intro:RELATED_TO:llm',
        sourceId: 'intro',
        targetId: 'llm',
        relationType: 'RELATED_TO',
      },
    ]

    const result = transformGraphData(nodes, edges)

    expect(result.nodes.find((node) => node.id === 'intro')?.data.chapterTitle).toBe('人工智能导论')
    expect(result.nodes.find((node) => node.id === 'llm')?.data.chapterTitle).toBe('大模型')
    expect(result.edges[0]?.data.crossCourse).toBe(true)
    expect(result.edges[0]?.data.sourceCourse).toBe('人工智能导论')
    expect(result.edges[0]?.data.targetCourse).toBe('大模型')
  })
  it('blends node fill from course background to primary color as mastery rises', () => {
    const palette = getCoursePalette('鏈哄櫒瀛︿範')
    const nodes = [
      {
        id: 'low',
        routeKnowledgeId: 'low',
        name: '鏈哄櫒瀛︿範鍩虹',
        canonicalName: '鏈哄櫒瀛︿範鍩虹',
        chapterTitle: '鏈哄櫒瀛︿範',
        entityType: 'concept',
        x: 0,
        y: 0,
      },
      {
        id: 'high',
        routeKnowledgeId: 'high',
        name: '鏈哄櫒瀛︿範杩涢樁',
        canonicalName: '鏈哄櫒瀛︿範杩涢樁',
        chapterTitle: '鏈哄櫒瀛︿範',
        entityType: 'concept',
        x: 120,
        y: 0,
      },
    ]

    const result = transformGraphData(nodes, [], {
      progressMap: {
        low: { masteryScore: 0, answeredCount: 0 },
        high: { masteryScore: 100, answeredCount: 20 },
      },
    })

    const lowNode = result.nodes.find((node) => node.id === 'low')
    const highNode = result.nodes.find((node) => node.id === 'high')

    expect(lowNode.style.fill).toBe('rgba(232, 237, 252, 0.99)')
    expect(highNode.style.fill).toBe('rgba(108, 142, 239, 0.99)')
    expect(lowNode.style.ringColor).toBe(palette.ring)
    expect(highNode.style.ringColor).toBe(palette.ring)
    expect(lowNode.style.masteryRatio).toBe(0)
    expect(highNode.style.masteryRatio).toBe(1)
  })
})
