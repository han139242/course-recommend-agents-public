<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import 'katex/dist/katex.min.css'

import { generateKnowledgeCard } from '../api/assistant'
import { fetchStudentQuestions, submitStudentAnswer } from '../api/learning'
import BridgeView from '../components/detail/BridgeView.vue'
import { useAiChat } from '../composables/useAiChat'
import { renderWithKatex } from '../composables/useKatex'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'
import { getCoursePalette } from '../utils/colorMap'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../utils/courseMeta'

const route = useRoute()
const graphStore = useGraphStore()
const userStore = useUserStore()
const frontier = ref(null)
const recommendedPath = ref([])
const draftNote = ref('')

// ── 练习测验 ──
const questions = ref([])
const questionsLoading = ref(false)
const selectedAnswers = ref({})
const submittedQuestions = ref(new Set())
const submitResults = ref({})

// ── AI 对话 ──
const aiExpanded = ref(false)
const aiInputText = ref('')
const knowledgeCard = ref(null)
const knowledgeCardLoading = ref(false)
const aiMessagesRef = ref(null)

const knowledgeId = computed(() => decodeURIComponent(route.params.knowledgeId || ''))
const detail = computed(() => graphStore.detailCache[knowledgeId.value] || null)
const progress = computed(() => userStore.getProgressMeta(knowledgeId.value))
const frontierSummary = computed(() =>
  frontier.value?.digestSummary || frontier.value?.message || '前沿摘要暂未返回，后续可继续补充。',
)
const frontierTopics = computed(() => frontier.value?.topics || [])
const frontierPapers = computed(() => frontier.value?.papers || [])
const frontierStatusLabel = computed(() => {
  const status = String(frontier.value?.status || '').trim().toLowerCase()
  if (status === 'ready') return '已就绪'
  if (status === 'stale') return '待刷新'
  if (status === 'failed') return '生成失败'
  return '暂未就绪'
})

const selectedDetail = computed(() => detail.value)
const isAuthenticated = computed(() => userStore.isAuthenticated)

const { messages: aiMessages, sending: aiSending, quickPrompts, send: aiSend } = useAiChat(selectedDetail, isAuthenticated)

function getCourseLabel(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}

function getRecommendationTarget(node) {
  const preferred = String(node?.canonicalName || node?.name || '').trim()
  return preferred || getCourseLabel(node)
}

// ── 推荐路径 ──
const crossCourseItems = computed(() =>
  recommendedPath.value.filter((item) => item.reasonCode === 'cross_course_transfer'),
)
const regularPathItems = computed(() =>
  recommendedPath.value.filter((item) => item.reasonCode !== 'cross_course_transfer'),
)

function getRecommendedPathLabel(item) {
  if (!item) return ''
  if (item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse) {
    return `${item.sourceCourse} → ${item.targetCourse} · ${item.title}`
  }
  return item.title
}

function getReasonLabel(code) {
  const map = {
    cross_course_transfer: '跨课程迁移',
    gap_bridge: '缺口桥接',
    forgetting_review: '遗忘复习',
    weakness: '薄弱补强',
    context_neighbor: '关联推荐',
    graph_driven: '图谱驱动',
  }
  return map[code] || '推荐'
}

function getCourseDotStyle(courseTitle) {
  const palette = getCoursePalette(courseTitle)
  return { background: palette.primary }
}

// ── 练习测验逻辑 ──
async function loadQuestions() {
  if (!knowledgeId.value) return
  questionsLoading.value = true
  try {
    const data = await fetchStudentQuestions({ knowledgeId: knowledgeId.value, pageSize: 10 })
    questions.value = data?.list || []
  } catch {
    questions.value = []
  } finally {
    questionsLoading.value = false
  }
}

function selectOption(questionId, optionKey) {
  if (submittedQuestions.value.has(questionId)) return
  selectedAnswers.value = { ...selectedAnswers.value, [questionId]: optionKey }
}

async function submitAnswer(question) {
  const selected = selectedAnswers.value[question.questionId]
  if (!selected || submittedQuestions.value.has(question.questionId)) return

  try {
    const result = await submitStudentAnswer({
      questionId: question.questionId,
      knowledgeId: question.knowledgeId || knowledgeId.value,
      selectedOption: selected,
      answeredAt: new Date().toISOString(),
    })
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = { ...submitResults.value, [question.questionId]: result }
  } catch {
    // 静默处理
  }
}

// ── AI 对话逻辑 ──
function toggleAi() {
  aiExpanded.value = !aiExpanded.value
}

async function handleAiSend() {
  const text = aiInputText.value.trim()
  if (!text) return
  aiInputText.value = ''
  await aiSend(text)
}

function handleQuickPrompt(prompt) {
  aiInputText.value = ''
  aiSend(prompt)
}

async function loadKnowledgeCard() {
  if (knowledgeCard.value || knowledgeCardLoading.value) return
  knowledgeCardLoading.value = true
  try {
    const data = await generateKnowledgeCard({ knowledgeId: knowledgeId.value })
    knowledgeCard.value = data
  } catch {
    knowledgeCard.value = null
  } finally {
    knowledgeCardLoading.value = false
  }
}

// ── 前沿日期格式化 ──
function formatFrontierDate(value) {
  const normalized = String(value || '').trim()
  if (!normalized) return ''
  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) return normalized
  return parsed.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

// ── 数据加载 ──
watch(
  knowledgeId,
  (value) => {
    draftNote.value = userStore.notes[value] || ''
    selectedAnswers.value = {}
    submittedQuestions.value = new Set()
    submitResults.value = {}
    knowledgeCard.value = null
  },
  { immediate: true },
)

async function loadDetail() {
  await graphStore.ensureNodeDetail(knowledgeId.value)
  frontier.value = await graphStore.ensureFrontier(knowledgeId.value)
  const current = graphStore.detailCache[knowledgeId.value]
  const target = getRecommendationTarget(current)
  const recommendation = await graphStore.ensureRecommendedPath(target, knowledgeId.value, 'detail')
  recommendedPath.value = recommendation?.path || []
}

function saveNote() {
  userStore.saveNote(knowledgeId.value, draftNote.value)
}

watch(
  knowledgeId,
  async () => {
    if (!graphStore.catalogNodes.length) {
      await graphStore.loadCatalog().catch(() => null)
    }
    await loadDetail()
    await loadQuestions()
  },
  { immediate: true },
)

watch(
  () => graphStore.frontierCache[knowledgeId.value],
  (value) => {
    frontier.value = value || null
  },
  { immediate: true },
)

watch(
  detail,
  (value) => {
    if (!value?.routeKnowledgeId) return
    draftNote.value = userStore.notes[value.routeKnowledgeId] || draftNote.value
  },
  { immediate: true },
)

watch(
  aiMessages,
  () => {
    nextTick(() => {
      if (aiMessagesRef.value) {
        aiMessagesRef.value.scrollTop = aiMessagesRef.value.scrollHeight
      }
    })
  },
  { deep: true },
)
</script>

<template>
  <section class="detail-view page-frame">
    <header class="detail-view__head page-card">
      <div>
        <p class="eyebrow">
          <RouterLink to="/" class="inline-link">首页</RouterLink> &gt; 知识详情 &gt; {{ detail?.canonicalName || detail?.name || knowledgeId }}
        </p>
        <h1>{{ detail?.canonicalName || detail?.name || knowledgeId }}</h1>
        <p>{{ getCourseLabel(detail) }} · {{ detail?.entityType || '知识点' }}</p>
      </div>
      <div class="detail-view__actions">
        <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
        <button type="button" class="primary-btn" @click="userStore.toggleBookmark(knowledgeId)">
          {{ userStore.bookmarks.includes(knowledgeId) ? '已收藏' : '收藏' }}
        </button>
      </div>
    </header>

    <div class="detail-view__grid">
      <article class="page-card detail-article">
        <h2>知识点说明</h2>
        <div class="rich-text" v-html="renderWithKatex(detail?.summary || detail?.description || '暂无内容')"></div>

        <div class="detail-meta-list">
          <div>
            <span class="panel-label">前置知识</span>
            <p>{{ (detail?.prerequisites || []).join('、') || '暂无' }}</p>
          </div>
          <div>
            <span class="panel-label">应用场景</span>
            <p>{{ (detail?.scenarios || []).join('、') || '暂无' }}</p>
          </div>
          <div>
            <span class="panel-label">推荐理由</span>
            <p>{{ detail?.recommendReason || '建议结合课程主线继续深入。' }}</p>
          </div>
        </div>

        <!-- 练习测验区域 -->
        <section class="quiz-section">
          <h2>练习测验</h2>
          <p v-if="questionsLoading" class="quiz-section__hint">加载题目中...</p>
          <p v-else-if="!questions.length" class="quiz-section__hint">暂无与此知识点关联的练习题。</p>
          <div v-else class="quiz-list">
            <div v-for="(q, qi) in questions" :key="q.questionId" class="quiz-item">
              <p class="quiz-item__stem">
                <span class="quiz-item__index">{{ qi + 1 }}.</span>
                {{ q.stem }}
                <span v-if="q.difficulty" class="quiz-item__diff">难度 {{ q.difficulty }}</span>
              </p>
              <div class="quiz-item__options">
                <button
                  v-for="opt in q.options"
                  :key="opt.key"
                  type="button"
                  class="quiz-option"
                  :class="{
                    'quiz-option--selected': selectedAnswers[q.questionId] === opt.key,
                    'quiz-option--submitted': submittedQuestions.has(q.questionId),
                  }"
                  @click="selectOption(q.questionId, opt.key)"
                >
                  <span class="quiz-option__key">{{ opt.key }}</span>
                  <span>{{ opt.text }}</span>
                </button>
              </div>
              <div class="quiz-item__footer">
                <button
                  v-if="!submittedQuestions.has(q.questionId)"
                  type="button"
                  class="primary-btn"
                  :disabled="!selectedAnswers[q.questionId]"
                  @click="submitAnswer(q)"
                >
                  提交答案
                </button>
                <span v-else class="quiz-item__submitted">已提交，答案已记录</span>
              </div>
            </div>
            <p class="quiz-section__mastery-hint">答题结果会自动更新你的掌握度分析</p>
          </div>
        </section>

        <!-- 内嵌 AI 对话 -->
        <section class="ai-inline-section">
          <div class="ai-inline-section__header">
            <h2>AI 讲解助手</h2>
            <button type="button" class="secondary-btn" @click="toggleAi">
              {{ aiExpanded ? '收起' : '展开对话' }}
            </button>
          </div>

          <div v-if="!aiExpanded" class="ai-inline-section__preview">
            <p>点击"展开对话"，AI 助手将围绕当前知识点进行讲解、答疑和学习建议。</p>
            <div class="ai-inline-section__quick">
              <button
                v-for="prompt in quickPrompts.slice(0, 3)"
                :key="prompt"
                type="button"
                class="quick-prompt"
                @click="aiExpanded = true; handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>
          </div>

          <div v-else class="ai-inline-chat">
            <div ref="aiMessagesRef" class="ai-inline-chat__messages">
              <div
                v-for="(msg, mi) in aiMessages"
                :key="mi"
                class="ai-inline-msg"
                :class="`ai-inline-msg--${msg.role}`"
              >
                <span v-if="msg.agentLabel" class="ai-inline-msg__agent">{{ msg.agentLabel }}</span>
                <div class="rich-text" v-html="renderWithKatex(msg.content)"></div>
              </div>
              <p v-if="aiSending" class="ai-inline-chat__typing">AI 正在思考...</p>
            </div>

            <div class="ai-inline-chat__quick">
              <button
                v-for="prompt in quickPrompts"
                :key="prompt"
                type="button"
                class="quick-prompt"
                @click="handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>

            <div class="ai-inline-chat__input">
              <input
                v-model="aiInputText"
                type="text"
                placeholder="输入你的问题..."
                @keydown.enter="handleAiSend"
              />
              <button type="button" class="primary-btn" :disabled="aiSending || !aiInputText.trim()" @click="handleAiSend">
                发送
              </button>
            </div>
          </div>
        </section>
      </article>

      <aside class="detail-side">
        <section class="page-card detail-side__card">
          <h3>学习状态</h3>
          <p>掌握度 {{ Math.round(progress.masteryScore || 0) }}%</p>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${Math.round(progress.masteryScore || 0)}%` }"></span>
          </div>
          <button type="button" class="secondary-btn" @click="userStore.markMastered(knowledgeId)">标记为已掌握</button>
        </section>

        <section class="page-card detail-side__card">
          <h3>前沿扩展</h3>
          <div class="frontier-status-row">
            <span class="frontier-status-chip" :class="`frontier-status-chip--${frontier?.status || 'not_ready'}`">
              {{ frontierStatusLabel }}
            </span>
            <span v-if="frontier?.generatedAt" class="muted-text">
              最近更新 {{ formatFrontierDate(frontier.generatedAt) }}
            </span>
          </div>
          <p>{{ frontierSummary }}</p>

          <div v-if="frontierTopics.length" class="frontier-section">
            <span class="panel-label">关联主题</span>
            <div class="path-chain">
              <span v-for="topic in frontierTopics" :key="topic.topicId || topic.name">
                {{ topic.name }}
              </span>
            </div>
          </div>

          <div v-if="frontierPapers.length" class="frontier-section frontier-paper-list">
            <span class="panel-label">代表论文</span>
            <a
              v-for="paper in frontierPapers"
              :key="paper.paperId || paper.title"
              class="frontier-paper-link"
              :href="paper.url || '#'"
              :target="paper.url ? '_blank' : undefined"
              :rel="paper.url ? 'noreferrer' : undefined"
            >
              <strong>{{ paper.title }}</strong>
              <span>{{ [paper.venue, paper.year].filter(Boolean).join(' · ') || '未标注来源' }}</span>
            </a>
          </div>
        </section>

        <!-- 跨课程推荐 -->
        <section v-if="crossCourseItems.length" class="page-card detail-side__card">
          <h3>跨课程知识桥梁</h3>
          <div class="bridge-list">
            <BridgeView
              v-for="item in crossCourseItems"
              :key="item.knowledgeId || item.title"
              :item="{
                sourceCourse: item.sourceCourse,
                targetCourse: item.targetCourse,
                detail: item.reason || item.title,
              }"
            />
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>推荐路径</h3>
          <p v-if="!recommendedPath.length" class="muted-text">暂无推荐路径</p>
          <div v-else class="path-chain path-chain--tagged">
            <span
              v-for="item in regularPathItems"
              :key="item.knowledgeId || item.title"
              class="path-chain__item"
            >
              <span class="path-chain__reason">{{ getReasonLabel(item.reasonCode) }}</span>
              {{ item.title }}
            </span>
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>我的笔记</h3>
          <textarea v-model="draftNote" rows="7" placeholder="写下自己的理解、推导过程和易错点..."></textarea>
          <button type="button" class="primary-btn" @click="saveNote">保存笔记</button>
        </section>
      </aside>
    </div>
  </section>
</template>
