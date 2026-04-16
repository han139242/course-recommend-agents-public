import { request } from './request'

export function fetchFullGraph(limit = 2000) {
  return request(`/graph/full?limit=${encodeURIComponent(limit)}`)
}

export function fetchSubgraph(root, limit = 12) {
  const query = new URLSearchParams({
    root: String(root || ''),
    limit: String(limit),
  })
  return request(`/graph/subgraph?${query.toString()}`)
}

export function fetchKnowledgeDetail(knowledgeId) {
  return request(`/knowledge/${encodeURIComponent(knowledgeId)}`)
}

export function fetchFrontierBriefing(knowledge) {
  return request(`/frontier/briefing?knowledge=${encodeURIComponent(knowledge)}`)
}
