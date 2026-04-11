<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { domains } from '../data/courseData'
import { recommendedPath, assistantPrompts } from '../data/learning'
import { authState, fetchDashboardSummary, fetchLearningProgress } from '../auth'
import { resolveFriendlyKnowledgeRoute } from '../utils/knowledgeLookup'
import { buildDisplayDomains } from '../utils/learningAnalytics'
import { rankPracticeProjects } from '../utils/projectRecommend'

const router = useRouter()
const activeDomainId = ref(domains[0]?.id || '')
const searchQuery = ref('')
const aiBarInput = ref('')
const recommendationLevelRank = {
  core: 0,
  advanced: 1,
  frontier: 1,
  foundation: 2,
  practice: 3,
}

function goAssistant() {
  const q = aiBarInput.value.trim()
  router.push(q ? `/assistant?q=${encodeURIComponent(q)}` : '/assistant')
}
const isScrolling = ref(false)
const dashboardPathItems = ref(recommendedPath)
const progressItems = ref([])
let observer = null
let scrollTimeout = null

const displayDomains = computed(() => (
  authState.token
    ? buildDisplayDomains(progressItems.value)
    : domains
))

const filteredDomains = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return displayDomains.value
  return displayDomains.value
    .map((d) => ({
      ...d,
      points: d.points.filter(
        (p) => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q),
      ),
    }))
    .filter((d) => d.points.length > 0)
})

/** 全量域掌握度上的项目匹配排序（实时重算） */
const practiceRanked = computed(() => rankPracticeProjects(displayDomains.value))

/** 搜索过滤后仍保持推荐顺序 */
const practiceDomainFiltered = computed(() => {
  const fd = filteredDomains.value.find((d) => d.id === 'practice')
  if (!fd) return []
  const ids = new Set(fd.points.map((p) => p.id))
  return practiceRanked.value.filter((p) => ids.has(p.id))
})

function buildLearningPath(domainsToRank) {
  const candidates = domainsToRank
    .filter((domain) => domain.id !== 'practice')
    .flatMap((domain) =>
      domain.points.map((point) => ({
        ...point,
        domainId: domain.id,
        domainName: domain.name,
        mastery: Math.max(0, Math.min(100, Number(point.mastery || 0))),
      })),
    )
    .sort((left, right) => {
      if (left.mastery !== right.mastery) return left.mastery - right.mastery
      if (Boolean(left.recommend) !== Boolean(right.recommend)) return Number(Boolean(right.recommend)) - Number(Boolean(left.recommend))
      const leftLevel = recommendationLevelRank[left.level] ?? 99
      const rightLevel = recommendationLevelRank[right.level] ?? 99
      if (leftLevel !== rightLevel) return leftLevel - rightLevel
      return left.name.localeCompare(right.name, 'zh-CN')
    })

  return candidates.slice(0, 3).map((point) => {
    const stage =
      point.mastery <= 20
        ? '优先补弱'
        : point.mastery < 60
          ? '补弱推荐'
          : '继续进阶'

    const detail =
      point.mastery <= 0
        ? `你还没有开始 ${point.name}，建议先从 ${point.domainName} 里的这个关键知识点补起。`
        : point.mastery < 60
          ? `当前掌握度 ${point.mastery}%，建议优先补强 ${point.name}，再继续推进更高难度内容。`
          : `当前掌握度 ${point.mastery}%，可以继续巩固 ${point.name}，并向后续进阶内容推进。`

    return {
      stage,
      title: point.name,
      detail,
      route: point.route,
    }
  })
}

const visiblePathItems = computed(() => (
  authState.token
    ? buildLearningPath(displayDomains.value)
    : dashboardPathItems.value
))

const totalPoints = computed(() => displayDomains.value.reduce((s, d) => s + d.points.length, 0))
const masteredCount = computed(
  () => displayDomains.value.reduce((s, d) => s + d.points.filter((p) => p.mastery >= 60).length, 0),
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
  dashboardPathItems.value = recommendedPath

  if (!authState.token) {
    return
  }

  try {
    const data = await fetchDashboardSummary()
    if ((data.recommendedPath || data.path)?.length) {
      dashboardPathItems.value = (data.recommendedPath || data.path).map((item) => ({
        ...item,
        route: resolveFriendlyKnowledgeRoute(item.route, item.title),
      }))
    }
  } catch {
    /* use defaults */
  }
}

async function loadLearningSnapshot() {
  if (!authState.token) {
    progressItems.value = []
    return
  }
  try {
    const data = await fetchLearningProgress()
    progressItems.value = data.list || []
  } catch {
    progressItems.value = []
  }
}

onMounted(() => {
  loadDashboard()
  loadLearningSnapshot()
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
    for (const d of displayDomains.value) {
      const el = document.getElementById(`domain-${d.id}`)
      if (el) observer.observe(el)
    }
  })
})

onBeforeUnmount(() => {
  if (observer) observer.disconnect()
  clearTimeout(scrollTimeout)
})

watch(() => authState.token, () => {
  loadDashboard()
  loadLearningSnapshot()
})
</script>

<template>
  <div class="learn-page">
    <section class="learn-hero">
      <div class="learn-hero-left">
        <h1 class="learn-title">
          <strong>突破课程边界的人工智能专业知识点推荐智能体</strong>
        </h1>
        <p class="learn-desc">
          从知识点出发组织学习，不再被课程顺序和学期边界限制。<br />
          哪里不会就从哪里开始，学完立刻知道下一步。
        </p>
      </div>
    </section>

    <form class="ai-bar" @submit.prevent="goAssistant">
      <span class="ai-bar-label">AI 学伴</span>
      <input
        v-model="aiBarInput"
        class="ai-bar-input"
        :placeholder="assistantPrompts[0]"
      />
      <button type="submit" class="ai-bar-go">→</button>
    </form>

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
            <h3 class="point-name">{{ p.name }}</h3>
            <p class="point-desc">{{ p.description }}</p>
          </RouterLink>
        </div>
      </section>

      <p v-if="filteredDomains.length === 0" class="learn-empty">没有找到匹配的知识点</p>
    </div>

    <section class="cross-section">
      <div class="cross-heading">
        <h2>跨课程推荐路径</h2>
        <p>根据你的薄弱点、目标方向和兴趣偏好，动态生成补弱、进阶和场景化路径。</p>
      </div>
      <div class="cross-list">
        <RouterLink
          v-for="item in visiblePathItems"
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
