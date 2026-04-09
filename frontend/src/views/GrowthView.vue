<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchCompetencyProfile } from '../auth'

const loading = ref(true)
const errorMessage = ref('')
const profileData = ref({
  radarAxes: [],
  growthInsights: [],
  growthSummaryCards: [],
  heatmapDomains: [],
  heatmapRows: [],
})

const growthRadarAxes = computed(() => profileData.value.radarAxes || [])
const growthInsights = computed(() => profileData.value.growthInsights || [])
const growthSummaryCards = computed(() => profileData.value.growthSummaryCards || [])
const heatmapDomains = computed(() => profileData.value.heatmapDomains || [])
const heatmapRows = computed(() => profileData.value.heatmapRows || [])

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
      <span class="section-tag">成长分析</span>
      <h1>能力画像与知识点掌握热力</h1>
      <p>把学习行为、练习结果和测评表现聚合成能力变化，帮助用户知道自己现在处在什么位置、下一步该补哪里。</p>
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
            <h2>跨课程能力表现</h2>
          </div>
          <p>能力不是按课程拆开，而是看数学基础、机器学习、深度学习、知识图谱和场景能力是否同步成长。</p>
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
            <span class="section-tag">知识点热力图</span>
            <h2>当前掌握分布</h2>
          </div>
          <p>颜色越深代表掌握越高，浅色区域是后续补弱和推荐的重点。</p>
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
