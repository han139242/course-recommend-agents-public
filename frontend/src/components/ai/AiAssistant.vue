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
const visibleHints = computed(() => (props.proactiveHints || []).slice(0, 2))
const lastAssistantMessage = computed(() =>
  [...messages.value].reverse().find((message) => message.role === 'assistant' && message.mode) || null,
)
const visibleRecommendedPath = computed(() => (props.recommendedPath || []).slice(0, 3))
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
  return 'AI 会根据当前上下文整理下一步主线'
})
const insightCount = computed(() =>
  Number(Boolean(props.learnerProfile)) + visibleHints.value.length + Number(Boolean(visibleRecommendedPath.value.length)),
)
const panelStatus = computed(() => {
  if (sending.value) {
    return {
      tone: 'pending',
      label: '正在响应',
      detail: '会优先尝试当前可用的在线模型，必要时自动切到本地兜底。',
    }
  }

  switch (lastAssistantMessage.value?.mode) {
    case 'online':
      return {
        tone: 'online',
        label: '在线回答',
        detail: '后端已命中当前可用的在线模型，这条回答走的是线上通道。',
      }
    case 'action':
      return {
        tone: 'action',
        label: '页面执行',
        detail: '图谱控制指令已经下发到页面，适合继续让 AI 帮你展开、聚焦或高亮。',
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
            detail: '后端会优先选择当前可用的在线模型，图谱控制指令可以直接作用到页面。',
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
  const sorted = [...props.knowledgeMentions].sort((a, b) => b.length - a.length)
  let remaining = String(text || '')
  const parts = []

  while (remaining) {
    const hit = sorted.find((item) => item && remaining.includes(item))
    if (!hit) {
      parts.push({ type: 'text', value: remaining })
      break
    }

    const index = remaining.indexOf(hit)
    if (index > 0) {
      parts.push({ type: 'text', value: remaining.slice(0, index) })
    }
    parts.push({ type: 'link', value: hit })
    remaining = remaining.slice(index + hit.length)
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
      智能提示：{{ visibleHints[0].title }}
    </button>
    <button type="button" class="ai-fab" @click="open = !open">AI</button>

    <div v-if="open" class="ai-panel">
      <div class="ai-panel__head">
        <div class="ai-panel__title">
          <strong>AI 学习助手</strong>
          <div class="ai-panel__status-row">
            <span class="ai-panel__status" :class="`ai-panel__status--${panelStatus.tone}`">
              {{ panelStatus.label }}
            </span>
            <span class="ai-panel__caption">{{ panelStatus.detail }}</span>
          </div>
        </div>
        <button type="button" class="icon-btn" @click="open = false">关闭</button>
      </div>

      <div v-if="learnerProfile || visibleHints.length || visibleRecommendedPath.length" class="ai-panel__section ai-panel__section--insights">
        <button
          type="button"
          class="ai-panel__fold"
          :aria-expanded="insightsOpen ? 'true' : 'false'"
          @click="insightsOpen = !insightsOpen"
        >
          <span class="panel-label">学习洞察</span>
          <span>{{ insightsOpen ? '收起' : `展开 ${insightCount} 条` }}</span>
        </button>
      </div>

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
            <span class="ai-hint-card__reason">{{ hint.title }}</span>
          </div>
          <p>{{ hint.detail }}</p>
          <button
            v-if="hint.knowledgeId"
            type="button"
            class="inline-link inline-link--ai"
            @click="emit('focus-knowledge', hint.knowledgeId)"
          >
            在图谱中查看
          </button>
        </article>

        <section v-if="visibleRecommendedPath.length" class="ai-path-card">
          <div class="ai-path-card__head">
            <span class="ai-learner-card__eyebrow">AI 推荐路径</span>
            <button
              v-if="recommendedRouteIds.length > 1"
              type="button"
              class="inline-link inline-link--ai ai-panel__path-link"
              @click="showRecommendedPath"
            >
              在图谱中查看
            </button>
          </div>

          <article
            v-for="item in visibleRecommendedPath"
            :key="item.knowledgeId || item.title"
            class="ai-path-step"
          >
            <div class="ai-path-step__meta">
              <span>{{ item.reasonLabel || item.stage || '推荐' }}</span>
              <strong v-if="item.score">{{ Math.round(Number(item.score || 0) * 100) }}%</strong>
            </div>
            <button
              type="button"
              class="ai-path-step__title"
              @click="emit('focus-knowledge', item.knowledgeId || item.title)"
            >
              {{ item.title }}
            </button>
            <p>{{ item.detail || '这是当前最值得优先推进的一步。' }}</p>
          </article>
        </section>
      </div>

      <div class="ai-panel__section">
        <span class="panel-label">对话</span>
      </div>

      <div ref="messageListRef" class="ai-panel__body">
        <article
          v-for="(message, index) in messages"
          :key="`${index}-${message.role}`"
          class="ai-message"
          :class="`ai-message--${message.role}`"
        >
          <div class="ai-message__meta">
            <span class="ai-message__role">{{ message.role === 'assistant' ? 'AI' : '你' }}</span>
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
                {{ part.value }}
              </button>
              <span v-else>{{ part.value }}</span>
            </template>
          </p>
        </article>
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
          {{ prompt }}
        </button>
      </div>

      <div class="ai-panel__composer">
        <textarea
          v-model="draft"
          rows="3"
          placeholder="输入你想问的问题..."
          @keydown.ctrl.enter.prevent="submit"
        ></textarea>
        <button type="button" class="primary-btn" :disabled="sending" @click="submit">
          {{ sending ? '发送中...' : '发送' }}
        </button>
      </div>
    </div>
  </div>
</template>
