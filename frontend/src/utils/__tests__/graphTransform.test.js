import { transformGraphData } from '../graphTransform'

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
})
