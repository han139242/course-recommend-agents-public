<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { domains } from '../data/courseData'
import { quickStats, recommendedPath, assistantPrompts } from '../data/learning'
import { fetchDashboardSummary } from '../auth'
import { rankPracticeProjects } from '../utils/projectRecommend'

const router = useRouter()
const activeDomainId = ref(domains[0]?.id || '')
const searchQuery = ref('')
const aiBarInput = ref('')

function goAssistant() {
  const q = aiBarInput.value.trim()
  router.push(q ? `/assistant?q=${encodeURIComponent(q)}` : '/assistant')
}
const isScrolling = ref(false)
const stats = ref(quickStats)
const pathItems = ref(recommendedPath)
let observer = null
let scrollTimeout = null

const filteredDomains = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return domains
  return domains
    .map((d) => ({
      ...d,
      points: d.points.filter(
        (p) => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q),
      ),
    }))
    .filter((d) => d.points.length > 0)
})

/** 全量域掌握度上的项目匹配排序（实时重算） */
const practiceRanked = computed(() => rankPracticeProjects(domains))

/** 搜索过滤后仍保持推荐顺序 */
const practiceDomainFiltered = computed(() => {
  const fd = filteredDomains.value.find((d) => d.id === 'practice')
  if (!fd) return []
  const ids = new Set(fd.points.map((p) => p.id))
  return practiceRanked.value.filter((p) => ids.has(p.id))
})

const recommendSnapshot = ref('')

function touchRecommendSnapshot() {
  recommendSnapshot.value = new Date().toLocaleString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const totalPoints = computed(() => domains.reduce((s, d) => s + d.points.length, 0))
const masteredCount = computed(
  () => domains.reduce((s, d) => s + d.points.filter((p) => p.mastery >= 60).length, 0),
)

function scrollToDomain(id) {
  isScrolling.value = true
  activeDomainId.value = id
  const el = document.getElementById(`domain-${id}`)
  if (el) {
    const top = el.getBoundingClientRect().top + window.scrollY - 120
    window.scrollTo({ top, behavior: 'smooth' })
  }
  clearTimeout(scrollTimeout)
  scrollTimeout = setTimeout(() => {
    isScrolling.value = false
  }, 800)
}

async function loadDashboard() {
  try {
    const data = await fetchDashboardSummary()
    if ((data.quickStats || data.stats)?.length) {
      stats.value = data.quickStats || data.stats
    }
    if ((data.recommendedPath || data.path)?.length) {
      pathItems.value = data.recommendedPath || data.path
    }
  } catch {
    /* use defaults */
  }
}

onMounted(() => {
  touchRecommendSnapshot()
  loadDashboard()
  nextTick(() => {
    observer = new IntersectionObserver(
      (entries) => {
        if (isScrolling.value) return
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        if (visible.length) {
          activeDomainId.value = visible[0].target.dataset.domainId
        }
      },
      { rootMargin: '-120px 0px -60% 0px', threshold: 0 },
    )
    for (const d of domains) {
      const el = document.getElementById(`domain-${d.id}`)
      if (el) observer.observe(el)
    }
  })
})

onBeforeUnmount(() => {
  if (observer) observer.disconnect()
  clearTimeout(scrollTimeout)
})
</script>

<template>
  <div class="learn-page">
    <!-- Hero -->
    <section class="learn-hero">
      <div class="learn-hero-left">
        <h1 class="learn-title">
          突破课程边界的<br /><strong>AI 专业知识图谱</strong>
        </h1>
        <p class="learn-desc">
          从知识点出发组织学习，不再被课程顺序和学期边界限制。<br />
          哪里不会就从哪里开始，学完立刻知道下一步。
        </p>
      </div>
      <div class="learn-stats">
        <div class="learn-stat">
          <strong>{{ masteredCount }}/{{ totalPoints }}</strong>
          <span>已掌握</span>
        </div>
        <div class="learn-stat" v-for="s in stats" :key="s.label">
          <strong>{{ s.value }}</strong>
          <span>{{ s.label }}</span>
        </div>
      </div>
    </section>

    <!-- AI Companion Entry -->
    <form class="ai-bar" @submit.prevent="goAssistant">
      <span class="ai-bar-label">AI 学伴</span>
      <input
        v-model="aiBarInput"
        class="ai-bar-input"
        :placeholder="assistantPrompts[0]"
      />
      <button type="submit" class="ai-bar-go">→</button>
    </form>

    <!-- Domain Navigation (sticky) -->
    <nav class="domain-nav">
      <button
        v-for="d in domains"
        :key="d.id"
        class="domain-tab"
        :class="{ active: activeDomainId === d.id }"
        @click="scrollToDomain(d.id)"
      >
        {{ d.name }}
      </button>

      <div class="domain-nav-search">
        <input v-model="searchQuery" placeholder="搜索知识点..." />
      </div>
    </nav>

    <!-- Domain Sections -->
    <div class="domain-content">
      <section
        v-for="domain in filteredDomains"
        :key="domain.id"
        :id="`domain-${domain.id}`"
        :data-domain-id="domain.id"
        class="domain-section"
      >
        <div class="domain-heading">
          <div>
            <h2>{{ domain.name }}</h2>
            <p class="domain-desc">{{ domain.desc }}</p>
          </div>
          <span class="domain-count">{{ domain.points.length }} 个知识点</span>
        </div>

        <div class="point-grid">
          <RouterLink
            v-for="p in domain.points"
            :key="p.id"
            :to="p.route"
            class="point-card"
          >
            <div class="point-head">
              <span class="point-level" :class="p.level">{{ p.levelLabel }}</span>
              <span v-if="p.recommend" class="point-rec">{{ p.recommend }}</span>
            </div>
            <h3 class="point-name">{{ p.name }}</h3>
            <p class="point-desc">{{ p.description }}</p>
            <div class="point-bar-row">
              <div class="point-bar">
                <div class="point-bar-fill" :style="{ width: `${p.mastery}%` }"></div>
              </div>
              <span class="point-pct">{{ p.mastery }}%</span>
            </div>
          </RouterLink>
        </div>
      </section>

      <p v-if="filteredDomains.length === 0" class="learn-empty">没有找到匹配的知识点</p>
    </div>

    <!-- Cross-course Recommendations -->
    <section class="cross-section">
      <div class="cross-heading">
        <h2>跨课程推荐路径</h2>
        <p>根据你的薄弱点、目标方向和兴趣偏好，动态生成补弱、进阶和场景化路径。</p>
      </div>
      <div class="cross-list">
        <RouterLink
          v-for="item in pathItems"
          :key="item.title"
          :to="item.route"
          class="cross-item"
        >
          <span class="cross-type">{{ item.stage }}</span>
          <div class="cross-body">
            <h3>{{ item.title }}</h3>
            <p>{{ item.detail }}</p>
          </div>
        </RouterLink>
      </div>
    </section>
  </div>
</template>
