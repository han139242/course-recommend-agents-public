<script setup>
import { computed, ref, watch } from 'vue'

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
  proactiveHints: {
    type: Array,
    default: () => [],
  },
  learnerProfile: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['focus-knowledge', 'run-actions', 'update:modelValue'])
const draft = ref('')
const detailRef = computed(() => props.detail)
const authRef = computed(() => props.isAuthenticated)
const { messages, quickPrompts, sending, send } = useAiChat(detailRef, authRef, {
  onActions(actions) {
    emit('run-actions', actions)
  },
})
const lastPromptId = ref(null)
const visibleHints = computed(() => (props.proactiveHints || []).slice(0, 2))

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
        <strong>AI 学习助手</strong>
        <button type="button" class="icon-btn" @click="open = false">关闭</button>
      </div>

      <div v-if="learnerProfile || visibleHints.length" class="ai-panel__insights">
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
      </div>

      <div class="ai-panel__section">
        <span class="panel-label">对话</span>
      </div>

      <div class="ai-panel__body">
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
