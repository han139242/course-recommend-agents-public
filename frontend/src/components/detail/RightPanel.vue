<script setup>
import { Bot, Bookmark, BookmarkCheck, ChevronLeft, ChevronRight, ExternalLink, MousePointerClick } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import BridgeView from './BridgeView.vue'
import { useGraphStore } from '../../stores/graphStore'
import { useUserStore } from '../../stores/userStore'
import { getEntityLabel } from '../../utils/colorMap'
import { DEFAULT_COURSE_TITLE, normalizeCourseTitle } from '../../utils/courseMeta'

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

const statusText = computed(() => {
  if (masteryPercent.value >= 80) return '已掌握'
  if (masteryPercent.value >= 40) return '学习中'
  if (Number(props.progress?.answeredCount || 0) > 0) return '待巩固'
  return '未开始'
})

function getStatusMeta(routeKnowledgeId) {
  const progress = userStore.getProgressMeta(routeKnowledgeId)
  const mastery = Number(progress.masteryScore || 0)
  const answeredCount = Number(progress.answeredCount || 0)
  if (mastery >= 80) return '已掌握'
  if (mastery < 30 && answeredCount > 0) return '待巩固'
  if (mastery >= 40) return '学习中'
  return '未开始'
}

const crossCourseRecommends = computed(() => {
  const currentCourse = normalizeCourseTitle(props.detail?.chapterTitle, DEFAULT_COURSE_TITLE)
  return (props.recommendedPath || [])
    .map((item) => {
      const node = graphStore.routeNodeMap.get(item.knowledgeId || item.title)
      const displayTitle =
        String(node?.canonicalName || node?.name || item.title || item.knowledgeId || '').trim()
      const sourceCourse = normalizeCourseTitle(item.sourceCourse || currentCourse, DEFAULT_COURSE_TITLE)
      const targetCourse = normalizeCourseTitle(item.targetCourse || node?.chapterTitle, DEFAULT_COURSE_TITLE)
      return {
        ...item,
        displayTitle,
        sourceCourse,
        targetCourse,
        courseLabel: item.reasonCode === 'cross_course_transfer'
          ? `${sourceCourse} -> ${targetCourse}`
          : targetCourse,
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

const pathRouteIds = computed(() =>
  (props.recommendedPath || []).map((item) => item.knowledgeId || item.title).filter(Boolean),
)

function saveNote() {
  if (!props.detail?.routeKnowledgeId) return
  userStore.saveNote(props.detail.routeKnowledgeId, draftNote.value)
}
</script>

<template>
  <aside class="right-panel" :class="{ 'right-panel--collapsed': collapsed }">
    <div class="panel-header">
      <div class="panel-header__title">
        <Bot :size="17" aria-hidden="true" />
        <strong>学习提示</strong>
      </div>
      <button type="button" class="panel-toggle" :title="collapsed ? '展开详情' : '收起详情'" @click="emit('toggle')">
        <ChevronLeft v-if="collapsed" :size="16" aria-hidden="true" />
        <ChevronRight v-else :size="16" aria-hidden="true" />
      </button>
    </div>

    <div v-if="!collapsed" class="panel-stack">
      <div v-if="detail" class="panel-card detail-card">
        <div class="detail-title-row">
          <div>
            <span class="status-pill" :class="`status-pill--${statusText === '已掌握' ? 'mastered' : statusText === '待巩固' ? 'weak' : statusText === '学习中' ? 'learning' : 'idle'}`">{{ statusText }}</span>
            <h2>{{ detail.canonicalName || detail.name }}</h2>
            <p>{{ detail.chapterTitle || '其他课程' }} · {{ getEntityLabel(detail.entityType) }}</p>
          </div>
          <button type="button" class="icon-btn" :title="userStore.bookmarks.includes(detail.routeKnowledgeId) ? '取消收藏' : '收藏'" @click="userStore.toggleBookmark(detail.routeKnowledgeId)">
            <BookmarkCheck v-if="userStore.bookmarks.includes(detail.routeKnowledgeId)" :size="18" aria-hidden="true" />
            <Bookmark v-else :size="18" aria-hidden="true" />
          </button>
        </div>

        <div class="detail-progress">
          <span>掌握度 {{ masteryPercent }}%</span>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${masteryPercent}%` }"></span>
          </div>
        </div>

        <p class="detail-copy">
          {{ detail.description || detail.summary || '当前知识点还没有补充说明，可以先结合前置知识和练习题建立基本理解。' }}
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
            <span>{{ getStatusMeta(item) }}</span>
            {{ item }}
          </button>
          <span v-if="!(detail.prerequisites || []).length" class="muted-text">暂无明确前置知识。</span>
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
            <span>{{ getStatusMeta(item.routeKnowledgeId || item) }}</span>
            {{ item.label || item }}
          </button>
          <span v-if="!nextNodes.length" class="muted-text">暂无明确后续节点。</span>
        </div>

        <div v-if="crossCourseRecommends.length" class="detail-section recommend-section">
          <span class="panel-label">跨课程衔接</span>
          <article class="recommend-card" v-for="item in crossCourseRecommends" :key="item.knowledgeId || item.title">
            <div class="recommend-card__head">
              <span class="recommend-card__badge">{{ item.reasonLabel || '推荐' }}</span>
              <span class="recommend-card__course">{{ item.courseLabel || item.targetCourse || '当前课程' }}</span>
            </div>
            <BridgeView
              v-if="item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse"
              :item="item"
            />
            <button type="button" class="recommend-card__title" @click="emit('select-related', item.knowledgeId || item.title)">
              {{ item.displayTitle || item.title }}
            </button>
            <p class="recommend-card__detail">
              {{ item.detail || '这一步适合作为当前知识点之后的延伸学习。' }}
            </p>
          </article>
          <button
            v-if="pathRouteIds.length"
            type="button"
            class="inline-link inline-link--ai"
            @click="emit('show-recommend-path', pathRouteIds)"
          >
            在图谱中查看完整路径
          </button>
        </div>

        <div class="detail-section">
          <span class="panel-label">我的笔记</span>
          <textarea v-model="draftNote" rows="5" placeholder="记录理解、易错点或下一步计划"></textarea>
          <button type="button" class="secondary-btn" @click="saveNote">保存笔记</button>
        </div>

        <div class="detail-actions">
          <RouterLink class="primary-btn" :to="`/knowledge/${encodeURIComponent(detail.routeKnowledgeId)}`">
            <ExternalLink :size="16" aria-hidden="true" />
            开始学习
          </RouterLink>
          <button type="button" class="secondary-btn" @click="emit('open-ai')">问 AI 助教</button>
        </div>
      </div>

      <div v-else class="panel-card empty-card empty-card--guide">
        <div class="empty-card__icon">
          <MousePointerClick :size="28" aria-hidden="true" />
        </div>
        <strong>选择一个知识点</strong>
        <p>点击图谱中的节点后，这里会显示说明、前置知识、推荐下一步和学习笔记。</p>
        <div class="empty-card__tips">
          <span>单击：查看详情</span>
          <span>双击：展开关联</span>
          <span>右键：加入路径或收藏</span>
        </div>
      </div>
    </div>
  </aside>
</template>
