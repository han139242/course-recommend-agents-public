import { request } from './request'

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

export function fetchResourceJob(jobId) {
  return request(`/resources/jobs/${jobId}`)
}

export function exportResourceJobMarkdown(jobId) {
  return request(`/resources/jobs/${jobId}/export`)
}

export function fetchResourcePresentationOutline(jobId) {
  return request(`/resources/jobs/${jobId}/presentation-outline`)
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

export function recordLearningEvent(payload) {
  return request('/resources/events', {
    method: 'POST',
    body: payload,
  })
}
