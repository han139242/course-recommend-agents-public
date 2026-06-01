import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', () => {
  const STORAGE_KEY = 'course-recommend-theme'

  // Determine initial theme: check localStorage, then system color preference, defaulting to 'dark'
  const savedTheme = localStorage.getItem(STORAGE_KEY)
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
  const initialTheme = savedTheme || (prefersDark ? 'dark' : 'dark') // Keeping 'dark' as default first-render preference

  const theme = ref(initialTheme)

  // Initialize theme by setting it on html document element
  function initTheme() {
    document.documentElement.setAttribute('data-theme', theme.value)
  }

  function setTheme(nextTheme) {
    if (nextTheme !== 'light' && nextTheme !== 'dark') return
    theme.value = nextTheme
    localStorage.setItem(STORAGE_KEY, nextTheme)
    document.documentElement.setAttribute('data-theme', nextTheme)
  }

  function toggleTheme() {
    const nextTheme = theme.value === 'dark' ? 'light' : 'dark'
    setTheme(nextTheme)
  }

  return {
    theme,
    initTheme,
    setTheme,
    toggleTheme,
  }
})
