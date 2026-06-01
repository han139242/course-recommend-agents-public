<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { request } from '../api/request'

const loading = ref(true)
const error = ref('')
const data = ref(null)

const metrics = [
  { key: 'hitAt5', label: 'Hit@5' },
  { key: 'ndcgAt5', label: 'NDCG@5' },
  { key: 'mrr', label: 'MRR' },
  { key: 'prerequisiteViolationRate', label: '先修违规率', lowerIsBetter: true },
]

const baselines = computed(() => data.value?.baselines || [])

function formatMetric(value) {
  return `${(Number(value || 0) * 100).toFixed(1)}%`
}

function bestBaseline(metricKey, lowerIsBetter = false) {
  if (!baselines.value.length) return null
  return baselines.value.reduce((best, item) => {
    if (!best) return item
    const current = Number(item.metrics?.[metricKey] || 0)
    const bestValue = Number(best.metrics?.[metricKey] || 0)
    return lowerIsBetter ? (current < bestValue ? item : best) : (current > bestValue ? item : best)
  }, null)
}

onMounted(async () => {
  try {
    data.value = await request('/evaluation/demo')
  } catch (e) {
    error.value = e.message || '评测数据加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="eval-view page-frame">
    <header class="dashboard-header">
      <div>
        <span class="eyebrow">Teacher Console</span>
        <h1>教研评测</h1>
        <p>该页面用于教师或管理员查看推荐策略表现，不作为学生端主入口。</p>
      </div>
      <RouterLink class="secondary-btn" to="/">返回学生端</RouterLink>
    </header>

    <div v-if="loading" class="dashboard-empty-state">
      <strong>正在加载评测数据</strong>
      <p>请稍候。</p>
    </div>

    <div v-else-if="error" class="dashboard-empty-state dashboard-empty-state--error">
      <strong>加载失败</strong>
      <p>{{ error }}</p>
    </div>

    <template v-else-if="data">
      <section class="eval-summary-grid">
        <article class="eval-summary-card">
          <span>策略数量</span>
          <strong>{{ baselines.length }}</strong>
        </article>
        <article class="eval-summary-card">
          <span>候选知识点</span>
          <strong>{{ data.task?.candidates?.length || 0 }}</strong>
        </article>
        <article class="eval-summary-card">
          <span>先修约束</span>
          <strong>{{ data.task?.prerequisites?.length || 0 }}</strong>
        </article>
        <article class="eval-summary-card">
          <span>指标数量</span>
          <strong>{{ metrics.length }}</strong>
        </article>
      </section>

      <section class="dashboard-panel eval-table-section">
        <div class="dashboard-section-head">
          <div>
            <span class="eyebrow">Benchmark</span>
            <h2>策略对比</h2>
          </div>
        </div>

        <div class="eval-table-wrap">
          <table class="eval-table">
            <thead>
              <tr>
                <th>策略</th>
                <th v-for="m in metrics" :key="m.key">{{ m.label }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="baseline in baselines" :key="baseline.name">
                <td><strong>{{ baseline.name }}</strong></td>
                <td
                  v-for="m in metrics"
                  :key="m.key"
                  :class="{ 'eval-table__cell--best': bestBaseline(m.key, m.lowerIsBetter)?.name === baseline.name }"
                >
                  {{ formatMetric(baseline.metrics?.[m.key]) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </section>
</template>
