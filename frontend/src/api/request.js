import { readStorage, STORAGE_KEYS } from '../utils/storage'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

export class ApiError extends Error {
  constructor(message, status = 500, payload = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.payload = payload
  }
}

function getToken() {
  const session = readStorage(STORAGE_KEYS.auth, null)
  return session?.token || ''
}

function buildHeaders(headers = {}, auth = true) {
  const nextHeaders = { ...headers }
  const token = auth ? getToken() : ''

  if (token) {
    nextHeaders.Authorization = `Bearer ${token}`
  }

  return nextHeaders
}

async function parsePayload(response) {
  const contentType = response.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    return response.json()
  }
  return response.text()
}

export async function request(path, options = {}) {
  const {
    auth = true,
    headers = {},
    body,
    ...restOptions
  } = options

  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData
  const requestHeaders = buildHeaders(headers, auth)

  if (body !== undefined && !isFormData && !requestHeaders['Content-Type']) {
    requestHeaders['Content-Type'] = 'application/json'
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...restOptions,
    headers: requestHeaders,
    body: body === undefined ? undefined : isFormData || typeof body === 'string' ? body : JSON.stringify(body),
  })

  const payload = await parsePayload(response)

  if (!response.ok) {
    const message = payload?.message || payload?.error || '请求失败，请稍后重试。'
    throw new ApiError(message, response.status, payload)
  }

  if (payload && typeof payload === 'object' && 'code' in payload && payload.code !== 0) {
    throw new ApiError(payload.message || '请求失败，请稍后重试。', response.status, payload)
  }

  if (payload && typeof payload === 'object' && 'data' in payload) {
    return payload.data
  }

  return payload
}

export async function streamRequest(path, options = {}) {
  const {
    auth = true,
    headers = {},
    body,
    onMessage,
    onEvent,
    signal,
    ...restOptions
  } = options

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...restOptions,
    headers: {
      Accept: 'text/event-stream',
      ...buildHeaders(headers, auth),
      'Content-Type': 'application/json',
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  })

  if (!response.ok) {
    const payload = await parsePayload(response).catch(() => null)
    throw new ApiError(payload?.message || payload?.error || 'AI 助手暂时不可用。', response.status, payload)
  }

  if (!response.body) {
    throw new ApiError('浏览器当前不支持流式响应。', response.status)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const chunks = buffer.split('\n\n')
    buffer = chunks.pop() || ''

    for (const chunk of chunks) {
      const line = chunk
        .split('\n')
        .map((item) => item.trim())
        .find((item) => item.startsWith('data:'))

      if (!line) continue

      const content = line.replace(/^data:\s*/, '')
      if (content === '[DONE]') return

      const payload = JSON.parse(content)
      const token = payload?.choices?.[0]?.delta?.content || ''
      if (token && typeof onMessage === 'function') {
        onMessage(token, payload)
      }
      if (typeof onEvent === 'function') {
        onEvent(payload)
      }
    }
  }
}
