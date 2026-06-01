import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import './style.css'
import './student-theme.css'
import { useUserStore } from './stores/userStore'
import { useThemeStore } from './stores/themeStore'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Initialize global theme
const themeStore = useThemeStore()
themeStore.initTheme()

const userStore = useUserStore()
userStore.bootstrap().finally(() => {
  app.mount('#app')
})
