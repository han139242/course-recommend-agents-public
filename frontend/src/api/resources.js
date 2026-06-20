import { request, streamRequest } from './request'

function toQueryString(params = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    query.set(key, String(value))
  })
  const search = query.toString()
  return search ? `?${search}` : ''
}

export function generateResources(payload) {
  return request('/resources/generate', {
    method: 'POST',
    body: payload,
  })
}

// generateQuiz 生成知识点结构化互动练习题。
export function generateQuiz(payload) {
  return request('/resources/quiz', {
    method: 'POST',
    body: payload,
  })
}

export function fetchResourceJob(jobId) {
  return request(`/resources/jobs/${jobId}`)
}

// streamResourceJob 以 SSE 订阅资源生成任务进度，onEvent 会收到每次的任务快照。
export function streamResourceJob(jobId, { onEvent, signal } = {}) {
  return streamRequest(`/resources/jobs/${jobId}/stream`, {
    method: 'GET',
    onEvent,
    signal,
  })
}

export function exportResourceJobMarkdown(jobId) {
  return request(`/resources/jobs/${jobId}/export`)
}

export function fetchResourcePresentationOutline(jobId) {
  return request(`/resources/jobs/${jobId}/presentation-outline`)
}

export function fetchResourceDemoScript(jobId) {
  return request(`/resources/jobs/${jobId}/demo-script`)
}

export function fetchResources(params = {}) {
  return request(`/resources${toQueryString(params)}`)
}

export function fetchResource(resourceId) {
  return request(`/resources/${resourceId}`)
}

export function fetchResourceFeed() {
  return request('/resources/feed')
}

export function fetchResourceSafetyReport(resourceId) {
  return request(`/resources/${resourceId}/safety-report`)
}

export function submitResourceFeedback(resourceId, payload) {
  return request(`/resources/${resourceId}/feedback`, {
    method: 'POST',
    body: payload,
  })
}

export function fetchResourceRecommendations() {
  return request('/resources/recommendations')
}

export function fetchResourceEffectSummary() {
  return request('/resources/effect-summary')
}

export function recordLearningEvent(payload) {
  return request('/resources/events', {
    method: 'POST',
    body: payload,
  })
}
