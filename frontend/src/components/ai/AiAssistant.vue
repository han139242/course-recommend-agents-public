<script setup>
import { Bot, ChevronDown, MessageCircle, Send, Sparkles, X } from '@lucide/vue'
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
const lastPromptId = ref(null)

const detailRef = computed(() => props.detail)
const authRef = computed(() => props.isAuthenticated)
const { messages, quickPrompts, sending, send } = useAiChat(detailRef, authRef, {
  onActions(actions) {
    emit('run-actions', actions)
  },
})

const open = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const mentionEntries = computed(() => {
  const map = new Map()

  Object.entries(props.knowledgeLookup || {}).forEach(([value, meta]) => {
    const normalized = String(value || '').trim()
    if (!normalized) return
    map.set(normalized, {
      value: normalized,
      label: String(meta?.label || meta?.canonicalName || meta?.name || normalized).trim() || normalized,
    })
  })

  ;(props.knowledgeMentions || []).forEach((value) => {
    const normalized = String(value || '').trim()
    if (!normalized || map.has(normalized)) return
    map.set(normalized, { value: normalized, label: normalized })
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

function resolveHintTitle(hint = {}) {
  const label = resolveKnowledgeLabel(hint.knowledgeId, '')
  const title = replaceKnowledgeLabels(hint.title || '')
  if (label && title && title !== label) return `${title}：${label}`
  return title || label || '学习提醒'
}

function buildPathTitle(item = {}) {
  return resolveKnowledgeLabel(item.knowledgeId, replaceKnowledgeLabels(item.title || '推荐知识点'))
}

const localStruggleHint = ref(null)

const visibleHints = computed(() => {
  const list = []
  if (localStruggleHint.value) {
    list.push({
      ...localStruggleHint.value,
      displayTitle: localStruggleHint.value.title,
      displayDetail: localStruggleHint.value.detail,
    })
  }
  const propHints = (props.proactiveHints || []).map((hint) => ({
    ...hint,
    displayTitle: resolveHintTitle(hint),
    displayDetail: replaceKnowledgeLabels(hint.detail || ''),
  }))
  return [...list, ...propHints].slice(0, 2)
})
const lastAssistantMessage = computed(() =>
  [...messages.value].reverse().find((message) => message.role === 'assistant' && message.mode) || null,
)
const visibleRecommendedPath = computed(() =>
  (props.recommendedPath || []).slice(0, 3).map((item) => ({
    ...item,
    displayTitle: buildPathTitle(item),
    displayDetail: replaceKnowledgeLabels(item.detail || ''),
    displayReasonLabel: String(item.reasonLabel || '').trim() || '推荐',
  })),
)

const recommendedRouteIds = computed(() =>
  (props.recommendedPath || []).map((item) => item.knowledgeId || item.title).filter(Boolean),
)

const panelStatus = computed(() => {
  if (sending.value) return { tone: 'pending', label: '正在回答', detail: 'AI 助教正在结合当前知识点整理解释。' }
  return props.isAuthenticated
    ? { tone: 'online', label: '可继续提问', detail: '可围绕当前节点、路径或练习结果继续追问。' }
    : { tone: 'idle', label: '访客模式', detail: '登录后可以结合个人进度获得更准确的建议。' }
})

function submit() {
  const content = draft.value.trim()
  if (!content) return
  send(content)
  draft.value = ''
  open.value = true
}

function handlePrompt(prompt) {
  open.value = true
  send(prompt)
}

function scrollMessagesToBottom() {
  nextTick(() => {
    if (messageListRef.value) {
      messageListRef.value.scrollTop = messageListRef.value.scrollHeight
    }
  })
}

watch(messages, scrollMessagesToBottom, { deep: true })

watch(
  () => props.autoPrompt,
  (prompt) => {
    if (!prompt?.text || prompt.id === lastPromptId.value) return
    lastPromptId.value = prompt.id
    open.value = true
    send(prompt.text)
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

function triggerStruggleHint(nodeName) {
  localStruggleHint.value = {
    title: '测验挑战卡关？',
    detail: `检测到你在“${nodeName}”测验中遇到阻碍。点击让我为你提供通俗大白话的讲解！`,
    knowledgeId: nodeName,
    reasonCode: 'weakness',
    isStruggle: true,
  }
  open.value = false
}

function handleBubbleClick() {
  const primaryHint = visibleHints.value[0]
  open.value = true
  if (primaryHint && primaryHint.isStruggle) {
    send(`我刚刚在学习“${primaryHint.knowledgeId}”时测验卡关了，请用极度通俗易懂的大白话、结合生活中的生动比喻，为我全面讲解一下这个概念，并说明它的实际用途。`)
    localStruggleHint.value = null
  }
}

defineExpose({
  triggerStruggleHint,
})
</script>

<template>
  <div class="ai-assistant" :class="{ 'ai-assistant--open': open }">
    <button v-if="!open && visibleHints.length" type="button" class="ai-proactive-bubble" @click="handleBubbleClick">
      <Sparkles :size="16" aria-hidden="true" />
      <span>{{ visibleHints[0].displayTitle }}</span>
    </button>

    <button type="button" class="ai-fab" :aria-expanded="open ? 'true' : 'false'" @click="open = !open">
      <X v-if="open" :size="20" aria-hidden="true" />
      <MessageCircle v-else :size="20" aria-hidden="true" />
    </button>

    <section v-if="open" class="ai-panel">
      <header class="ai-panel__head">
        <div>
          <span class="eyebrow">AI Tutor</span>
          <strong>学习助教</strong>
        </div>
        <button type="button" class="icon-btn" title="关闭" @click="open = false">
          <X :size="18" aria-hidden="true" />
        </button>
      </header>

      <div class="ai-panel__status" :class="`ai-panel__status--${panelStatus.tone}`">
        <Bot :size="16" aria-hidden="true" />
        <div>
          <strong>{{ panelStatus.label }}</strong>
          <span>{{ panelStatus.detail }}</span>
        </div>
      </div>

      <button type="button" class="ai-panel__fold" @click="insightsOpen = !insightsOpen">
        <span>学习建议</span>
        <ChevronDown :size="16" :class="{ expanded: insightsOpen }" aria-hidden="true" />
      </button>

      <div v-if="insightsOpen" class="ai-insight-stack">
        <article v-if="learnerProfile" class="ai-learner-card">
          <strong>{{ learnerProfile.label || learnerProfile.style || '学习画像' }}</strong>
          <p>{{ learnerProfile.summary || '系统会结合你的学习记录调整后续建议。' }}</p>
        </article>

        <article v-for="hint in visibleHints" :key="hint.title || hint.knowledgeId" class="ai-hint-card">
          <span class="status-pill status-pill--learning">{{ hint.reasonLabel || hint.reasonCode || '提醒' }}</span>
          <strong>{{ hint.displayTitle }}</strong>
          <p>{{ hint.displayDetail || '建议先处理这个学习点，再继续往后推进。' }}</p>
          <button v-if="hint.knowledgeId" type="button" class="inline-link inline-link--ai" @click="emit('focus-knowledge', hint.knowledgeId)">
            定位知识点
          </button>
        </article>

        <article v-if="visibleRecommendedPath.length" class="ai-path-card">
          <div class="ai-path-card__head">
            <strong>推荐路径</strong>
            <button type="button" class="ai-panel__path-link" @click="emit('show-recommend-path', recommendedRouteIds)">
              高亮路径
            </button>
          </div>
          <button
            v-for="(item, index) in visibleRecommendedPath"
            :key="item.knowledgeId || item.title || index"
            type="button"
            class="ai-path-step"
            @click="emit('focus-knowledge', item.knowledgeId || item.title)"
          >
            <span>{{ index + 1 }}</span>
            <div>
              <strong class="ai-path-step__title">{{ item.displayTitle }}</strong>
              <small>{{ item.displayReasonLabel }}</small>
            </div>
          </button>
        </article>
      </div>

      <div ref="messageListRef" class="ai-panel__messages">
        <article
          v-for="(message, index) in messages"
          :key="index"
          class="ai-message"
          :class="`ai-message--${message.role}`"
        >
          <span v-if="message.agentLabel" class="ai-message__agent">{{ message.agentLabel }}</span>
          <p>{{ replaceKnowledgeLabels(message.content) }}</p>
        </article>
        <p v-if="sending" class="muted-text">正在生成讲解...</p>
      </div>

      <div class="panel-label">快捷提问</div>
      <div class="ai-panel__quick">
        <button
          v-for="prompt in quickPrompts.slice(0, 3)"
          :key="prompt"
          type="button"
          class="quick-prompt"
          @click="handlePrompt(prompt)"
        >
          {{ prompt }}
        </button>
      </div>

      <form class="ai-panel__composer" @submit.prevent="submit">
        <input v-model="draft" type="text" placeholder="问一个学习问题" />
        <button type="submit" class="primary-btn" :disabled="sending || !draft.trim()">
          <Send :size="16" aria-hidden="true" />
          发送
        </button>
      </form>
    </section>
  </div>
</template>
