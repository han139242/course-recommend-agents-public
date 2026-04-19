<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import { useUserStore } from '../stores/userStore'
import { getCoursePalette } from '../utils/colorMap'

const userStore = useUserStore()
const radarCanvas = ref(null)

onMounted(async () => {
  await Promise.all([
    userStore.loadProgress().catch(() => null),
    userStore.loadDashboard().catch(() => null),
    userStore.loadCompetency().catch(() => null),
  ])

  window.addEventListener('resize', handleRadarResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleRadarResize)
})

const quickStats = computed(() => userStore.quickStats || [])
const dashboardSummary = computed(() => userStore.dashboardSummary || null)
const knowledgeCards = computed(() => dashboardSummary.value?.knowledgeCards || [])
const recommendedPath = computed(() => dashboardSummary.value?.recommendedPath || [])

const profile = computed(() => userStore.competencyProfile?.profile || null)
const radarAxes = computed(() => userStore.competencyProfile?.radarAxes || [])
const growthInsights = computed(() => userStore.competencyProfile?.growthInsights || [])
const growthCards = computed(() => userStore.competencyProfile?.growthSummaryCards || [])
const heatmapDomains = computed(() => userStore.competencyProfile?.heatmapDomains || [])
const heatmapRows = computed(() => userStore.competencyProfile?.heatmapRows || [])

const progressItems = computed(() => {
  const seen = new Set()
  return (userStore.progressList || []).filter((item) => {
    const key = String(item.knowledgeId || item.routeKnowledgeId || item.canonicalName || item.name || '').trim()
    if (!key || seen.has(key)) return false
    seen.add(key)
    return true
  })
})

const progressStats = computed(() => {
  let mastered = 0
  let learning = 0
  let weak = 0
  let total = 0

  progressItems.value.forEach((item) => {
    total += 1
    const mastery = Number(item.masteryScore || item.mastery || 0)
    const answered = Number(item.answeredCount || 0)
    if (mastery >= 80) mastered += 1
    else if (mastery >= 40) learning += 1
    if (mastery < 30 && answered > 0) weak += 1
  })

  return { total, mastered, learning, weak }
})

const totalProgress = computed(() => {
  const { total, mastered, learning } = progressStats.value
  if (!total) return 0
  return Math.round(((mastered + learning * 0.6) / total) * 100)
})

const averageRadarScore = computed(() => {
  if (!radarAxes.value.length) return 0
  const total = radarAxes.value.reduce((sum, item) => sum + Number(item.value || 0), 0)
  return Math.round(total / radarAxes.value.length)
})

const learnerName = computed(() => {
  const profileName = String(profile.value?.name || '').trim()
  if (profileName) return profileName

  const storeName = String(userStore.user?.name || userStore.user?.username || '').trim()
  if (storeName) return storeName

  const email = String(userStore.user?.email || '').trim()
  if (email.includes('@')) return email.split('@')[0]

  return '学习者'
})

const heroSummary = computed(() => {
  if (String(profile.value?.summary || '').trim()) {
    return profile.value.summary
  }

  if (progressStats.value.total > 0) {
    return `已追踪 ${progressStats.value.total} 个知识点，当前完成度 ${totalProgress.value}% 。建议优先处理薄弱点，再把推荐路径串成一条连续主线。`
  }

  return '这里不是简单的数据陈列，而是你的学习驾驶舱。完成几次练习后，这里会开始呈现能力画像、热区变化和下一步最值得投入的方向。'
})

function inferMetricTone(label = '', index = 0) {
  const text = String(label).toLowerCase()
  if (text.includes('掌握')) return 'success'
  if (text.includes('薄弱') || text.includes('风险')) return 'warning'
  if (text.includes('进度') || text.includes('%')) return 'accent'
  if (text.includes('能力')) return 'info'
  return ['accent', 'info', 'warning', 'success'][index % 4]
}

function buildMetricHelper(label = '') {
  const text = String(label)
  if (text.includes('掌握')) return '已经形成稳定输出的知识节点'
  if (text.includes('薄弱')) return '建议优先回到前置链查漏补缺'
  if (text.includes('学习中')) return '继续把相关知识点连成一条主线'
  if (text.includes('进度')) return '综合已掌握与进行中的完成估算'
  if (text.includes('能力')) return '由答题记录与图谱关系共同生成'
  return '仪表盘会随着学习记录持续刷新'
}

const metricCards = computed(() => {
  if (quickStats.value.length) {
    return quickStats.value.slice(0, 4).map((item, index) => ({
      eyebrow: index === 0 ? 'Focus' : `Metric 0${index + 1}`,
      label: item.label,
      value: item.value,
      helper: buildMetricHelper(item.label),
      tone: inferMetricTone(item.label, index),
    }))
  }

  return [
    {
      eyebrow: 'Focus',
      label: '学习中知识点',
      value: progressStats.value.learning,
      helper: '维持理解连续性的主战场',
      tone: 'accent',
    },
    {
      eyebrow: 'Stable',
      label: '已掌握知识点',
      value: progressStats.value.mastered,
      helper: '可以拿来做迁移和类比的基础',
      tone: 'success',
    },
    {
      eyebrow: 'Risk',
      label: '薄弱知识点',
      value: progressStats.value.weak,
      helper: '优先补齐前置关系，避免后续卡住',
      tone: 'warning',
    },
    {
      eyebrow: 'Progress',
      label: '总体掌握进度',
      value: `${totalProgress.value}%`,
      helper: '按掌握与在学状态综合估算',
      tone: 'info',
    },
  ]
})

const sortedAxes = computed(() => [...radarAxes.value].sort((left, right) => Number(right.value || 0) - Number(left.value || 0)))
const strongestAxes = computed(() => sortedAxes.value.slice(0, 3))
const weakestAxes = computed(() => [...sortedAxes.value].reverse().slice(0, 3))

const normalizedWeeklyActivity = computed(() => {
  const raw = dashboardSummary.value?.weeklyTrend
  const fallbackLabels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

  const source = Array.isArray(raw) && raw.length
    ? raw.map((item, index) => ({
        label: String(item.label || fallbackLabels[index] || `第${index + 1}天`),
        value: Number(item.value || 0),
      }))
    : fallbackLabels.map((label, index) => ({
        label,
        value: progressStats.value.total > 0
          ? Math.max(0, Math.min(100, Math.round((progressStats.value.learning * 14 + progressStats.value.mastered * 9 + index * 7) % 100)))
          : 0,
      }))

  const peak = Math.max(0, ...source.map((item) => item.value))
  return source.map((item) => ({
    ...item,
    height: peak > 0 ? Math.max(16, Math.round((item.value / peak) * 100)) : 14,
  }))
})

const weeklyTotals = computed(() => {
  const total = normalizedWeeklyActivity.value.reduce((sum, item) => sum + Number(item.value || 0), 0)
  const peak = Math.max(0, ...normalizedWeeklyActivity.value.map((item) => Number(item.value || 0)))
  const average = normalizedWeeklyActivity.value.length ? Math.round(total / normalizedWeeklyActivity.value.length) : 0
  return { total, peak, average }
})

const weakKnowledge = computed(() =>
  progressItems.value
    .filter((item) => Number(item.answeredCount || 0) > 0)
    .sort((left, right) => {
      const masteryGap = Number(left.masteryScore || 0) - Number(right.masteryScore || 0)
      if (masteryGap !== 0) return masteryGap
      return Number(right.answeredCount || 0) - Number(left.answeredCount || 0)
    })
    .slice(0, 4),
)

function resolveRoute(route = '', fallbackId = '') {
  const normalized = String(route || '').trim()
  if (normalized) return normalized
  const id = String(fallbackId || '').trim()
  return id ? `/knowledge/${encodeURIComponent(id)}` : '/'
}

const focusActions = computed(() => {
  if (recommendedPath.value.length) {
    return recommendedPath.value.slice(0, 4).map((item, index) => ({
      title: item.title || `推荐动作 ${index + 1}`,
      detail: item.detail || '建议从该节点切入，并继续展开它的前置与相邻知识。',
      meta: item.reasonLabel || item.stage || `第 ${index + 1} 步`,
      score: item.score != null ? `${Math.round(Number(item.score || 0) * 100)}%` : '',
      route: resolveRoute(item.route, item.knowledgeId || item.title),
    }))
  }

  if (weakKnowledge.value.length) {
    return weakKnowledge.value.map((item, index) => ({
      title: item.canonicalName || item.name || `薄弱知识点 ${index + 1}`,
      detail: `当前掌握度 ${Math.round(Number(item.masteryScore || 0))}% ，建议先回到它的前置关系补齐基础。`,
      meta: '优先补强',
      score: `${Math.round(Number(item.masteryScore || 0))}%`,
      route: resolveRoute('', item.routeKnowledgeId || item.knowledgeId || item.canonicalName || item.name),
    }))
  }

  return [
    {
      title: '先完成一组知识点练习',
      detail: '答题后这里会自动出现能力画像、热区矩阵和更具体的推荐动作。',
      meta: '启动建议',
      score: '',
      route: '/',
    },
    {
      title: '从图谱中展开一条课程主线',
      detail: '不要同时学很多散点，先沿着一条前置链往下走，效果会更稳定。',
      meta: '学习节奏',
      score: '',
      route: '/',
    },
  ]
})

const displayKnowledgeCards = computed(() => {
  const baseCards = knowledgeCards.value.length
    ? knowledgeCards.value
    : recommendedPath.value.slice(0, 4).map((item, index) => ({
        id: item.knowledgeId || item.title || `path-${index}`,
        type: item.stage || '推荐路径',
        progress: Number(item.score || 0) * 100,
        title: item.title,
        domain: item.reasonLabel || item.stage || '图谱推荐',
        tag: item.detail || '建议结合图谱关系继续深入',
        route: resolveRoute(item.route, item.knowledgeId || item.title),
      }))

  return baseCards.slice(0, 4).map((card) => ({
    ...card,
    progressValue: Math.max(0, Math.min(100, Math.round(Number(card.progress || 0)))),
    palette: getCoursePalette(card.domain || card.tag || card.title),
    route: resolveRoute(card.route, card.id || card.title),
  }))
})

const growthHighlights = computed(() => {
  if (growthInsights.value.length) {
    return growthInsights.value.slice(0, 4).map((item) => ({
      title: item.label,
      value: `${Math.round(Number(item.value || 0))}%`,
      detail: item.note || '当前能力维度正在持续变化。',
      ratio: Math.max(0, Math.min(100, Math.round(Number(item.value || 0)))),
    }))
  }

  if (growthCards.value.length) {
    return growthCards.value.slice(0, 4).map((item) => ({
      title: item.label,
      value: item.value,
      detail: '这是当前画像阶段给出的摘要判断。',
      ratio: 68,
    }))
  }

  if (strongestAxes.value.length) {
    return strongestAxes.value.map((item) => ({
      title: item.name,
      value: `${Math.round(Number(item.value || 0))}%`,
      detail: '当前表现相对稳定，可作为继续迁移的支点。',
      ratio: Math.max(0, Math.min(100, Math.round(Number(item.value || 0)))),
    }))
  }

  return []
})

const goalItems = computed(() => {
  if (profile.value?.goals?.length) {
    return profile.value.goals.slice(0, 4)
  }

  if (focusActions.value.length) {
    return focusActions.value.slice(0, 3).map((item) => `围绕「${item.title}」安排一段连续学习时间`)
  }

  return [
    '先在图谱中完成一次节点展开，建立自己的学习主线',
    '完成几道练习题，让能力画像开始有真实反馈',
    '把薄弱知识点和推荐路径连接起来，而不是零散地看概念',
  ]
})

const recentItems = computed(() => {
  if (profile.value?.recentRecords?.length) {
    return profile.value.recentRecords.slice(0, 4).map((item, index) => ({
      title: `最近记录 ${index + 1}`,
      detail: item,
    }))
  }

  if (progressItems.value.length) {
    return progressItems.value
      .slice()
      .sort((left, right) => Number(right.answeredCount || 0) - Number(left.answeredCount || 0))
      .slice(0, 4)
      .map((item) => ({
        title: item.canonicalName || item.name || item.knowledgeId || '知识点',
        detail: `${Math.round(Number(item.masteryScore || 0))}% 掌握度 · ${Number(item.answeredCount || 0)} 次练习`,
      }))
  }

  return [
    {
      title: '还没有学习记录',
      detail: '完成几次练习之后，这里会自动变成一条清晰的学习时间线。',
    },
  ]
})

const progressRingStyle = computed(() => {
  const progress = Math.max(0, Math.min(100, totalProgress.value))
  return {
    background: `conic-gradient(#8b5cf6 0 ${progress}%, #38bdf8 ${progress}% ${Math.min(100, progress + 12)}%, rgba(255,255,255,0.08) ${Math.min(100, progress + 12)}% 100%)`,
  }
})

const heatmapPeak = computed(() => {
  const values = heatmapRows.value.flatMap((row) => (row.values || []).map((value) => Number(value || 0)))
  return Math.max(1, ...values, 0)
})

function heatValueStyle(value) {
  const ratio = Math.max(0, Math.min(1, Number(value || 0) / heatmapPeak.value))
  return {
    background: `linear-gradient(180deg, rgba(139, 92, 246, ${0.18 + ratio * 0.52}), rgba(56, 189, 248, ${0.08 + ratio * 0.34}))`,
    borderColor: `rgba(139, 92, 246, ${0.14 + ratio * 0.28})`,
    boxShadow: ratio > 0.65 ? '0 10px 22px rgba(56, 189, 248, 0.12)' : 'none',
  }
}

function handleRadarResize() {
  nextTick(() => {
    if (radarAxes.value.length >= 3) {
      drawRadar()
    }
  })
}

function drawRadar() {
  const canvas = radarCanvas.value
  if (!canvas || radarAxes.value.length < 3) return

  const parentWidth = canvas.parentElement?.clientWidth || 320
  const size = Math.max(248, Math.min(340, parentWidth))
  const dpr = window.devicePixelRatio || 1
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  canvas.width = size * dpr
  canvas.height = size * dpr
  canvas.style.width = `${size}px`
  canvas.style.height = `${size}px`

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, size, size)

  const cx = size / 2
  const cy = size / 2
  const maxRadius = size * 0.31
  const axes = radarAxes.value
  const angleStep = (Math.PI * 2) / axes.length
  const startAngle = -Math.PI / 2

  ctx.save()
  const halo = ctx.createRadialGradient(cx, cy, maxRadius * 0.15, cx, cy, maxRadius * 1.35)
  halo.addColorStop(0, 'rgba(139, 92, 246, 0.22)')
  halo.addColorStop(1, 'rgba(15, 23, 42, 0)')
  ctx.fillStyle = halo
  ctx.beginPath()
  ctx.arc(cx, cy, maxRadius * 1.35, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()

  for (let level = 1; level <= 4; level += 1) {
    const radius = (maxRadius / 4) * level
    ctx.beginPath()
    axes.forEach((_, index) => {
      const angle = startAngle + angleStep * index
      const x = cx + radius * Math.cos(angle)
      const y = cy + radius * Math.sin(angle)
      if (index === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })
    ctx.closePath()
    ctx.strokeStyle = `rgba(148, 163, 184, ${0.08 + level * 0.03})`
    ctx.lineWidth = 1
    ctx.stroke()
  }

  axes.forEach((axis, index) => {
    const angle = startAngle + angleStep * index
    const x = cx + maxRadius * Math.cos(angle)
    const y = cy + maxRadius * Math.sin(angle)

    ctx.beginPath()
    ctx.moveTo(cx, cy)
    ctx.lineTo(x, y)
    ctx.strokeStyle = 'rgba(148, 163, 184, 0.12)'
    ctx.lineWidth = 1
    ctx.stroke()

    const labelRadius = maxRadius + 24
    const lx = cx + labelRadius * Math.cos(angle)
    const ly = cy + labelRadius * Math.sin(angle)
    ctx.fillStyle = 'rgba(226, 232, 240, 0.88)'
    ctx.font = '600 12px "IBM Plex Sans", "PingFang SC", sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(axis.name, lx, ly)
  })

  ctx.beginPath()
  axes.forEach((axis, index) => {
    const angle = startAngle + angleStep * index
    const ratio = Math.max(0, Math.min(1, Number(axis.value || 0) / 100))
    const x = cx + maxRadius * ratio * Math.cos(angle)
    const y = cy + maxRadius * ratio * Math.sin(angle)
    if (index === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  })
  ctx.closePath()

  const fill = ctx.createLinearGradient(cx - maxRadius, cy - maxRadius, cx + maxRadius, cy + maxRadius)
  fill.addColorStop(0, 'rgba(56, 189, 248, 0.28)')
  fill.addColorStop(1, 'rgba(139, 92, 246, 0.34)')
  ctx.fillStyle = fill
  ctx.fill()
  ctx.strokeStyle = 'rgba(167, 139, 250, 0.92)'
  ctx.lineWidth = 2.2
  ctx.stroke()

  axes.forEach((axis, index) => {
    const angle = startAngle + angleStep * index
    const ratio = Math.max(0, Math.min(1, Number(axis.value || 0) / 100))
    const x = cx + maxRadius * ratio * Math.cos(angle)
    const y = cy + maxRadius * ratio * Math.sin(angle)

    ctx.beginPath()
    ctx.arc(x, y, 4, 0, Math.PI * 2)
    ctx.fillStyle = '#e2e8f0'
    ctx.fill()
    ctx.beginPath()
    ctx.arc(x, y, 2.2, 0, Math.PI * 2)
    ctx.fillStyle = '#8b5cf6'
    ctx.fill()
  })

  ctx.beginPath()
  ctx.arc(cx, cy, 32, 0, Math.PI * 2)
  ctx.fillStyle = 'rgba(9, 14, 28, 0.96)'
  ctx.fill()
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.08)'
  ctx.lineWidth = 1
  ctx.stroke()

  ctx.fillStyle = 'rgba(148, 163, 184, 0.82)'
  ctx.font = '600 11px "IBM Plex Sans", "PingFang SC", sans-serif'
  ctx.textAlign = 'center'
  ctx.fillText('画像', cx, cy - 8)
  ctx.fillStyle = '#f8fafc'
  ctx.font = '700 18px "IBM Plex Sans", "PingFang SC", sans-serif'
  ctx.fillText(`${averageRadarScore.value}%`, cx, cy + 14)
}

watch(
  radarAxes,
  () => {
    handleRadarResize()
  },
  { immediate: true, deep: true },
)
</script>

<template>
  <section class="dashboard-view page-frame">
    <div class="dashboard-shell">
      <header class="page-card dashboard-hero">
        <div class="dashboard-hero__copy">
          <p class="eyebrow">学习驾驶舱</p>
          <h1>{{ learnerName }}的学习概览</h1>
          <p class="dashboard-hero__summary">{{ heroSummary }}</p>

          <div class="dashboard-hero__chips">
            <span class="dashboard-chip">{{ progressStats.total }} 个知识点已纳入追踪</span>
            <span class="dashboard-chip">{{ radarAxes.length || 0 }} 个能力维度已激活</span>
            <span class="dashboard-chip">{{ focusActions.length }} 条当前建议可直接执行</span>
          </div>
        </div>

        <div class="dashboard-hero__side">
          <div class="dashboard-progress-ring" :style="progressRingStyle">
            <div class="dashboard-progress-ring__inner">
              <strong>{{ totalProgress }}%</strong>
              <span>掌握进度</span>
            </div>
          </div>

          <div class="dashboard-hero__actions">
            <RouterLink class="primary-btn" to="/">返回图谱继续学习</RouterLink>
            <RouterLink class="secondary-btn" to="/">查看知识图谱</RouterLink>
          </div>
        </div>
      </header>

      <section class="dashboard-metrics">
        <article
          v-for="stat in metricCards"
          :key="`${stat.label}-${stat.eyebrow}`"
          class="page-card dashboard-metric-card"
          :class="`dashboard-metric-card--${stat.tone}`"
        >
          <span class="dashboard-metric-card__eyebrow">{{ stat.eyebrow }}</span>
          <strong>{{ stat.value }}</strong>
          <span class="dashboard-metric-card__label">{{ stat.label }}</span>
          <p>{{ stat.helper }}</p>
        </article>
      </section>

      <section class="dashboard-layout">
        <article class="page-card dashboard-panel dashboard-panel--radar">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">能力画像</p>
              <h2>能力驾驶舱</h2>
            </div>
            <span class="dashboard-section-badge">{{ averageRadarScore }}% 平均表现</span>
          </div>

          <div class="dashboard-radar-block">
            <div class="dashboard-radar-stage">
              <div v-if="radarAxes.length >= 3" class="dashboard-radar-canvas-wrap">
                <canvas ref="radarCanvas"></canvas>
              </div>
              <div v-else class="dashboard-empty-state dashboard-empty-state--compact">
                <strong>能力画像还没成型</strong>
                <p>完成几次练习后，这里会开始显示你在哪些维度已经稳定，哪些维度还需要补强。</p>
                <RouterLink class="primary-btn" to="/">去图谱里开始学习</RouterLink>
              </div>
            </div>

            <div class="dashboard-axis-column">
              <div class="dashboard-axis-list">
                <div class="dashboard-axis-list__head">
                  <span class="eyebrow">优势轴</span>
                  <strong>当前最稳的部分</strong>
                </div>
                <div v-if="strongestAxes.length" class="dashboard-axis-pills">
                  <div v-for="axis in strongestAxes" :key="axis.name" class="dashboard-axis-pill dashboard-axis-pill--strong">
                    <span>{{ axis.name }}</span>
                    <strong>{{ Math.round(axis.value) }}%</strong>
                  </div>
                </div>
                <p v-else class="dashboard-muted-copy">还没有足够数据来判断稳定优势。</p>
              </div>

              <div class="dashboard-axis-list">
                <div class="dashboard-axis-list__head">
                  <span class="eyebrow">补强轴</span>
                  <strong>接下来最值得投入</strong>
                </div>
                <div v-if="weakestAxes.length" class="dashboard-axis-pills">
                  <div v-for="axis in weakestAxes" :key="axis.name" class="dashboard-axis-pill dashboard-axis-pill--weak">
                    <span>{{ axis.name }}</span>
                    <strong>{{ Math.round(axis.value) }}%</strong>
                  </div>
                </div>
                <p v-else class="dashboard-muted-copy">开始练习后，这里会自动生成需要优先补的能力轴。</p>
              </div>
            </div>
          </div>
        </article>

        <article class="page-card dashboard-panel dashboard-panel--actions">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">下一步</p>
              <h2>推荐动作</h2>
            </div>
            <span class="dashboard-section-badge">按优先级排序</span>
          </div>

          <div class="dashboard-action-list">
            <RouterLink
              v-for="item in focusActions"
              :key="`${item.meta}-${item.title}`"
              class="dashboard-action-card"
              :to="item.route"
            >
              <div class="dashboard-action-card__meta">
                <span>{{ item.meta }}</span>
                <strong v-if="item.score">{{ item.score }}</strong>
              </div>
              <h3>{{ item.title }}</h3>
              <p>{{ item.detail }}</p>
            </RouterLink>
          </div>
        </article>

        <article class="page-card dashboard-panel dashboard-panel--trend">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">学习节奏</p>
              <h2>本周趋势</h2>
            </div>
            <span class="dashboard-section-badge">连续观察比单天爆发更重要</span>
          </div>

          <div class="dashboard-trend-summary">
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.total }}</strong>
              <span>本周活跃值</span>
            </div>
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.peak }}</strong>
              <span>单日峰值</span>
            </div>
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.average }}</strong>
              <span>日均节奏</span>
            </div>
          </div>

          <div class="trend-bars trend-bars--large dashboard-trend-bars">
            <div v-for="item in normalizedWeeklyActivity" :key="item.label" class="trend-bar dashboard-trend-bar">
              <span class="trend-bar__col dashboard-trend-bar__col" :style="{ height: `${item.height}%` }"></span>
              <strong>{{ item.value }}</strong>
              <small>{{ item.label }}</small>
            </div>
          </div>
        </article>

        <article v-if="heatmapRows.length && heatmapDomains.length" class="page-card dashboard-panel dashboard-panel--heatmap">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">热区矩阵</p>
              <h2>知识热区</h2>
            </div>
            <span class="dashboard-section-badge">颜色越亮，当前关注度越高</span>
          </div>

          <div class="dashboard-heatmap">
            <div class="dashboard-heatmap__head">
              <span class="dashboard-heatmap__corner">知识点</span>
              <span v-for="domain in heatmapDomains" :key="domain" class="dashboard-heatmap__domain">{{ domain }}</span>
            </div>

            <div v-for="row in heatmapRows" :key="row.label" class="dashboard-heatmap__row">
              <span class="dashboard-heatmap__label">{{ row.label }}</span>
              <span
                v-for="(value, index) in row.values"
                :key="`${row.label}-${index}`"
                class="dashboard-heatmap__cell"
                :style="heatValueStyle(value)"
              >
                {{ Math.round(value) }}
              </span>
            </div>
          </div>
        </article>

        <article class="page-card dashboard-panel dashboard-panel--cards">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">推荐卡片</p>
              <h2>值得马上处理的知识点</h2>
            </div>
            <span class="dashboard-section-badge">{{ displayKnowledgeCards.length }} 张卡片</span>
          </div>

          <div v-if="displayKnowledgeCards.length" class="dashboard-knowledge-grid">
            <RouterLink
              v-for="card in displayKnowledgeCards"
              :key="card.id || card.title"
              class="dashboard-knowledge-card"
              :to="card.route"
              :style="{
                '--card-primary': card.palette.primary,
                '--card-soft': card.palette.light,
                '--card-ring': card.palette.ring,
              }"
            >
              <div class="dashboard-knowledge-card__meta">
                <span class="dashboard-knowledge-card__tag">{{ card.type || card.domain || '推荐' }}</span>
                <strong>{{ card.progressValue }}%</strong>
              </div>
              <h3>{{ card.title }}</h3>
              <p>{{ card.tag || card.domain || '建议结合图谱关系继续推进' }}</p>
              <div class="progress-track">
                <span class="progress-fill" :style="{ width: `${card.progressValue}%` }"></span>
              </div>
            </RouterLink>
          </div>

          <div v-else class="dashboard-empty-state">
            <strong>推荐卡片还没生成</strong>
            <p>开始在图谱里学习后，这里会自动整理出最适合你当下推进的知识卡片。</p>
            <RouterLink class="secondary-btn" to="/">去图谱里找一个起点</RouterLink>
          </div>
        </article>

        <article class="page-card dashboard-panel dashboard-panel--insights">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">成长洞察</p>
              <h2>近期画像解读</h2>
            </div>
            <span class="dashboard-section-badge">从趋势看，而不是只看一组分数</span>
          </div>

          <div v-if="growthHighlights.length" class="dashboard-insight-grid">
            <article v-for="item in growthHighlights" :key="item.title" class="dashboard-insight-card">
              <div class="dashboard-insight-card__head">
                <span>{{ item.title }}</span>
                <strong>{{ item.value }}</strong>
              </div>
              <div class="progress-track">
                <span class="progress-fill" :style="{ width: `${item.ratio}%` }"></span>
              </div>
              <p>{{ item.detail }}</p>
            </article>
          </div>

          <div v-else class="dashboard-empty-state dashboard-empty-state--compact">
            <strong>成长洞察会逐步形成</strong>
            <p>先做几次练习，系统才能判断你的稳定强项、薄弱区和近期变化趋势。</p>
          </div>
        </article>

        <article class="page-card dashboard-panel dashboard-panel--records">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">学习备忘</p>
              <h2>目标与记录</h2>
            </div>
            <span class="dashboard-section-badge">让当前任务和最近变化同屏出现</span>
          </div>

          <div class="dashboard-records-layout">
            <div class="dashboard-note-block">
              <h3>当前目标</h3>
              <ul class="dashboard-list">
                <li v-for="goal in goalItems" :key="goal">{{ goal }}</li>
              </ul>
            </div>

            <div class="dashboard-note-block">
              <h3>最近记录</h3>
              <div class="dashboard-record-list">
                <article v-for="item in recentItems" :key="`${item.title}-${item.detail}`" class="dashboard-record-item">
                  <strong>{{ item.title }}</strong>
                  <p>{{ item.detail }}</p>
                </article>
              </div>
            </div>
          </div>
        </article>
      </section>
    </div>
  </section>
</template>
