<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchLearningProgress, fetchStudentQuestions, submitStudentAnswer } from '../auth'
import { domains } from '../data/courseData'

const PAGE_SIZE = 6

const selectedKnowledgeId = ref('')
const questionPage = ref(1)
const questionState = ref({
  list: [],
  total: 0,
  page: 1,
  pageSize: PAGE_SIZE,
})
const progressState = ref({
  list: [],
  degraded: false,
})

const questionsLoading = ref(false)
const progressLoading = ref(false)
const questionsError = ref('')
const progressError = ref('')
const submitError = ref('')
const submittingQuestionId = ref('')

const selectedAnswers = ref({})
const answeredThisSession = ref({})
const submissionHints = ref({})
const questionStartedAt = ref({})

const knowledgeOptions = computed(() =>
  domains
    .filter((domain) => domain.id !== 'practice')
    .flatMap((domain) =>
      domain.points.map((point) => ({
        id: point.id,
        name: point.name,
        domainName: domain.name,
        route: point.route,
      })),
    ),
)

const knowledgeOptionMap = computed(() =>
  Object.fromEntries(knowledgeOptions.value.map((item) => [item.id, item])),
)

const selectedKnowledgeMeta = computed(() => knowledgeOptionMap.value[selectedKnowledgeId.value] || null)

const totalAnsweredCount = computed(() =>
  progressState.value.list.reduce((sum, item) => sum + Number(item.answeredCount || 0), 0),
)

const masteredKnowledgeCount = computed(() =>
  progressState.value.list.filter((item) => Number(item.masteryScore || 0) >= 80).length,
)

const progressCompletionRate = computed(() => {
  const list = progressState.value.list
  if (!list.length) return 0
  const total = list.reduce((sum, item) => sum + Number(item.masteryScore || 0), 0)
  return Math.round(total / list.length)
})

const totalPages = computed(() => {
  const total = Number(questionState.value.total || 0)
  const pageSize = Number(questionState.value.pageSize || PAGE_SIZE)
  return Math.max(1, Math.ceil(total / pageSize))
})

function resolveKnowledgeMeta(id, fallbackName = '') {
  const trimmed = String(id || '').trim()
  const local = knowledgeOptionMap.value[trimmed]
  if (local) {
    return local
  }
  return {
    id: trimmed,
    name: fallbackName || trimmed || '未命名知识点',
    domainName: '学习进度',
    route: trimmed ? `/knowledge/${trimmed}` : '/',
  }
}

function formatDifficulty(value) {
  const num = Number(value)
  if (!Number.isFinite(num)) return '未标注'
  if (num < 0.35) return '基础'
  if (num < 0.7) return '进阶'
  return '挑战'
}

function statusLabel(status) {
  if (status === 'solid') return '已稳固'
  if (status === 'progressing') return '学习中'
  return '未开始'
}

function statusClass(status) {
  if (status === 'solid') return 'solid'
  if (status === 'progressing') return 'progressing'
  return 'not-started'
}

function formatAnsweredAt(value) {
  if (!value) return '暂无记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '暂无记录'
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function recordQuestionStart(questions) {
  const now = Date.now()
  const next = { ...questionStartedAt.value }
  for (const question of questions) {
    if (!next[question.questionId]) {
      next[question.questionId] = now
    }
  }
  questionStartedAt.value = next
}

async function loadQuestions() {
  questionsLoading.value = true
  questionsError.value = ''
  try {
    const data = await fetchStudentQuestions({
      knowledgeId: selectedKnowledgeId.value || undefined,
      page: questionPage.value,
      pageSize: PAGE_SIZE,
    })
    questionState.value = {
      list: data.list || [],
      total: data.total || 0,
      page: data.page || questionPage.value,
      pageSize: data.pageSize || PAGE_SIZE,
    }
    recordQuestionStart(questionState.value.list)
  } catch (error) {
    questionsError.value = error.message || '练习题加载失败'
    questionState.value = {
      list: [],
      total: 0,
      page: questionPage.value,
      pageSize: PAGE_SIZE,
    }
  } finally {
    questionsLoading.value = false
  }
}

async function loadProgress() {
  progressLoading.value = true
  progressError.value = ''
  try {
    const data = await fetchLearningProgress()
    progressState.value = {
      list: data.list || [],
      degraded: Boolean(data.degraded),
    }
  } catch (error) {
    progressError.value = error.message || '学习进度加载失败'
    progressState.value = {
      list: [],
      degraded: false,
    }
  } finally {
    progressLoading.value = false
  }
}

async function refreshPracticeData() {
  await Promise.all([loadQuestions(), loadProgress()])
}

async function submitQuestion(question) {
  const selectedOption = selectedAnswers.value[question.questionId]
  if (!selectedOption || submittingQuestionId.value) return

  submittingQuestionId.value = question.questionId
  submitError.value = ''
  submissionHints.value = {
    ...submissionHints.value,
    [question.questionId]: '',
  }

  const startedAt = questionStartedAt.value[question.questionId]
  const now = Date.now()
  const durationMs = Number.isFinite(startedAt) ? Math.max(0, now - startedAt) : undefined

  try {
    const response = await submitStudentAnswer({
      questionId: question.questionId,
      knowledgeId: question.knowledgeId,
      selectedOption,
      durationMs,
      answeredAt: new Date(now).toISOString(),
    })

    answeredThisSession.value = {
      ...answeredThisSession.value,
      [question.questionId]: true,
    }
    submissionHints.value = {
      ...submissionHints.value,
      [question.questionId]: response.degraded
        ? '答案已提交，系统会稍后补齐进度同步。'
        : '答案已提交，学习进度已更新。',
    }
    await loadProgress()
  } catch (error) {
    submitError.value = error.message || '提交答案失败'
  } finally {
    submittingQuestionId.value = ''
  }
}

function changePage(nextPage) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === questionPage.value) {
    return
  }
  questionPage.value = nextPage
}

function selectAnswer(questionId, optionKey) {
  selectedAnswers.value = {
    ...selectedAnswers.value,
    [questionId]: optionKey,
  }
}

watch(selectedKnowledgeId, () => {
  questionPage.value = 1
  loadQuestions()
})

watch(questionPage, () => {
  loadQuestions()
})

onMounted(() => {
  refreshPracticeData()
})
</script>

<template>
  <section class="practice-page">
    <div class="detail-hero practice-hero">
      <span class="section-tag">练习测试</span>
      <h1>学生练习与学习进度</h1>
      <p>围绕当前知识点拉题、提交答案，并把作答结果实时汇入你的学习进度。</p>
    </div>

    <div class="practice-summary-grid">
      <article class="stat-card">
        <span>当前题库</span>
        <strong>{{ questionState.total }}</strong>
      </article>
      <article class="stat-card">
        <span>累计作答</span>
        <strong>{{ totalAnsweredCount }}</strong>
      </article>
      <article class="stat-card">
        <span>已稳固知识点</span>
        <strong>{{ masteredKnowledgeCount }}</strong>
      </article>
      <article class="stat-card">
        <span>平均掌握度</span>
        <strong>{{ progressCompletionRate }}%</strong>
      </article>
    </div>

    <div class="practice-layout">
      <main class="practice-main">
        <div class="practice-toolbar">
          <div>
            <span class="section-tag">智能练习</span>
            <h2>按知识点筛选题目</h2>
          </div>
          <div class="practice-filters">
            <select v-model="selectedKnowledgeId" class="practice-select">
              <option value="">全部知识点</option>
              <option
                v-for="item in knowledgeOptions"
                :key="item.id"
                :value="item.id"
              >
                {{ item.name }} · {{ item.domainName }}
              </option>
            </select>
          </div>
        </div>

        <div v-if="selectedKnowledgeMeta" class="practice-filter-tip">
          当前只显示与
          <RouterLink :to="selectedKnowledgeMeta.route" class="practice-filter-link">
            {{ selectedKnowledgeMeta.name }}
          </RouterLink>
          相关的练习题。
        </div>

        <p v-if="questionsError" class="auth-error">{{ questionsError }}</p>
        <p v-if="submitError" class="auth-error">{{ submitError }}</p>

        <div v-if="questionsLoading" class="practice-empty">正在加载练习题...</div>

        <div v-else-if="questionState.list.length === 0" class="practice-empty">
          当前筛选条件下还没有可练习的题目。
        </div>

        <div v-else class="practice-question-list">
          <article
            v-for="question in questionState.list"
            :key="question.questionId"
            class="practice-question-card"
          >
            <div class="practice-question-head">
              <div class="practice-question-meta">
                <span class="practice-difficulty">{{ formatDifficulty(question.difficulty) }}</span>
                <RouterLink
                  :to="resolveKnowledgeMeta(question.knowledgeId).route"
                  class="practice-knowledge-link"
                >
                  {{ resolveKnowledgeMeta(question.knowledgeId).name }}
                </RouterLink>
              </div>
              <span class="practice-question-id">#{{ question.questionId }}</span>
            </div>

            <h3 class="practice-question-stem">{{ question.stem }}</h3>

            <div class="practice-option-list">
              <label
                v-for="option in question.options"
                :key="`${question.questionId}-${option.key}`"
                class="practice-option"
                :class="{ active: selectedAnswers[question.questionId] === option.key }"
              >
                <input
                  :name="question.questionId"
                  type="radio"
                  :value="option.key"
                  :checked="selectedAnswers[question.questionId] === option.key"
                  :disabled="Boolean(answeredThisSession[question.questionId])"
                  @change="selectAnswer(question.questionId, option.key)"
                />
                <span class="practice-option-key">{{ option.key }}</span>
                <span>{{ option.text }}</span>
              </label>
            </div>

            <div class="practice-question-actions">
              <button
                class="auth-submit practice-submit-btn"
                :disabled="!selectedAnswers[question.questionId] || Boolean(answeredThisSession[question.questionId]) || submittingQuestionId === question.questionId"
                @click="submitQuestion(question)"
              >
                {{
                  answeredThisSession[question.questionId]
                    ? '已提交'
                    : submittingQuestionId === question.questionId
                      ? '提交中...'
                      : '提交答案'
                }}
              </button>
              <p v-if="submissionHints[question.questionId]" class="practice-submit-hint">
                {{ submissionHints[question.questionId] }}
              </p>
            </div>
          </article>
        </div>

        <div v-if="questionState.list.length > 0" class="practice-pagination">
          <button class="header-logout" :disabled="questionPage <= 1 || questionsLoading" @click="changePage(questionPage - 1)">
            上一页
          </button>
          <span>第 {{ questionPage }} / {{ totalPages }} 页</span>
          <button class="header-logout" :disabled="questionPage >= totalPages || questionsLoading" @click="changePage(questionPage + 1)">
            下一页
          </button>
        </div>
      </main>

      <aside class="practice-side">
        <div class="detail-side-card practice-side-card">
          <span class="section-tag">学习进度</span>
          <h3>知识点掌握快照</h3>
          <p v-if="progressState.degraded" class="practice-side-note">
            当前进度为降级视图，部分实时数据可能稍后同步。
          </p>
          <p v-if="progressError" class="auth-error">{{ progressError }}</p>
          <p v-else-if="progressLoading" class="practice-empty practice-empty--side">正在同步进度...</p>
          <p v-else-if="progressState.list.length === 0" class="practice-empty practice-empty--side">
            你还没有作答记录，先从左侧题目开始练习。
          </p>

          <div v-else class="practice-progress-list">
            <article
              v-for="item in progressState.list.slice(0, 8)"
              :key="item.knowledgeId"
              class="practice-progress-item"
            >
              <div class="practice-progress-top">
                <div>
                  <RouterLink :to="resolveKnowledgeMeta(item.knowledgeId, item.canonicalName).route" class="practice-progress-link">
                    {{ resolveKnowledgeMeta(item.knowledgeId, item.canonicalName).name }}
                  </RouterLink>
                  <p>{{ resolveKnowledgeMeta(item.knowledgeId, item.canonicalName).domainName }}</p>
                </div>
                <span :class="['practice-status-pill', statusClass(item.status)]">
                  {{ statusLabel(item.status) }}
                </span>
              </div>

              <div class="point-bar-row">
                <div class="point-bar">
                  <div class="point-bar-fill" :style="{ width: `${Math.max(0, Math.min(100, Number(item.masteryScore || 0)))}%` }"></div>
                </div>
                <span class="point-pct">{{ Math.round(Number(item.masteryScore || 0)) }}%</span>
              </div>

              <div class="practice-progress-meta">
                <span>作答 {{ item.answeredCount || 0 }} 次</span>
                <span>正确率 {{ Math.round(Number(item.correctRate || 0) * 100) }}%</span>
              </div>
              <p class="practice-progress-time">最近作答：{{ formatAnsweredAt(item.lastAnsweredAt) }}</p>
            </article>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
