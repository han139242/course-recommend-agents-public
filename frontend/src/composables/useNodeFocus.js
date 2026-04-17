export function buildGraphStateMap({
  nodes = [],
  edges = [],
  focusedNodeId = null,
  selectedNodeId = null,
  highlightFilter = null,
  highlightPath = [],
  courseFilter = '全部',
  progressMap = {},
  bookmarks = [],
}) {
  const states = {}
  const bookmarkSet = new Set(bookmarks)
  const pathNodeSet = new Set(highlightPath)
  const pathEdgePairs = new Set()
  const nodeById = new Map(nodes.map((node) => [node.id, node]))

  highlightPath.forEach((routeKnowledgeId, index) => {
    const nextRouteKnowledgeId = highlightPath[index + 1]
    if (!routeKnowledgeId || !nextRouteKnowledgeId) return
    pathEdgePairs.add(`${routeKnowledgeId}->${nextRouteKnowledgeId}`)
    pathEdgePairs.add(`${nextRouteKnowledgeId}->${routeKnowledgeId}`)
  })

  const adjacencyNodes = new Set()
  const adjacencyEdges = new Set()

  if (focusedNodeId && nodeById.has(focusedNodeId)) {
    adjacencyNodes.add(focusedNodeId)

    edges.forEach((edge) => {
      const edgeId = edge.id || `${edge.source}-${edge.target}`
      if (edge.source === focusedNodeId || edge.target === focusedNodeId) {
        adjacencyEdges.add(edgeId)
        if (nodeById.has(edge.source)) adjacencyNodes.add(edge.source)
        if (nodeById.has(edge.target)) adjacencyNodes.add(edge.target)
      }
    })
  }

  const filterMatches = new Set()
  if (highlightFilter) {
    nodes.forEach((node) => {
      const progress = progressMap[node.data?.routeKnowledgeId] || {}
      const mastery = Number(progress.masteryScore || node.data?.masteryScore || 0)
      const answeredCount = Number(progress.answeredCount || node.data?.answeredCount || 0)
      const bookmarked = bookmarkSet.has(node.data?.routeKnowledgeId)
      const weak = mastery < 30 && answeredCount > 0
      const matched =
        (highlightFilter === 'mastered' && mastery >= 80) ||
        (highlightFilter === 'learning' && mastery >= 40 && mastery < 80) ||
        (highlightFilter === 'weak' && weak) ||
        (highlightFilter === 'bookmarked' && bookmarked)

      if (matched) filterMatches.add(node.id)
    })
  }

  nodes.forEach((node) => {
    const id = node.id
    const routeKnowledgeId = node.data?.routeKnowledgeId
    const chapterTitle = node.data?.chapterTitle
    const courseMatched = courseFilter === '全部' || courseFilter === chapterTitle
    const activeByFocus = !focusedNodeId || adjacencyNodes.has(id)
    const activeByFilter = !highlightFilter || filterMatches.has(id)
    const active = courseMatched && activeByFocus && activeByFilter

    states[id] = []
    if (!active) states[id].push('dimmed')
    if (focusedNodeId && adjacencyNodes.has(id)) states[id].push('active')
    if (selectedNodeId && selectedNodeId === id) states[id].push('selected')
    if (pathNodeSet.has(routeKnowledgeId)) states[id].push('path')
  })

  edges.forEach((edge) => {
    const id = edge.id
    const sourceNode = nodeById.get(edge.source)
    const targetNode = nodeById.get(edge.target)
    const sourceRoute = sourceNode?.data?.routeKnowledgeId
    const targetRoute = targetNode?.data?.routeKnowledgeId
    const sourceCourse = sourceNode?.data?.chapterTitle || ''
    const targetCourse = targetNode?.data?.chapterTitle || ''
    const crossCourse = Boolean(sourceCourse && targetCourse && sourceCourse !== targetCourse)
    const courseMatched =
      courseFilter === '全部' ||
      sourceCourse === courseFilter ||
      targetCourse === courseFilter
    const activeByFocus = !focusedNodeId || adjacencyEdges.has(id)
    const activeByFilter = !highlightFilter || (filterMatches.has(edge.source) && filterMatches.has(edge.target))
    const inPath = Boolean(sourceRoute && targetRoute) && pathEdgePairs.has(`${sourceRoute}->${targetRoute}`)

    states[id] = []
    if (!(courseMatched && activeByFocus && activeByFilter) && !inPath) states[id].push('dimmed')
    if (focusedNodeId && adjacencyEdges.has(id)) states[id].push('active')
    if (inPath) states[id].push('path')
    if (crossCourse) states[id].push('cross-course')
    if (crossCourse && focusedNodeId && adjacencyEdges.has(id)) states[id].push('cross-course-active')
    if (crossCourse && states[id].includes('dimmed')) states[id].push('cross-course-dimmed')
  })

  return states
}
