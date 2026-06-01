<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  breakdown: {
    type: Object,
    default: null,
  },
})

const expanded = ref(false)

function scoreValue(value) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return 0
  return Math.min(Math.max(numeric, 0), 1)
}

const factors = computed(() => {
  if (!props.breakdown) return []
  return [
    { key: 'weakness', label: '薄弱程度', value: scoreValue(props.breakdown.weakness), color: '#ef4444' },
    { key: 'prerequisiteReadiness', label: '前置基础', value: scoreValue(props.breakdown.prerequisiteReadiness), color: '#f59e0b' },
    { key: 'targetRelevance', label: '目标匹配', value: scoreValue(props.breakdown.targetRelevance), color: '#38bdf8' },
    { key: 'forgettingRisk', label: '复习价值', value: scoreValue(props.breakdown.forgettingRisk), color: '#8b5cf6' },
    { key: 'crossCourseTransfer', label: '知识迁移', value: scoreValue(props.breakdown.crossCourseTransfer), color: '#22c55e' },
    { key: 'frontierRelevance', label: '扩展关联', value: scoreValue(props.breakdown.frontierRelevance), color: '#14b8a6' },
    { key: 'masteredPenalty', label: '熟练度修正', value: scoreValue(props.breakdown.masteredPenalty), color: '#64748b' },
  ]
})

const overallScore = computed(() => {
  if (!props.breakdown) return 0
  return Math.round(scoreValue(props.breakdown.overall) * 100)
})

const animatedScore = ref(0)
let animating = false

function animateScore(target) {
  if (animating) return
  animating = true
  const start = animatedScore.value
  const diff = target - start
  if (diff === 0) {
    animating = false
    return
  }
  const duration = 600
  const startTime = performance.now()
  function step(now) {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    animatedScore.value = Math.round(start + diff * eased)
    if (progress < 1) {
      requestAnimationFrame(step)
    } else {
      animatedScore.value = target
      animating = false
    }
  }
  requestAnimationFrame(step)
}

watch(expanded, (val) => {
  if (val) animateScore(overallScore.value)
})

function toggle() {
  expanded.value = !expanded.value
}

function percent(v) {
  return Math.round(scoreValue(v) * 100) + '%'
}
</script>

<template>
  <div v-if="breakdown" class="score-breakdown">
    <button
      type="button"
      class="score-breakdown__toggle"
      :aria-expanded="expanded"
      @click="toggle"
    >
      <span class="score-breakdown__label">推荐依据</span>
      <span class="score-breakdown__overall" :class="{ 'score-breakdown__overall--anim': expanded }">
        {{ expanded ? animatedScore : overallScore }}
      </span>
      <span class="score-breakdown__arrow" :class="{ expanded }">⌄</span>
    </button>
    <div v-if="expanded" class="score-breakdown__bars">
      <div v-for="f in factors" :key="f.key" class="sf-bar">
        <div class="sf-bar__header">
          <span class="sf-bar__name">{{ f.label }}</span>
          <span class="sf-bar__value">{{ percent(f.value) }}</span>
        </div>
        <div class="sf-bar__track">
          <div
            class="sf-bar__fill"
            :style="{ width: percent(f.value), background: f.color }"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.score-breakdown {
  margin-top: 8px;
  border: 1px solid var(--border-medium);
  border-radius: 8px;
  overflow: hidden;
}

.score-breakdown__toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  border: none;
  background: rgba(255, 255, 255, 0.03);
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
}

.score-breakdown__toggle:hover {
  background: var(--bg-hover);
}

.score-breakdown__label {
  font-weight: 600;
}

.score-breakdown__overall {
  margin-left: auto;
  font-weight: 700;
  color: var(--accent);
}

.score-breakdown__arrow {
  transition: transform 0.2s;
  color: var(--text-muted);
}

.score-breakdown__arrow.expanded {
  transform: rotate(180deg);
}

.score-breakdown__bars {
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sf-bar__header {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  margin-bottom: 2px;
}

.sf-bar__name {
  color: var(--text-muted);
}

.sf-bar__value {
  font-weight: 600;
  color: var(--text-primary);
}

.sf-bar__track {
  height: 6px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 3px;
  overflow: hidden;
}

.sf-bar__fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s ease;
}
</style>
