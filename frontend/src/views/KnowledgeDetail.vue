<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import 'katex/dist/katex.min.css'

import { renderWithKatex } from '../composables/useKatex'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'

const route = useRoute()
const graphStore = useGraphStore()
const userStore = useUserStore()
const frontier = ref(null)
const recommendedPath = ref([])
const draftNote = ref('')

const knowledgeId = computed(() => decodeURIComponent(route.params.knowledgeId || ''))
const detail = computed(() => graphStore.detailCache[knowledgeId.value] || null)
const progress = computed(() => userStore.getProgressMeta(knowledgeId.value))

watch(
  knowledgeId,
  (value) => {
    draftNote.value = userStore.notes[value] || ''
  },
  { immediate: true },
)

async function loadDetail() {
  await graphStore.ensureNodeDetail(knowledgeId.value)
  frontier.value = await graphStore.ensureFrontier(knowledgeId.value)
  const current = graphStore.detailCache[knowledgeId.value]
  const target = current?.chapterTitle || current?.canonicalName || current?.name || ''
  const recommendation = await graphStore.ensureRecommendedPath(target, knowledgeId.value, 'detail')
  recommendedPath.value = recommendation?.path || []
}

function saveNote() {
  userStore.saveNote(knowledgeId.value, draftNote.value)
}

onMounted(() => {
  if (!graphStore.rawNodes.length) {
    graphStore.loadFullGraph().catch(() => null)
  }
  loadDetail()
})
</script>

<template>
  <section class="detail-view page-frame">
    <header class="detail-view__head page-card">
      <div>
        <p class="eyebrow">深度学习页</p>
        <h1>{{ detail?.canonicalName || detail?.name || knowledgeId }}</h1>
        <p>{{ detail?.chapterTitle || '其他课程' }} · {{ detail?.entityType || '知识点' }}</p>
      </div>
      <div class="detail-view__actions">
        <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
        <button type="button" class="primary-btn" @click="userStore.toggleBookmark(knowledgeId)">
          {{ userStore.bookmarks.includes(knowledgeId) ? '已收藏' : '收藏' }}
        </button>
      </div>
    </header>

    <div class="detail-view__grid">
      <article class="page-card detail-article">
        <h2>知识点说明</h2>
        <div class="rich-text" v-html="renderWithKatex(detail?.summary || detail?.description || '暂无内容')"></div>

        <div class="detail-meta-list">
          <div>
            <span class="panel-label">前置知识</span>
            <p>{{ (detail?.prerequisites || []).join('、') || '暂无' }}</p>
          </div>
          <div>
            <span class="panel-label">应用场景</span>
            <p>{{ (detail?.scenarios || []).join('、') || '暂无' }}</p>
          </div>
          <div>
            <span class="panel-label">推荐理由</span>
            <p>{{ detail?.recommendReason || '建议结合课程主线继续深入。' }}</p>
          </div>
        </div>
      </article>

      <aside class="detail-side">
        <section class="page-card detail-side__card">
          <h3>学习状态</h3>
          <p>掌握度 {{ Math.round(progress.masteryScore || 0) }}%</p>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${Math.round(progress.masteryScore || 0)}%` }"></span>
          </div>
          <button type="button" class="secondary-btn" @click="userStore.markMastered(knowledgeId)">标记为已掌握</button>
        </section>

        <section class="page-card detail-side__card">
          <h3>前沿扩展</h3>
          <p>{{ frontier?.summary || frontier?.content || '前沿摘要暂未返回，后续可继续补充。' }}</p>
        </section>

        <section class="page-card detail-side__card">
          <h3>推荐路径</h3>
          <div class="path-chain">
            <span v-for="item in recommendedPath" :key="item.knowledgeId || item.title">{{ item.title }}</span>
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>我的笔记</h3>
          <textarea v-model="draftNote" rows="7" placeholder="写下自己的理解、推导过程和易错点..."></textarea>
          <button type="button" class="primary-btn" @click="saveNote">保存笔记</button>
        </section>
      </aside>
    </div>
  </section>
</template>

