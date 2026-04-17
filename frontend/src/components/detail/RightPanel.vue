<script setup>
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import BridgeView from './BridgeView.vue'
import { useUserStore } from '../../stores/userStore'
import { useGraphStore } from '../../stores/graphStore'

const props = defineProps({
  collapsed: Boolean,
  detail: Object,
  progress: Object,
  nextNodes: {
    type: Array,
    default: () => [],
  },
  recommendedPath: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['toggle', 'select-related', 'open-ai', 'show-recommend-path'])
const userStore = useUserStore()
const graphStore = useGraphStore()
const draftNote = ref('')
const entityTypeLabelMap = {
  concept: '概念',
  method: '方法',
  algorithm: '算法',
  application: '应用',
  topic: '主题',
}

watch(
  () => props.detail?.routeKnowledgeId,
  (routeKnowledgeId) => {
    draftNote.value = userStore.notes[routeKnowledgeId] || ''
  },
  { immediate: true },
)

const masteryPercent = computed(() =>
  Math.round(props.progress?.masteryScore || props.detail?.masteryScore || 0),
)

function getStatusMeta(routeKnowledgeId) {
  const progress = userStore.getProgressMeta(routeKnowledgeId)
  const mastery = Number(progress.masteryScore || 0)
  const answeredCount = Number(progress.answeredCount || 0)

  if (mastery >= 80) return '✅'
  if (mastery < 30 && answeredCount > 0) return '⚠️'
  if (mastery >= 40) return '📖'
  return '⬜'
}

const crossCourseRecommends = computed(() => {
  const currentCourse = props.detail?.chapterTitle || ''
  return (props.recommendedPath || [])
    .map((item) => {
      const node = graphStore.routeNodeMap.get(item.knowledgeId || item.title)
      return {
        ...item,
        sourceCourse: item.sourceCourse || currentCourse || '',
        targetCourse: item.targetCourse || node?.chapterTitle || '',
      }
    })
    .filter((item) => {
      if (!item?.title && !item?.knowledgeId) return false
      if (item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse) {
        return item.sourceCourse !== item.targetCourse
      }
      return Boolean(item.targetCourse && item.targetCourse !== currentCourse)
    })
    .slice(0, 3)
})

function saveNote() {
  if (!props.detail?.routeKnowledgeId) return
  userStore.saveNote(props.detail.routeKnowledgeId, draftNote.value)
}
</script>

<template>
  <aside class="right-panel" :class="{ 'right-panel--collapsed': collapsed }">
    <div class="panel-header">
      <strong>节点详情</strong>
      <button type="button" class="panel-toggle" @click="emit('toggle')">
        {{ collapsed ? '展开' : '折叠' }}
      </button>
    </div>

    <div v-if="!collapsed" class="panel-stack">
      <div v-if="detail" class="panel-card detail-card">
        <div class="detail-title-row">
          <div>
            <h2>{{ detail.canonicalName || detail.name }}</h2>
            <p>{{ detail.chapterTitle || '其他课程' }} · {{ entityTypeLabelMap[detail.entityType] || '知识点' }}</p>
          </div>
          <button type="button" class="icon-btn" @click="userStore.toggleBookmark(detail.routeKnowledgeId)">
            {{ userStore.bookmarks.includes(detail.routeKnowledgeId) ? '★' : '☆' }}
          </button>
        </div>

        <div class="detail-progress">
          <span>掌握度 {{ masteryPercent }}%</span>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${masteryPercent}%` }"></span>
          </div>
        </div>

        <p class="detail-copy">
          {{ detail.description || detail.summary || '当前知识点暂时还没有补充说明。' }}
        </p>

        <div class="detail-section">
          <span class="panel-label">前置知识</span>
          <button
            v-for="item in detail.prerequisites || []"
            :key="item"
            type="button"
            class="inline-link"
            @click="emit('select-related', item)"
          >
            {{ getStatusMeta(item) }} {{ item }}
          </button>
        </div>

        <div class="detail-section">
          <span class="panel-label">后续可学</span>
          <button
            v-for="item in nextNodes"
            :key="item.routeKnowledgeId || item"
            type="button"
            class="inline-link"
            @click="emit('select-related', item.routeKnowledgeId || item)"
          >
            {{ getStatusMeta(item.routeKnowledgeId || item) }} {{ item.label || item }}
          </button>
          <span v-if="!nextNodes.length" class="muted-text">当前还没有明确的后续可学节点。</span>
        </div>

        <div class="detail-section">
          <span class="panel-label">相关知识</span>
          <button
            v-for="item in detail.related || []"
            :key="item"
            type="button"
            class="inline-link"
            @click="emit('select-related', item)"
          >
            {{ getStatusMeta(item) }} {{ item }}
          </button>
        </div>

        <div v-if="crossCourseRecommends.length" class="detail-section recommend-section">
          <span class="panel-label">智能推荐 · 突破课程边界</span>
          <div class="recommend-card" v-for="item in crossCourseRecommends" :key="item.knowledgeId || item.title">
            <div class="recommend-card__head">
              <span class="recommend-card__badge">{{ item.reasonLabel || '推荐' }}</span>
              <span class="recommend-card__course">{{ item.stage || '跨课程' }}</span>
            </div>
            <div
              v-if="item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse"
            >
              <BridgeView :item="item" />
            </div>
            <button type="button" class="recommend-card__title" @click="emit('select-related', item.knowledgeId || item.title)">
              {{ item.title }}
            </button>
            <p v-if="item.reasonCode !== 'cross_course_transfer'" class="recommend-card__detail">
              {{ item.detail || '基于你当前的学习路径，推荐探索此知识点' }}
            </p>
            <div class="recommend-card__bar">
              <div class="progress-track">
                <span class="progress-fill" :style="{ width: `${Math.round(item.masteryScore || 0)}%` }"></span>
              </div>
              <span class="recommend-card__score">掌握 {{ Math.round(item.masteryScore || 0) }}%</span>
            </div>
          </div>
          <button
            v-if="recommendedPath.length > 3"
            type="button"
            class="inline-link--ai"
            @click="emit('show-recommend-path', recommendedPath.map(p => p.knowledgeId || p.title))"
          >
            在图谱中查看完整推荐路径 →
          </button>
        </div>

        <div class="detail-section">
          <span class="panel-label">我的笔记</span>
          <textarea v-model="draftNote" rows="5" placeholder="记录你的理解、误区和启发..."></textarea>
          <button type="button" class="secondary-btn" @click="saveNote">保存笔记</button>
        </div>

        <div class="detail-actions">
          <RouterLink class="primary-btn" :to="`/knowledge/${encodeURIComponent(detail.routeKnowledgeId)}`">
            开始学习
          </RouterLink>
          <button type="button" class="secondary-btn" @click="emit('open-ai')">AI 讲解</button>
        </div>
      </div>

      <div v-else class="panel-card empty-card empty-card--guide">
        <div class="empty-card__icon">
          <span>?</span>
        </div>
        <strong>在图谱中点选知识点</strong>
        <p>智能体将为你分析该知识点的跨课程关联，推荐突破课程边界的学习路径。</p>
        <div class="empty-card__tips">
          <span>左键点击 → 聚焦邻域</span>
          <span>右键点击 → 学习操作</span>
          <span>双击知识点 → 展开关联节点</span>
          <span>再次双击已展开节点 → 收回关联节点</span>
        </div>
      </div>
    </div>
  </aside>
</template>
