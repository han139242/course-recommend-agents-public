<script setup>
import { ArrowLeft, BookOpenText, ExternalLink, Newspaper, RefreshCw, Search } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchFrontierLibrary } from '../api/graph'

const loading = ref(false)
const error = ref('')
const library = ref({
  status: 'not_ready',
  message: '',
  generatedAt: '',
  total: 0,
  items: [],
})
const query = ref('')
const activeTopic = ref('全部')

const fallbackItems = [
  {
    paperId: 'demo-rag-agent',
    title: 'Retrieval-Augmented Transformer Agents for Knowledge Graph Reasoning',
    abstract: '围绕检索增强生成、Transformer 智能体和知识图谱推理的代表性前沿方向，用于展示论文如何映射回课程知识点。',
    venue: 'arXiv',
    year: 2026,
    url: 'https://arxiv.org',
    source: 'demo',
    updatedAt: '',
    topics: [
      { topicId: 'rag-agent', name: 'RAG Agent', reason: '把检索、规划和工具调用组合成可执行智能体。' },
      { topicId: 'kg-reasoning', name: '知识图谱推理', reason: '让生成模型使用结构化知识降低幻觉。' },
    ],
    knowledgeRefs: [
      { knowledgeId: 'rag', name: '检索增强生成', course: '高级大模型应用开发', reason: '论文方法直接使用 RAG。' },
      { knowledgeId: 'transformer', name: 'Transformer', course: '大模型基础', reason: '模型主体依赖 Transformer。' },
      { knowledgeId: 'knowledge-graph', name: '知识图谱', course: '知识工程', reason: '使用图谱作为推理支撑。' },
    ],
  },
]

const items = computed(() => {
  const sourceItems = Array.isArray(library.value.items) && library.value.items.length
    ? library.value.items
    : fallbackItems
  const normalizedQuery = query.value.trim().toLowerCase()
  return sourceItems.filter((item) => {
    const text = [
      item.title,
      item.abstract,
      item.venue,
      ...(item.topics || []).map((topic) => topic.name),
      ...(item.knowledgeRefs || []).map((knowledge) => knowledge.name),
    ].join(' ').toLowerCase()
    const topicMatched =
      activeTopic.value === '全部' ||
      (item.topics || []).some((topic) => topic.name === activeTopic.value) ||
      (item.knowledgeRefs || []).some((knowledge) => knowledge.name === activeTopic.value)
    return topicMatched && (!normalizedQuery || text.includes(normalizedQuery))
  })
})

const topicOptions = computed(() => {
  const names = new Set(['全部'])
  ;(library.value.items?.length ? library.value.items : fallbackItems).forEach((item) => {
    ;(item.topics || []).forEach((topic) => names.add(topic.name))
    ;(item.knowledgeRefs || []).forEach((knowledge) => names.add(knowledge.name))
  })
  return [...names].slice(0, 24)
})

const stats = computed(() => {
  const allItems = library.value.items?.length ? library.value.items : fallbackItems
  const topicCount = new Set(allItems.flatMap((item) => (item.topics || []).map((topic) => topic.name))).size
  const knowledgeCount = new Set(allItems.flatMap((item) => (item.knowledgeRefs || []).map((knowledge) => knowledge.name))).size
  const latestYear = Math.max(...allItems.map((item) => Number(item.year || 0)), 0)
  return [
    { label: '前沿资料', value: allItems.length },
    { label: '研究主题', value: topicCount },
    { label: '关联知识点', value: knowledgeCount },
    { label: '最新年份', value: latestYear || '-' },
  ]
})

async function loadLibrary() {
  loading.value = true
  error.value = ''
  try {
    library.value = await fetchFrontierLibrary({ limit: 50 })
  } catch (err) {
    error.value = err?.message || '前沿资料暂时无法加载。'
  } finally {
    loading.value = false
  }
}

function knowledgeRoute(item) {
  return `/knowledge/${encodeURIComponent(item.knowledgeId || item.name || '')}`
}

onMounted(loadLibrary)
</script>

<template>
  <section class="frontier-library page-frame">
    <div class="frontier-library__shell">
      <header class="frontier-library__header">
        <div>
          <span class="eyebrow">Frontier Library</span>
          <h1>AI 前沿资料</h1>
          <p>按论文、研究主题和课程知识点组织最新 AI 资料，帮助你从课堂概念延伸到当代研究。</p>
        </div>
        <div class="frontier-library__actions">
          <RouterLink class="secondary-btn" to="/">
            <ArrowLeft :size="16" aria-hidden="true" />
            知识图谱
          </RouterLink>
          <button class="primary-btn" type="button" :disabled="loading" @click="loadLibrary">
            <RefreshCw :size="16" aria-hidden="true" />
            刷新
          </button>
        </div>
      </header>

      <section class="frontier-library__stats" aria-label="前沿资料概览">
        <article v-for="item in stats" :key="item.label" class="frontier-stat">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </article>
      </section>

      <section class="frontier-library__toolbar">
        <label class="frontier-search">
          <Search :size="16" aria-hidden="true" />
          <input v-model="query" type="search" placeholder="搜索论文、主题或知识点" />
        </label>
        <div class="frontier-topic-tabs" aria-label="主题筛选">
          <button
            v-for="topic in topicOptions"
            :key="topic"
            type="button"
            :class="{ active: activeTopic === topic }"
            @click="activeTopic = topic"
          >
            {{ topic }}
          </button>
        </div>
      </section>

      <p v-if="error" class="frontier-library__notice">{{ error }}</p>
      <p v-else-if="library.status !== 'ready' && !library.items?.length" class="frontier-library__notice">
        {{ library.message || '前沿资料尚未入库，当前显示演示数据。运行前沿同步链路后会自动展示真实资料。' }}
      </p>

      <section class="frontier-library__grid">
        <article v-for="item in items" :key="item.paperId || item.title" class="frontier-paper-card">
          <div class="frontier-paper-card__head">
            <span>{{ item.venue || item.source || 'Frontier' }}</span>
            <strong>{{ item.year || '最新' }}</strong>
          </div>
          <h2>{{ item.title }}</h2>
          <p>{{ item.abstract }}</p>

          <div v-if="item.topics?.length" class="frontier-chip-row">
            <span v-for="topic in item.topics" :key="topic.topicId || topic.name" class="frontier-chip">
              <Newspaper :size="13" aria-hidden="true" />
              {{ topic.name }}
            </span>
          </div>

          <div v-if="item.knowledgeRefs?.length" class="frontier-knowledge-links">
            <RouterLink
              v-for="knowledge in item.knowledgeRefs"
              :key="knowledge.knowledgeId || knowledge.name"
              :to="knowledgeRoute(knowledge)"
              class="frontier-knowledge-link"
            >
              <BookOpenText :size="14" aria-hidden="true" />
              <span>
                <strong>{{ knowledge.name }}</strong>
                <small>{{ knowledge.course || '课程知识点' }}</small>
              </span>
            </RouterLink>
          </div>

          <a v-if="item.url" class="frontier-paper-card__source" :href="item.url" target="_blank" rel="noreferrer">
            查看原文
            <ExternalLink :size="14" aria-hidden="true" />
          </a>
        </article>
      </section>
    </div>
  </section>
</template>
