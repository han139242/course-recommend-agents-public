<script setup>
import { Bot, CheckCircle2, ChevronRight, NotebookPen, Send, Star } from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import 'katex/dist/katex.min.css'

import { fetchStudentQuestions, submitStudentAnswer } from '../api/learning'
import ScoreBreakdown from '../components/ScoreBreakdown.vue'
import { useAiChat } from '../composables/useAiChat'
import { renderWithKatex } from '../composables/useKatex'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../utils/courseMeta'

const route = useRoute()
const graphStore = useGraphStore()
const userStore = useUserStore()

const recommendedPath = ref([])
const draftNote = ref('')
const questions = ref([])
const questionsLoading = ref(false)
const selectedAnswers = ref({})
const submittedQuestions = ref(new Set())
const submitResults = ref({})
const aiExpanded = ref(false)
const aiInputText = ref('')
const aiMessagesRef = ref(null)
const frontier = ref(null)
const frontierLoading = ref(false)

const knowledgeId = computed(() => decodeURIComponent(route.params.knowledgeId || ''))
const detail = computed(() => graphStore.detailCache[knowledgeId.value] || null)
const selectedDetail = computed(() => detail.value)
const progress = computed(() => userStore.getProgressMeta(knowledgeId.value))
const isAuthenticated = computed(() => userStore.isAuthenticated)

const detailTitle = computed(() => String(detail.value?.canonicalName || detail.value?.name || knowledgeId.value || '当前知识点').trim())
const detailCourseLabel = computed(() => resolveNodeCourseTitle(detail.value || {}, DEFAULT_COURSE_TITLE))
const detailTypeLabel = computed(() => {
  const typeMap = {
    concept: '概念',
    method: '方法',
    algorithm: '算法',
    application: '应用',
    model: '模型',
    tool: '工具',
    framework: '框架',
    topic: '主题',
  }
  return typeMap[detail.value?.entityType] || '知识点'
})

const masteryPercent = computed(() => Math.round(progress.value?.masteryScore || 0))
const masteryLabel = computed(() => {
  if (masteryPercent.value >= 80) return '已掌握'
  if (masteryPercent.value >= 40) return '学习中'
  if (Number(progress.value?.answeredCount || 0) > 0) return '待巩固'
  return '未开始'
})

const { messages: aiMessages, sending: aiSending, quickPrompts, send: aiSend } = useAiChat(selectedDetail, isAuthenticated)

function cleanText(value, fallback = '') {
  const normalized = String(value || '').replace(/\s+/g, ' ').trim()
  return normalized || fallback
}

function cleanList(values) {
  return Array.isArray(values) ? values.map((item) => cleanText(item)).filter(Boolean) : []
}

function buildPrerequisiteItems(node) {
  const source = cleanList(node?.prerequisites)
  if (source.length) return source.slice(0, 5)
  return [`${detailCourseLabel.value}基础概念`, `${detailTitle.value}的关键定义`]
}

function buildScenarioItems(node) {
  const source = cleanList(node?.scenarios)
  if (source.length) return source.slice(0, 5)
  return ['理解后续知识点的前置基础', '结合练习题完成概念到应用的迁移']
}

function buildRecommendReason(node) {
  return cleanText(
    node?.recommendReason,
    `建议先理解${buildPrerequisiteItems(node).slice(0, 2).join('、')}，再通过练习题确认掌握情况。`,
  )
}

const detailSummary = computed(() =>
  cleanText(
    detail.value?.summary || detail.value?.description,
    `${detailTitle.value}是${detailCourseLabel.value}中的重要${detailTypeLabel.value}，建议结合前置知识和典型练习逐步理解。`,
  ),
)

const prerequisiteItems = computed(() => buildPrerequisiteItems(detail.value))
const scenarioItems = computed(() => buildScenarioItems(detail.value))
const recommendReasonText = computed(() => buildRecommendReason(detail.value))

const regularPathItems = computed(() => recommendedPath.value.slice(0, 5))

function getCourseLabel(node) {
  return resolveNodeCourseTitle(node || {}, DEFAULT_COURSE_TITLE)
}

function summarizeSentence(value, fallback) {
  const text = cleanText(value, fallback)
  if (!text) return ''
  const segments = text.split(/[。！？!?；;]/).map((item) => item.trim()).filter(Boolean)
  return `${segments[0] || text}。`
}

function isPresentableQuestion(item) {
  const stem = cleanText(item?.stem)
  const options = Array.isArray(item?.options) ? item.options : []
  if (stem.length < 10) return false
  if (options.length < 3) return false
  return true
}

function normalizeLiveQuestions(items = []) {
  return (Array.isArray(items) ? items : [])
    .filter((item) => isPresentableQuestion(item))
    .map((item, index) => ({
      questionId: cleanText(item.questionId, `live-${knowledgeId.value}-${index}`),
      knowledgeId: cleanText(item.knowledgeId, knowledgeId.value),
      type: cleanText(item.type, 'single_choice'),
      stem: cleanText(item.stem),
      options: Array.isArray(item.options)
        ? item.options.map((option) => ({ key: cleanText(option?.key), text: cleanText(option?.text) })).filter((option) => option.key && option.text)
        : [],
      difficulty: Number(item.difficulty || 0),
      categoryLabel: ['核心理解', '关键辨析', '场景应用'][index % 3],
      focusLabel: [detailTitle.value, ...prerequisiteItems.value].filter(Boolean)[index % Math.max(1, [detailTitle.value, ...prerequisiteItems.value].filter(Boolean).length)],
      mode: 'live',
    }))
    .filter((item) => item.stem && item.options.length >= 2)
}

function buildShowcaseQuestions(node) {
  const title = cleanText(node?.canonicalName || node?.name, detailTitle.value)
  const course = getCourseLabel(node)
  const summary = summarizeSentence(
    node?.summary || node?.description,
    `${title}用于帮助学习者理解该知识点在 ${course} 中的核心作用，并支撑后续相关内容学习。`,
  )
  const prerequisiteText = buildPrerequisiteItems(node).slice(0, 2).join('、')
  const scenarioText = buildScenarioItems(node)[0]
  const recommendation = buildRecommendReason(node)

  return [
    {
      questionId: `showcase-${knowledgeId.value}-core`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `关于”${title}”，下列哪项最能概括它的核心含义？`,
      options: [
        { key: 'A', text: summary.replace(/。$/, '') },
        { key: 'B', text: `${title}主要用于替代全部前置知识的学习，不需要理解其内部逻辑。` },
        { key: 'C', text: `${title}只是一个孤立术语，与 ${course} 中其它知识点基本没有关系。` },
        { key: 'D', text: `${title}更适合作为界面展示名词记忆，无需结合应用场景理解。` },
      ],
      correctOption: 'A',
      explanation: `知识点说明强调：${summary} 因此应先把握它在当前课程体系中的核心定位。`,
      difficulty: 0.35,
      categoryLabel: '核心理解',
      focusLabel: title,
      mode: 'showcase',
    },
    {
      questionId: `showcase-${knowledgeId.value}-prerequisite`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `如果你准备继续深入学习”${title}”，下列哪组内容最值得优先补齐？`,
      options: [
        { key: 'A', text: prerequisiteText || `${course}基础概念与关键定义` },
        { key: 'B', text: '直接跳过先修知识，只记住结论与术语名称即可。' },
        { key: 'C', text: '优先学习与当前知识点关联较弱的外围话题，再回头补基础。' },
        { key: 'D', text: '只关注练习结果，不分析知识点之间的关系链。' },
      ],
      correctOption: 'A',
      explanation: `从学习衔接角度看，优先补齐 ${prerequisiteText || `${course}基础概念`}，更容易建立稳定理解。`,
      difficulty: 0.52,
      categoryLabel: '先修衔接',
      focusLabel: prerequisiteText || `${course}基础`,
      mode: 'showcase',
    },
    {
      questionId: `showcase-${knowledgeId.value}-scenario`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `在下面的学习或应用场景中，最适合引入”${title}”的是哪一项？`,
      options: [
        { key: 'A', text: scenarioText },
        { key: 'B', text: '完全不结合上下文，只把它当作独立名词做机械记忆。' },
        { key: 'C', text: '先忽略当前知识点，再把所有后续内容混在一起一次性学习。' },
        { key: 'D', text: `只围绕”是否收藏”做判断，而不考虑 ${title} 的学习价值。` },
      ],
      correctOption: 'A',
      explanation: `当前知识点更适合出现在”${scenarioText}”这类场景中，推荐理由也指出：${recommendation}`,
      difficulty: 0.6,
      categoryLabel: '场景应用',
      focusLabel: '典型应用',
      mode: 'showcase',
    },
  ]
}

const liveQuestions = computed(() => normalizeLiveQuestions(questions.value))
const showcaseQuestions = computed(() => buildShowcaseQuestions(detail.value))
const practiceQuestions = computed(() => {
  const merged = [...liveQuestions.value]
  const stemSet = new Set(merged.map((item) => item.stem))
  showcaseQuestions.value.forEach((item) => {
    if (merged.length >= 3) return
    if (stemSet.has(item.stem)) return
    merged.push(item)
  })
  return merged.slice(0, 3)
})

const usesShowcaseQuiz = computed(() => liveQuestions.value.length === 0)

function getQuestionResult(questionId) {
  return submitResults.value[questionId] || null
}

function getQuestionStatusText(question) {
  const result = getQuestionResult(question.questionId)
  if (!result) return ''
  if (question.mode === 'showcase') return result.correct ? '回答正确' : '建议复盘'
  return '已提交，结果已记录'
}

function selectOption(questionId, optionKey) {
  if (submittedQuestions.value.has(questionId)) return
  selectedAnswers.value = { ...selectedAnswers.value, [questionId]: optionKey }
}

async function submitAnswer(question) {
  const selected = selectedAnswers.value[question.questionId]
  if (!selected || submittedQuestions.value.has(question.questionId)) return

  if (question.mode === 'showcase') {
    const correct = selected === question.correctOption
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = { ...submitResults.value, [question.questionId]: { accepted: true, correct, explanation: question.explanation } }
    return
  }

  try {
    const result = await submitStudentAnswer({ questionId: question.questionId, knowledgeId: question.knowledgeId || knowledgeId.value, selectedOption: selected, answeredAt: new Date().toISOString() })
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = { ...submitResults.value, [question.questionId]: result }
  } catch {
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = { ...submitResults.value, [question.questionId]: { accepted: true } }
  }
}

function isShowcaseRevealVisible(question) {
  return question.mode === 'showcase' && submittedQuestions.value.has(question.questionId)
}

function isShowcaseCorrectOption(question, optionKey) {
  return question.mode === 'showcase' && submittedQuestions.value.has(question.questionId) && question.correctOption === optionKey
}

function isShowcaseIncorrectOption(question, optionKey) {
  return question.mode === 'showcase' && submittedQuestions.value.has(question.questionId) && selectedAnswers.value[question.questionId] === optionKey && question.correctOption !== optionKey
}

function isCorrect(question, optionKey) {
  if (!submittedQuestions.value.has(question.questionId)) return false
  if (question.mode === 'showcase') return question.correctOption === optionKey
  const result = getQuestionResult(question.questionId)
  return result?.correctAnswer === optionKey
}

function isIncorrect(question, optionKey) {
  if (!submittedQuestions.value.has(question.questionId)) return false
  if (question.mode === 'showcase') return selectedAnswers.value[question.questionId] === optionKey && question.correctOption !== optionKey
  const result = getQuestionResult(question.questionId)
  return selectedAnswers.value[question.questionId] === optionKey && result?.correctAnswer !== optionKey
}

function formatDifficultyLabel(difficulty) {
  const d = Number(difficulty)
  if (d <= 0.4) return '基础'
  if (d <= 0.6) return '中等'
  return '进阶'
}

function handleQuickPrompt(prompt) {
  aiExpanded.value = true
  aiSend(prompt)
}

function handleAiSend() {
  const content = aiInputText.value.trim()
  if (!content) return
  aiSend(content)
  aiInputText.value = ''
}

async function loadQuestions() {
  if (!knowledgeId.value) return
  questionsLoading.value = true
  try {
    const data = await fetchStudentQuestions({ knowledgeId: knowledgeId.value, pageSize: 6 })
    questions.value = data?.list || data?.items || []
  } catch {
    questions.value = []
  } finally {
    questionsLoading.value = false
  }
}

async function loadFrontier() {
  if (!knowledgeId.value) return
  frontierLoading.value = true
  frontier.value = await graphStore.ensureFrontier(knowledgeId.value)
  frontierLoading.value = false
}

async function loadDetail() {
  await graphStore.ensureNodeDetail(knowledgeId.value)
  const current = graphStore.detailCache[knowledgeId.value]
  const target = cleanText(current?.canonicalName || current?.name, detailTitle.value)
  const recommendation = await graphStore.ensureRecommendedPath(target, knowledgeId.value, 'detail')
  recommendedPath.value = recommendation?.path || []
}

function saveNote() {
  userStore.saveNote(knowledgeId.value, draftNote.value)
}

watch(
  knowledgeId,
  (value) => {
    draftNote.value = userStore.notes[value] || ''
    selectedAnswers.value = {}
    submittedQuestions.value = new Set()
    submitResults.value = {}
  },
  { immediate: true },
)

watch(
  knowledgeId,
  async () => {
    if (!graphStore.catalogNodes.length) await graphStore.loadCatalog().catch(() => null)
    await userStore.loadProgress().catch(() => null)
    await loadDetail()
    await loadQuestions()
    await loadFrontier()
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
      if (aiMessagesRef.value) aiMessagesRef.value.scrollTop = aiMessagesRef.value.scrollHeight
    })
  },
  { deep: true },
)
</script>

<template>
  <section class="detail-view page-frame">
    <header class="detail-view__head page-card">
      <div>
        <p class="eyebrow detail-breadcrumb">
          <RouterLink to="/" class="inline-link">知识地图</RouterLink>
          <ChevronRight :size="14" aria-hidden="true" />
          <span>{{ detailCourseLabel }}</span>
        </p>
        <h1>{{ detailTitle }}</h1>
        <p>{{ detailTypeLabel }} · 掌握度 {{ masteryPercent }}%</p>
      </div>
      <div class="detail-view__actions">
        <span class="status-pill" :class="`status-pill--${masteryLabel === '已掌握' ? 'mastered' : masteryLabel === '待巩固' ? 'weak' : masteryLabel === '学习中' ? 'learning' : 'idle'}`">
          {{ masteryLabel }}
        </span>
        <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
        <button type="button" class="primary-btn" @click="userStore.toggleBookmark(knowledgeId)">
          <Star :size="16" aria-hidden="true" />
          {{ userStore.bookmarks.includes(knowledgeId) ? '已收藏' : '收藏' }}
        </button>
      </div>
    </header>

    <div class="detail-view__grid">
      <article class="page-card detail-article">
        <section class="detail-section-block">
          <div class="detail-section-block__head">
            <h2>知识点说明</h2>
            <span class="detail-section-block__tag">{{ detailCourseLabel }}</span>
          </div>
          <div class="rich-text detail-summary-copy" v-html="renderWithKatex(detailSummary)"></div>

          <div class="detail-meta-list">
            <div>
              <span class="panel-label">前置知识</span>
              <div class="detail-pill-list">
                <span v-for="item in prerequisiteItems" :key="item" class="detail-pill">{{ item }}</span>
              </div>
            </div>
            <div>
              <span class="panel-label">典型场景</span>
              <div class="detail-pill-list">
                <span v-for="item in scenarioItems" :key="item" class="detail-pill detail-pill--soft">{{ item }}</span>
              </div>
            </div>
            <div>
              <span class="panel-label">学习建议</span>
              <p>{{ recommendReasonText }}</p>
            </div>
          </div>
        </section>

        <section class="quiz-section">
          <div class="quiz-section__head">
            <div>
              <h2>练习测验</h2>
              <p class="quiz-section__intro">先用短题确认理解，再决定是否进入下一步。</p>
            </div>
            <div class="quiz-section__legend">
              <span>核心理解</span>
              <span>前置衔接</span>
              <span>应用判断</span>
            </div>
          </div>

          <p v-if="questionsLoading && !practiceQuestions.length" class="quiz-section__hint">正在加载练习内容...</p>

          <div v-else class="quiz-list">
            <article v-for="question in practiceQuestions" :key="question.questionId" class="quiz-item">
              <div class="quiz-item__head">
                <div class="quiz-item__meta">
                  <span class="quiz-type-chip">{{ question.categoryLabel }}</span>
                  <span class="quiz-focus-chip">{{ question.focusLabel }}</span>
                </div>
                <span class="quiz-item__diff">{{ formatDifficultyLabel(question.difficulty) }}</span>
              </div>

              <p class="quiz-item__stem">{{ question.stem }}</p>

              <div class="quiz-item__options">
                <button
                  v-for="option in question.options"
                  :key="option.key"
                  type="button"
                  class="quiz-option"
                  :class="{
                    'quiz-option--selected': selectedAnswers[question.questionId] === option.key,
                    'quiz-option--submitted': submittedQuestions.has(question.questionId),
                    'quiz-option--correct': isCorrect(question, option.key),
                    'quiz-option--incorrect': isIncorrect(question, option.key),
                  }"
                  @click="selectOption(question.questionId, option.key)"
                >
                  <span class="quiz-option__key">{{ option.key }}</span>
                  <span>{{ option.text }}</span>
                </button>
              </div>

              <div class="quiz-item__footer">
                <button
                  v-if="!submittedQuestions.has(question.questionId)"
                  type="button"
                  class="primary-btn"
                  :disabled="!selectedAnswers[question.questionId]"
                  @click="submitAnswer(question)"
                >
                  提交答案
                </button>
                <span v-else class="quiz-item__submitted" :class="{ 'quiz-item__submitted--correct': getQuestionResult(question.questionId)?.correct }">
                  {{ getQuestionStatusText(question) }}
                </span>
              </div>

              <div v-if="submittedQuestions.has(question.questionId)" class="quiz-item__explanation">
                <span class="panel-label">解析</span>
                <p>{{ getQuestionResult(question.questionId)?.explanation || question.explanation }}</p>
              </div>
            </article>
          </div>
        </section>

        <section class="ai-inline-section">
          <div class="ai-inline-section__header">
            <div>
              <h2>AI 学习助教</h2>
              <p>围绕当前知识点提问，适合用来补定义、问例子、查下一步。</p>
            </div>
            <button type="button" class="secondary-btn" @click="aiExpanded = !aiExpanded">
              <Bot :size="16" aria-hidden="true" />
              {{ aiExpanded ? '收起' : '展开对话' }}
            </button>
          </div>

          <div v-if="!aiExpanded" class="ai-inline-section__preview">
            <div class="ai-inline-section__quick">
              <button
                v-for="prompt in quickPrompts.slice(0, 3)"
                :key="prompt"
                type="button"
                class="quick-prompt"
                @click="handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>
          </div>

          <div v-else class="ai-inline-chat">
            <div ref="aiMessagesRef" class="ai-inline-chat__messages">
              <article
                v-for="(message, index) in aiMessages"
                :key="index"
                class="ai-inline-msg"
                :class="`ai-inline-msg--${message.role}`"
              >
                <span v-if="message.agentLabel" class="ai-inline-msg__agent">{{ message.agentLabel }}</span>
                <div class="rich-text" v-html="renderWithKatex(message.content)"></div>
              </article>
              <p v-if="aiSending" class="ai-inline-chat__typing">正在生成讲解...</p>
            </div>

            <div class="ai-inline-chat__input">
              <input v-model="aiInputText" type="text" placeholder="输入你想追问的问题" @keydown.enter="handleAiSend" />
              <button type="button" class="primary-btn" :disabled="aiSending || !aiInputText.trim()" @click="handleAiSend">
                <Send :size="16" aria-hidden="true" />
                发送
              </button>
            </div>
          </div>
        </section>

        <section class="frontier-expand-section">
          <div class="frontier-expand-section__header">
            <div>
              <h2>前沿扩展</h2>
              <p>基于学术前沿数据，了解与本知识点相关的最新研究方向与代表论文。</p>
            </div>
            <span v-if="frontier?.status" class="frontier-status-chip" :class="`frontier-status-chip--${frontier.status}`">
              {{ frontier.status === 'ready' ? '已收录' : frontier.status === 'stale' ? '待更新' : '收录中' }}
            </span>
          </div>

          <p v-if="frontierLoading" class="quiz-section__hint">正在加载前沿内容...</p>

          <div v-else-if="frontier && frontier.digestSummary" class="frontier-section">
            <blockquote class="frontier-summary-card">
              <p>{{ frontier.digestSummary }}</p>
            </blockquote>

            <div v-if="frontier.topics?.length" class="frontier-topic-grid">
              <article v-for="topic in frontier.topics" :key="topic.topicId" class="frontier-topic-card">
                <strong>{{ topic.name }}</strong>
                <p>{{ topic.reason }}</p>
              </article>
            </div>

            <div v-if="frontier.papers?.length" class="frontier-paper-list">
              <a
                v-for="paper in frontier.papers"
                :key="paper.paperId"
                class="frontier-paper-link"
                :href="paper.url"
                target="_blank"
                rel="noopener noreferrer"
              >
                <div class="frontier-paper-link__body">
                  <strong>{{ paper.title }}</strong>
                  <span>{{ paper.venue }}{{ paper.year ? ` · ${paper.year}` : '' }}</span>
                </div>
                <span class="frontier-paper-link__tag">查看原文</span>
              </a>
            </div>
          </div>

          <p v-else-if="!frontierLoading" class="frontier-support-text">
            当前知识点暂未收录前沿扩展内容，系统会随着前沿论文数据更新逐步补充。
          </p>
        </section>
      </article>

      <aside class="detail-side">
        <section class="page-card detail-side__card">
          <h3>学习状态</h3>
          <p>掌握度 {{ masteryPercent }}%</p>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${masteryPercent}%` }"></span>
          </div>
          <button type="button" class="secondary-btn" @click="userStore.markMastered(knowledgeId)">
            <CheckCircle2 :size="16" aria-hidden="true" />
            标记为已掌握
          </button>
        </section>

        <section class="page-card detail-side__card">
          <h3>推荐路径</h3>
          <p v-if="!regularPathItems.length" class="muted-text">完成本页练习后，系统会继续调整推荐路径。</p>
          <div v-else class="path-score-list">
            <article v-for="(item, idx) in regularPathItems" :key="item.knowledgeId || idx" class="path-score-item">
              <RouterLink class="path-score-item__header" :to="`/knowledge/${encodeURIComponent(item.knowledgeId || item.title)}`">
                <span class="path-score-item__rank">{{ idx + 1 }}</span>
                <strong>{{ item.title || item.knowledgeId }}</strong>
                <span class="path-score-item__scoresum">{{ Math.round((item.score || 0) * 100) }}分</span>
              </RouterLink>
              <ScoreBreakdown v-if="item.scoringBreakdown" :breakdown="item.scoringBreakdown" />
            </article>
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>我的笔记</h3>
          <textarea v-model="draftNote" rows="7" placeholder="记录当前知识点的理解、易错点和后续计划"></textarea>
          <button type="button" class="primary-btn" @click="saveNote">
            <NotebookPen :size="16" aria-hidden="true" />
            保存笔记
          </button>
        </section>
      </aside>
    </div>
  </section>
</template>
