import { createApp } from 'vue'
import router from './router'
import './style.css'
import App from './App.vue'
import { bootstrapAuth } from './auth'

bootstrapAuth().finally(() => {
  createApp(App).use(router).mount('#app')
})
