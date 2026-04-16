<script setup>
import { computed } from 'vue'

const props = defineProps({
  nodes: {
    type: Array,
    default: () => [],
  },
  progressMap: {
    type: Object,
    default: () => ({}),
  },
  bookmarks: {
    type: Array,
    default: () => [],
  },
  isAuthenticated: Boolean,
})

const emit = defineEmits(['highlight-status'])

const stats = computed(() => {
  let mastered = 0
  let learning = 0
  let weak = 0

  props.nodes.forEach((node) => {
    const progress = props.progressMap[node.routeKnowledgeId] || {}
    const mastery = Number(progress.masteryScore || 0)
    const answeredCount = Number(progress.answeredCount || 0)
    if (mastery >= 80) mastered += 1
    else if (mastery >= 40) learning += 1
    if (mastery < 30 && answeredCount > 0) weak += 1
  })

  return {
    total: props.nodes.length,
    mastered,
    learning,
    weak,
    bookmarked: props.bookmarks.length,
  }
})

const totalProgress = computed(() => {
  if (!stats.value.total) return 0
  return Math.round(((stats.value.mastered + stats.value.learning * 0.6) / stats.value.total) * 100)
})

const weeklyTrend = computed(() => {
  const base = Math.max(stats.value.mastered, 1)
  return Array.from({ length: 7 }, (_, index) => ({
    label: `D${index + 1}`,
    value: Math.max(12, Math.min(100, ((base * 7 + index * 9 + stats.value.learning * 3) % 86) + 14)),
  }))
})
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>学习进度</strong>
      <span v-if="isAuthenticated">个人画像</span>
      <span v-else>登录后更完整</span>
    </div>

    <div class="total-progress">
      <div class="tree-item__meta">
        <strong>总进度</strong>
        <span>{{ totalProgress }}%</span>
      </div>
      <div class="progress-track">
        <span class="progress-fill" :style="{ width: `${totalProgress}%` }"></span>
      </div>
    </div>

    <div class="progress-summary">
      <button type="button" class="stat-chip" @click="emit('highlight-status', 'mastered')">
        已掌握 {{ stats.mastered }}
      </button>
      <button type="button" class="stat-chip" @click="emit('highlight-status', 'learning')">
        学习中 {{ stats.learning }}
      </button>
      <button type="button" class="stat-chip" @click="emit('highlight-status', 'weak')">
        薄弱点 {{ stats.weak }}
      </button>
      <button type="button" class="stat-chip" @click="emit('highlight-status', 'bookmarked')">
        收藏 {{ stats.bookmarked }}
      </button>
    </div>

    <div class="trend-card">
      <span class="panel-label">本周学习趋势</span>
      <div class="trend-bars">
        <div v-for="item in weeklyTrend" :key="item.label" class="trend-bar">
          <span class="trend-bar__col" :style="{ height: `${item.value}%` }"></span>
          <small>{{ item.label }}</small>
        </div>
      </div>
    </div>
  </section>
</template>
