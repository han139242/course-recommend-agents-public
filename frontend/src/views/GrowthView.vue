<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchCompetencyProfile, fetchLearningProgress } from '../auth'
import { buildAbilitySnapshot } from '../utils/learningAnalytics'

const loading = ref(true)
const errorMessage = ref('')
const fallbackSnapshot = ref(buildAbilitySnapshot())
const profileData = ref({
  radarAxes: [],
  growthInsights: [],
  growthSummaryCards: [],
  heatmapDomains: [],
  heatmapRows: [],
})

function hasUsableProfile(data) {
  return Boolean(
    (data?.radarAxes || []).length >= 3 &&
    (data?.growthSummaryCards || []).length &&
    (data?.heatmapDomains || []).length >= 3 &&
    (data?.heatmapRows || []).length,
  )
}

const growthRadarAxes = computed(() => (
  hasUsableProfile(profileData.value)
    ? profileData.value.radarAxes || []
    : fallbackSnapshot.value.radarAxes
))
const growthInsights = computed(() => (
  hasUsableProfile(profileData.value)
    ? profileData.value.growthInsights || []
    : fallbackSnapshot.value.growthInsights
))
const growthSummaryCards = computed(() => (
  hasUsableProfile(profileData.value)
    ? profileData.value.growthSummaryCards || []
    : fallbackSnapshot.value.growthSummaryCards
))
const heatmapDomains = computed(() => (
  hasUsableProfile(profileData.value)
    ? profileData.value.heatmapDomains || []
    : fallbackSnapshot.value.heatmapDomains
))
const heatmapRows = computed(() => (
  hasUsableProfile(profileData.value)
    ? profileData.value.heatmapRows || []
    : fallbackSnapshot.value.heatmapRows
))
const profileSummary = computed(() => (
  hasUsableProfile(profileData.value)
    ? '能力图谱由知识图谱映射与当前掌握度共同生成，重点看哪条路径已经形成稳定能力、哪条路径还需要补。'
    : fallbackSnapshot.value.profileSummary
))

const radarPolygon = computed(() => {
  const center = 130
  const radius = 88
  const axes = growthRadarAxes.value

  return axes
    .map((item, index) => {
      const angle = -Math.PI / 2 + (index * Math.PI * 2) / axes.length
      const scoreRadius = radius * (item.value / 100)
      const x = center + scoreRadius * Math.cos(angle)
      const y = center + scoreRadius * Math.sin(angle)
      return `${x},${y}`
    })
    .join(' ')
})

const radarAxes = computed(() => {
  const center = 130
  const radius = 88
  const axes = growthRadarAxes.value

  return axes.map((item, index) => {
    const angle = -Math.PI / 2 + (index * Math.PI * 2) / axes.length
    return {
      ...item,
      x: center + radius * Math.cos(angle),
      y: center + radius * Math.sin(angle),
      labelX: center + (radius + 26) * Math.cos(angle),
      labelY: center + (radius + 26) * Math.sin(angle),
      anchor: center + (radius + 26) * Math.cos(angle) >= center ? 'start' : 'end',
    }
  })
})

function guidePolygon(scale) {
  const center = 130
  const radius = 88 * scale
  const axes = growthRadarAxes.value

  return axes
    .map((_, index) => {
      const angle = -Math.PI / 2 + (index * Math.PI * 2) / axes.length
      const x = center + radius * Math.cos(angle)
      const y = center + radius * Math.sin(angle)
      return `${x},${y}`
    })
    .join(' ')
}

function heatStyle(value) {
  const alpha = Math.max(0.08, value / 100)
  return {
    background: `rgba(111, 120, 255, ${alpha})`,
    color: value > 55 ? '#ffffff' : '#1e2437',
  }
}

async function loadProfile() {
  loading.value = true
  errorMessage.value = ''

  try {
    const progress = await fetchLearningProgress().catch(() => ({ list: [] }))
    fallbackSnapshot.value = buildAbilitySnapshot(progress.list || [])
    profileData.value = await fetchCompetencyProfile()
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadProfile()
})
</script>

<template>
  <section class="growth-page">
    <div class="detail-hero">
      <span class="section-tag">能力图谱</span>
      <h1>从知识路径，反推出你的能力结构</h1>
      <p>{{ profileSummary }}</p>
    </div>

    <div class="growth-summary-grid">
      <article v-for="item in growthSummaryCards" :key="item.label" class="stat-card">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
      </article>
    </div>

    <div class="growth-layout">
      <article class="growth-radar-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">能力雷达图</span>
            <h2>按知识路径聚合的能力表现</h2>
          </div>
          <p>不是看你刷了多少页面，而是看每条知识路径上的关键节点是否真正打通，再聚合成能力轴。</p>
        </div>

        <p v-if="loading" class="auth-side-copy">正在同步成长画像...</p>
        <p v-else-if="errorMessage" class="auth-error">{{ errorMessage }}</p>

        <div class="growth-radar-wrap">
          <svg viewBox="0 0 260 260" class="growth-radar-svg" aria-label="能力雷达图">
            <polygon :points="guidePolygon(1)" class="radar-guide" />
            <polygon :points="guidePolygon(0.75)" class="radar-guide" />
            <polygon :points="guidePolygon(0.5)" class="radar-guide" />
            <polygon :points="guidePolygon(0.25)" class="radar-guide" />
            <line
              v-for="axis in radarAxes"
              :key="`${axis.name}-line`"
              x1="130"
              y1="130"
              :x2="axis.x"
              :y2="axis.y"
              class="radar-guide"
            />
            <polygon :points="radarPolygon" class="radar-shape growth-shape" />
            <text
              v-for="axis in radarAxes"
              :key="`${axis.name}-label`"
              :x="axis.labelX"
              :y="axis.labelY"
              :text-anchor="axis.anchor"
              class="growth-label"
            >
              {{ axis.name }}
            </text>
          </svg>
        </div>

        <div class="analysis-list">
          <div v-for="item in growthInsights" :key="item.label" class="analysis-item">
            <div class="analysis-top">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}%</strong>
            </div>
            <div class="progress-track soft">
              <div class="progress-fill" :style="{ width: `${item.value}%` }"></div>
            </div>
            <p>{{ item.note }}</p>
          </div>
        </div>
      </article>

      <article class="growth-heatmap-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">路径热力图</span>
            <h2>关键知识点分布</h2>
          </div>
          <p>横向看每条路径，纵向看关键知识点。浅色区域就是下一轮补强和推荐的重点。</p>
        </div>

        <div class="heatmap-table">
          <div class="heatmap-row header">
            <div class="heatmap-label">知识点</div>
            <div v-for="domain in heatmapDomains" :key="domain" class="heatmap-cell title">{{ domain }}</div>
          </div>

          <div v-for="row in heatmapRows" :key="row.label" class="heatmap-row">
            <div class="heatmap-label">{{ row.label }}</div>
            <div
              v-for="(value, index) in row.values"
              :key="`${row.label}-${heatmapDomains[index]}`"
              class="heatmap-cell"
              :style="heatStyle(value)"
            >
              {{ value }}
            </div>
          </div>
        </div>

        <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
      </article>
    </div>
  </section>
</template>
