import { ref } from 'vue'

const DEFAULT_IMAGE_PROMPT =
  '请结合上方知识图谱与我的掌握情况，分析这张图片：题目或内容在说什么、涉及哪些知识点、与先修/关联知识的关系如何；若发现薄弱点请指出，并给出可执行的下一步学习建议。使用中文。'

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
  const t = text.trim() || DEFAULT_IMAGE_PROMPT
  if (!imageDataUrl) return t
  return [
    { type: 'text', text: t },
    { type: 'image_url', image_url: { url: imageDataUrl } },
  ]
}

function messageToApiPayload(m) {
  if (m.role === 'user' && m.imageUrl) {
    return { role: 'user', content: buildUserApiContent(m.content || '', m.imageUrl) }
  }
  return { role: m.role, content: m.content }
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
      let s = options.systemPrompt
      if (extra.hasImage) {
        s +=
          '\n\n学生上传了图片（习题、板书或截图等）。请结合知识图谱上下文：识别题目或内容、关联知识点与先修依赖，指出可能的错因或薄弱点，并给出下一步学习建议。使用中文。'
      }
      return s
    }
    if (!options.knowledgeContext) {
      let s =
        '你是一个人工智能专业课程的学习助手，帮助学生理解知识点、规划学习路径、解答疑惑。回答要准确、清晰、有层次。使用中文回答。'
      if (extra.hasImage) {
        s +=
          '\n\n学生上传了图片。请结合对话上下文分析图片内容，并尽量关联到课程知识点与学习路径。使用中文。'
      }
      return s
    }
    const ctx = options.knowledgeContext
    const parts = [
      '你是一个人工智能专业课程的学习助手。',
      `当前学生正在学习知识点：「${ctx.name}」（${ctx.domain || ''}，${ctx.levelLabel || ''}）`,
    ]
    if (ctx.description) parts.push(`知识点描述：${ctx.description}`)
    if (ctx.prerequisites?.length) parts.push(`先修知识：${ctx.prerequisites.join('、')}`)
    if (ctx.related?.length) parts.push(`关联知识：${ctx.related.join('、')}`)
    if (ctx.scenarios?.length) parts.push(`应用场景：${ctx.scenarios.join('、')}`)
    if (ctx.mastery != null) parts.push(`当前掌握度：${ctx.mastery}%`)
    parts.push('')
    parts.push('请帮助学生理解这个知识点。回答要准确、清晰、有层次。使用中文回答。')
    parts.push('如果学生掌握度较低，从基础概念讲起；如果较高，可以深入细节和应用。')
    if (extra.hasImage) {
      parts.push('')
      parts.push(
        '学生上传了图片。请结合上述知识图谱信息：说明图片与当前知识点、先修、关联知识的关系，分析可能的错因或薄弱点，并给出下一步建议。',
      )
    }
    return parts.join('\n')
  }

  async function send(userText, opts = {}) {
    const imageDataUrl = opts.imageDataUrl
    const text = typeof userText === 'string' ? userText : ''
    if (!text.trim() && !imageDataUrl) return
    if (loading.value) return

    error.value = ''
    const displayText =
      text.trim() || (imageDataUrl ? '（已上传图片，请结合知识图谱分析）' : '')

    messages.value.push({
      role: 'user',
      content: displayText,
      imageUrl: imageDataUrl || undefined,
    })

    const assistantMsg = { role: 'assistant', content: '' }
    messages.value.push(assistantMsg)
    loading.value = true

    const hasImageInThread = messages.value.some((m) => m.role === 'user' && m.imageUrl)
    const apiMessages = [
      { role: 'system', content: buildSystemPrompt({ hasImage: hasImageInThread }) },
      ...messages.value
        .filter((m) => {
          if (m.role === 'assistant') return m.content
          if (m.role === 'user') return m.content || m.imageUrl
          return false
        })
        .map((m) => messageToApiPayload(m)),
    ]
    if (apiMessages[apiMessages.length - 1]?.role === 'assistant' && apiMessages[apiMessages.length - 1].content === '') {
      apiMessages.pop()
    }

    try {
      const resp = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messages: apiMessages, stream: true }),
      })

      if (!resp.ok) {
        throw new Error((await resp.text()) || `HTTP ${resp.status}`)
      }

      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed.startsWith('data: ')) continue
          const payload = trimmed.slice(6)
          if (payload === '[DONE]') continue
          try {
            const delta = JSON.parse(payload).choices?.[0]?.delta?.content
            if (delta) {
              assistantMsg.content += delta
              messages.value = [...messages.value]
            }
          } catch {
            /* skip */
          }
        }
      }

      if (buffer.trim().startsWith('data: ') && buffer.trim().slice(6) !== '[DONE]') {
        try {
          const delta = JSON.parse(buffer.trim().slice(6)).choices?.[0]?.delta?.content
          if (delta) {
            assistantMsg.content += delta
            messages.value = [...messages.value]
          }
        } catch {
          /* skip */
        }
      }

      if (!assistantMsg.content) {
        assistantMsg.content = '抱歉，AI 暂时没有生成回复，请稍后重试。'
        messages.value = [...messages.value]
      }
    } catch (err) {
      error.value = err.message || '请求 AI 服务失败'
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

function explainGenerateFetchError(err) {
  const raw = err?.message || String(err)
  if (raw === 'Failed to fetch' || err?.name === 'TypeError') {
    return [
      '无法连接生成服务（Failed to fetch）。',
      '开发环境请用「npm run dev」启动前端，并在 frontend/.env 中配置 DEEPSEEK_API_KEY；',
      '中间栏内容由 Vite 将请求代理到 DeepSeek Chat API（与 DEEPSEEK_MODEL 一致）。',
    ].join('')
  }
  return raw
}

export async function generateLearningContent(ctx) {
  if (contentCache.has(ctx.id)) {
    return contentCache.get(ctx.id)
  }

  const masteryHint =
    ctx.mastery < 40
      ? '学生掌握度较低，请从最基础的概念和直觉讲起，多用类比和简单例子。'
      : ctx.mastery > 70
        ? '学生掌握度较高，可以深入细节、公式推导和高级应用。'
        : '学生掌握度中等，兼顾核心概念和实际应用。'

  const systemMsg = [
    '你是人工智能专业课程的学习助手。请为学生生成关于「' + ctx.name + '」的学习内容。',
    '',
    '知识点信息：',
    '- 所属领域：' + (ctx.domain || ''),
    '- 难度等级：' + (ctx.levelLabel || ''),
    '- 描述：' + (ctx.description || ''),
    ctx.prerequisites?.length ? '- 先修知识：' + ctx.prerequisites.join('、') : '',
    ctx.related?.length ? '- 关联知识：' + ctx.related.join('、') : '',
    ctx.scenarios?.length ? '- 应用场景：' + ctx.scenarios.join('、') : '',
    '- 学生当前掌握度：' + (ctx.mastery ?? 0) + '%',
    '',
    masteryHint,
    '',
    '请严格按照以下 Markdown 格式输出（保留 ## 标题）：',
    '',
    '## 概念解释',
    '(用通俗易懂的语言解释核心概念，2-3段)',
    '',
    '## 关键要点',
    '(列出 3-5 个必须掌握的核心要点，每个要点一行)',
    '',
    '## 与先修知识的关系',
    '(解释这个知识点如何依赖和关联先修知识)',
    '',
    '## 应用举例',
    '(给出 1-2 个实际应用场景)',
  ]
    .filter(Boolean)
    .join('\n')

  const messages = [
    { role: 'system', content: systemMsg },
    { role: 'user', content: '请生成「' + ctx.name + '」的学习内容。' },
  ]

  try {
    const resp = await fetch('/api/generate-content', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages }),
    })

    if (!resp.ok) {
      throw new Error((await resp.text()) || `HTTP ${resp.status}`)
    }

    const data = await resp.json()
    const content = data.content || ''

    contentCache.set(ctx.id, content)
    return content
  } catch (err) {
    throw new Error(explainGenerateFetchError(err))
  }
}
