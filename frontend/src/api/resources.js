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

export function generateResourceImage(payload) {
  return request('/resources/media/image', {
    method: 'POST',
    body: payload,
  })
}

export function generateResourceVideo(payload) {
  return request('/resources/media/video', {
    method: 'POST',
    body: payload,
  })
}

export function fetchResourceJob(jobId) {
  return request(`/resources/jobs/${jobId}`)
}

export function retryResourceJob(jobId, payload = {}) {
  return request(`/resources/jobs/${jobId}/retry`, {
    method: 'POST',
    body: payload,
  })
}

export function cancelResourceJob(jobId) {
  return request(`/resources/jobs/${jobId}/cancel`, {
    method: 'POST',
  })
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

export function fetchJobReviewSummary(jobId) {
  return request(`/resources/jobs/${jobId}/review-summary`)
}

export function fetchJobEvidencePackage(jobId) {
  return request(`/resources/jobs/${jobId}/evidence-package`)
}

export function fetchStudyChecklist(jobId) {
  return request(`/resources/jobs/${jobId}/study-checklist`)
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

export function fetchResourceMediaStatus(resourceId) {
  return request(`/resources/${resourceId}/media-status`)
}

export function fetchResourceQualityReport(resourceId) {
  return request(`/resources/${resourceId}/quality-report`)
}

export function fetchResourceProvenance(resourceId) {
  return request(`/resources/${resourceId}/provenance`)
}

export function generateResourceVariant(resourceId, payload = {}) {
  return request(`/resources/${resourceId}/variant`, {
    method: 'POST',
    body: payload,
  })
}

export function compareResources(resourceIds) {
  return request('/resources/compare', {
    method: 'POST',
    body: { resourceIds },
  })
}

export function submitResourceFeedback(resourceId, payload) {
  return request(`/resources/${resourceId}/feedback`, {
    method: 'POST',
    body: payload,
  })
}

export function updateResourceLearningStatus(resourceId, payload) {
  return request(`/resources/${resourceId}/learning-status`, {
    method: 'PATCH',
    body: payload,
  })
}

export function fetchResourceRecommendations() {
  return request('/resources/recommendations')
}

export function fetchResourceEffectSummary() {
  return request('/resources/effect-summary')
}

// fetchLearningReport 获取可分享/可提交的 Markdown 学情学习报告。
export function fetchLearningReport() {
  return request('/resources/learning-report')
}

export function recordLearningEvent(payload) {
  return request('/resources/events', {
    method: 'POST',
    body: payload,
  })
}
