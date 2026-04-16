export const STORAGE_KEYS = {
  auth: 'kg-auth-session',
  bookmarks: 'kg-bookmarks',
  notes: 'kg-notes',
  customPaths: 'kg-custom-paths',
  masteryOverrides: 'kg-progress-overrides',
}

export function readStorage(key, fallback) {
  if (typeof window === 'undefined') return fallback

  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return fallback
    return JSON.parse(raw)
  } catch (error) {
    return fallback
  }
}

export function writeStorage(key, value) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(key, JSON.stringify(value))
}

export function removeStorage(key) {
  if (typeof window === 'undefined') return
  window.localStorage.removeItem(key)
}

export function createLocalId(prefix = 'item') {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}
