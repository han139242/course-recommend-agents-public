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

function formatPathName(name) {
  const colonIndex = name.indexOf('：')
  if (colonIndex !== -1 && colonIndex < name.length - 1) {
    const prefix = name.slice(0, colonIndex + 1)
    const suffix = name.slice(colonIndex + 1)
    return { prefix, suffix, hasSplit: true }
  }
  return { prefix: name, suffix: '', hasSplit: false }
}

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
      <strong>我的学习路径</strong>
      <button type="button" class="link-button" @click="emit('create-path')">新建</button>
    </div>

    <div v-if="customPaths.length" class="panel-stack panel-stack--compact">
      <article v-for="path in customPaths" :key="path.id" class="learning-path-card">
        <div class="learning-path-card__header">
          <input
            v-if="renaming.pathId === path.id"
            v-model="renaming.name"
            class="path-rename-input"
            @keydown.enter="commitRename"
            @keydown.escape="renaming = { pathId: '', name: '' }"
            @blur="commitRename"
          />
          <div v-else class="learning-path-card__title-wrapper">
            <template v-if="formatPathName(path.name).hasSplit">
              <div class="learning-path-card__title">
                {{ formatPathName(path.name).prefix }}
              </div>
              <div class="learning-path-card__title learning-path-card__title--suffix">
                {{ formatPathName(path.name).suffix }}
              </div>
            </template>
            <div v-else class="learning-path-card__title">{{ path.name }}</div>
          </div>
          <button
            v-if="renaming.pathId !== path.id"
            type="button"
            class="link-button link-button--compact"
            @click="startRename(path)"
          >
            重命名
          </button>
        </div>

        <div class="learning-path-card__progress">
          <div class="progress-bar-wrapper">
            <div class="progress-bar">
              <div
                class="progress-bar__fill"
                :style="{ width: `${(getPathSummary(path).completed / Math.max(getPathSummary(path).total, 1)) * 100}%` }"
              ></div>
            </div>
            <span class="progress-text">{{ getPathSummary(path).completed }}/{{ getPathSummary(path).total }} 已掌握</span>
          </div>
        </div>

        <div class="learning-path-card__actions">
          <button
            type="button"
            class="secondary-btn"
            :disabled="!path.nodes.length"
            @click="emit('show-path', path.nodes)"
          >
            在图谱中高亮
          </button>
          <button
            type="button"
            class="primary-btn"
            :disabled="!path.nodes.length"
            @click="handleStartPath(path.nodes)"
          >
            开始学习
          </button>
        </div>
      </article>
    </div>

    <div v-else class="empty-state">
      <p class="muted-text">还没有学习路径</p>
      <p class="muted-text" style="font-size: 0.8rem;">点击"新建"创建路径，或在右侧推荐区域保存路径</p>
    </div>
  </section>
</template>
