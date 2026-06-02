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

const sourceItems = computed(() => (Array.isArray(library.value.items) ? library.value.items.filter(isTrustedItem) : []))

const items = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase()
  return sourceItems.value.filter((item) => {
    const text = [
      item.title,
      item.abstract,
      item.venue,
      item.source,
      ...(item.authors || []),
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
  sourceItems.value.forEach((item) => {
    ;(item.topics || []).forEach((topic) => names.add(topic.name))
    ;(item.knowledgeRefs || []).forEach((knowledge) => names.add(knowledge.name))
  })
  return [...names].slice(0, 18)
})

const stats = computed(() => {
  const paperCount = sourceItems.value.filter((item) => item.source !== 'github' && item.venue !== 'GitHub').length
  const githubCount = sourceItems.value.filter((item) => item.source === 'github' || item.venue === 'GitHub').length
  const topicCount = new Set(sourceItems.value.flatMap((item) => (item.topics || []).map((topic) => topic.name))).size
  return [
    { label: '真实资料', value: sourceItems.value.length },
    { label: '论文条目', value: paperCount },
    { label: 'GitHub 更新', value: githubCount },
    { label: '研究主题', value: topicCount },
  ]
})

const generatedAtText = computed(() => formatDateTime(library.value.generatedAt))
const nextRefreshText = computed(() => formatDateTime(library.value.nextRefreshAt))
const hasItems = computed(() => sourceItems.value.length > 0)
const hasFilteredItems = computed(() => items.value.length > 0)
const rawItemCount = computed(() => (Array.isArray(library.value.items) ? library.value.items.length : 0))
const hiddenExampleCount = computed(() => Math.max(0, rawItemCount.value - sourceItems.value.length))
const emptyTitle = computed(() => (hiddenExampleCount.value > 0 ? '已隐藏示例资料' : '暂无真实前沿资料'))
const emptyDescription = computed(() =>
  hiddenExampleCount.value > 0
    ? '当前接口返回的仍是 example.org / example.com 旧示例数据，页面已按真实来源要求隐藏。重启后端服务后会改为从 arXiv、OpenAlex 和 GitHub 同步公开资料。'
    : '后端未能从 arXiv、OpenAlex 或 GitHub 获取到公开数据。请稍后刷新，或检查服务器网络访问。',
)

async function loadLibrary() {
  loading.value = true
  error.value = ''
  try {
    library.value = await fetchFrontierLibrary({ limit: 12 })
  } catch (err) {
    error.value = err?.message || '前沿资料暂时无法加载。'
  } finally {
    loading.value = false
  }
}

function knowledgeRoute(item) {
  return `/knowledge/${encodeURIComponent(item.knowledgeId || item.name || '')}`
}

function sourceLabel(item) {
  if (item.source === 'github' || item.venue === 'GitHub') return 'GitHub'
  if (item.source === 'arxiv' || item.venue === 'arXiv') return 'arXiv'
  if (item.source === 'openalex') return item.venue || 'OpenAlex'
  return item.venue || item.source || 'Frontier'
}

function sourceClass(item) {
  return item.source === 'github' || item.venue === 'GitHub' ? 'github' : 'paper'
}

function sourceActionLabel(item) {
  return item.source === 'github' || item.venue === 'GitHub' ? '查看仓库' : '查看原文'
}

function isTrustedItem(item) {
  const source = String(item?.source || item?.venue || '').toLowerCase()
  const link = String(item?.url || '').toLowerCase()
  if (source === 'arxiv' || source === 'github' || source === 'openalex') return true
  if (!link || link.includes('example.org') || link.includes('example.com')) return false
  return [
    'arxiv.org',
    'github.com',
    'openreview.net',
    'doi.org',
    'aclanthology.org',
    'proceedings.mlr.press',
    'neurips.cc',
    'ieee.org',
    'dl.acm.org',
  ].some((domain) => link.includes(domain))
}

function formatDateTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

function authorsText(item) {
  const authors = Array.isArray(item.authors) ? item.authors.filter(Boolean) : []
  if (!authors.length) return ''
  return authors.slice(0, 3).join('、') + (authors.length > 3 ? ' 等' : '')
}

onMounted(loadLibrary)
</script>

<template>
  <section class="frontier-library page-frame">
    <div class="frontier-library__shell">
      <header class="frontier-library__header">
        <div class="frontier-library__title">
          <span class="eyebrow">Frontier Watch</span>
          <h1>AI 前沿资料</h1>
          <p>少量精选，真实来源。每周从 arXiv 最新论文和 GitHub 公开更新中同步一次，并映射回课程知识点。</p>
          <div class="frontier-library__sync">
            <span>最近同步：{{ generatedAtText || '等待同步' }}</span>
            <span>下次更新：{{ nextRefreshText || '每 7 天' }}</span>
          </div>
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

      <p v-if="loading" class="frontier-library__notice">正在同步前沿资料...</p>
      <p v-else-if="error" class="frontier-library__notice frontier-library__notice--error">{{ error }}</p>
      <p v-else-if="library.message" class="frontier-library__notice">
        {{ library.message }}
      </p>

      <section v-if="!loading && !hasItems" class="frontier-empty-state">
        <strong>{{ emptyTitle }}</strong>
        <p>{{ emptyDescription }}</p>
      </section>

      <section v-else-if="hasItems && !hasFilteredItems" class="frontier-empty-state">
        <strong>没有匹配结果</strong>
        <p>换一个关键词或切回“全部”主题。</p>
      </section>

      <section class="frontier-library__grid">
        <article v-for="item in items" :key="item.paperId || item.title" class="frontier-paper-card">
          <div class="frontier-paper-card__head">
            <span :class="['frontier-source-pill', `frontier-source-pill--${sourceClass(item)}`]">
              {{ sourceLabel(item) }}
            </span>
            <strong>{{ item.year || '最新' }}</strong>
          </div>
          <h2>{{ item.title }}</h2>
          <div class="frontier-paper-card__meta">
            <span v-if="authorsText(item)">{{ authorsText(item) }}</span>
            <span v-if="formatDateTime(item.publishedAt || item.updatedAt)">
              {{ formatDateTime(item.publishedAt || item.updatedAt) }}
            </span>
          </div>
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
            {{ sourceActionLabel(item) }}
            <ExternalLink :size="14" aria-hidden="true" />
          </a>
        </article>
      </section>
    </div>
  </section>
</template>
