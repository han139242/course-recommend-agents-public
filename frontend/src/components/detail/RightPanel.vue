<script setup>
import { Bot, Bookmark, BookmarkCheck, ChevronLeft, ChevronRight, ExternalLink, MousePointerClick } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import BridgeView from './BridgeView.vue'
import { useGraphStore } from '../../stores/graphStore'
import { useUserStore } from '../../stores/userStore'
import { getEntityLabel } from '../../utils/colorMap'
import { DEFAULT_COURSE_TITLE, normalizeCourseTitle } from '../../utils/courseMeta'
import { getQuizForNode } from '../../data/quizBank'

const props = defineProps({
  collapsed: Boolean,
  detail: Object,
  progress: Object,
  nextNodes: {
    type: Array,
    default: () => [],
  },
  recommendedPath: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits([
  'toggle',
  'select-related',
  'open-ai',
  'show-recommend-path',
  'open-ai-with-prompt',
  'quiz-mastered',
  'quiz-failed'
])
const userStore = useUserStore()
const graphStore = useGraphStore()
const draftNote = ref('')

// Quiz reactive state
const quizState = ref('idle')
const quizCurrentIndex = ref(0)
const quizSelectedOption = ref(null)
const quizAnswers = ref([])
const quizScore = ref(0)

const quizQuestions = computed(() => {
  if (!props.detail) return []
  return getQuizForNode(props.detail.canonicalName || props.detail.name || props.detail.routeKnowledgeId)
})

const currentQuestion = computed(() => {
  return quizQuestions.value[quizCurrentIndex.value] || null
})

function startQuiz() {
  quizState.value = 'active'
  quizCurrentIndex.value = 0
  quizSelectedOption.value = null
  quizAnswers.value = []
  quizScore.value = 0
}

function selectOption(index) {
  quizSelectedOption.value = index
}

function nextQuestion() {
  if (quizSelectedOption.value === null) return
  const correct = quizSelectedOption.value === currentQuestion.value.answer
  quizAnswers.value.push(correct)
  if (correct) {
    quizScore.value++
  }
  quizCurrentIndex.value++
  quizSelectedOption.value = null
}

function submitQuiz() {
  if (quizSelectedOption.value === null) return
  const correct = quizSelectedOption.value === currentQuestion.value.answer
  quizAnswers.value.push(correct)
  if (correct) {
    quizScore.value++
  }

  const passed = quizScore.value === quizQuestions.value.length
  if (passed) {
    quizState.value = 'success'
    if (props.detail?.routeKnowledgeId) {
      userStore.markMastered(props.detail.routeKnowledgeId)
    }
    emit('quiz-mastered', {
      nodeId: props.detail.routeKnowledgeId,
      name: props.detail.canonicalName || props.detail.name
    })
  } else {
    quizState.value = 'failed'
    emit('quiz-failed', {
      nodeId: props.detail.routeKnowledgeId,
      name: props.detail.canonicalName || props.detail.name,
      score: quizScore.value,
      total: quizQuestions.value.length
    })
  }
}

function resetQuiz() {
  quizState.value = 'idle'
  quizCurrentIndex.value = 0
  quizSelectedOption.value = null
  quizAnswers.value = []
  quizScore.value = 0
}

const contextChips = computed(() => {
  const name = props.detail?.canonicalName || props.detail?.name || ''
  if (!name) return []
  return [
    `请用通俗的大白话解释一下“${name}”是什么意思？`,
    `“${name}”在实际工业界或科研项目里有什么经典的应用场景？`,
    `要学好“${name}”，我需要掌握哪些核心的前提基础知识？`
  ]
})

watch(
  () => props.detail?.routeKnowledgeId,
  (routeKnowledgeId) => {
    draftNote.value = userStore.notes[routeKnowledgeId] || ''
    resetQuiz()
  },
  { immediate: true },
)

const masteryPercent = computed(() =>
  Math.round(props.progress?.masteryScore || props.detail?.masteryScore || 0),
)

const statusText = computed(() => {
  if (masteryPercent.value >= 80) return '已掌握'
  if (masteryPercent.value >= 40) return '学习中'
  if (Number(props.progress?.answeredCount || 0) > 0) return '待巩固'
  return '未开始'
})

function getStatusMeta(routeKnowledgeId) {
  const progress = userStore.getProgressMeta(routeKnowledgeId)
  const mastery = Number(progress.masteryScore || 0)
  const answeredCount = Number(progress.answeredCount || 0)
  if (mastery >= 80) return '已掌握'
  if (mastery < 30 && answeredCount > 0) return '待巩固'
  if (mastery >= 40) return '学习中'
  return '未开始'
}

const crossCourseRecommends = computed(() => {
  const currentCourse = normalizeCourseTitle(props.detail?.chapterTitle, DEFAULT_COURSE_TITLE)
  return (props.recommendedPath || [])
    .map((item) => {
      const node = graphStore.routeNodeMap.get(item.knowledgeId || item.title)
      const displayTitle =
        String(node?.canonicalName || node?.name || item.title || item.knowledgeId || '').trim()
      const sourceCourse = normalizeCourseTitle(item.sourceCourse || currentCourse, DEFAULT_COURSE_TITLE)
      const targetCourse = normalizeCourseTitle(item.targetCourse || node?.chapterTitle, DEFAULT_COURSE_TITLE)
      return {
        ...item,
        displayTitle,
        sourceCourse,
        targetCourse,
        courseLabel: item.reasonCode === 'cross_course_transfer'
          ? `${sourceCourse} -> ${targetCourse}`
          : targetCourse,
      }
    })
    .filter((item) => {
      if (!item?.title && !item?.knowledgeId) return false
      if (item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse) {
        return item.sourceCourse !== item.targetCourse
      }
      return Boolean(item.targetCourse && item.targetCourse !== currentCourse)
    })
    .slice(0, 3)
})

const pathRouteIds = computed(() =>
  (props.recommendedPath || []).map((item) => item.knowledgeId || item.title).filter(Boolean),
)

function saveNote() {
  if (!props.detail?.routeKnowledgeId) return
  userStore.saveNote(props.detail.routeKnowledgeId, draftNote.value)
}
</script>

<template>
  <aside class="right-panel" :class="{ 'right-panel--collapsed': collapsed }">
    <div class="panel-header">
      <div class="panel-header__title">
        <Bot :size="17" aria-hidden="true" />
        <strong>学习提示</strong>
      </div>
      <button type="button" class="panel-toggle" :title="collapsed ? '展开详情' : '收起详情'" @click="emit('toggle')">
        <ChevronLeft v-if="collapsed" :size="16" aria-hidden="true" />
        <ChevronRight v-else :size="16" aria-hidden="true" />
      </button>
    </div>

    <div v-if="!collapsed" class="panel-stack">
      <div v-if="detail" class="panel-card detail-card">
        <div class="detail-title-row">
          <div>
            <span class="status-pill" :class="`status-pill--${statusText === '已掌握' ? 'mastered' : statusText === '待巩固' ? 'weak' : statusText === '学习中' ? 'learning' : 'idle'}`">{{ statusText }}</span>
            <h2>{{ detail.canonicalName || detail.name }}</h2>
            <p>{{ detail.chapterTitle || '其他课程' }} · {{ getEntityLabel(detail.entityType) }}</p>
          </div>
          <button type="button" class="icon-btn" :title="userStore.bookmarks.includes(detail.routeKnowledgeId) ? '取消收藏' : '收藏'" @click="userStore.toggleBookmark(detail.routeKnowledgeId)">
            <BookmarkCheck v-if="userStore.bookmarks.includes(detail.routeKnowledgeId)" :size="18" aria-hidden="true" />
            <Bookmark v-else :size="18" aria-hidden="true" />
          </button>
        </div>

        <div class="detail-progress">
          <span>掌握度 {{ masteryPercent }}%</span>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${masteryPercent}%` }"></span>
          </div>
        </div>

        <p class="detail-copy">
          {{ detail.description || detail.summary || '当前知识点还没有补充说明，可以先结合前置知识和练习题建立基本理解。' }}
        </p>

        <div class="detail-section">
          <span class="panel-label">前置知识</span>
          <button
            v-for="item in detail.prerequisites || []"
            :key="item"
            type="button"
            class="inline-link"
            @click="emit('select-related', item)"
          >
            <span>{{ getStatusMeta(item) }}</span>
            {{ item }}
          </button>
          <span v-if="!(detail.prerequisites || []).length" class="muted-text">暂无明确前置知识。</span>
        </div>

        <div class="detail-section">
          <span class="panel-label">后续可学</span>
          <button
            v-for="item in nextNodes"
            :key="item.routeKnowledgeId || item"
            type="button"
            class="inline-link"
            @click="emit('select-related', item.routeKnowledgeId || item)"
          >
            <span>{{ getStatusMeta(item.routeKnowledgeId || item) }}</span>
            {{ item.label || item }}
          </button>
          <span v-if="!nextNodes.length" class="muted-text">暂无明确后续节点。</span>
        </div>

        <div v-if="crossCourseRecommends.length" class="detail-section recommend-section">
          <span class="panel-label">跨课程衔接</span>
          <article class="recommend-card" v-for="item in crossCourseRecommends" :key="item.knowledgeId || item.title">
            <div class="recommend-card__head">
              <span class="recommend-card__badge">{{ item.reasonLabel || '推荐' }}</span>
              <span class="recommend-card__course">{{ item.courseLabel || item.targetCourse || '当前课程' }}</span>
            </div>
            <BridgeView
              v-if="item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse"
              :item="item"
            />
            <button type="button" class="recommend-card__title" @click="emit('select-related', item.knowledgeId || item.title)">
              {{ item.displayTitle || item.title }}
            </button>
            <p class="recommend-card__detail">
              {{ item.detail || '这一步适合作为当前知识点之后的延伸学习。' }}
            </p>
          </article>
          <button
            v-if="pathRouteIds.length"
            type="button"
            class="inline-link inline-link--ai"
            @click="emit('show-recommend-path', pathRouteIds)"
          >
            在图谱中查看完整路径
          </button>
        </div>

        <!-- Quiz Section -->
        <div class="detail-section quiz-section">
          <span class="panel-label">掌握度挑战</span>

          <!-- Idle State -->
          <div v-if="quizState === 'idle'" class="quiz-idle-card">
            <p class="quiz-desc">
              本关卡包含 3 道专属选择题。全部答对即可瞬间达成 100% 掌握度，并在图谱中触发点亮特效！
            </p>
            <button type="button" class="primary-btn" style="width: 100%; justify-content: center;" @click="startQuiz">
              开始掌握度测验
            </button>
          </div>

          <!-- Active Quiz State -->
          <div v-else-if="quizState === 'active' && currentQuestion" class="quiz-active-card">
            <div class="quiz-progress-bar">
              <span>问题 {{ quizCurrentIndex + 1 }} / {{ quizQuestions.length }}</span>
              <div class="progress-track" style="height: 4px;">
                <span class="progress-fill" :style="{ width: `${((quizCurrentIndex) / quizQuestions.length) * 100}%` }"></span>
              </div>
            </div>

            <p class="quiz-question">{{ currentQuestion.question }}</p>

            <div class="quiz-options">
              <button
                v-for="(option, index) in currentQuestion.options"
                :key="index"
                type="button"
                class="quiz-option-btn"
                :class="{ 'quiz-option-btn--selected': quizSelectedOption === index }"
                @click="selectOption(index)"
              >
                {{ String.fromCharCode(65 + index) }}. {{ option }}
              </button>
            </div>

            <div class="quiz-actions">
              <button
                v-if="quizCurrentIndex < quizQuestions.length - 1"
                type="button"
                class="primary-btn"
                :disabled="quizSelectedOption === null"
                @click="nextQuestion"
              >
                下一题
              </button>
              <button
                v-else
                type="button"
                class="primary-btn"
                :disabled="quizSelectedOption === null"
                @click="submitQuiz"
              >
                提交评估
              </button>
            </div>
          </div>

          <!-- Success State -->
          <div v-else-if="quizState === 'success'" class="quiz-success-card">
            <div class="quiz-success-icon">🏆</div>
            <strong>挑战通关！达成 100% 掌握</strong>
            <p>你答对了解释中的全部问题，已完美掌握本核心节点。</p>
            <button type="button" class="secondary-btn" style="width: 100%; justify-content: center;" @click="resetQuiz">
              重新挑战
            </button>
          </div>

          <!-- Failed State -->
          <div v-else-if="quizState === 'failed'" class="quiz-failed-card">
            <div class="quiz-failed-icon">❌</div>
            <strong>挑战失败 (答对 {{ quizScore }} / {{ quizQuestions.length }})</strong>
            <p>别灰心！你可以点击下方智能追问让 AI 助手为你梳理薄弱环节，或重新挑战本测验。</p>
            <div class="detail-actions" style="margin-top: 10px;">
              <button type="button" class="secondary-btn" style="flex: 1; justify-content: center;" @click="resetQuiz">
                再试一次
              </button>
              <button type="button" class="primary-btn" style="flex: 1; justify-content: center;" @click="emit('open-ai')">
                寻求 AI 讲解
              </button>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <span class="panel-label">我的笔记</span>
          <textarea v-model="draftNote" rows="5" placeholder="记录理解、易错点或下一步计划"></textarea>
          <button type="button" class="secondary-btn" @click="saveNote">保存笔记</button>
        </div>

        <div class="detail-actions">
          <RouterLink class="primary-btn" :to="`/knowledge/${encodeURIComponent(detail.routeKnowledgeId)}`">
            <ExternalLink :size="16" aria-hidden="true" />
            开始学习
          </RouterLink>
          <button type="button" class="secondary-btn" @click="emit('open-ai')">问 AI 助教</button>
        </div>

        <!-- AI Smart Context Chips -->
        <div class="detail-section ai-context-chips" style="margin-top: 14px;">
          <span class="panel-label">智能探索追问</span>
          <div class="context-chips-list">
            <button
              v-for="chip in contextChips"
              :key="chip"
              type="button"
              class="context-chip-btn"
              @click="emit('open-ai-with-prompt', chip)"
            >
              {{ chip }}
            </button>
          </div>
        </div>
      </div>

      <div v-else class="panel-card empty-card empty-card--guide">
        <div class="empty-card__icon">
          <MousePointerClick :size="28" aria-hidden="true" />
        </div>
        <strong>选择一个知识点</strong>
        <p>点击图谱中的节点后，这里会显示说明、前置知识、推荐下一步和学习笔记。</p>
        <div class="empty-card__tips">
          <span>单击：查看详情</span>
          <span>双击：展开关联</span>
          <span>右键：加入路径或收藏</span>
        </div>
      </div>
    </div>
  </aside>
</template>
