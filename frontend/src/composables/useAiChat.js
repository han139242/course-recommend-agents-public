import { ref } from 'vue'

const DEFAULT_IMAGE_PROMPT =
  '请结合上方知识图谱与我的掌握情况，分析这张图片中的题目或内容，说明它涉及哪些知识点、与先修或关联知识有什么关系，并给出可执行的下一步学习建议。请使用中文。'

const RETRYABLE_STATUS = new Set([429, 500, 502, 503, 504])
const ENV_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
const API_BASE_URL_CANDIDATES = ENV_API_BASE_URL
  ? [ENV_API_BASE_URL]
  : ['/api', 'http://localhost:8080/api', 'http://localhost:3000/api', 'http://localhost:3002/api', 'http://localhost:3001/api']

let resolvedApiBaseUrl = API_BASE_URL_CANDIDATES[0]
let apiBaseUrlConfirmed = Boolean(ENV_API_BASE_URL)

function sleep(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function shouldRetryStatus(status) {
  return RETRYABLE_STATUS.has(status)
}

async function readErrorText(resp) {
  const raw = await resp.text()
  if (!raw) return `HTTP ${resp.status}`

  try {
    const parsed = JSON.parse(raw)
    if (typeof parsed?.error === 'string' && parsed.error.trim()) return parsed.error
    if (typeof parsed?.message === 'string' && parsed.message.trim()) return parsed.message
    if (typeof parsed?.error?.message === 'string' && parsed.error.message.trim()) return parsed.error.message
  } catch {
    // Fall through to raw text.
  }

  return raw
}

function getApiBaseCandidates() {
  return [
    resolvedApiBaseUrl,
    ...API_BASE_URL_CANDIDATES.filter((item) => item !== resolvedApiBaseUrl),
  ]
}

function explainAiFetchError(err, context = 'AI 服务') {
  const raw = err?.message || String(err)
  if (/AI proxy is not configured/i.test(raw)) {
    return 'AI 服务尚未配置。请在后端环境变量中设置 DEEPSEEK_API_KEY、DEEPSEEK_BASE_URL 和 DEEPSEEK_MODEL。'
  }
  if (/terminated|socket|networkerror when attempting to fetch resource/i.test(raw)) {
    return `${context}连接被中断了，通常是响应超时或上游模型波动。请稍后再试一次。`
  }
  if (raw === 'Failed to fetch' || err?.name === 'TypeError') {
    return `无法连接${context}（Failed to fetch）。请确认后端 API 已启动；如果前端是单独运行的，请使用 npm run dev 启动以启用 /api 代理，或设置 VITE_API_BASE_URL。`
  }
  if (/HTTP 502|502 Bad Gateway|upstream/i.test(raw)) {
    return `${context}上游暂时不可用，系统已经尝试自动重试。请稍后再试一次。`
  }
  return raw
}

async function fetchWithRetry(path, options = {}, retryOptions = {}) {
  const {
    retries = 1,
    initialDelayMs = 400,
  } = retryOptions

  let lastError = null

  for (const baseUrl of getApiBaseCandidates()) {
    let delayMs = initialDelayMs

    for (let attempt = 0; attempt <= retries; attempt += 1) {
      try {
        const resp = await fetch(`${baseUrl}${path}`, options)
        if (resp.ok) {
          resolvedApiBaseUrl = baseUrl
          apiBaseUrlConfirmed = true
          return resp
        }

        const message = await readErrorText(resp)
        if (attempt < retries && shouldRetryStatus(resp.status)) {
          await sleep(delayMs)
          delayMs *= 2
          continue
        }

        if (resp.status === 404 && !apiBaseUrlConfirmed && !ENV_API_BASE_URL) {
          lastError = new Error(message || `HTTP ${resp.status}`)
          break
        }

        throw new Error(message || `HTTP ${resp.status}`)
      } catch (err) {
        if (err?.name === 'AbortError') {
          throw err
        }
        lastError = err
        if (attempt >= retries) {
          break
        }
        await sleep(delayMs)
        delayMs *= 2
      }
    }
  }

  throw lastError || new Error('请求失败')
}

function extractStreamDelta(line) {
  const trimmed = line.trim()
  if (!trimmed.startsWith('data: ')) return ''

  const payload = trimmed.slice(6)
  if (!payload || payload === '[DONE]') return ''

  try {
    return JSON.parse(payload).choices?.[0]?.delta?.content || ''
  } catch {
    return ''
  }
}

async function readStreamingContent(response, onDelta) {
  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('AI 服务返回了空响应')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let content = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      const delta = extractStreamDelta(line)
      if (!delta) continue
      content += delta
      onDelta?.(content, delta)
    }
  }

  const tailDelta = extractStreamDelta(buffer)
  if (tailDelta) {
    content += tailDelta
    onDelta?.(content, tailDelta)
  }

  return content
}

async function streamChatCompletion(messages, options = {}) {
  const resp = await fetchWithRetry('/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      messages,
      stream: true,
      ...(options.maxTokens ? { maxTokens: options.maxTokens } : {}),
    }),
    signal: options.signal,
  }, {
    retries: options.retries ?? 1,
    initialDelayMs: options.initialDelayMs ?? 500,
  })

  return readStreamingContent(resp, options.onDelta)
}

async function fetchGeneratedContent(messages, options = {}) {
  const resp = await fetchWithRetry('/generate-content', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      messages,
      ...(options.maxTokens ? { maxTokens: options.maxTokens } : {}),
    }),
    signal: options.signal,
  }, {
    retries: options.retries ?? 1,
    initialDelayMs: options.initialDelayMs ?? 500,
  })

  const data = await resp.json()
  return typeof data?.content === 'string' ? data.content : ''
}

/**
 * Resize image to reduce payload; returns data URL (jpeg).
 */
export function imageFileToDataUrl(file, maxDim = 1280, quality = 0.82) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const dataUrl = reader.result
      const img = new Image()
      img.onload = () => {
        let w = img.naturalWidth
        let h = img.naturalHeight
        if (w <= 0 || h <= 0) {
          resolve(typeof dataUrl === 'string' ? dataUrl : '')
          return
        }
        if (w <= maxDim && h <= maxDim) {
          resolve(typeof dataUrl === 'string' ? dataUrl : '')
          return
        }
        if (w > h) {
          h = Math.round((h * maxDim) / w)
          w = maxDim
        } else {
          w = Math.round((w * maxDim) / h)
          h = maxDim
        }
        const canvas = document.createElement('canvas')
        canvas.width = w
        canvas.height = h
        const ctx = canvas.getContext('2d')
        if (!ctx) {
          resolve(typeof dataUrl === 'string' ? dataUrl : '')
          return
        }
        ctx.drawImage(img, 0, 0, w, h)
        resolve(canvas.toDataURL('image/jpeg', quality))
      }
      img.onerror = () => reject(new Error('图片无法解析'))
      img.src = typeof dataUrl === 'string' ? dataUrl : ''
    }
    reader.onerror = () => reject(reader.error || new Error('读取文件失败'))
    reader.readAsDataURL(file)
  })
}

function buildUserApiContent(text, imageDataUrl) {
  const promptText = text.trim() || DEFAULT_IMAGE_PROMPT
  if (!imageDataUrl) return promptText
  return [
    { type: 'text', text: promptText },
    { type: 'image_url', image_url: { url: imageDataUrl } },
  ]
}

function messageToApiPayload(message) {
  if (message.role === 'user' && message.imageUrl) {
    return {
      role: 'user',
      content: buildUserApiContent(message.content || '', message.imageUrl),
    }
  }
  return { role: message.role, content: message.content }
}

function buildKnowledgeSystemPrompt(ctx, extra = {}) {
  const parts = [
    '你是一个人工智能专业课程的学习助手。',
    `当前学生正在学习的知识点：${ctx.name || '未命名知识点'}${ctx.domain ? `（${ctx.domain}）` : ''}${ctx.levelLabel ? `，等级：${ctx.levelLabel}` : ''}。`,
  ]

  if (ctx.description) parts.push(`知识点描述：${ctx.description}`)
  if (ctx.prerequisites?.length) parts.push(`先修知识：${ctx.prerequisites.join('、')}`)
  if (ctx.related?.length) parts.push(`关联知识：${ctx.related.join('、')}`)
  if (ctx.scenarios?.length) parts.push(`应用场景：${ctx.scenarios.join('、')}`)
  if (ctx.mastery != null) parts.push(`当前掌握度：${ctx.mastery}%`)

  parts.push('')
  parts.push('请帮助学生理解这个知识点。回答要准确、清晰、有层次，并使用中文。')
  parts.push('除非学生明确要求展开，否则优先用 120 到 220 字给出紧凑回答。')
  parts.push('如果学生掌握度较低，请从基础概念讲起；如果较高，可以深入到关键细节和实际应用。')

  if (extra.hasImage) {
    parts.push('')
    parts.push('学生上传了一张图片。请结合当前知识点、先修知识和关联知识，分析图片内容、指出可能的薄弱点，并给出下一步学习建议。')
  }

  return parts.join('\n')
}

function buildGenericSystemPrompt(extra = {}) {
  let prompt = '你是一个人工智能专业课程的学习助手，帮助学生理解知识点、规划学习路径、解答疑问。回答要准确、清晰、有层次，并使用中文。除非学生明确要求展开，否则优先用 120 到 220 字给出紧凑回答。'
  if (extra.hasImage) {
    prompt += '\n\n学生上传了一张图片。请结合对话上下文分析图片内容，并尽量关联到课程知识点和学习路径。'
  }
  return prompt
}

/**
 * Streaming chat composable for the AI companion sidebar.
 */
export function useAiChat(options = {}) {
  const messages = ref([])
  const loading = ref(false)
  const error = ref('')

  function buildSystemPrompt(extra = {}) {
    if (options.systemPrompt) {
      let prompt = options.systemPrompt
      if (extra.hasImage) {
        prompt += '\n\n学生上传了一张图片。请结合上下文识别题目或内容，分析涉及的知识点、先修依赖和可能薄弱点，并给出下一步学习建议。'
      }
      return prompt
    }

    if (!options.knowledgeContext) {
      return buildGenericSystemPrompt(extra)
    }

    return buildKnowledgeSystemPrompt(options.knowledgeContext, extra)
  }

  async function send(userText, opts = {}) {
    const imageDataUrl = opts.imageDataUrl
    const text = typeof userText === 'string' ? userText : ''
    if (!text.trim() && !imageDataUrl) return
    if (loading.value) return

    error.value = ''
    const displayText = text.trim() || (imageDataUrl ? '已上传图片，请结合知识图谱分析。' : '')

    messages.value.push({
      role: 'user',
      content: displayText,
      imageUrl: imageDataUrl || undefined,
    })

    const assistantMsg = { role: 'assistant', content: '' }
    messages.value.push(assistantMsg)
    loading.value = true

    const hasImageInThread = messages.value.some((message) => message.role === 'user' && message.imageUrl)
    const apiMessages = [
      { role: 'system', content: buildSystemPrompt({ hasImage: hasImageInThread }) },
      ...messages.value
        .filter((message) => {
          if (message.role === 'assistant') return Boolean(message.content)
          if (message.role === 'user') return Boolean(message.content || message.imageUrl)
          return false
        })
        .map((message) => messageToApiPayload(message)),
    ]

    if (apiMessages[apiMessages.length - 1]?.role === 'assistant' && apiMessages[apiMessages.length - 1].content === '') {
      apiMessages.pop()
    }

    try {
      const content = await streamChatCompletion(apiMessages, {
        retries: 1,
        initialDelayMs: 500,
        maxTokens: opts.maxTokens ?? 260,
        onDelta(nextContent) {
          assistantMsg.content = nextContent
          messages.value = [...messages.value]
        },
      })

      if (!content) {
        assistantMsg.content = '抱歉，AI 暂时没有生成回复，请稍后重试。'
        messages.value = [...messages.value]
      }
    } catch (err) {
      error.value = explainAiFetchError(err)
      assistantMsg.content = `出错了：${error.value}`
      messages.value = [...messages.value]
    } finally {
      loading.value = false
    }
  }

  function clearMessages() {
    messages.value = []
    error.value = ''
  }

  return { messages, loading, error, send, clearMessages }
}

/**
 * Non-streaming content generation for the middle column.
 */
const contentCache = new Map()

function toBulletLines(items = [], emptyText = '当前可先围绕核心定义、基本操作和典型场景建立主线理解。') {
  if (!Array.isArray(items) || items.length === 0) {
    return `- ${emptyText}`
  }
  return items
    .slice(0, 3)
    .map((item) => `- ${String(item || '').trim()}`)
    .filter((line) => line !== '-')
    .join('\n')
}

export function buildInstantLearningContent(ctx = {}) {
  const name = String(ctx?.name || '当前知识点').trim()
  const domain = String(ctx?.domain || '当前领域').trim()
  const description = String(ctx?.description || '').trim()
  const levelLabel = String(ctx?.levelLabel || '基础').trim()
  const mastery = Number(ctx?.mastery ?? 0)
  const prerequisites = Array.isArray(ctx?.prerequisites) ? ctx.prerequisites : []
  const related = Array.isArray(ctx?.related) ? ctx.related : []
  const scenarios = Array.isArray(ctx?.scenarios) ? ctx.scenarios : []

  const masteryAdvice =
    mastery < 40
      ? '你当前掌握度偏低，建议先把定义、基本符号和最小例子学扎实，再进入应用。'
      : mastery > 70
        ? '你当前已经有一定基础，可以把重点放在常见误区、工程实现和跨知识点连接上。'
        : '你当前已有一定理解，适合用“概念 + 例子 + 应用”三步法快速巩固。'

  const relationLines = prerequisites.length
    ? `它通常建立在 ${prerequisites.slice(0, 3).join('、')} 这些先修内容之上。先修越稳，后续理解越顺。`
    : related.length
      ? `它和 ${related.slice(0, 3).join('、')} 等主题联系紧密，可以放到同一条学习链路里一起理解。`
      : `可以先抓住它的核心定义、输入输出和使用场景，再逐步连接到上下游知识。`

  const scenarioLines = scenarios.length
    ? `- ${scenarios.slice(0, 2).join('\n- ')}`
    : `- 在 ${domain} 相关任务中作为基础能力使用\n- 作为后续模型训练、分析或工程实现的前置步骤`

  return [
    '## 概念解释',
    `${name}属于「${domain}」中的${levelLabel}知识点。${description || `学习它时，重点是先弄清楚它解决什么问题、输入输出是什么，以及为什么它会出现在真实任务里。`} ${masteryAdvice}`,
    '',
    '## 关键要点',
    toBulletLines(
      [
        `${name}的核心定义、目标和基本作用`,
        prerequisites[0] ? `和先修知识「${prerequisites[0]}」之间的衔接关系` : `${name}在整条学习路径中的位置`,
        scenarios[0] ? `在「${scenarios[0]}」这类场景里的具体用法` : `${name}在真实项目中的典型落地方式`,
      ],
    ),
    '',
    '## 与先修知识的关系',
    relationLines,
    '',
    '## 应用举例',
    scenarioLines,
    '',
    '> 正在生成更完整的 AI 增强版内容...',
  ].join('\n')
}

function buildContentCacheKey(ctx) {
  return JSON.stringify({
    id: String(ctx?.id || '').trim(),
    name: String(ctx?.name || '').trim(),
    domain: String(ctx?.domain || '').trim(),
    levelLabel: String(ctx?.levelLabel || '').trim(),
    description: String(ctx?.description || '').trim(),
    mastery: Number(ctx?.mastery ?? 0),
    prerequisites: Array.isArray(ctx?.prerequisites) ? ctx.prerequisites : [],
    related: Array.isArray(ctx?.related) ? ctx.related : [],
    scenarios: Array.isArray(ctx?.scenarios) ? ctx.scenarios : [],
  })
}

function explainGenerateFetchError(err) {
  return explainAiFetchError(err, '生成服务')
}

export async function generateLearningContent(ctx, options = {}) {
  const force = Boolean(options.force)
  const cacheKey = buildContentCacheKey(ctx)

  if (!force && contentCache.has(cacheKey)) {
    const cached = contentCache.get(cacheKey)
    options.onProgress?.(cached)
    return cached
  }

  const masteryHint =
    ctx.mastery < 40
      ? '学生掌握度较低，请从最基础的概念和直觉讲起，多用类比和简单例子。'
      : ctx.mastery > 70
        ? '学生掌握度较高，可以深入细节、公式推导和高级应用。'
        : '学生掌握度中等，兼顾核心概念和实际应用。'

  const systemMsg = [
    `你是人工智能专业课程的学习助手。请为学生生成关于「${ctx.name}」的学习内容。`,
    '',
    '知识点信息：',
    `- 所属领域：${ctx.domain || ''}`,
    `- 难度等级：${ctx.levelLabel || ''}`,
    `- 描述：${ctx.description || ''}`,
    ctx.prerequisites?.length ? `- 先修知识：${ctx.prerequisites.join('、')}` : '',
    ctx.related?.length ? `- 关联知识：${ctx.related.join('、')}` : '',
    ctx.scenarios?.length ? `- 应用场景：${ctx.scenarios.join('、')}` : '',
    `- 学生当前掌握度：${ctx.mastery ?? 0}%`,
    '',
    masteryHint,
    '',
    '请输出精简版内容，总长度尽量控制在 220 到 320 个汉字内，避免冗长铺陈。',
    '请严格按照以下 Markdown 格式输出，并保留这些二级标题：',
    '',
    '## 概念解释',
    '用通俗易懂的语言解释核心概念，1 段即可。',
    '',
    '## 关键要点',
    '列出 3 个必须掌握的核心要点，每个要点一行，尽量简短。',
    '',
    '## 与先修知识的关系',
    '用 1 到 2 句话解释这个知识点如何依赖和关联先修知识。',
    '',
    '## 应用举例',
    '给出 1 个最典型的实际应用场景。',
  ]
    .filter(Boolean)
    .join('\n')

  const messages = [
    { role: 'system', content: systemMsg },
    { role: 'user', content: `请生成「${ctx.name}」的学习内容。` },
  ]

  try {
    let content = await fetchGeneratedContent(messages, {
      signal: options.signal,
      retries: 1,
      initialDelayMs: 500,
      maxTokens: 320,
    })

    if (String(content || '').trim()) {
      options.onProgress?.(content)
    } else {
      content = await streamChatCompletion(messages, {
        signal: options.signal,
        retries: 1,
        initialDelayMs: 500,
        maxTokens: 320,
        onDelta(nextContent) {
          options.onProgress?.(nextContent)
        },
      })
    }

    if (!String(content || '').trim()) {
      throw new Error('AI 本次没有返回有效内容，请重试一次。')
    }

    contentCache.set(cacheKey, content)
    return content
  } catch (err) {
    throw new Error(explainGenerateFetchError(err))
  }
}

