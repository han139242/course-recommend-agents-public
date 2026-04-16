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
})

const emit = defineEmits(['focus-knowledge', 'update:modelValue'])
const draft = ref('')
const detailRef = computed(() => props.detail)
const authRef = computed(() => props.isAuthenticated)
const { messages, quickPrompts, sending, send } = useAiChat(detailRef, authRef)
const lastPromptId = ref(null)

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
    <button type="button" class="ai-fab" @click="open = !open">AI</button>

    <div v-if="open" class="ai-panel">
      <div class="ai-panel__head">
        <strong>AI 学习助手</strong>
        <button type="button" class="icon-btn" @click="open = false">关闭</button>
      </div>

      <div class="ai-panel__body">
        <article
          v-for="(message, index) in messages"
          :key="`${index}-${message.role}`"
          class="ai-message"
          :class="`ai-message--${message.role}`"
        >
          <span class="ai-message__role">{{ message.role === 'assistant' ? 'AI' : '你' }}</span>
          <p>
            <template v-for="(part, partIndex) in splitMessage(message.content)" :key="`${index}-${partIndex}`">
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
