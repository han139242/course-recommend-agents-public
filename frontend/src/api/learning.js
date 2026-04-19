import { request } from './request'

export function fetchLearningProgress() {
  return request('/learning/progress')
}

export function fetchDashboardSummary() {
  return request('/dashboard/summary')
}

export function fetchCompetencyProfile() {
  return request('/competency/profile')
}

export function fetchStudentQuestions(params = {}) {
  const query = new URLSearchParams()
  if (params.knowledgeId) query.set('knowledgeId', String(params.knowledgeId))
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return request(`/learning/questions${suffix}`)
}

export function submitStudentAnswer(payload) {
  return request('/learning/answers', {
    method: 'POST',
    body: payload,
  })
}
