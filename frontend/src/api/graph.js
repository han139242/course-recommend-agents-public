import { request } from './request'

export function fetchGraphCatalog() {
  return request('/graph/catalog')
}

export function fetchGraphSeeds(count = 14) {
  return request(`/graph/seeds?count=${encodeURIComponent(count)}`)
}

export function fetchGraphNeighbors(nodeId, options = {}) {
  const query = new URLSearchParams({
    nodeId: String(nodeId || ''),
    depth: String(options.depth ?? 1),
    direction: String(options.direction || 'both'),
    limit: String(options.limit ?? 20),
  })
  return request(`/graph/neighbors?${query.toString()}`)
}

export function fetchFullGraph(limit = 2000) {
  return request(`/graph/full?limit=${encodeURIComponent(limit)}`)
}

export function fetchKnowledgeDetail(knowledgeId) {
  return request(`/knowledge/${encodeURIComponent(knowledgeId)}`)
}

export function fetchFrontierBriefing(query = {}) {
  const normalized =
    typeof query === 'string'
      ? {
          knowledgeId: query,
          knowledge: query,
        }
      : (query || {})

  const params = new URLSearchParams()
  if (normalized.knowledgeId) {
    params.set('knowledgeId', String(normalized.knowledgeId))
  }
  if (normalized.knowledge) {
    params.set('knowledge', String(normalized.knowledge))
  }

  const suffix = params.toString() ? `?${params.toString()}` : ''
  return request(`/frontier/briefing${suffix}`)
}
