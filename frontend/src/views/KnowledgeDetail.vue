<script setup>
import { computed, ref, shallowRef, watch, nextTick, onBeforeUnmount } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { domains } from '../data/courseData'
import { knowledgeDetails } from '../data/learning'
import { fetchKnowledgeDetail } from '../auth'
import { buildKnowledgeLookupCandidates } from '../utils/knowledgeLookup'
import { getKnowledgeMasteryProfile } from '../utils/knowledgeExperience'
import { useAiChat, generateLearningContent, imageFileToDataUrl, buildInstantLearningContent } from '../composables/useAiChat'
import { renderLatex } from '../composables/useKatex'
import 'katex/dist/katex.min.css'

const route = useRoute()
const knowledgeId = computed(() => String(route.params.knowledgeId || ''))

// Data resolution

const currentPoint = computed(() => {
  for (const d of domains) {
    const p = d.points.find((pt) => pt.id === knowledgeId.value)
    if (p) return { ...p, domain: d.name, domainId: d.id }
  }
  return null
})

const detailSnapshot = ref(null)

const detailData = computed(() => detailSnapshot.value || knowledgeDetails[knowledgeId.value] || null)

const displayPoint = computed(() => {
  if (currentPoint.value) {
    return currentPoint.value
  }

  if (!detailData.value) {
    return null
  }

  const domainName =
    detailData.value.parentTopics?.[0] ||
    detailData.value.chapterTitle ||
    detailData.value.entityType ||
    '知识点'

  return {
    id: detailData.value.routeKnowledgeId || detailData.value.id || knowledgeId.value,
    name: detailData.value.name || detailData.value.canonicalName || knowledgeId.value,
    domain: domainName,
    domainId: '',
    level: 'core',
    levelLabel: '核心',
    mastery: Number(detailData.value.masteryScore || 0),
    description: detailData.value.description || detailData.value.summary || '',
    recommend: detailData.value.recommendReason || '',
    route: `/knowledge/${detailData.value.routeKnowledgeId || detailData.value.id || knowledgeId.value}`,
  }
})

const currentMastery = computed(() => {
  const score = Number(detailData.value?.masteryScore)
  return Number.isFinite(score) ? score : (displayPoint.value?.mastery ?? 0)
})

const masteryProfile = computed(() => getKnowledgeMasteryProfile(displayPoint.value, currentMastery.value))

const displayDescription = computed(() => {
  return detailData.value?.description || displayPoint.value?.description || ''
})

const prerequisitePoints = computed(() => {
  if (!detailData.value?.prerequisites) return []
  return detailData.value.prerequisites.map((name) => {
    for (const d of domains) {
      const p = d.points.find((pt) => pt.name === name || pt.name.includes(name))
      if (p) return { ...p, domain: d.name }
    }
    return { name, mastery: 0, domain: '未收录' }
  })
})

const nextSteps = computed(() => {
  if (!displayPoint.value?.domainId) return []
  const currentDomain = domains.find((d) => d.id === displayPoint.value.domainId)
  if (!currentDomain) return []
  return currentDomain.points
    .filter((p) => p.id !== knowledgeId.value && p.mastery < 60)
    .slice(0, 3)
})

function knowledgeLookupCandidates() {
  return buildKnowledgeLookupCandidates({
    knowledgeId: knowledgeId.value,
    currentPoint: currentPoint.value,
    detailSnapshot: detailSnapshot.value,
  })
}


// Middle column: AI-generated content

const AI_CONTENT_STORAGE_PREFIX = 'knowledge-detail-ai-content:v1'
const aiContent = ref('')
const contentLoading = ref(false)
const contentError = ref('')
let contentRequestSerial = 0
let contentAbortController = null

const aiContentStorageIds = computed(() => {
  return [...new Set(
    [
      detailData.value?.routeKnowledgeId,
      displayPoint.value?.id,
      detailData.value?.id,
      knowledgeId.value,
    ]
      .map((item) => String(item || '').trim())
      .filter(Boolean),
  )]
})

function readStoredAiContent() {
  if (typeof window === 'undefined') return ''

  for (const id of aiContentStorageIds.value) {
    const storageKey = `${AI_CONTENT_STORAGE_PREFIX}:${id}`
    let raw = ''
    try {
      raw = window.localStorage.getItem(storageKey) || ''
      if (!raw) continue
      const parsed = JSON.parse(raw)
      const content = typeof parsed === 'string' ? parsed : parsed?.content
      if (typeof content === 'string' && content.trim()) {
        return content.trim()
      }
    } catch {
      if (raw.trim()) {
        return raw.trim()
      }
    }
  }

  return ''
}

function persistAiContent(content) {
  const nextContent = String(content || '').trim()
  if (!nextContent || typeof window === 'undefined') return

  const payload = JSON.stringify({
    content: nextContent,
    updatedAt: Date.now(),
  })

  for (const id of aiContentStorageIds.value) {
    const storageKey = `${AI_CONTENT_STORAGE_PREFIX}:${id}`
    try {
      window.localStorage.setItem(storageKey, payload)
    } catch {
      return
    }
  }
}

function restoreStoredAiContent() {
  const stored = readStoredAiContent()
  aiContent.value = stored
  return Boolean(stored)
}

const displayedAiContent = computed(() => {
  if (aiContent.value.trim()) return aiContent.value

  const pt = displayPoint.value || currentPoint.value
  if (!pt) return ''

  const detail = detailData.value || {}
  return buildInstantLearningContent({
    id: pt.id,
    name: pt.name,
    domain: pt.domain,
    levelLabel: pt.levelLabel,
    description: displayDescription.value || pt.description || '',
    mastery: currentMastery.value,
    prerequisites: detail?.prerequisites || [],
    related: detail?.related || [],
    scenarios: detail?.scenarios || [],
  })
})

async function loadContent(force = false) {
  const pt = displayPoint.value
  const detail = detailData.value
  if (!pt) return

  const requestSerial = ++contentRequestSerial

  if (contentAbortController) {
    contentAbortController.abort()
  }
  contentAbortController = new AbortController()

  contentLoading.value = true
  contentError.value = ''

  try {
    const content = await generateLearningContent({
      id: pt.id,
      name: pt.name,
      domain: pt.domain,
      levelLabel: pt.levelLabel,
      description: displayDescription.value,
      mastery: currentMastery.value,
      prerequisites: detail?.prerequisites || [],
      related: detail?.related || [],
      scenarios: detail?.scenarios || [],
    }, {
      force,
      signal: contentAbortController.signal,
      onProgress(nextContent) {
        if (requestSerial !== contentRequestSerial) return
        if (String(nextContent || '').trim()) {
          aiContent.value = nextContent
        }
      },
    })
    if (requestSerial !== contentRequestSerial) return
    aiContent.value = content
    persistAiContent(content)
  } catch (err) {
    if (err?.name === 'AbortError' || requestSerial !== contentRequestSerial) {
      return
    }
    contentError.value = err.message || '生成学习内容失败'
  } finally {
    if (requestSerial === contentRequestSerial) {
      contentLoading.value = false
      contentAbortController = null
    }
  }
}

// Right column: AI chat

const chatRef = ref(null)
const draft = ref('')
const fileInput = ref(null)
const pendingImage = ref('')
const chatInstance = shallowRef(null)
const chatMessages = computed(() => chatInstance.value?.messages.value || [])
const chatLoading = computed(() => chatInstance.value?.loading.value || false)

function initChat() {
  const ctx = displayPoint.value
  const detail = detailData.value
  chatInstance.value = useAiChat({
    knowledgeContext: {
      name: ctx?.name || knowledgeId.value,
      domain: ctx?.domain || '',
      levelLabel: ctx?.levelLabel || '',
      description: detail?.description || ctx?.description || detail?.summary || '',
      mastery: currentMastery.value,
      prerequisites: detail?.prerequisites || [],
      related: detail?.related || [],
      scenarios: detail?.scenarios || [],
    },
  })
}

const suggestedQuestions = computed(() => {
  const name = displayPoint.value?.name || '这个知识点'
  return [
    `${name}的核心思想是什么？`,
    '和先修知识有什么关系？',
    '实际中怎么用？',
  ]
})

async function handleSend(text) {
  const input = typeof text === 'string' ? text : draft.value
  if (!chatInstance.value || (!input.trim() && !pendingImage.value)) return
  const img = pendingImage.value
  draft.value = ''
  pendingImage.value = ''
  await chatInstance.value.send(input, { imageDataUrl: img || undefined })
  await nextTick()
  if (chatRef.value) chatRef.value.scrollTop = chatRef.value.scrollHeight
}

// Lifecycle

async function loadKnowledgeDetail() {
  if (!knowledgeId.value) return
  const fallback = knowledgeDetails[knowledgeId.value] || null
  detailSnapshot.value = fallback
  if (currentPoint.value) {
    return
  }
  for (const candidate of knowledgeLookupCandidates()) {
    try {
      detailSnapshot.value = await fetchKnowledgeDetail(candidate)
      return
    } catch {
      // Try the next identifier so slug pages can fall back to canonical Chinese names.
    }
  }
  detailSnapshot.value = fallback
}

let pageLoadSerial = 0

watch(knowledgeId, () => {
  const loadSerial = ++pageLoadSerial
  contentAbortController?.abort()
  contentAbortController = null
  contentLoading.value = false
  contentError.value = ''
  aiContent.value = ''
  detailSnapshot.value = knowledgeDetails[knowledgeId.value] || null
  restoreStoredAiContent()
  initChat()
  draft.value = ''
  pendingImage.value = ''

  void loadKnowledgeDetail().then(() => {
    if (loadSerial !== pageLoadSerial) return
    const hasStoredContent = restoreStoredAiContent()
    initChat()
    if (!hasStoredContent) {
      void loadContent()
    }
  })
}, { immediate: true })

onBeforeUnmount(() => {
  contentAbortController?.abort()
})

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

// Helpers

function renderMarkdown(md) {
  if (!md) return ''
  let html = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/^## (.+)$/gm, '<h3 class="ai-section-title">$1</h3>')
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/^- (.+)$/gm, '<li>$1</li>')
    .replace(/(<li>.*<\/li>\n?)+/g, (m) => `<ul>${m}</ul>`)
    .replace(/\n{2,}/g, '</p><p>')
    .replace(/\n/g, '<br>')
    .replace(/^/, '<p>')
    .replace(/$/, '</p>')
  return renderLatex(html)
}

function formatChat(text) {
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
</script>

<template>
  <section class="kd-page" v-if="displayPoint">
    <div class="kd-three-col">

      <!-- Left: knowledge meta -->
      <aside class="kd-left">
        <div class="kd-card">
          <span class="section-tag">当前知识点</span>
          <div class="kd-pos-info">
            <div class="kd-pos-row">
              <span class="kd-pos-label">卡片类型</span>
              <span class="kd-pos-value">{{ displayPoint.cardTypeLabel || '知识卡' }}</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">所属领域</span>
              <span class="kd-pos-value">{{ displayPoint.domain }}</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">等级</span>
              <span :class="['point-level', displayPoint.level]">{{ displayPoint.levelLabel }}</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">掌握度</span>
              <span class="kd-pos-value kd-mastery">{{ currentMastery }}%</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">当前判定</span>
              <span :class="['kd-mastery-stage', masteryProfile.stageKey]">{{ masteryProfile.stageLabel }}</span>
            </div>
          </div>
        </div>

        <div class="kd-card" v-if="prerequisitePoints.length">
          <span class="section-tag">先修状态</span>
          <div class="kd-prereq-list">
            <div
              v-for="p in prerequisitePoints"
              :key="p.name"
              class="kd-prereq-item"
              :class="{ weak: p.mastery != null && p.mastery < 50 }"
            >
              <span class="kd-prereq-name">{{ p.name }}</span>
              <span v-if="p.mastery != null" :class="['kd-prereq-pct', { low: p.mastery < 50 }]">{{ p.mastery }}%</span>
            </div>
          </div>
        </div>

        <div class="kd-card" v-if="nextSteps.length">
          <span class="section-tag">推荐下一步</span>
          <div class="kd-next-list">
            <RouterLink v-for="s in nextSteps" :key="s.id" :to="s.route" class="kd-next-item">
              <span>{{ s.name }}</span>
              <span class="kd-next-mastery">{{ s.mastery }}%</span>
            </RouterLink>
          </div>
        </div>

        <RouterLink to="/" class="kd-back-link">返回首页</RouterLink>
      </aside>

      <!-- Middle: AI-generated content -->
      <main class="kd-center">
        <div class="kd-center-header">
          <div class="kd-header-top">
            <span :class="['point-level', displayPoint.level]">{{ displayPoint.levelLabel }}</span>
            <span class="kd-card-type-badge">{{ displayPoint.cardTypeLabel || '知识卡' }}</span>
            <span v-if="displayPoint.recommend" class="point-rec">{{ displayPoint.recommend }}</span>
            <span class="kd-domain-badge">{{ displayPoint.domain }}</span>
          </div>
          <h1 class="kd-title">{{ displayPoint.name }}</h1>
          <p class="kd-desc">{{ displayDescription }}</p>
        </div>

        <div class="kd-ai-content">
          <div class="kd-ai-content-head">
            <span class="section-tag">AI 生成学习内容</span>
            <button v-if="!contentLoading" class="kd-refresh-btn" @click="loadContent(true)">重新生成</button>
          </div>

          <div v-if="contentLoading && !displayedAiContent" class="kd-content-loading">
            <div class="kd-spinner"></div>
            <span>AI 正在根据你的掌握情况生成个性化学习内容...</span>
          </div>

          <div v-else class="kd-content-body-wrap">
            <div v-if="contentError" class="kd-content-error-inline">
              <p>{{ contentError }}</p>
              <button class="kd-retry-btn" @click="loadContent(true)">重试</button>
            </div>
            <div v-if="contentLoading" class="kd-content-streaming-hint">
              <div class="kd-spinner kd-spinner--inline"></div>
              <span>正在继续生成内容...</span>
            </div>
            <div class="kd-content-body" v-html="renderMarkdown(displayedAiContent)"></div>
          </div>
        </div>
      </main>

      <!-- Right: AI chat -->
      <aside class="kd-right">
        <div class="kd-chat-panel">
          <div class="kd-chat-head">
            <span class="section-tag">AI 学伴</span>
          </div>

          <div class="kd-chat-suggest" v-if="chatMessages.length === 0">
            <p class="kd-chat-hint">关于「{{ displayPoint.name }}」，你想了解什么？</p>
            <button
              v-for="q in suggestedQuestions"
              :key="q"
              class="kd-suggest-btn"
              @click="handleSend(q)"
            >{{ q }}</button>
          </div>

          <div class="kd-chat-messages" ref="chatRef">
            <div v-for="(msg, idx) in chatMessages" :key="idx" :class="['kd-msg', msg.role]">
              <div class="kd-msg-avatar">{{ msg.role === 'user' ? '我' : 'AI' }}</div>
              <div class="kd-msg-body">
                <div class="kd-msg-content">
                  <img v-if="msg.role === 'user' && msg.imageUrl" :src="msg.imageUrl" class="kd-msg-image" alt="" />
                  <div v-html="formatChat(msg.content)"></div>
                </div>
              </div>
            </div>
            <div v-if="chatLoading && chatMessages.length > 0 && !chatMessages[chatMessages.length - 1]?.content" class="kd-msg assistant">
              <div class="kd-msg-avatar">AI</div>
              <div class="kd-msg-body"><div class="kd-msg-content kd-typing">思考中...</div></div>
            </div>
          </div>

          <div v-if="pendingImage" class="kd-image-preview">
            <img :src="pendingImage" alt="待发送图片" />
            <button type="button" class="kd-image-remove" @click="clearPendingImage" aria-label="移除图片">×</button>
          </div>
          <div class="kd-chat-input">
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              capture="environment"
              class="kd-file-input"
              @change="onImageSelected"
            />
            <button
              type="button"
              class="kd-attach-btn"
              title="拍照/上传图片"
              :disabled="chatLoading"
              @click="triggerImagePick"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
            </button>
            <input
              v-model="draft"
              type="text"
              placeholder="输入问题，可配图..."
              @keydown.enter.prevent="handleSend()"
              :disabled="chatLoading"
            />
            <button
              class="kd-send-btn"
              @click="handleSend()"
              :disabled="chatLoading || (!draft.trim() && !pendingImage)"
            >发送</button>
          </div>
        </div>
      </aside>

    </div>
  </section>

  <section class="kd-page" v-else>
    <div class="kd-fallback">
      <h2>未找到该知识点</h2>
      <p>请返回首页选择一个知识点开始学习。</p>
      <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
    </div>
  </section>
</template>
