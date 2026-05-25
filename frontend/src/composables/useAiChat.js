import { computed, reactive, ref, watch } from 'vue'

import { streamAssistantChat } from '../api/assistant'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../utils/courseMeta'

function buildSystemPrompt(detail) {
  if (!detail) {
    return [
      '你是知识图谱学习助手，请结合当前页面上下文，用中文给出清晰、鼓励式的讲解。',
      '当前还没有锁定具体知识点，请优先从全局图谱、课程关联和下一步学习建议出发回答。',
    ].join('\n')
  }

  return [
    '你是知识图谱学习助手，请用中文回答。',
    `当前知识点：${detail.canonicalName || detail.name}`,
    `课程：${resolveNodeCourseTitle(detail, DEFAULT_COURSE_TITLE)}`,
    `简介：${detail.description || detail.summary || '暂无描述'}`,
    `前置知识：${(detail.prerequisites || []).join('、') || '暂无'}`,
    `掌握度：${detail.masteryScore ?? 0}`,
  ].join('\n')
}

function createSeedMessage(detail) {
  if (!detail) {
    return {
      role: 'assistant',
      agent: 'explorer',
      agentLabel: '知识探索助手',
      mode: 'seed',
      content:
        '现在是全局图谱视角。你可以直接问我“先学什么”“帮我概览当前图谱”或者“推荐下一步学习路径”，不需要先点击节点。',
    }
  }

  const name = detail?.canonicalName || detail?.name || '当前知识点'
  return {
    role: 'assistant',
    agent: 'explorer',
    agentLabel: '知识探索助手',
    mode: 'seed',
    content: `我们现在聚焦在“${name}”。如果你愿意，我可以帮你解释概念、梳理前置知识，或者给出下一步学习建议。`,
  }
}

function buildLocalFallbackReply(detail, question) {
  const normalizedQuestion = String(question || '').trim()

  if (!detail) {
    const sections = [
      '在线讲解服务暂时不稳定，我先基于当前全局图谱给你一个本地回答。',
      '现在还没有锁定具体知识点，所以更适合先做全局梳理，再决定下一步聚焦哪里。',
    ]

    if (/下一步|后续|怎么学|路线/.test(normalizedQuestion)) {
      sections.push('下一步建议：先从你当前最熟悉的一门课切入，再逐步展开它的一跳邻居，把前置知识和后续去向串起来。')
    } else if (/概览|总览|图谱|结构/.test(normalizedQuestion)) {
      sections.push('图谱使用方式：先看课程簇，再选一个核心节点展开邻居，最后沿前置链或相关推荐往下走，会比一上来盯着全图更清晰。')
    } else {
      sections.push('你可以继续直接问我学习路线、课程关系、某门课该先看哪里，或者随时点一个节点让我做更具体的讲解。')
    }

    return sections.join('\n\n')
  }

  const name = detail?.canonicalName || detail?.name || '当前知识点'
  const summary = detail?.summary || detail?.description || `${name} 暂时还没有完整摘要。`
  const prerequisites = (detail?.prerequisites || []).filter(Boolean)
  const recommendReason = String(detail?.recommendReason || '').trim()
  const sections = [
    `在线讲解服务暂时不稳定，我先基于当前知识图谱为你整理“${name}”的本地讲解。`,
    `核心理解：${summary}`,
  ]

  if (prerequisites.length) {
    sections.push(`建议先确认这些前置知识：${prerequisites.join('、')}。`)
  }

  if (/前置|先学|基础/.test(normalizedQuestion) && prerequisites.length) {
    sections.push(`如果你现在要系统学习它，建议顺序是：${prerequisites.join(' -> ')} -> ${name}。`)
  } else if (/下一步|后续|怎么学|路线/.test(normalizedQuestion)) {
    sections.push(
      recommendReason
        ? `下一步建议：${recommendReason}`
        : `下一步建议：先把 ${name} 的定义和前置知识吃透，再进入相关知识与应用场景。`,
    )
  } else if (/对比|区别/.test(normalizedQuestion)) {
    sections.push(`如果你想做对比，最适合把“${name}”和它的前置知识、相关知识一起看：先比定义，再比作用，最后比它们在图谱里的连接位置。`)
  } else {
    sections.push('学习建议：先弄清“它是什么”，再弄清“它依赖什么”，最后看“它会通向什么”。这样学最稳。')
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
      '当前项目暂未配置可用的在线讲解服务。',
      '请检查后端环境配置后重启服务。',
      '相关密钥只保留在服务端即可。',
    ].join('\n')
  }

  return `提示：在线讲解服务当前返回了异常，原因是：${message || '未知错误'}`
}

function buildEmptyReplyHint(detail, question) {
  return `${buildLocalFallbackReply(detail, question)}\n\n提示：这次在线讲解返回了空内容，已自动切换到本地讲解。`
}

function normalizeActionPrompt(input = '') {
  return String(input || '')
    .trim()
    .replace(/\s+/g, ' ')
}

function buildActionReplyHint(actions = []) {
  const [action] = actions
  switch (String(action?.type || '').trim()) {
    case 'focus_knowledge':
      return action?.target ? `已为你聚焦到“${action.target}”。` : '已为你聚焦到当前知识点。'
    case 'expand_knowledge':
      return action?.target ? `已为你展开“${action.target}”的关联节点。` : '已为你展开当前知识点的关联节点。'
    case 'highlight_prerequisite_chain':
      return action?.target ? `已为你高亮“${action.target}”的前置知识链。` : '已为你高亮当前知识点的前置知识链。'
    case 'open_knowledge_detail':
      return action?.target ? `已为你打开“${action.target}”的详情页。` : '已为你打开当前知识点的详情页。'
    case 'filter_course':
      return action?.courseTitle ? `已为你切换到“${action.courseTitle}”课程图谱。` : '已为你按课程筛选当前图谱。'
    case 'fit_graph_view':
      return '已为你适配当前图谱视图。'
    case 'reset_graph_scene':
      return '已为你重置当前图谱场景。'
    default:
      return '已为你执行图谱操作。'
  }
}

function isExplanationStylePrompt(input = '') {
  return /解释|讲解|介绍|分析|总结|概览|概述|是什么|为什么|如何|怎么|推荐|建议|对比|区别|学习路线|学习路径/i.test(
    normalizeActionPrompt(input),
  )
}

function looksLikeActionSummarySection(section = '') {
  const text = String(section || '').trim()
  if (!text) return false

  if (/^(已为你尝试|已为你|我已经让前端|我已让前端|我已经帮你|我已帮你|已经为你|已帮你)/.test(text)) {
    return true
  }

  if (/^\*\*已执行的操作/.test(text)) {
    return true
  }

  return /(前端|页面).*(展开|高亮|聚焦|定位|过滤|筛选|切换|适配|重置|打开)/.test(text)
}

function isPureActionReply(content = '') {
  const sections = String(content || '')
    .trim()
    .split(/\n\s*\n/)
    .map((item) => item.trim())
    .filter(Boolean)

  if (!sections.length) return false

  return sections.every((section) => looksLikeActionSummarySection(section) || /^[-*]\s+/.test(section))
}

function actionMatchesPrompt(text, action = {}) {
  const normalized = normalizeActionPrompt(text)
  const type = String(action?.type || '').trim()

  switch (type) {
    case 'focus_knowledge':
      return /聚焦|定位|选中|切到|切换到|跳到|跳转到|带我看|帮我看|看一下|看下|看看|拉到|移到|对准/.test(normalized)
    case 'expand_knowledge':
      return /展开|扩展|展开一下|展开下|关联节点|邻居|周围节点|关联网络/.test(normalized)
    case 'highlight_prerequisite_chain':
      return /前置知识链|前置链|先修链|前置关系|先修关系|高亮.*前置|看(一下|下)?.*(前置|先修)/.test(normalized)
    case 'open_knowledge_detail':
      return /打开|详情|详情页|详情面板|知识卡|跳转/.test(normalized)
    case 'filter_course':
      return /课程|切到|切换到|图谱切到|筛选|过滤|课程视图/.test(normalized)
    case 'fit_graph_view':
      return /适配视图|fit view|看全图|完整视图|缩放到全图|缩放图谱|居中全图|整张图|全景/.test(normalized)
    case 'reset_graph_scene':
      return /重置|恢复全局|恢复默认视图|清空高亮|重置页面|重置场景|回到初始/.test(normalized)
    default:
      return false
  }
}

function shouldExecuteGraphActions(input, actions = [], content = '') {
  const text = normalizeActionPrompt(input)
  if (!text || !Array.isArray(actions) || !actions.length) return false

  const explicitControlRequest =
    /聚焦|定位|展开|扩展|高亮|打开|查看|看一下|看下|帮我看|带我看|切到|切换到|跳到|跳转到|筛选|过滤|适配视图|fit view|重置|缩放|居中|看全图|看完整图|展示|显示|拉到|移到|选中|前置知识链|前置链|先修链/i.test(
      text,
    ) || actions.some((action) => actionMatchesPrompt(text, action))

  if (!explicitControlRequest) return false
  if (!isExplanationStylePrompt(text)) return true
  return isPureActionReply(content)
}

function stripLeadingActionSummary(content) {
  const text = String(content || '').trim()
  if (!text) return ''

  const segments = text
    .split(/\n\s*\n/)
    .map((item) => item.trim())
    .filter(Boolean)
  if (!segments.length) return ''

  let startIndex = 0
  while (startIndex < segments.length && looksLikeActionSummarySection(segments[startIndex])) {
    startIndex += 1
  }

  if (startIndex === 0) return text
  return segments.slice(startIndex).join('\n\n').trim()
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

    const reply = reactive({
      role: 'assistant',
      content: '',
      agent: '',
      agentLabel: '',
      mode: 'online',
    })
    baseMessages.push(reply)

    const payload = [
      { role: 'system', content: buildSystemPrompt(selectedDetail.value) },
      ...baseMessages.map((message) => ({ role: message.role, content: message.content })),
    ]

    const controller = new AbortController()
    sending.value = true
    let pendingActions = []

    try {
      await streamAssistantChat(payload, {
        authenticated: isAuthenticated.value,
        signal: controller.signal,
        onMessage(chunk) {
          reply.content += chunk
        },
        onEvent(eventPayload) {
          if (eventPayload?.type !== 'assistant_meta') return

          if (eventPayload.agent || eventPayload.agentLabel) {
            reply.agent = String(eventPayload.agent || '').trim()
            reply.agentLabel = String(eventPayload.agentLabel || '').trim()
          }

          if (Array.isArray(eventPayload.actions)) {
            pendingActions = eventPayload.actions
          }
        },
      })

      const shouldExecuteActions = shouldExecuteGraphActions(text, pendingActions, reply.content)
      if (shouldExecuteActions) {
        options.onActions?.(pendingActions)
      } else if (pendingActions.length > 0) {
        reply.content = stripLeadingActionSummary(reply.content)
      }

      const normalizedReply = String(reply.content || '').trim()
      if (!normalizedReply) {
        if (shouldExecuteActions && pendingActions.length > 0) {
          reply.mode = 'action'
          reply.content = buildActionReplyHint(pendingActions)
        } else {
          reply.mode = 'fallback'
          reply.content = buildEmptyReplyHint(selectedDetail.value, text)
        }
      } else if (/^在线讲解服务暂时不稳定/.test(normalizedReply)) {
        reply.mode = 'fallback'
      } else if (shouldExecuteActions) {
        reply.mode = 'action'
      } else {
        reply.mode = 'online'
      }
    } catch (error) {
      reply.mode = 'fallback'
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
