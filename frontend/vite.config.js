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
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('@antv/g6') || id.includes('@antv/g-webgl')) {
            return 'antv'
          }
          if (id.includes('katex')) {
            return 'katex'
          }
          if (id.includes('node_modules/vue') || id.includes('vue-router') || id.includes('pinia')) {
            return 'vue'
          }
          return undefined
        },
      },
    },
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
