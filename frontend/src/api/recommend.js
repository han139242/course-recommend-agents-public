import { request } from './request'

export function fetchRecommendedPath(params = {}) {
  const query = new URLSearchParams()

  if (params.target) query.set('target', String(params.target))
  if (params.knowledgeId) query.set('knowledgeId', String(params.knowledgeId))
  if (params.limit) query.set('limit', String(params.limit))
  if (params.scene) query.set('scene', String(params.scene))

  const suffix = query.toString() ? `?${query.toString()}` : ''
  return request(`/recommend/path${suffix}`)
}
