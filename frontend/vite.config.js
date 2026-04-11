import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

const envDir = '..'
const env = loadEnv('', envDir, '')
const apiProxyTarget = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'

export default defineConfig({
  envDir,
  plugins: [vue()],
  build: {
    outDir: '../backend/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
})
