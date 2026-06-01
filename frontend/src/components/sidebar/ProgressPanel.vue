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

const progressItems = computed(() => [
  { key: 'mastered', label: '已掌握', value: stats.value.mastered },
  { key: 'learning', label: '学习中', value: stats.value.learning },
  { key: 'weak', label: '待巩固', value: stats.value.weak },
  { key: 'bookmarked', label: '收藏', value: stats.value.bookmarked },
])
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>学习进度</strong>
      <span>{{ isAuthenticated ? '个人同步' : '登录后同步' }}</span>
    </div>

    <div class="total-progress">
      <div class="tree-item__meta">
        <strong>总体进度</strong>
        <span>{{ totalProgress }}%</span>
      </div>
      <div class="progress-track">
        <span class="progress-fill" :style="{ width: `${totalProgress}%` }"></span>
      </div>
    </div>

    <div class="progress-summary">
      <button
        v-for="item in progressItems"
        :key="item.key"
        type="button"
        class="stat-chip"
        @click="emit('highlight-status', item.key)"
      >
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
      </button>
    </div>
  </section>
</template>
