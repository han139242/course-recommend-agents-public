import { computed, reactive } from 'vue'

const AUTH_STORAGE_KEY = 'course-recommend-auth'
const ENV_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
/** 开发环境优先走同源 /api，由 Vite 代理到后端，避免 CORS 与直连失败 */
const API_BASE_URL_CANDIDATES = ENV_API_BASE_URL
  ? [ENV_API_BASE_URL]
  : ['/api', 'http://localhost:3000/api', 'http://localhost:3002/api', 'http://localhost:3001/api']

export const authState = reactive({
  token: '',
  user: null,
  ready: false,
})

let bootstrapPromise = null
let resolvedApiBaseUrl = API_BASE_URL_CANDIDATES[0]

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
    try {
      const response = await fetch(`${baseUrl}${path}`, {
        ...options,
        headers,
      })

      const payload = await response.json().catch(() => ({}))

      if (!response.ok) {
        const message = payload.message || '请求失败，请稍后重试。'
        if (response.status === 404 && !ENV_API_BASE_URL) {
          lastError = new Error(message)
          continue
        }
        throw new Error(message)
      }

      if (payload.code && payload.code !== 0) {
        throw new Error(payload.message || '请求失败，请稍后重试。')
      }

      resolvedApiBaseUrl = baseUrl
      return payload.data ?? {}
    } catch (error) {
      const msg = error?.message || ''
      if (msg === 'Failed to fetch' || error?.name === 'TypeError') {
        lastError = new Error(
          '无法连接后端（Failed to fetch）。请确认已启动 API 服务（默认端口 3000），开发前端请使用 npm run dev 以启用 /api 代理。',
        )
      } else {
        lastError = error
      }
    }
  }

  throw lastError || new Error('请求失败，请稍后重试。')
}

async function fetchCurrentUser() {
  const data = await apiRequest('/auth/me', {
    method: 'GET',
  })

  authState.user = data.user
  persistAuth()
  return data.user
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

  setSession(data.token, data.user)
  authState.ready = true
  return data
}

export async function register(payload) {
  const data = await apiRequest('/auth/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  })

  setSession(data.token, data.user)
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
  return {
    ...data,
    quickStats: data.quickStats || data.stats || [],
    stats: data.stats || data.quickStats || [],
    recommendedPath: data.recommendedPath || data.path || [],
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

export async function fetchRecommendedPath() {
  const data = await apiRequest('/recommend/path', {
    method: 'GET',
  })
  return {
    ...data,
    path: data.path || data.recommendedPath || [],
  }
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
