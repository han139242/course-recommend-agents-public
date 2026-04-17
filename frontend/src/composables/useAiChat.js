import { computed, reactive, ref, watch } from 'vue'

import { streamAssistantChat } from '../api/assistant'

function buildSystemPrompt(detail) {
  if (!detail) {
    return '你是知识图谱学习助手，请结合当前页面上下文，用中文进行清晰、鼓励式的讲解。'
  }

  return [
    '你是知识图谱学习助手，请用中文回答。',
    `当前知识点：${detail.canonicalName || detail.name}`,
    `课程：${detail.chapterTitle || '未分类'}`,
    `简介：${detail.description || detail.summary || '暂无描述'}`,
    `前置知识：${(detail.prerequisites || []).join('、') || '暂无'}`,
    `掌握度：${detail.masteryScore ?? 0}`,
  ].join('\n')
}

function createSeedMessage(detail) {
  const name = detail?.canonicalName || detail?.name || '当前知识点'
  return {
    role: 'assistant',
    agent: 'explorer',
    agentLabel: '知识探索助手',
    content: `我们现在聚焦在“${name}”。如果你愿意，我可以帮你解释概念、梳理前置知识，或者给出下一步学习建议。`,
  }
}

function buildLocalFallbackReply(detail, question) {
  const name = detail?.canonicalName || detail?.name || '当前知识点'
  const summary = detail?.summary || detail?.description || `${name} 暂时还没有完整摘要。`
  const prerequisites = (detail?.prerequisites || []).filter(Boolean)
  const recommendReason = detail?.recommendReason || ''
  const normalizedQuestion = String(question || '').trim()

  const sections = [
    `在线 AI 暂时不稳定，我先基于当前知识图谱为你整理“${name}”的本地讲解。`,
    `核心理解：${summary}`,
  ]

  if (prerequisites.length) {
    sections.push(`建议先确认这些前置知识：${prerequisites.join('、')}。`)
  }

  if (/前置|先学|基础/.test(normalizedQuestion) && prerequisites.length) {
    sections.push(`如果你现在要系统学习它，建议顺序是：${prerequisites.join(' -> ')} -> ${name}。`)
  } else if (/下一步|后续|怎么学|路线/.test(normalizedQuestion)) {
    sections.push(recommendReason ? `下一步建议：${recommendReason}` : `下一步建议：先把 ${name} 的定义和前置知识吃透，再进入相关知识与应用场景。`)
  } else if (/对比|区别/.test(normalizedQuestion)) {
    sections.push(`如果你想做对比，最适合拿“${name}”和它的前置知识、相关知识一起看：先比定义，再比作用，最后比在图谱里的连接位置。`)
  } else {
    sections.push(`学习建议：先弄清“它是什么”，再弄清“它依赖什么”，最后看“它会通向什么”。这样学最稳。`)
    if (recommendReason) {
      sections.push(`补充建议：${recommendReason}`)
    }
  }

  return sections.join('\n\n')
}

function buildErrorHint(error) {
  const message = String(error?.message || '').trim()

  if (Number(error?.status) === 503 && /OPENAI_API_KEY|DEEPSEEK_API_KEY/i.test(message)) {
    return [
      '当前项目还没有配置可用的后端 AI Key。',
      '请在项目根目录的 `.env` 里配置 `OPENAI_API_KEY`（或兼容的 `DEEPSEEK_API_KEY`），然后重启后端服务。',
      '密钥只放后端，不要放到前端。',
    ].join('\n')
  }

  return `提示：线上模型当前返回了异常，原因是：${message || '未知错误'}`
}

export function useAiChat(selectedDetail, isAuthenticated, options = {}) {
  const sessions = reactive({})
  const sending = ref(false)

  const activeSessionKey = computed(() => selectedDetail.value?.routeKnowledgeId || '__global__')

  watch(
    activeSessionKey,
    (key) => {
      if (!sessions[key]) {
        sessions[key] = [createSeedMessage(selectedDetail.value)]
      }
    },
    { immediate: true },
  )

  const messages = computed(() => sessions[activeSessionKey.value] || [])

  const quickPrompts = computed(() => {
    const detail = selectedDetail.value
    if (!detail) {
      return ['帮我快速理解当前图谱', '推荐一条学习路线', '告诉我该先看哪里', '帮我复盘知识结构']
    }

    const name = detail.canonicalName || detail.name
    return [
      `帮我解释 ${name}`,
      `对比 ${name} 和相关知识`,
      `推荐 ${name} 的下一步学习`,
      `高亮 ${name} 的前置链`,
      `展开 ${name} 的关联节点`,
    ]
  })

  async function send(content) {
    const text = String(content || '').trim()
    if (!text || sending.value) return

    const key = activeSessionKey.value
    if (!sessions[key]) {
      sessions[key] = [createSeedMessage(selectedDetail.value)]
    }

    const baseMessages = sessions[key]
    baseMessages.push({ role: 'user', content: text })
    const reply = { role: 'assistant', content: '', agent: '', agentLabel: '' }
    baseMessages.push(reply)

    const payload = [
      { role: 'system', content: buildSystemPrompt(selectedDetail.value) },
      ...baseMessages.map((message) => ({ role: message.role, content: message.content })),
    ]

    const controller = new AbortController()
    sending.value = true

    try {
      await streamAssistantChat(payload, {
        authenticated: isAuthenticated.value,
        signal: controller.signal,
        onMessage(chunk) {
          reply.content += chunk
        },
        onEvent(payload) {
          if (payload?.type === 'assistant_meta') {
            if (payload.agent || payload.agentLabel) {
              reply.agent = String(payload.agent || '').trim()
              reply.agentLabel = String(payload.agentLabel || '').trim()
            }
            if (Array.isArray(payload.actions)) {
              options.onActions?.(payload.actions)
            }
          }
        },
      })
    } catch (error) {
      reply.content =
        reply.content ||
        `${buildLocalFallbackReply(selectedDetail.value, text)}\n\n${buildErrorHint(error)}`
    } finally {
      sending.value = false
    }
  }

  return {
    messages,
    sending,
    quickPrompts,
    send,
  }
}
