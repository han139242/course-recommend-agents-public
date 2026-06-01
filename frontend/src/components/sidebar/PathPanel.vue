<script setup>
import { computed, ref } from 'vue'

import ScoreBreakdown from '../ScoreBreakdown.vue'

const props = defineProps({
  recommendedPath: {
    type: Array,
    default: () => [],
  },
  customPaths: {
    type: Array,
    default: () => [],
  },
  progressMap: {
    type: Object,
    default: () => ({}),
  },
  nodes: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits([
  'show-path',
  'create-path',
  'remove-node',
  'reorder-path',
  'focus-node',
  'start-path',
  'rename-path',
])

const renaming = ref({ pathId: '', name: '' })

const recommendedRouteIds = computed(() =>
  props.recommendedPath.map((item) => item.knowledgeId || item.title).filter(Boolean),
)

const nodeLookup = computed(() => {
  const lookup = new Map()

  ;(props.nodes || []).forEach((node) => {
    const routeKnowledgeId = String(node?.routeKnowledgeId || '').trim()
    const fallbackId = String(node?.id || '').trim()
    const label = String(node?.canonicalName || node?.name || routeKnowledgeId || fallbackId).trim()
    const courseLabel = String(node?.chapterTitle || node?.courseTitle || '').trim()

    if (routeKnowledgeId) lookup.set(routeKnowledgeId, { label, courseLabel })
    if (fallbackId && !lookup.has(fallbackId)) lookup.set(fallbackId, { label, courseLabel })
  })

  return lookup
})

function getNodeDisplayMeta(nodeId) {
  const normalized = String(nodeId || '').trim()
  return nodeLookup.value.get(normalized) || {
    label: normalized.length > 18 ? `${normalized.slice(0, 8)}...${normalized.slice(-6)}` : normalized,
    courseLabel: '',
  }
}

function getNodeMeta(routeKnowledgeId) {
  const progress = props.progressMap[String(routeKnowledgeId || '').trim()] || {}
  const mastery = Number(progress.masteryScore || 0)
  const answeredCount = Number(progress.answeredCount || 0)

  if (mastery >= 80) return { label: '已掌握', tone: 'mastered' }
  if (mastery < 30 && answeredCount > 0) return { label: '待巩固', tone: 'weak' }
  if (mastery >= 40) return { label: '学习中', tone: 'learning' }
  return { label: '未开始', tone: 'idle' }
}

function getPathSummary(path) {
  const total = Array.isArray(path?.nodes) ? path.nodes.length : 0
  const completed = (path?.nodes || []).filter((nodeId) => getNodeMeta(nodeId).tone === 'mastered').length
  return { total, completed }
}

function startRename(path) {
  renaming.value = { pathId: path.id, name: path.name || '' }
}

function commitRename() {
  const name = renaming.value.name.trim()
  if (name) {
    emit('rename-path', { pathId: renaming.value.pathId, name })
  }
  renaming.value = { pathId: '', name: '' }
}

function handleStartPath(nodeIds) {
  const routeIds = (nodeIds || []).filter(Boolean)
  if (routeIds.length) emit('start-path', routeIds)
}

function recommendedTitle(item) {
  return String(item?.displayTitle || item?.title || item?.knowledgeId || '推荐知识点').trim()
}
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>学习路径</strong>
      <button type="button" class="link-button" @click="emit('create-path')">新建</button>
    </div>

    <div class="path-block">
      <span class="panel-label">推荐下一步</span>

      <template v-if="recommendedRouteIds.length">
        <div class="path-score-list">
          <article
            v-for="(item, idx) in recommendedPath.slice(0, 4)"
            :key="item.knowledgeId || idx"
            class="path-score-item"
          >
            <button type="button" class="path-score-item__header" @click="emit('focus-node', item.knowledgeId || item.title)">
              <span class="path-score-item__rank">{{ idx + 1 }}</span>
              <strong>{{ recommendedTitle(item) }}</strong>
              <span class="path-score-item__scoresum">{{ Math.round((item.score || 0) * 100) }}分</span>
            </button>
            <ScoreBreakdown v-if="item.scoringBreakdown" :breakdown="item.scoringBreakdown" />
          </article>
        </div>

        <div class="path-block__actions">
          <button type="button" class="secondary-btn" @click="emit('show-path', recommendedRouteIds)">高亮路径</button>
          <button type="button" class="primary-btn" @click="handleStartPath(recommendedRouteIds)">开始学习</button>
        </div>
      </template>

      <p v-else class="muted-text">选中知识点后，系统会根据当前进度给出下一步建议。</p>
    </div>

    <div class="path-block">
      <span class="panel-label">我的路径</span>

      <div v-if="customPaths.length" class="panel-stack panel-stack--compact">
        <article v-for="path in customPaths" :key="path.id" class="custom-path-card">
          <div class="custom-path-card__head">
            <div class="path-card__meta">
              <input
                v-if="renaming.pathId === path.id"
                v-model="renaming.name"
                class="path-rename-input"
                @keydown.enter="commitRename"
                @keydown.escape="renaming = { pathId: '', name: '' }"
                @blur="commitRename"
              />
              <template v-else>
                <strong>{{ path.name }}</strong>
                <button type="button" class="link-button link-button--compact" @click="startRename(path)">重命名</button>
              </template>
              <span class="path-progress">{{ getPathSummary(path).completed }}/{{ getPathSummary(path).total }} 已掌握</span>
            </div>

            <div class="path-card__actions">
              <button type="button" class="link-button" :disabled="!path.nodes.length" @click="emit('show-path', path.nodes)">高亮</button>
              <button type="button" class="secondary-btn" :disabled="!path.nodes.length" @click="handleStartPath(path.nodes)">继续</button>
            </div>
          </div>

          <div v-if="path.nodes.length" class="path-stack">
            <article v-for="nodeId in path.nodes" :key="`${path.id}-${nodeId}`" class="path-node">
              <button type="button" class="path-node__main" @click="emit('focus-node', nodeId)">
                <strong>{{ getNodeDisplayMeta(nodeId).label }}</strong>
                <span>{{ getNodeDisplayMeta(nodeId).courseLabel || getNodeMeta(nodeId).label }}</span>
              </button>
              <span class="status-pill" :class="`status-pill--${getNodeMeta(nodeId).tone}`">{{ getNodeMeta(nodeId).label }}</span>
              <button type="button" class="path-node__remove" @click="emit('remove-node', { pathId: path.id, knowledgeId: nodeId })">移除</button>
            </article>
          </div>
          <p v-else class="muted-text">右键知识点可加入这条路径。</p>
        </article>
      </div>

      <p v-else class="muted-text">还没有自定义路径。可以先用推荐路径开始学习。</p>
    </div>
  </section>
</template>
