<script setup>
import { Activity, ArrowRight, BookOpenCheck, Brain, Clock3, Target } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { fetchUnifiedPath } from '../api/recommend'
import { fetchResourceRecommendations, fetchResources } from '../api/resources'
import { useUserStore } from '../stores/userStore'

const userStore = useUserStore()
const route = useRoute()
const radarCanvas = ref(null)
const ready = ref(false)
const unifiedPath = ref([])
const unifiedInsights = ref([])
const resourceList = ref([])
const resourceRecommendations = ref([])
const resourceRecommendationSummary = ref('')
const resourceFeedbackSummary = ref(null)

const RESOURCE_TYPE_LABELS = {
  explanation_doc: '讲解文档',
  mind_map: '思维导图',
  practice_questions: '练习题',
  coding_case: '代码案例',
  reading_material: '拓展阅读',
  video_script: '视频/动画脚本',
}

const DEMO_PROGRESS_ITEMS = [
  { knowledgeId: 'gradient_descent', canonicalName: '梯度下降', chapterTitle: '机器学习', masteryScore: 88, answeredCount: 12, correctRate: 0.92 },
  { knowledgeId: 'loss_function', canonicalName: '损失函数', chapterTitle: '机器学习', masteryScore: 82, answeredCount: 9, correctRate: 0.89 },
  { knowledgeId: 'matrix_vector', canonicalName: '矩阵与向量运算', chapterTitle: '数学基础', masteryScore: 76, answeredCount: 8, correctRate: 0.81 },
  { knowledgeId: 'model_evaluation', canonicalName: '模型评估', chapterTitle: '机器学习', masteryScore: 64, answeredCount: 7, correctRate: 0.74 },
  { knowledgeId: 'knowledge_graph', canonicalName: '知识图谱', chapterTitle: '知识工程', masteryScore: 58, answeredCount: 6, correctRate: 0.67 },
  { knowledgeId: 'backpropagation', canonicalName: '反向传播', chapterTitle: '深度学习', masteryScore: 47, answeredCount: 5, correctRate: 0.6 },
  { knowledgeId: 'attention_mechanism', canonicalName: '注意力机制', chapterTitle: '大模型基础', masteryScore: 43, answeredCount: 4, correctRate: 0.58 },
  { knowledgeId: 'overfitting', canonicalName: '过拟合与正则化', chapterTitle: '模型泛化', masteryScore: 28, answeredCount: 5, correctRate: 0.42 },
]

const DEMO_DASHBOARD_SUMMARY = {
  weeklyTrend: [
    { label: '周一', value: 36 },
    { label: '周二', value: 48 },
    { label: '周三', value: 42 },
    { label: '周四', value: 63 },
    { label: '周五', value: 57 },
    { label: '周六', value: 72 },
    { label: '周日', value: 66 },
  ],
  recommendedPath: [
    {
      knowledgeId: 'overfitting',
      title: '过拟合与正则化',
      detail: '先区分训练误差和测试误差，再学习正则化、数据划分和早停。',
      reasonLabel: '优先巩固',
      score: 0.91,
      route: '/knowledge/overfitting',
    },
    {
      knowledgeId: 'backpropagation',
      title: '反向传播',
      detail: '把损失函数、梯度下降和链式法则串起来，理解训练闭环。',
      reasonLabel: '前置衔接',
      score: 0.84,
      route: '/knowledge/backpropagation',
    },
    {
      knowledgeId: 'attention_mechanism',
      title: '注意力机制',
      detail: '在神经网络基础上继续进入 Transformer 的关键结构。',
      reasonLabel: '继续推进',
      score: 0.78,
      route: '/knowledge/attention_mechanism',
    },
  ],
  knowledgeCards: [
    { id: 'overfitting', type: '优先巩固', progress: 28, title: '过拟合与正则化', domain: '模型泛化', route: '/knowledge/overfitting' },
    { id: 'attention_mechanism', type: '继续推进', progress: 43, title: '注意力机制', domain: '大模型基础', route: '/knowledge/attention_mechanism' },
    { id: 'backpropagation', type: '强化理解', progress: 47, title: '反向传播', domain: '深度学习', route: '/knowledge/backpropagation' },
  ],
}

const DEMO_RADAR_AXES = [
  { name: '数学基础', value: 72 },
  { name: '编程数据', value: 64 },
  { name: '机器学习', value: 81 },
  { name: '深度学习', value: 58 },
  { name: '语言知识', value: 51 },
  { name: '工程实践', value: 66 },
]

function hasMeaningfulProgress(item = {}) {
  return Number(item.answeredCount || 0) > 0 || Number(item.masteryScore ?? item.mastery ?? 0) > 0
}

onMounted(async () => {
  await Promise.all([
    userStore.loadProgress().catch(() => null),
    userStore.loadDashboard().catch(() => null),
    userStore.loadCompetency().catch(() => null),
  ])

  fetchUnifiedPath({ limit: 6 }).then((data) => {
    unifiedPath.value = data?.path || []
    unifiedInsights.value = data?.insights || []
  }).catch(() => {})

  loadResourceLoop()

  window.addEventListener('resize', handleRadarResize)
  ready.value = true
  await nextTick()
  handleRadarResize()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleRadarResize)
})

async function loadResourceLoop() {
  if (!userStore.isAuthenticated) return
  try {
    const [listPayload, recommendPayload] = await Promise.all([
      fetchResources({ limit: 100 }),
      fetchResourceRecommendations(),
    ])
    resourceList.value = listPayload?.list || []
    resourceRecommendations.value = recommendPayload?.list || []
    resourceRecommendationSummary.value = recommendPayload?.summary || ''
    resourceFeedbackSummary.value = recommendPayload?.feedbackSummary || null
  } catch {
    resourceList.value = []
    resourceRecommendations.value = []
    resourceRecommendationSummary.value = ''
    resourceFeedbackSummary.value = null
  }
}

const resourceStats = computed(() => {
  const summary = resourceFeedbackSummary.value || {}
  const feedbackTotal = Number(summary.total || 0)
  const helpfulCount = Number(summary.helpfulCount || 0)
  const avg = Number(summary.averageRating || 0)
  return {
    total: resourceList.value.length,
    completed: Number(summary.completedCount || 0),
    averageRatingText: avg > 0 ? avg.toFixed(1) : '暂无',
    hard: Number(summary.difficulty?.hard || 0),
    helpfulRateText: feedbackTotal > 0 ? `${Math.round((helpfulCount / feedbackTotal) * 100)}%` : '暂无',
  }
})

const resourceNextSteps = computed(() =>
  (resourceRecommendations.value || []).slice(0, 3).map((item) => ({
    id: item.id,
    title: item.title,
    reason: item.reason,
    typesLabel: (item.resourceTypes || []).map((type) => RESOURCE_TYPE_LABELS[type] || type).join(' · '),
    route: {
      name: 'resource-center',
      query: {
        knowledgeId: item.knowledgeId || '',
        knowledgeName: item.knowledgeName || '',
        courseTitle: item.courseTitle || '',
        target: item.target || '',
      },
    },
  })),
)

const shouldUseDemoData = computed(() => {
  const list = Array.isArray(userStore.progressList) ? userStore.progressList : []
  return !list.some((item) => hasMeaningfulProgress(item))
})

const progressItems = computed(() => (shouldUseDemoData.value ? DEMO_PROGRESS_ITEMS : userStore.progressList || []))
const dashboardSummary = computed(() => (shouldUseDemoData.value ? DEMO_DASHBOARD_SUMMARY : userStore.dashboardSummary || {}))
const isCompetencyView = computed(() => route?.name === 'competency-map' || route?.path === '/competency')

const learnerName = computed(() =>
  userStore.user?.name || userStore.user?.username || userStore.user?.email || '同学',
)

const progressStats = computed(() => {
  const stats = { total: 0, mastered: 0, learning: 0, weak: 0, answered: 0 }
  progressItems.value.forEach((item) => {
    const mastery = Number(item.masteryScore ?? item.mastery ?? 0)
    const answered = Number(item.answeredCount || 0)
    stats.total += 1
    stats.answered += answered
    if (mastery >= 80) stats.mastered += 1
    else if (mastery >= 40) stats.learning += 1
    if (mastery < 35 && answered > 0) stats.weak += 1
  })
  return stats
})

const totalProgress = computed(() => {
  if (!progressItems.value.length) return 0
  const total = progressItems.value.reduce((sum, item) => sum + Number(item.masteryScore ?? item.mastery ?? 0), 0)
  return Math.round(total / progressItems.value.length)
})

const metricCards = computed(() => [
  {
    label: '总体掌握度',
    value: `${totalProgress.value}%`,
    helper: '按已记录知识点平均掌握度计算',
    icon: Target,
    tone: 'primary',
  },
  {
    label: '已掌握知识点',
    value: progressStats.value.mastered,
    helper: '掌握度达到 80% 以上',
    icon: BookOpenCheck,
    tone: 'success',
  },
  {
    label: '待巩固',
    value: progressStats.value.weak,
    helper: '建议优先复习和做题',
    icon: Brain,
    tone: 'warning',
  },
  {
    label: '练习记录',
    value: progressStats.value.answered,
    helper: '累计答题和练习反馈',
    icon: Activity,
    tone: 'info',
  },
])

const rawRadarAxes = computed(() => {
  const current = userStore.competencyProfile?.radarAxes || []
  if (current.length) return current
  return DEMO_RADAR_AXES
})

const radarAxes = computed(() =>
  rawRadarAxes.value
    .map((item) => ({
      name: String(item.name || '').trim(),
      value: Math.max(0, Math.min(100, Math.round(Number(item.value || 0)))),
    }))
    .filter((item) => item.name),
)

const averageRadarScore = computed(() => {
  if (!radarAxes.value.length) return 0
  const total = radarAxes.value.reduce((sum, item) => sum + item.value, 0)
  return Math.round(total / radarAxes.value.length)
})

const sortedRadarAxes = computed(() =>
  [...radarAxes.value].sort((left, right) => right.value - left.value),
)

const strongestAxis = computed(() => sortedRadarAxes.value[0] || null)

const weakestAxes = computed(() =>
  [...radarAxes.value].sort((left, right) => left.value - right.value).slice(0, 3),
)

const weakAxisCount = computed(() => radarAxes.value.filter((axis) => axis.value < 60).length)

const focusActions = computed(() => {
  const source = unifiedPath.value.length ? unifiedPath.value : dashboardSummary.value?.recommendedPath || []
  return source.slice(0, 4).map((item, index) => ({
    title: item.title || item.displayTitle || item.knowledgeId || `推荐任务 ${index + 1}`,
    detail: item.detail || item.reason || '建议结合知识图谱和练习题继续推进。',
    meta: isCompetencyView.value ? item.reasonLabel || item.stage || '能力补强' : item.reasonLabel || item.stage || '下一步',
    route: item.route || `/knowledge/${encodeURIComponent(item.knowledgeId || item.title || '')}`,
    score: item.score != null ? `${Math.round(Number(item.score || 0) * 100)}分` : '',
  }))
})

const knowledgeCards = computed(() => {
  if (isCompetencyView.value) {
    const source = weakItems.value.length
      ? weakItems.value
      : [...progressItems.value]
        .sort((left, right) => Number(left.masteryScore ?? left.mastery ?? 0) - Number(right.masteryScore ?? right.mastery ?? 0))
        .slice(0, 6)
    return source.slice(0, 6).map((item) => {
      const knowledgeId = item.knowledgeId || item.routeKnowledgeId || item.canonicalName || item.name
      const mastery = Math.max(0, Math.min(100, Math.round(Number(item.masteryScore ?? item.mastery ?? 0))))
      return {
        id: knowledgeId,
        title: item.canonicalName || item.name || item.knowledgeId || '知识点',
        domain: item.chapterTitle || item.domain || '能力支撑知识',
        type: mastery < 60 ? '能力短板' : '能力支撑',
        progressValue: mastery,
        route: `/knowledge/${encodeURIComponent(knowledgeId || '')}`,
      }
    })
  }

  return (dashboardSummary.value?.knowledgeCards || []).slice(0, 6).map((card) => ({
    id: card.id || card.knowledgeId || card.title,
    title: card.title || card.knowledgeId || '知识点',
    domain: card.domain || card.type || '课程知识',
    type: card.type || '推荐',
    progressValue: Math.max(0, Math.min(100, Math.round(Number(card.progress || card.masteryScore || 0)))),
    route: card.route || `/knowledge/${encodeURIComponent(card.id || card.knowledgeId || card.title || '')}`,
  }))
})

const weeklyActivity = computed(() => {
  const source = dashboardSummary.value?.weeklyTrend || []
  if (source.length) return source
  return DEMO_DASHBOARD_SUMMARY.weeklyTrend
})

const normalizedWeeklyActivity = computed(() => {
  const max = Math.max(...weeklyActivity.value.map((item) => Number(item.value || 0)), 1)
  return weeklyActivity.value.map((item) => ({
    label: item.label,
    value: Number(item.value || 0),
    height: Math.max(10, Math.round((Number(item.value || 0) / max) * 100)),
  }))
})

const weakItems = computed(() =>
  [...progressItems.value]
    .filter((item) => Number(item.masteryScore ?? item.mastery ?? 0) < 60)
    .sort((a, b) => Number(a.masteryScore ?? a.mastery ?? 0) - Number(b.masteryScore ?? b.mastery ?? 0))
    .slice(0, 4),
)

const insightItems = computed(() => {
  if (isCompetencyView.value) {
    const competencyInsights = (userStore.competencyProfile?.growthInsights || []).slice(0, 3).map((item) => ({
      title: item.label || '能力成长建议',
      detail: item.note || `当前能力值 ${Math.round(Number(item.value || 0))}%，建议结合关联知识点继续训练。`,
    }))
    if (competencyInsights.length) return competencyInsights

    if (weakestAxes.value.length) {
      return weakestAxes.value.map((axis) => ({
        title: `补强${axis.name}`,
        detail: `当前能力值 ${axis.value}%，建议优先学习下方关联知识点，并用练习反馈校准能力画像。`,
      }))
    }

    return [
      { title: '能力画像待生成', detail: '完成一次练习或学习记录后，系统会把知识点掌握度映射到能力维度。' },
    ]
  }

  const fromApi = unifiedInsights.value.slice(0, 3).map((item) => ({
    title: item.title || item.reasonLabel || '学习建议',
    detail: item.detail || item.summary || '建议结合当前进度调整学习顺序。',
  }))
  if (fromApi.length) return fromApi
  return [
    { title: '先补薄弱点', detail: '掌握度低于 60% 的知识点建议优先做一轮练习。' },
    { title: '路径不要跳太快', detail: '先修知识没有稳定前，不建议直接进入复杂模型结构。' },
    { title: '保持复习节奏', detail: '已掌握内容也需要隔几天回顾，避免遗忘导致路径断层。' },
  ]
})

const pageCopy = computed(() => {
  if (isCompetencyView.value) {
    return {
      eyebrow: 'Competency Map',
      title: `${learnerName.value}的能力图谱`,
      summary: '把知识点掌握度映射到能力维度，直接看清优势能力、短板能力和对应补强路径。',
      secondaryAction: '打开知识地图',
      secondaryRoute: '/',
      primaryAction: '补强薄弱能力',
      primaryRoute: focusActions.value[0]?.route || '/',
    }
  }

  return {
    eyebrow: 'Learning Overview',
    title: `${learnerName.value}的学习总览`,
    summary: '这里聚合进度、薄弱点和下一步建议，帮助你尽快回到学习状态。',
    secondaryAction: '打开知识地图',
    secondaryRoute: '/',
    primaryAction: '继续学习',
    primaryRoute: focusActions.value[0]?.route || '/',
  }
})

const sectionCopy = computed(() => {
  if (isCompetencyView.value) {
    return {
      radarEyebrow: 'Competency Map',
      radarTitle: '能力雷达图',
      radarBadge: `${averageRadarScore.value}% 平均`,
      actionsEyebrow: 'Ability Path',
      actionsTitle: '能力补强路径',
      actionsBadge: '按短板优先',
      trendEyebrow: 'Training Rhythm',
      trendTitle: '能力训练节奏',
      weakEyebrow: 'Knowledge Links',
      weakTitle: '薄弱能力关联知识',
      weakBadge: `${weakItems.value.length} 个知识点`,
      cardsEyebrow: 'Support Cards',
      cardsTitle: '能力支撑知识点',
      cardsBadge: `${knowledgeCards.value.length} 张`,
      insightsEyebrow: 'Growth Notes',
      insightsTitle: '能力成长建议',
    }
  }

  return {
    radarEyebrow: 'Ability Map',
    radarTitle: '能力分布',
    radarBadge: `${averageRadarScore.value}% 平均`,
    actionsEyebrow: 'Next Step',
    actionsTitle: '下一步学习',
    actionsBadge: '按优先级排序',
    trendEyebrow: 'Weekly Rhythm',
    trendTitle: '本周节奏',
    weakEyebrow: 'Practice Feedback',
    weakTitle: '需要巩固',
    weakBadge: `${weakItems.value.length} 项`,
    cardsEyebrow: 'Focus Cards',
    cardsTitle: '重点知识卡片',
    cardsBadge: `${knowledgeCards.value.length} 张`,
    insightsEyebrow: 'Learning Notes',
    insightsTitle: '学习建议',
  }
})

function drawRadar() {
  const canvas = radarCanvas.value
  if (!canvas || radarAxes.value.length < 3) return

  const rect = canvas.getBoundingClientRect()
  const ratio = window.devicePixelRatio || 1
  const size = Math.max(220, Math.min(rect.width || 320, 360))
  canvas.width = size * ratio
  canvas.height = size * ratio
  canvas.style.height = `${size}px`

  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0)
  ctx.clearRect(0, 0, size, size)

  const cx = size / 2
  const cy = size / 2
  const radius = size * 0.34
  const count = radarAxes.value.length
  const font = "'IBM Plex Sans', 'Microsoft YaHei', sans-serif"

  ctx.save()
  for (let level = 1; level <= 4; level += 1) {
    ctx.beginPath()
    radarAxes.value.forEach((_, index) => {
      const angle = -Math.PI / 2 + (Math.PI * 2 * index) / count
      const r = (radius * level) / 4
      const x = cx + Math.cos(angle) * r
      const y = cy + Math.sin(angle) * r
      if (index === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })
    ctx.closePath()
    ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)'
    ctx.lineWidth = 1
    ctx.stroke()
  }

  ctx.beginPath()
  radarAxes.value.forEach((axis, index) => {
    const angle = -Math.PI / 2 + (Math.PI * 2 * index) / count
    const r = radius * (axis.value / 100)
    const x = cx + Math.cos(angle) * r
    const y = cy + Math.sin(angle) * r
    if (index === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  })
  ctx.closePath()
  ctx.fillStyle = 'rgba(45, 212, 191, 0.18)'
  ctx.strokeStyle = '#2dd4bf'
  ctx.lineWidth = 2
  ctx.fill()
  ctx.stroke()

  ctx.font = `600 12px ${font}`
  ctx.fillStyle = 'rgba(226, 232, 240, 0.82)'
  const labelPadding = 12
  radarAxes.value.forEach((axis, index) => {
    const angle = -Math.PI / 2 + (Math.PI * 2 * index) / count
    const rawX = cx + Math.cos(angle) * (radius + 28)
    const y = cy + Math.sin(angle) * (radius + 28)
    const textWidth = ctx.measureText(axis.name).width
    let x = rawX
    let align = 'center'

    if (rawX > cx + 8) {
      align = 'left'
      x = Math.min(rawX, size - labelPadding - textWidth)
    } else if (rawX < cx - 8) {
      align = 'right'
      x = Math.max(rawX, labelPadding + textWidth)
    } else {
      x = Math.max(labelPadding + textWidth / 2, Math.min(rawX, size - labelPadding - textWidth / 2))
    }

    ctx.textAlign = align
    ctx.textBaseline = 'middle'
    ctx.fillText(axis.name, x, y)
  })
  ctx.restore()
}

function handleRadarResize() {
  requestAnimationFrame(drawRadar)
}

watch(radarAxes, handleRadarResize, { deep: true })
</script>

<template>
  <section class="dashboard-view page-frame" :class="{ 'dashboard-view--ready': ready }">
    <div class="dashboard-shell">
      <header class="dashboard-header">
        <div>
          <span class="eyebrow">{{ pageCopy.eyebrow }}</span>
          <h1>{{ pageCopy.title }}</h1>
          <p>{{ pageCopy.summary }}</p>
        </div>
        <div class="dashboard-header__actions">
          <RouterLink class="secondary-btn" :to="pageCopy.secondaryRoute">{{ pageCopy.secondaryAction }}</RouterLink>
          <RouterLink class="primary-btn" :to="pageCopy.primaryRoute">
            {{ pageCopy.primaryAction }}
            <ArrowRight :size="16" aria-hidden="true" />
          </RouterLink>
        </div>
      </header>

      <section v-if="isCompetencyView" class="dashboard-competency-strip" aria-label="能力图谱诊断">
        <article>
          <span>平均能力</span>
          <strong>{{ averageRadarScore }}%</strong>
        </article>
        <article>
          <span>优势能力</span>
          <strong>{{ strongestAxis?.name || '待生成' }}</strong>
        </article>
        <article>
          <span>短板维度</span>
          <strong>{{ weakAxisCount }}</strong>
        </article>
        <article>
          <span>关联知识点</span>
          <strong>{{ knowledgeCards.length }}</strong>
        </article>
      </section>

      <section class="dashboard-metrics">
        <article v-for="card in metricCards" :key="card.label" class="dashboard-metric-card" :class="`dashboard-metric-card--${card.tone}`">
          <component :is="card.icon" class="dashboard-metric-card__icon" :size="22" aria-hidden="true" />
          <span>{{ card.label }}</span>
          <strong>{{ card.value }}</strong>
          <p>{{ card.helper }}</p>
        </article>
      </section>

      <section class="dashboard-grid">
        <article v-if="!isCompetencyView" class="dashboard-panel dashboard-panel--resource">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">Resource Loop</span>
              <h2>资源学习效果</h2>
            </div>
            <span class="dashboard-section-badge">{{ resourceStats.total }} 份资源</span>
          </div>
          <div class="dashboard-competency-strip dashboard-resource-stats">
            <article>
              <span>已生成资源</span>
              <strong>{{ resourceStats.total }}</strong>
            </article>
            <article>
              <span>已完成学习</span>
              <strong>{{ resourceStats.completed }}</strong>
            </article>
            <article>
              <span>平均评分</span>
              <strong>{{ resourceStats.averageRatingText }}</strong>
            </article>
            <article>
              <span>偏难反馈</span>
              <strong>{{ resourceStats.hard }}</strong>
            </article>
            <article>
              <span>有帮助比例</span>
              <strong>{{ resourceStats.helpfulRateText }}</strong>
            </article>
          </div>
          <p v-if="resourceRecommendationSummary" class="dashboard-resource-summary">{{ resourceRecommendationSummary }}</p>
          <div v-if="resourceNextSteps.length" class="dashboard-action-list">
            <RouterLink
              v-for="(item, index) in resourceNextSteps"
              :key="item.id"
              class="dashboard-action-card"
              :to="item.route"
            >
              <span class="dashboard-action-card__step">{{ index + 1 }}</span>
              <div>
                <div class="dashboard-action-card__meta">
                  <span>下一步资源包</span>
                  <strong v-if="item.typesLabel">{{ item.typesLabel }}</strong>
                </div>
                <h3>{{ item.title }}</h3>
                <p>{{ item.reason }}</p>
              </div>
            </RouterLink>
          </div>
          <div v-else class="dashboard-empty-state">
            <strong>还没有资源学习记录</strong>
            <p>到资源中心生成个性化资源包并提交反馈后，这里会展示学习效果与下一步推荐。</p>
          </div>
          <div class="dashboard-resource-foot">
            <RouterLink class="secondary-btn" to="/resources">
              打开资源中心
              <ArrowRight :size="16" aria-hidden="true" />
            </RouterLink>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--radar">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.radarEyebrow }}</span>
              <h2>{{ sectionCopy.radarTitle }}</h2>
            </div>
            <span class="dashboard-section-badge">{{ sectionCopy.radarBadge }}</span>
          </div>
          <div class="dashboard-radar-layout">
            <div class="dashboard-radar-canvas-wrap">
              <canvas ref="radarCanvas"></canvas>
            </div>
            <div class="dashboard-axis-pills">
              <div v-for="axis in radarAxes" :key="axis.name" class="dashboard-axis-pill">
                <span>{{ axis.name }}</span>
                <div class="dashboard-axis-pill__bar">
                  <i :style="{ width: `${axis.value}%` }"></i>
                </div>
                <strong>{{ axis.value }}%</strong>
              </div>
            </div>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--actions">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.actionsEyebrow }}</span>
              <h2>{{ sectionCopy.actionsTitle }}</h2>
            </div>
            <span class="dashboard-section-badge">{{ sectionCopy.actionsBadge }}</span>
          </div>
          <div class="dashboard-action-list">
            <RouterLink v-for="(item, index) in focusActions" :key="`${item.title}-${index}`" class="dashboard-action-card" :to="item.route">
              <span class="dashboard-action-card__step">{{ index + 1 }}</span>
              <div>
                <div class="dashboard-action-card__meta">
                  <span>{{ item.meta }}</span>
                  <strong v-if="item.score">{{ item.score }}</strong>
                </div>
                <h3>{{ item.title }}</h3>
                <p>{{ item.detail }}</p>
              </div>
            </RouterLink>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--trend">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.trendEyebrow }}</span>
              <h2>{{ sectionCopy.trendTitle }}</h2>
            </div>
            <Clock3 :size="18" aria-hidden="true" />
          </div>
          <div class="trend-bars dashboard-trend-bars">
            <div v-for="item in normalizedWeeklyActivity" :key="item.label" class="trend-bar">
              <strong>{{ item.value }}</strong>
              <span class="trend-bar__col" :style="{ height: `${item.height}%` }"></span>
              <small>{{ item.label }}</small>
            </div>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--weak">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.weakEyebrow }}</span>
              <h2>{{ sectionCopy.weakTitle }}</h2>
            </div>
            <span class="dashboard-section-badge">{{ sectionCopy.weakBadge }}</span>
          </div>
          <div v-if="weakItems.length" class="dashboard-weak-list">
            <RouterLink
              v-for="item in weakItems"
              :key="item.knowledgeId || item.routeKnowledgeId || item.canonicalName"
              class="dashboard-weak-item"
              :to="`/knowledge/${encodeURIComponent(item.knowledgeId || item.routeKnowledgeId || item.canonicalName || item.name)}`"
            >
              <div>
                <strong>{{ item.canonicalName || item.name || item.knowledgeId }}</strong>
                <span>{{ item.chapterTitle || '课程知识' }}</span>
              </div>
              <b>{{ Math.round(Number(item.masteryScore ?? item.mastery ?? 0)) }}%</b>
            </RouterLink>
          </div>
          <div v-else class="dashboard-empty-state">
            <strong>暂时没有明显薄弱点</strong>
            <p>继续完成练习后，这里会自动整理优先复习内容。</p>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--cards">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.cardsEyebrow }}</span>
              <h2>{{ sectionCopy.cardsTitle }}</h2>
            </div>
            <span class="dashboard-section-badge">{{ sectionCopy.cardsBadge }}</span>
          </div>
          <div v-if="knowledgeCards.length" class="dashboard-knowledge-grid">
            <RouterLink v-for="card in knowledgeCards" :key="card.id || card.title" class="dashboard-knowledge-card" :to="card.route">
              <div class="dashboard-knowledge-card__meta">
                <span>{{ card.type }}</span>
                <strong>{{ card.progressValue }}%</strong>
              </div>
              <h3>{{ card.title }}</h3>
              <p>{{ card.domain }}</p>
              <div class="progress-track">
                <span class="progress-fill" :style="{ width: `${card.progressValue}%` }"></span>
              </div>
            </RouterLink>
          </div>
          <div v-else class="dashboard-empty-state">
            <strong>还没有生成知识卡片</strong>
            <p>从知识地图开始学习后，系统会整理适合当前阶段的重点卡片。</p>
          </div>
        </article>

        <article class="dashboard-panel dashboard-panel--insights">
          <div class="dashboard-section-head">
            <div>
              <span class="eyebrow">{{ sectionCopy.insightsEyebrow }}</span>
              <h2>{{ sectionCopy.insightsTitle }}</h2>
            </div>
          </div>
          <div class="dashboard-insight-grid">
            <article v-for="item in insightItems" :key="item.title" class="dashboard-insight-card">
              <strong>{{ item.title }}</strong>
              <p>{{ item.detail }}</p>
            </article>
          </div>
        </article>
      </section>
    </div>
  </section>
</template>
