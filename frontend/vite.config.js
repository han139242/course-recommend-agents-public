import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

function messagesIncludeImage(messages) {
  if (!Array.isArray(messages)) return false
  return messages.some((m) => {
    const c = m?.content
    if (typeof c === 'string') return false
    if (!Array.isArray(c)) return false
    return c.some((part) => part?.type === 'image_url' && part?.image_url?.url)
  })
}

function deepseekProxyPlugin() {
  let apiKey, baseUrl, chatModel, visionModel

  async function readBody(req) {
    const chunks = []
    for await (const chunk of req) chunks.push(chunk)
    return JSON.parse(Buffer.concat(chunks).toString())
  }

  function resolveModel(messages) {
    return messagesIncludeImage(messages) ? visionModel : chatModel
  }

  async function forwardToDeepSeek(messages, stream) {
    const model = resolveModel(messages)
    return fetch(`${baseUrl}/v1/chat/completions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${apiKey}`,
      },
      body: JSON.stringify({ model, messages, stream }),
    })
  }

  return {
    name: 'deepseek-proxy',
    configureServer(server) {
      const env = loadEnv('', process.cwd(), '')
      apiKey = env.DEEPSEEK_API_KEY || ''
      baseUrl = env.DEEPSEEK_BASE_URL || 'https://api.deepseek.com'
      chatModel = env.DEEPSEEK_MODEL || 'deepseek-chat'
      visionModel = env.DEEPSEEK_VISION_MODEL || chatModel

      // Streaming chat endpoint
      server.middlewares.use('/api/chat', async (req, res) => {
        if (req.method !== 'POST') {
          res.statusCode = 405
          return res.end('Method Not Allowed')
        }
        try {
          const body = await readBody(req)
          const upstream = await forwardToDeepSeek(body.messages || [], true)

          if (!upstream.ok) {
            res.statusCode = upstream.status
            return res.end(await upstream.text())
          }

          res.setHeader('Content-Type', 'text/event-stream')
          res.setHeader('Cache-Control', 'no-cache')
          res.setHeader('Connection', 'keep-alive')

          const reader = upstream.body.getReader()
          const decoder = new TextDecoder()
          while (true) {
            const { done, value } = await reader.read()
            if (done) break
            res.write(decoder.decode(value, { stream: true }))
          }
          res.end()
        } catch (err) {
          res.statusCode = 502
          res.end(JSON.stringify({ error: err.message }))
        }
      })

      // Non-streaming content generation endpoint
      server.middlewares.use('/api/generate-content', async (req, res) => {
        if (req.method !== 'POST') {
          res.statusCode = 405
          return res.end('Method Not Allowed')
        }
        try {
          const body = await readBody(req)
          const upstream = await forwardToDeepSeek(body.messages || [], false)

          if (!upstream.ok) {
            res.statusCode = upstream.status
            return res.end(await upstream.text())
          }

          res.setHeader('Content-Type', 'application/json')
          const data = await upstream.json()
          const content = data.choices?.[0]?.message?.content || ''
          res.end(JSON.stringify({ content }))
        } catch (err) {
          res.statusCode = 502
          res.end(JSON.stringify({ error: err.message }))
        }
      })
    },
  }
}

export default defineConfig({
  plugins: [vue(), deepseekProxyPlugin()],
  server: {
    proxy: {
      // 将业务后端 /api/* 转到 Node（开发环境避免跨域 Failed to fetch）
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
        /** 不代理：由上方 deepseek 中间件处理 */
        bypass(req) {
          const u = req.url || ''
          if (u.startsWith('/api/chat') || u.startsWith('/api/generate-content')) {
            return false
          }
        },
      },
    },
  },
})
