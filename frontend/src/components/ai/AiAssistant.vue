<script setup>
import { computed, nextTick, ref, watch } from 'vue'

import { useAiChat } from '../../composables/useAiChat'

const props = defineProps({
  detail: Object,
  isAuthenticated: Boolean,
  modelValue: Boolean,
  autoPrompt: {
    type: Object,
    default: null,
  },
  knowledgeMentions: {
    type: Array,
    default: () => [],
  },
  knowledgeLookup: {
    type: Object,
    default: () => ({}),
  },
  recommendedPath: {
    type: Array,
    default: () => [],
  },
  proactiveHints: {
    type: Array,
    default: () => [],
  },
  learnerProfile: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['focus-knowledge', 'run-actions', 'show-recommend-path', 'update:modelValue'])
const draft = ref('')
const messageListRef = ref(null)
const insightsOpen = ref(false)
const detailRef = computed(() => props.detail)
const authRef = computed(() => props.isAuthenticated)
const { messages, quickPrompts, sending, send } = useAiChat(detailRef, authRef, {
  onActions(actions) {
    emit('run-actions', actions)
  },
})
const lastPromptId = ref(null)
const mentionEntries = computed(() => {
  const map = new Map()

  Object.entries(props.knowledgeLookup || {}).forEach(([value, meta]) => {
    const normalized = String(value || '').trim()
    if (!normalized) return
    map.set(normalized, {
      value: normalized,
      label:
        String(meta?.label || meta?.canonicalName || meta?.name || normalized).trim() || normalized,
    })
  })

  ;(props.knowledgeMentions || []).forEach((value) => {
    const normalized = String(value || '').trim()
    if (!normalized || map.has(normalized)) return
    map.set(normalized, {
      value: normalized,
      label: normalized,
    })
  })

  return [...map.values()].sort((left, right) => right.value.length - left.value.length)
})

function resolveKnowledgeLabel(value = '', fallback = '') {
  const normalized = String(value || '').trim()
  const fallbackValue = String(fallback || '').trim()
  if (!normalized) return fallbackValue

  const entry = props.knowledgeLookup?.[normalized]
  return String(entry?.label || entry?.canonicalName || entry?.name || normalized).trim() || fallbackValue
}

function replaceKnowledgeLabels(text = '') {
  let output = String(text || '')
  mentionEntries.value.forEach(({ value, label }) => {
    if (!value || !label || value === label || !output.includes(value)) return
    output = output.split(value).join(label)
  })
  return output
}

function findMentionInText(...values) {
  const source = values
    .map((item) => String(item || '').trim())
    .filter(Boolean)
    .join(' ')

  if (!source) return null
  return mentionEntries.value.find((item) => item.value && source.includes(item.value)) || null
}

function resolveInsightPrefix(reasonCode = '', fallback = '') {
  switch (String(reasonCode || '').trim()) {
    case 'weakness':
      return '薄弱补强'
    case 'forgetting_review':
      return '回温复习'
    case 'cross_course_transfer':
      return '跨课迁移'
    case 'gap_bridge':
      return '前置补桥'
    default:
      return String(fallback || '').trim()
  }
}

function hasMachineId(text = '') {
  return /[0-9a-f]{16,}/i.test(String(text || ''))
}

function buildHintTitle(hint = {}) {
  const explicitId = String(hint.knowledgeId || '').trim()
  const mention = findMentionInText(explicitId, hint.title, hint.detail, hint.route)
  const label = resolveKnowledgeLabel(explicitId || mention?.value || '', mention?.label || '')
  const prefix = resolveInsightPrefix(hint.reasonCode, hint.title)
  const normalizedTitle = replaceKnowledgeLabels(hint.title || '')

  if (label && prefix) {
    return `${prefix}：${label}`
  }

  if (label && hasMachineId(normalizedTitle)) {
    return label
  }

  return normalizedTitle || label || '学习提醒'
}

function buildPathTitle(item = {}) {
  const explicitId = String(item.knowledgeId || '').trim()
  const mention = findMentionInText(explicitId, item.title, item.detail, item.route)
  const label = resolveKnowledgeLabel(explicitId || mention?.value || '', mention?.label || '')
  const normalizedTitle = replaceKnowledgeLabels(item.title || '')

  if (label && hasMachineId(normalizedTitle)) {
    return label
  }

  return label || normalizedTitle || '推荐知识点'
}

const visibleHints = computed(() =>
  (props.proactiveHints || []).slice(0, 2).map((hint) => ({
    ...hint,
    displayTitle: buildHintTitle(hint),
    displayDetail: replaceKnowledgeLabels(hint.detail || ''),
  })),
)
const lastAssistantMessage = computed(() =>
  [...messages.value].reverse().find((message) => message.role === 'assistant' && message.mode) || null,
)
const visibleRecommendedPath = computed(() =>
  (props.recommendedPath || []).slice(0, 3).map((item) => ({
    ...item,
    displayTitle: buildPathTitle(item),
    displayDetail: replaceKnowledgeLabels(item.detail || ''),
    displayReasonLabel: String(item.reasonLabel || '').trim() || '图谱推荐',
    displayCourseLabel:
      String(item.sourceCourse || '').trim() && String(item.targetCourse || '').trim()
        ? String(item.sourceCourse || '').trim() === String(item.targetCourse || '').trim()
          ? String(item.targetCourse || '').trim()
          : `${String(item.sourceCourse || '').trim()} → ${String(item.targetCourse || '').trim()}`
        : String(item.targetCourse || item.sourceCourse || '').trim(),
  })),
)
const recommendedRouteIds = computed(() =>
  (props.recommendedPath || []).map((item) => item.knowledgeId || item.title).filter(Boolean),
)
const roadmapItems = computed(() =>
  visibleRecommendedPath.value.map((item, index) => {
    const rawScore = Math.round(Number(item.score || 0) * 100)
    const scorePercent = Math.max(0, Math.min(100, rawScore))
    const rawMastery = Math.round(Number(item.masteryScore || 0))
    const masteryPercent = Math.max(0, Math.min(100, rawMastery))
    const hasTransfer =
      String(item.sourceCourse || '').trim() &&
      String(item.targetCourse || '').trim() &&
      String(item.sourceCourse || '').trim() !== String(item.targetCourse || '').trim()
    const courseLabel = hasTransfer
      ? `${item.sourceCourse} -> ${item.targetCourse}`
      : String(item.targetCourse || item.sourceCourse || '').trim()

    return {
      ...item,
      stepIndex: index + 1,
      scorePercent,
      masteryPercent,
      stageLabel: item.reasonLabel || item.stage || '推荐',
      courseLabel,
      priorityLabel: index === 0 ? '当前优先' : index === 1 ? '紧接着学' : `第 ${index + 1} 步`,
      summary: item.detail || '这是当前最值得优先推进的一步。',
    }
  }),
)
const hiddenStepCount = computed(() =>
  Math.max((props.recommendedPath || []).length - roadmapItems.value.length, 0),
)
const roadmapHeadline = computed(() => {
  const currentName = String(props.detail?.canonicalName || props.detail?.name || '').trim()
  const total = recommendedRouteIds.value.length
  if (currentName && total) {
    return `围绕 ${currentName} 的 ${total} 步推进建议`
  }
  if (total) {
    return `当前最值得优先推进的 ${total} 步主线`
  }
  return '系统会根据当前上下文整理下一步主线'
})
const insightCount = computed(() =>
  Number(Boolean(props.learnerProfile)) + visibleHints.value.length + Number(Boolean(visibleRecommendedPath.value.length)),
)
const panelStatus = computed(() => {
  if (sending.value) {
    return {
      tone: 'pending',
      label: '正在响应',
      detail: '会优先尝试当前可用的在线服务，必要时自动切到本地兜底。',
    }
  }

  switch (lastAssistantMessage.value?.mode) {
    case 'online':
      return {
        tone: 'online',
        label: '在线回答',
        detail: '当前回答已通过在线服务生成，可继续追问或联动图谱操作。',
      }
    case 'action':
      return {
        tone: 'action',
        label: '页面执行',
        detail: '图谱控制指令已经下发到页面，适合继续展开、聚焦或高亮。',
      }
    case 'fallback':
      return {
        tone: 'fallback',
        label: '本地兜底',
        detail: '网络异常时会自动回退到本地上下文回答，避免面板卡死。',
      }
    default:
      return props.isAuthenticated
        ? {
            tone: 'idle',
            label: '已就绪',
            detail: '系统已准备就绪，图谱控制指令可以直接作用到页面。',
          }
        : {
            tone: 'idle',
            label: '已就绪',
            detail: '未登录时也能控制图谱，网络异常时会自动切到本地兜底。',
          }
  }
})
const open = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

function submit() {
  if (!draft.value.trim()) return
  send(draft.value)
  draft.value = ''
  open.value = true
}

function handlePrompt(prompt) {
  open.value = true
  send(prompt)
}

function scrollMessagesToBottom() {
  const element = messageListRef.value
  if (!element) return
  element.scrollTop = element.scrollHeight
}

function resolveMessageModeLabel(message) {
  switch (message?.mode) {
    case 'fallback':
      return '本地兜底'
    case 'action':
      return '页面执行'
    case 'online':
      return '在线回答'
    default:
      return ''
  }
}

watch(
  () => props.autoPrompt,
  (payload) => {
    if (!payload?.id || !payload?.text) return
    if (lastPromptId.value === payload.id) return
    lastPromptId.value = payload.id
    open.value = true
    send(payload.text)
  },
  { deep: true, immediate: true },
)

watch(
  () => open.value,
  async (value) => {
    if (!value) return
    await nextTick()
    scrollMessagesToBottom()
  },
)

watch(
  () => messages.value.map((message) => `${message.role}:${String(message.content || '').length}`).join('|'),
  async () => {
    if (!open.value) return
    await nextTick()
    scrollMessagesToBottom()
  },
)

function splitMessage(text) {
  let remaining = String(text || '')
  const parts = []

  while (remaining) {
    const hit = mentionEntries.value.find((item) => item.value && remaining.includes(item.value))
    if (!hit) {
      parts.push({ type: 'text', value: replaceKnowledgeLabels(remaining) })
      break
    }

    const index = remaining.indexOf(hit.value)
    if (index > 0) {
      parts.push({ type: 'text', value: replaceKnowledgeLabels(remaining.slice(0, index)) })
    }
    parts.push({ type: 'link', value: hit.value, label: hit.label })
    remaining = remaining.slice(index + hit.value.length)
  }

  return parts
}

function showRecommendedPath() {
  if (!recommendedRouteIds.value.length) return
  emit('show-recommend-path', recommendedRouteIds.value)
}
</script>

<template>
  <div class="ai-assistant" :class="{ 'ai-assistant--open': open }">
    <button
      v-if="!open && visibleHints.length"
      type="button"
      class="ai-proactive-bubble"
      @click="open = true"
    >
      <span class="ai-proactive-bubble__icon">&#x2728;</span>
      <span class="ai-proactive-bubble__text">{{ visibleHints[0].displayTitle || visibleHints[0].title }}</span>
    </button>
    <button type="button" class="ai-fab" @click="open = !open">
      <span class="ai-fab__icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2a4 4 0 0 1 4 4v1a3 3 0 0 1 3 3v1a2 2 0 0 1-2 2h-1l-1.5 4.5a1.5 1.5 0 0 1-1.42 1H10.92a1.5 1.5 0 0 1-1.42-1L8 13H7a2 2 0 0 1-2-2v-1a3 3 0 0 1 3-3V6a4 4 0 0 1 4-4z"/>
          <circle cx="9.5" cy="9" r="1" fill="currentColor" stroke="none"/>
          <circle cx="14.5" cy="9" r="1" fill="currentColor" stroke="none"/>
        </svg>
      </span>
      <span class="ai-fab__pulse"></span>
    </button>

    <Transition name="ai-panel-transition">
      <div v-if="open" class="ai-panel">
        <div class="ai-panel__head">
          <div class="ai-panel__brand">
            <div class="ai-panel__avatar">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 2a4 4 0 0 1 4 4v1a3 3 0 0 1 3 3v1a2 2 0 0 1-2 2h-1l-1.5 4.5a1.5 1.5 0 0 1-1.42 1H10.92a1.5 1.5 0 0 1-1.42-1L8 13H7a2 2 0 0 1-2-2v-1a3 3 0 0 1 3-3V6a4 4 0 0 1 4-4z"/>
                <circle cx="9.5" cy="9" r="1" fill="currentColor" stroke="none"/>
                <circle cx="14.5" cy="9" r="1" fill="currentColor" stroke="none"/>
              </svg>
            </div>
            <div class="ai-panel__title">
              <strong>学习助手</strong>
              <div class="ai-panel__status-row">
                <span class="ai-panel__status" :class="`ai-panel__status--${panelStatus.tone}`">
                  <span class="ai-panel__status-dot"></span>
                  {{ panelStatus.label }}
                </span>
                <span class="ai-panel__caption">{{ panelStatus.detail }}</span>
              </div>
            </div>
          </div>
          <button type="button" class="ai-panel__close" @click="open = false">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg>
          </button>
        </div>

        <div v-if="learnerProfile || visibleHints.length || visibleRecommendedPath.length" class="ai-panel__section ai-panel__section--insights">
          <button
            type="button"
            class="ai-panel__fold"
            :class="{ 'ai-panel__fold--open': insightsOpen }"
            :aria-expanded="insightsOpen ? 'true' : 'false'"
            @click="insightsOpen = !insightsOpen"
          >
            <span class="ai-panel__fold-left">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
              <span class="panel-label">学习洞察</span>
            </span>
            <span class="ai-panel__fold-right">
              <span>{{ insightsOpen ? '收起' : `${insightCount} 条` }}</span>
              <svg class="ai-panel__fold-arrow" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M4 6l4 4 4-4"/></svg>
            </span>
          </button>
        </div>

        <Transition name="ai-insights-transition">
          <div v-if="insightsOpen && (learnerProfile || visibleHints.length || visibleRecommendedPath.length)" class="ai-panel__insights">
            <div v-if="learnerProfile" class="ai-learner-card">
              <span class="ai-learner-card__eyebrow">学习风格</span>
              <strong>{{ learnerProfile.label || '当前画像' }}</strong>
              <p>{{ learnerProfile.summary || '系统会根据你最近的学习记录动态调整讲解方式。' }}</p>
            </div>

            <article
              v-for="hint in visibleHints"
              :key="`${hint.reasonCode}-${hint.knowledgeId || hint.title}`"
              class="ai-hint-card"
              :class="`ai-hint-card--${hint.severity || 'medium'}`"
            >
              <div class="ai-hint-card__meta">
                <span class="ai-hint-card__severity">{{ hint.severity || '提醒' }}</span>
                <span class="ai-hint-card__reason">{{ hint.displayTitle || hint.title }}</span>
              </div>
              <p>{{ hint.displayDetail || hint.detail }}</p>
              <button
                v-if="hint.knowledgeId"
                type="button"
                class="inline-link inline-link--ai"
                @click="emit('focus-knowledge', hint.knowledgeId)"
              >
                在图谱中查看 &rarr;
              </button>
            </article>

            <section v-if="visibleRecommendedPath.length" class="ai-path-card">
              <div class="ai-path-card__head">
                <span class="ai-learner-card__eyebrow">推荐路径</span>
                <button
                  v-if="recommendedRouteIds.length > 1"
                  type="button"
                  class="inline-link inline-link--ai ai-panel__path-link"
                  @click="showRecommendedPath"
                >
                  在图谱中查看 &rarr;
                </button>
              </div>

              <article
                v-for="(item, stepIdx) in visibleRecommendedPath"
                :key="item.knowledgeId || item.title"
                class="ai-path-step"
              >
                <div class="ai-path-step__num">{{ stepIdx + 1 }}</div>
                <div class="ai-path-step__content">
                  <div class="ai-path-step__meta">
                    <span>{{ `推荐理由：${item.displayReasonLabel || item.reasonLabel || '图谱推荐'}` }}</span>
                    <span v-if="item.displayCourseLabel">{{ `课程：${item.displayCourseLabel}` }}</span>
                    <strong v-if="item.score">{{ Math.round(Number(item.score || 0) * 100) }}%</strong>
                  </div>
                  <button
                    type="button"
                    class="ai-path-step__title"
                    @click="emit('focus-knowledge', item.knowledgeId || item.title)"
                  >
                    {{ item.displayTitle || item.title }}
                  </button>
                  <p>{{ item.displayDetail || item.detail || '这是当前最值得优先推进的一步。' }}</p>
                </div>
              </article>
            </section>
          </div>
        </Transition>

        <div class="ai-panel__divider">
          <span class="ai-panel__divider-label">对话</span>
        </div>

        <div ref="messageListRef" class="ai-panel__body">
          <div v-if="!messages.length" class="ai-panel__empty">
            <div class="ai-panel__empty-icon">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </div>
            <p>向学习助手提问，获取知识讲解、学习建议或控制图谱。</p>
          </div>
          <article
            v-for="(message, index) in messages"
            :key="`${index}-${message.role}`"
            class="ai-message"
            :class="`ai-message--${message.role}`"
          >
            <div class="ai-message__bubble">
              <div class="ai-message__meta">
                <span class="ai-message__avatar" :class="`ai-message__avatar--${message.role}`">
                  {{ message.role === 'assistant' ? '助' : '我' }}
                </span>
                <span
                  v-if="message.role === 'assistant' && message.agentLabel"
                  class="ai-message__agent"
                >
                  {{ message.agentLabel }}
                </span>
                <span
                  v-if="message.role === 'assistant' && resolveMessageModeLabel(message)"
                  class="ai-message__badge"
                  :class="`ai-message__badge--${message.mode || 'default'}`"
                >
                  {{ resolveMessageModeLabel(message) }}
                </span>
              </div>
              <p>
                <template
                  v-for="(part, partIndex) in splitMessage(message.content)"
                  :key="`${index}-${partIndex}`"
                >
                  <button
                    v-if="part.type === 'link'"
                    type="button"
                    class="inline-link inline-link--ai"
                    @click="emit('focus-knowledge', part.value)"
                  >
                    {{ part.label || part.value }}
                  </button>
                  <span v-else>{{ part.value }}</span>
                </template>
              </p>
            </div>
          </article>

          <div v-if="sending" class="ai-typing">
            <span class="ai-typing__dot"></span>
            <span class="ai-typing__dot"></span>
            <span class="ai-typing__dot"></span>
          </div>
        </div>

        <div class="ai-panel__section ai-panel__section--quick">
          <span class="panel-label">快捷提问</span>
        </div>

        <div class="ai-panel__quick">
          <button
            v-for="prompt in quickPrompts"
            :key="prompt"
            type="button"
            class="quick-prompt"
            @click="handlePrompt(prompt)"
          >
            <svg class="quick-prompt__icon" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M1 12l5-10 5 10M3.5 7.5h5"/></svg>
            {{ prompt }}
          </button>
        </div>

        <div class="ai-panel__composer">
          <div class="ai-panel__input-wrap">
            <textarea
              v-model="draft"
              rows="2"
              placeholder="问一个问题... Ctrl+Enter 发送"
              @keydown.ctrl.enter.prevent="submit"
            ></textarea>
          </div>
          <button type="button" class="ai-send-btn" :disabled="sending || !draft.trim()" @click="submit">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 2L11 13"/><path d="M22 2l-7 20-4-9-9-4z"/></svg>
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>
