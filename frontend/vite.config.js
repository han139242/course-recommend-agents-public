import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

const envDir = '..'
const env = loadEnv('', envDir, '')
const apiProxyTarget = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'

export default defineConfig({
  envDir,
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    clearMocks: true,
  },
  build: {
    outDir: '../backend/web/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('vis-network') || id.includes('vis-data') || id.includes('vis-util') || id.includes('keycharm') || id.includes('component-emitter') || id.includes('uuid') || id.includes('hammerjs')) {
            return 'graph-engine'
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
