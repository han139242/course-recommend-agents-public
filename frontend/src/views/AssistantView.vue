<script setup>
import { ref, nextTick, computed } from 'vue'
import { useRoute } from 'vue-router'
import { domains } from '../data/courseData'
import { useAiChat, imageFileToDataUrl } from '../composables/useAiChat'
import { renderLatex } from '../composables/useKatex'
import 'katex/dist/katex.min.css'

const route = useRoute()
const chatRef = ref(null)
const fileInput = ref(null)
const pendingImage = ref('')
const draft = ref((route.query.q ?? '').toString())

const aiModelLabel = import.meta.env.VITE_AI_MODEL_LABEL || 'DeepSeek Chat（deepseek-chat）'

const systemPrompt = computed(() => {
  const domainSummary = domains
    .map((d) => {
      const avg = Math.round(d.points.reduce((s, p) => s + p.mastery, 0) / (d.points.length || 1))
      const weak = d.points.filter((p) => p.mastery < 50).map((p) => p.name)
      return `${d.name}（平均掌握 ${avg}%${weak.length ? `，薄弱：${weak.join('、')}` : ''}）`
    })
    .join('\n')

  return [
    '你是一个人工智能专业课程的学习助手，名叫“AI 学伴”。',
    '你的任务是帮助学生规划学习路径、解答知识点疑惑、分析薄弱环节。',
    '',
    '当前学生的能力画像：',
    domainSummary,
    '',
    '请根据学生的提问，结合上面的能力数据给出有针对性的建议。',
    '回答要准确、清晰、有层次，并使用中文。',
  ].join('\n')
})

const { messages, loading, error, send, clearMessages } = useAiChat({
  systemPrompt: systemPrompt.value,
})

const exampleQuestions = [
  '如果我想做算法工程师，下一步该补哪些知识点？',
  '我梯度下降总学不明白，应该回补哪些先修内容？',
  '怎样从机器学习过渡到大模型应用开发？',
  '帮我分析一下我目前的薄弱环节',
]

async function handleSend(text) {
  const input = typeof text === 'string' ? text : draft.value
  if (!input.trim() && !pendingImage.value) return

  const imageDataUrl = pendingImage.value
  draft.value = ''
  pendingImage.value = ''

  await send(input, { imageDataUrl: imageDataUrl || undefined })
  await nextTick()
  if (chatRef.value) {
    chatRef.value.scrollTop = chatRef.value.scrollHeight
  }
}

function triggerImagePick() {
  fileInput.value?.click()
}

async function onImageSelected(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file || !file.type.startsWith('image/')) return

  if (file.size > 6 * 1024 * 1024) {
    window.alert('图片请小于 6MB')
    return
  }

  try {
    pendingImage.value = await imageFileToDataUrl(file)
  } catch (err) {
    window.alert(err.message || '图片处理失败')
  }
}

function clearPendingImage() {
  pendingImage.value = ''
}

function handleKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

function formatContent(text) {
  if (!text) return ''
  let html = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
  return renderLatex(html)
}

if (draft.value.trim()) {
  handleSend(draft.value)
}
</script>

<template>
  <section class="ast-page">
    <div class="ast-header">
      <span class="section-tag">AI 助学室</span>
      <h1>跨课程学习助手</h1>
      <p>围绕知识点补弱、进阶学习和场景目标提供建议，帮助你快速找到下一步该学什么。</p>
      <p class="ast-model-note">
        本页对话由 <strong>{{ aiModelLabel }}</strong> 流式回答；与「知识学习」里 AI 生成摘要、配图答疑使用同一套
        DeepSeek 接口。下方可输入问题或上传图片。
      </p>
    </div>

    <div class="ast-body">
      <div v-if="messages.length === 0" class="ast-welcome">
        <div class="ast-welcome-icon">AI</div>
        <h2>你好，我是你的 AI 学伴</h2>
        <p>我了解你的学习进度和能力画像，可以帮你分析薄弱点、规划路径、解答问题。试试下面这些问题：</p>
        <div class="ast-example-grid">
          <button
            v-for="q in exampleQuestions"
            :key="q"
            class="ast-example-btn"
            @click="handleSend(q)"
          >{{ q }}</button>
        </div>
      </div>

      <div v-else class="ast-chat-area" ref="chatRef">
        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          :class="['kd-msg', msg.role]"
        >
          <div class="kd-msg-avatar">{{ msg.role === 'user' ? '我' : 'AI' }}</div>
          <div class="kd-msg-body">
            <div class="kd-msg-content">
              <img v-if="msg.role === 'user' && msg.imageUrl" :src="msg.imageUrl" class="kd-msg-image" alt="" />
              <div v-html="formatContent(msg.content)"></div>
            </div>
          </div>
        </div>

        <div v-if="loading && messages.length > 0 && messages[messages.length - 1]?.content === ''" class="kd-msg assistant">
          <div class="kd-msg-avatar">AI</div>
          <div class="kd-msg-body">
            <div class="kd-msg-content kd-typing">正在思考...</div>
          </div>
        </div>
      </div>

      <div class="ast-input-area">
        <div v-if="pendingImage" class="ast-image-preview">
          <img :src="pendingImage" alt="待发送图片" />
          <button type="button" class="kd-image-remove" @click="clearPendingImage" aria-label="移除图片">×</button>
        </div>

        <div class="ast-input-row">
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            capture="environment"
            class="kd-file-input"
            @change="onImageSelected"
          />
          <button type="button" class="kd-attach-btn ast-attach" title="拍照/上传图片" :disabled="loading" @click="triggerImagePick">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
          </button>
          <textarea
            v-model="draft"
            class="ast-textarea"
            rows="2"
            placeholder="输入你当前学不懂的知识点、目标方向，或上传题目照片..."
            @keydown="handleKeydown"
            :disabled="loading"
          ></textarea>
          <button class="kd-send-btn" @click="handleSend()" :disabled="loading || (!draft.trim() && !pendingImage)">
            发送
          </button>
        </div>

        <div class="ast-input-hint">
          <button v-if="messages.length > 0" class="ast-clear-btn" @click="clearMessages">清空对话</button>
          <span>按 Enter 发送，Shift+Enter 换行</span>
        </div>

        <p v-if="error" class="auth-error">{{ error }}</p>
      </div>
    </div>
  </section>
</template>
