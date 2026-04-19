import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { fetchCurrentUser, loginRequest, registerRequest } from '../api/auth'
import { fetchCompetencyProfile, fetchDashboardSummary, fetchLearningProgress } from '../api/learning'
import { createLocalId, readStorage, removeStorage, STORAGE_KEYS, writeStorage } from '../utils/storage'

const LEGACY_AUTH_STORAGE_KEY = 'course-recommend-auth'

function normalizeProgressList(payload) {
  const list = payload?.list || payload?.items || payload?.progress || payload || []
  return Array.isArray(list) ? list : []
}

function buildProgressIndex(list = [], overrides = {}) {
  const map = {}

  list.forEach((item) => {
    const keys = [item.knowledgeId, item.routeKnowledgeId, item.canonicalName, item.name]
      .map((value) => String(value || '').trim())
      .filter(Boolean)

    keys.forEach((key) => {
      map[key] = {
        masteryScore: Number(item.masteryScore ?? item.mastery ?? 0),
        answeredCount: Number(item.answeredCount || 0),
        correctRate: Number(item.correctRate || 0),
        status: item.status || 'not_started',
        masteryText: item.masteryText || '',
      }
    })
  })

  Object.entries(overrides || {}).forEach(([key, value]) => {
    map[key] = {
      ...(map[key] || {}),
      ...value,
    }
  })

  return map
}

export const useUserStore = defineStore('user', () => {
  const token = ref('')
  const user = ref(null)
  const ready = ref(false)

  const progressList = ref([])
  const dashboardSummary = ref(null)
  const competencyProfile = ref(null)

  const bookmarks = ref(readStorage(STORAGE_KEYS.bookmarks, []))
  const notes = ref(readStorage(STORAGE_KEYS.notes, {}))
  const customPaths = ref(readStorage(STORAGE_KEYS.customPaths, []))
  const masteryOverrides = ref(readStorage(STORAGE_KEYS.masteryOverrides, {}))

  function persistSession() {
    if (!token.value || !user.value) {
      removeStorage(STORAGE_KEYS.auth)
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem(LEGACY_AUTH_STORAGE_KEY)
      }
      return
    }

    const payload = {
      token: token.value,
      user: user.value,
    }

    writeStorage(STORAGE_KEYS.auth, payload)
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(LEGACY_AUTH_STORAGE_KEY, JSON.stringify(payload))
    }
  }

  function hydrateSession() {
    const session =
      readStorage(STORAGE_KEYS.auth, null) ||
      readStorage(LEGACY_AUTH_STORAGE_KEY, null)

    token.value = session?.token || session?.accessToken || ''
    user.value = session?.user || null

    if (token.value && user.value) {
      persistSession()
    }
  }

  const isAuthenticated = computed(() => Boolean(token.value && user.value))
  const progressMap = computed(() => buildProgressIndex(progressList.value, masteryOverrides.value))

  const quickStats = computed(() => dashboardSummary.value?.quickStats || dashboardSummary.value?.stats || [])

  async function bootstrap() {
    if (ready.value) return
    hydrateSession()

    if (!token.value) {
      ready.value = true
      return
    }

    try {
      const profile = await fetchCurrentUser()
      user.value = profile?.user || profile
      persistSession()
    } catch (error) {
      logout()
    } finally {
      ready.value = true
    }
  }

  async function login(email, password) {
    const payload = await loginRequest({ email, password })
    token.value = payload.accessToken || payload.token || ''
    user.value = payload.user || null
    persistSession()
    return payload
  }

  async function register(email, password, role = 'student') {
    const payload = await registerRequest({ email, password, role })
    token.value = payload.accessToken || payload.token || ''
    user.value = payload.user || null
    persistSession()
    return payload
  }

  function logout() {
    token.value = ''
    user.value = null
    progressList.value = []
    dashboardSummary.value = null
    competencyProfile.value = null
    removeStorage(STORAGE_KEYS.auth)
  }

  async function loadProgress() {
    if (!isAuthenticated.value) {
      progressList.value = []
      return []
    }

    const payload = await fetchLearningProgress()
    progressList.value = normalizeProgressList(payload)
    return progressList.value
  }

  async function loadDashboard() {
    if (!isAuthenticated.value) {
      dashboardSummary.value = null
      return null
    }

    dashboardSummary.value = await fetchDashboardSummary()
    return dashboardSummary.value
  }

  async function loadCompetency() {
    if (!isAuthenticated.value) {
      competencyProfile.value = null
      return null
    }

    competencyProfile.value = await fetchCompetencyProfile()
    return competencyProfile.value
  }

  function toggleBookmark(knowledgeId) {
    const key = String(knowledgeId || '').trim()
    if (!key) return

    const next = new Set(bookmarks.value)
    if (next.has(key)) next.delete(key)
    else next.add(key)

    bookmarks.value = [...next]
    writeStorage(STORAGE_KEYS.bookmarks, bookmarks.value)
  }

  function saveNote(knowledgeId, content) {
    const key = String(knowledgeId || '').trim()
    notes.value = {
      ...notes.value,
      [key]: content,
    }
    writeStorage(STORAGE_KEYS.notes, notes.value)
  }

  function createCustomPath(name = '我的路径') {
    const path = {
      id: createLocalId('path'),
      name,
      nodes: [],
      createdAt: new Date().toISOString(),
    }
    customPaths.value = [path, ...customPaths.value]
    writeStorage(STORAGE_KEYS.customPaths, customPaths.value)
    return path
  }

  function addToPath(pathId, knowledgeId) {
    customPaths.value = customPaths.value.map((path) => {
      if (path.id !== pathId) return path
      if (path.nodes.includes(knowledgeId)) return path
      return {
        ...path,
        nodes: [...path.nodes, knowledgeId],
      }
    })
    writeStorage(STORAGE_KEYS.customPaths, customPaths.value)
  }

  function removeFromPath(pathId, knowledgeId) {
    customPaths.value = customPaths.value.map((path) => {
      if (path.id !== pathId) return path
      return {
        ...path,
        nodes: path.nodes.filter((nodeId) => nodeId !== knowledgeId),
      }
    })
    writeStorage(STORAGE_KEYS.customPaths, customPaths.value)
  }

  function reorderPath(pathId, fromIndex, toIndex) {
    customPaths.value = customPaths.value.map((path) => {
      if (path.id !== pathId) return path
      const nodes = [...path.nodes]
      const [item] = nodes.splice(fromIndex, 1)
      nodes.splice(toIndex, 0, item)
      return {
        ...path,
        nodes,
      }
    })
    writeStorage(STORAGE_KEYS.customPaths, customPaths.value)
  }

  function markMastered(knowledgeId) {
    masteryOverrides.value = {
      ...masteryOverrides.value,
      [knowledgeId]: {
        masteryScore: 100,
        answeredCount: Math.max(Number(masteryOverrides.value?.[knowledgeId]?.answeredCount || 1), 1),
        status: 'mastered',
        masteryText: '已掌握',
      },
    }
    writeStorage(STORAGE_KEYS.masteryOverrides, masteryOverrides.value)
  }

  function getProgressMeta(knowledgeId) {
    return progressMap.value[String(knowledgeId || '').trim()] || {
      masteryScore: 0,
      answeredCount: 0,
      status: 'not_started',
      masteryText: '未学习',
    }
  }

  return {
    token,
    user,
    ready,
    progressList,
    progressMap,
    quickStats,
    dashboardSummary,
    competencyProfile,
    bookmarks,
    notes,
    customPaths,
    masteryOverrides,
    isAuthenticated,
    bootstrap,
    login,
    register,
    logout,
    loadProgress,
    loadDashboard,
    loadCompetency,
    toggleBookmark,
    saveNote,
    createCustomPath,
    addToPath,
    removeFromPath,
    reorderPath,
    markMastered,
    getProgressMeta,
  }
})
