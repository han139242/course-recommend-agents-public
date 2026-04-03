<script setup>
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { fetchFrontierBriefing, fetchKnowledgeDetail, fetchRecommendedPath } from '../auth'

const route = useRoute()
const loading = ref(true)
const errorMessage = ref('')
const currentKnowledge = ref(null)
const recommendedPath = ref([])
const frontierBriefing = ref(null)

const currentMasteryText = computed(() => {
  const count = currentKnowledge.value?.prerequisites?.length || 0
  if (!currentKnowledge.value) {
    return '正在同步知识点详情...'
  }
  if (count >= 2) {
    return `建议先梳理 ${count} 个关键先修知识点后再深入学习。`
  }
  return '当前可直接继续学习并结合关联知识点拓展。'
})

async function loadDetail(id) {
  loading.value = true
  errorMessage.value = ''

  try {
    const [detail, path] = await Promise.all([fetchKnowledgeDetail(id), fetchRecommendedPath()])
    const frontier = await fetchFrontierBriefing(detail.canonicalName)

    currentKnowledge.value = detail
    recommendedPath.value = path.path || []
    frontierBriefing.value = frontier
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

watch(
  () => route.params.knowledgeId,
  (id) => {
    loadDetail(String(id || ''))
  },
  { immediate: true },
)
</script>

<template>
  <section class="detail-page">
    <div class="detail-hero">
      <span class="section-tag">{{ currentKnowledge?.entityType || '知识点详情' }}</span>
      <h1>{{ currentKnowledge?.canonicalName || '加载中...' }}</h1>
      <p>{{ currentKnowledge?.summary || '正在获取知识点说明...' }}</p>
    </div>

    <div class="detail-layout">
      <article class="detail-main">
        <p v-if="loading" class="auth-side-copy">正在加载知识点详情...</p>
        <p v-else-if="errorMessage" class="auth-error">{{ errorMessage }}</p>

        <div class="detail-block">
          <span class="panel-label">当前状态</span>
          <h2>{{ currentMasteryText }}</h2>
        </div>

        <div class="detail-block">
          <span class="panel-label">推荐原因</span>
          <h3>
            {{
              currentKnowledge?.prerequisites?.length
                ? `该知识点与 ${currentKnowledge.prerequisites.slice(0, 2).join('、')} 等先修内容直接相关。`
                : '该知识点可作为当前学习主线中的重点延展节点。'
            }}
          </h3>
        </div>

        <div class="detail-block">
          <span class="panel-label">先修知识点</span>
          <ul class="detail-list">
            <li v-for="item in currentKnowledge?.prerequisites || []" :key="item">{{ item }}</li>
          </ul>
        </div>

        <div class="detail-block">
          <span class="panel-label">关联知识点</span>
          <ul class="detail-list">
            <li v-for="item in currentKnowledge?.related || []" :key="item">{{ item }}</li>
          </ul>
        </div>

        <div class="detail-block">
          <span class="panel-label">应用场景</span>
          <ul class="detail-list">
            <li v-for="item in currentKnowledge?.scenarios || []" :key="item">{{ item }}</li>
          </ul>
        </div>

        <div class="detail-block">
          <span class="panel-label">课后扩展</span>
          <h3>{{ frontierBriefing?.message || '前沿知识链路将在下一阶段接入。' }}</h3>
          <ul class="detail-list" v-if="frontierBriefing?.frontierTopics?.length">
            <li v-for="topic in frontierBriefing.frontierTopics" :key="topic">{{ topic }}</li>
          </ul>
          <ul class="detail-list" v-if="frontierBriefing?.papers?.length">
            <li v-for="paper in frontierBriefing.papers" :key="paper.title">
              {{ paper.title }}（{{ paper.venue }} {{ paper.year }}）- {{ paper.relation }}
            </li>
          </ul>
        </div>
      </article>

      <aside class="detail-side">
        <div class="detail-side-card">
          <span class="section-tag">智能推荐路径</span>
          <div class="detail-links">
            <RouterLink
              v-for="item in recommendedPath"
              :key="`${item.stage}-${item.title}`"
              :to="item.route"
              class="detail-link"
            >
              <strong>{{ item.stage }}</strong>
              <span>{{ item.title }}</span>
            </RouterLink>
          </div>
        </div>

        <div class="detail-side-card">
          <span class="section-tag">继续操作</span>
          <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
        </div>
      </aside>
    </div>
  </section>
</template>
