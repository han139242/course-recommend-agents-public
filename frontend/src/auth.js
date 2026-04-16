import { computed, reactive } from 'vue'
import { resolveFriendlyKnowledgeRoute } from './utils/knowledgeLookup'

const AUTH_STORAGE_KEY = 'course-recommend-auth'
const ENV_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
/** 开发环境优先走同源 /api，由 Vite 代理到后端，避免 CORS 与直连失败 */
const API_BASE_URL_CANDIDATES = ENV_API_BASE_URL
  ? [ENV_API_BASE_URL]
  : ['/api', 'http://localhost:8080/api', 'http://localhost:3000/api', 'http://localhost:3002/api', 'http://localhost:3001/api']

export const authState = reactive({
  token: '',
  user: null,
  ready: false,
})

let bootstrapPromise = null
let resolvedApiBaseUrl = API_BASE_URL_CANDIDATES[0]
let apiBaseUrlConfirmed = Boolean(ENV_API_BASE_URL)

function persistAuth() {
  if (typeof window === 'undefined') {
    return
  }

  if (!authState.token || !authState.user) {
    window.localStorage.removeItem(AUTH_STORAGE_KEY)
    return
  }

  window.localStorage.setItem(
    AUTH_STORAGE_KEY,
    JSON.stringify({
      token: authState.token,
      user: authState.user,
    }),
  )
}

function setSession(token, user) {
  authState.token = token
  authState.user = user
  persistAuth()
}

function isUsableRecommendationItem(item) {
  const title = String(item?.title || '').trim()
  const route = String(item?.route || '').trim()
  if (!title || !route) return false
  if (title === 'default-target') return false
  if (route.includes('default-target')) return false
  return true
}

function normalizeRecommendationItems(items) {
  if (!Array.isArray(items)) return []
  return items
    .filter(isUsableRecommendationItem)
    .map((item) => ({
      ...item,
      mastery: Number(item.mastery ?? item.masteryScore ?? 0),
      masteryScore: Number(item.masteryScore ?? item.mastery ?? 0),
      correctRate: Number(item.correctRate ?? 0),
      answeredCount: Number(item.answeredCount ?? 0),
      knowledgeId: String(item.knowledgeId || '').trim(),
      route: resolveFriendlyKnowledgeRoute(item.route, item.title),
    }))
}

export function clearSession() {
  authState.token = ''
  authState.user = null
  persistAuth()
}

async function apiRequest(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }

  if (authState.token) {
    headers.Authorization = `Bearer ${authState.token}`
  }

  const candidates = [
    resolvedApiBaseUrl,
    ...API_BASE_URL_CANDIDATES.filter((item) => item !== resolvedApiBaseUrl),
  ]

  let lastError = null

  for (const baseUrl of candidates) {
    let timeoutId = null
    try {
      const controller = new AbortController()
      const timeoutMs = Number(options.timeoutMs || 0)
      if (timeoutMs > 0) {
        timeoutId = window.setTimeout(() => {
          controller.abort(new Error('request_timeout'))
        }, timeoutMs)
      }

      const response = await fetch(`${baseUrl}${path}`, {
        ...options,
        headers,
        signal: options.signal || controller.signal,
      })

      const payload = await response.json().catch(() => ({}))
      if (timeoutId) window.clearTimeout(timeoutId)

      if (!response.ok) {
        const message = payload.message || '请求失败，请稍后重试。'
        if (response.status === 404 && !apiBaseUrlConfirmed && !ENV_API_BASE_URL) {
          lastError = new Error(message)
          continue
        }
        throw new Error(message)
      }

      if (payload.code && payload.code !== 0) {
        throw new Error(payload.message || '请求失败，请稍后重试。')
      }

      resolvedApiBaseUrl = baseUrl
      apiBaseUrlConfirmed = true
      return payload.data ?? {}
    } catch (error) {
      if (error?.name === 'AbortError') {
        lastError = new Error('请求超时，请确认后端服务正常运行后重试。')
        continue
      }
      const msg = error?.message || ''
      if (msg === 'Failed to fetch' || error?.name === 'TypeError') {
        lastError = new Error(
          '无法连接后端（Failed to fetch）。请确认已启动 API 服务，并使用 npm run dev 启动前端以启用 /api 代理。',
        )
      } else {
        lastError = error
      }
    } finally {
      if (timeoutId) window.clearTimeout(timeoutId)
    }
  }

  throw lastError || new Error('请求失败，请稍后重试。')
}

async function fetchCurrentUser() {
  const data = await apiRequest('/auth/me', {
    method: 'GET',
  })
  // /auth/me 成功时 data 即为 AuthUserView（无 user 嵌套）
  const user = data?.user ?? data
  if (!user || typeof user !== 'object') {
    throw new Error('无法解析当前用户信息')
  }
  authState.user = user
  persistAuth()
  return user
}

export async function bootstrapAuth() {
  if (bootstrapPromise) {
    return bootstrapPromise
  }

  bootstrapPromise = (async () => {
    if (typeof window !== 'undefined') {
      const raw = window.localStorage.getItem(AUTH_STORAGE_KEY)

      if (raw) {
        try {
          const parsed = JSON.parse(raw)
          authState.token = parsed.token || ''
          authState.user = parsed.user || null
        } catch (error) {
          clearSession()
        }
      }
    }

    if (!authState.token) {
      authState.ready = true
      return
    }

    try {
      await fetchCurrentUser()
    } catch (error) {
      clearSession()
    } finally {
      authState.ready = true
    }
  })()

  return bootstrapPromise
}

export async function login(payload) {
  const data = await apiRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify(payload),
  })

  setSession(data.accessToken || data.token, data.user)
  authState.ready = true
  return data
}

export async function register(payload) {
  const data = await apiRequest('/auth/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  })

  setSession(data.accessToken || data.token, data.user)
  authState.ready = true
  return data
}

export async function refreshCurrentUser() {
  return fetchCurrentUser()
}

export async function fetchDashboardSummary() {
  const data = await apiRequest('/dashboard/summary', {
    method: 'GET',
  })
  const recommendedPath = normalizeRecommendationItems(data.recommendedPath || data.path || [])
  return {
    ...data,
    quickStats: data.quickStats || data.stats || [],
    stats: data.stats || data.quickStats || [],
    recommendedPath,
    knowledgeCards: data.knowledgeCards || [],
  }
}

export async function fetchGraphSubgraph(root = '', limit = 12) {
  const params = new URLSearchParams()
  if (root) {
    params.set('root', root)
  }
  params.set('limit', String(limit))
  return apiRequest(`/graph/subgraph?${params.toString()}`, {
    method: 'GET',
  })
}

export async function fetchFullGraph(limit = 120) {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  return apiRequest(`/graph/full?${params.toString()}`, {
    method: 'GET',
    timeoutMs: 12000,
  })
}

export async function fetchKnowledgeDetail(id) {
  return apiRequest(`/knowledge/${id}`, {
    method: 'GET',
  })
}

export async function fetchCompetencyProfile() {
  return apiRequest('/competency/profile', {
    method: 'GET',
  })
}

export async function fetchRecommendedPath(params = {}) {
  const query = new URLSearchParams()
  if (params.target) query.set('target', String(params.target))
  if (params.scene) query.set('scene', String(params.scene))
  if (params.knowledgeId) query.set('knowledgeId', String(params.knowledgeId))
  if (params.limit) query.set('limit', String(params.limit))
  const suffix = query.toString() ? `?${query.toString()}` : ''

  const data = await apiRequest(`/recommend/path${suffix}`, {
    method: 'GET',
  })
  return {
    ...data,
    path: normalizeRecommendationItems(data.path || data.recommendedPath || []),
  }
}

export async function fetchStudentQuestions(params = {}) {
  const query = new URLSearchParams()
  if (params.knowledgeId) {
    query.set('knowledgeId', String(params.knowledgeId))
  }
  if (params.page) {
    query.set('page', String(params.page))
  }
  if (params.pageSize) {
    query.set('pageSize', String(params.pageSize))
  }
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return apiRequest(`/learning/questions${suffix}`, {
    method: 'GET',
  })
}

export async function submitStudentAnswer(payload) {
  return apiRequest('/learning/answers', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function fetchLearningProgress() {
  return apiRequest('/learning/progress', {
    method: 'GET',
  })
}

export async function fetchFrontierBriefing(knowledge = '') {
  const params = new URLSearchParams()
  if (knowledge) {
    params.set('knowledge', knowledge)
  }
  return apiRequest(`/frontier/briefing?${params.toString()}`, {
    method: 'GET',
  })
}

export const isAuthenticated = computed(() => Boolean(authState.token && authState.user))
