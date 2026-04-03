import { computed, reactive } from 'vue'

const AUTH_STORAGE_KEY = 'course-recommend-auth'
const ENV_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
const API_BASE_URL_CANDIDATES = ENV_API_BASE_URL
  ? [ENV_API_BASE_URL]
  : ['http://localhost:3000/api', 'http://localhost:3002/api', 'http://localhost:3001/api']

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
      lastError = error
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
  return apiRequest('/dashboard/summary', {
    method: 'GET',
  })
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
  return apiRequest('/recommend/path', {
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
