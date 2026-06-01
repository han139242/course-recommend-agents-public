<script setup>
import { ChevronLeft, ChevronRight, Library } from '@lucide/vue'

import BookmarkPanel from './BookmarkPanel.vue'
import KnowledgeTree from './KnowledgeTree.vue'
import PathPanel from './PathPanel.vue'
import ProgressPanel from './ProgressPanel.vue'

defineProps({
  collapsed: Boolean,
  courseGroups: {
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
  bookmarks: {
    type: Array,
    default: () => [],
  },
  customPaths: {
    type: Array,
    default: () => [],
  },
  recommendedPath: {
    type: Array,
    default: () => [],
  },
  isAuthenticated: Boolean,
})

const emit = defineEmits([
  'toggle',
  'focus-course',
  'focus-node',
  'highlight-status',
  'show-path',
  'bookmark-select',
  'create-path',
  'add-path-node',
  'reorder-path',
  'remove-path-node',
  'start-path',
  'rename-path',
])
</script>

<template>
  <aside class="left-panel" :class="{ 'left-panel--collapsed': collapsed }">
    <div class="panel-header">
      <div class="panel-header__title">
        <Library :size="17" aria-hidden="true" />
        <strong>学习导航</strong>
      </div>
      <button type="button" class="panel-toggle" :title="collapsed ? '展开导航' : '收起导航'" @click="emit('toggle')">
        <ChevronRight v-if="collapsed" :size="16" aria-hidden="true" />
        <ChevronLeft v-else :size="16" aria-hidden="true" />
      </button>
    </div>

    <div v-if="!collapsed" class="panel-stack">
      <div class="aside-intro">
        <strong>从课程到知识点</strong>
        <span>按课程查看知识地图，结合进度和推荐路径继续学习。</span>
      </div>

      <KnowledgeTree
        :course-groups="courseGroups"
        :nodes="nodes"
        :progress-map="progressMap"
        @focus-course="emit('focus-course', $event)"
        @focus-node="emit('focus-node', $event)"
      />

      <ProgressPanel
        :nodes="nodes"
        :progress-map="progressMap"
        :bookmarks="bookmarks"
        :is-authenticated="isAuthenticated"
        @highlight-status="emit('highlight-status', $event)"
      />

      <PathPanel
        :recommended-path="recommendedPath"
        :custom-paths="customPaths"
        :progress-map="progressMap"
        :nodes="nodes"
        @show-path="emit('show-path', $event)"
        @create-path="emit('create-path')"
        @reorder-path="emit('reorder-path', $event)"
        @remove-node="emit('remove-path-node', $event)"
        @focus-node="emit('focus-node', $event)"
        @start-path="emit('start-path', $event)"
        @rename-path="emit('rename-path', $event)"
      />

      <BookmarkPanel
        :bookmarks="bookmarks"
        :nodes="nodes"
        @select="emit('bookmark-select', $event)"
      />
    </div>
  </aside>
</template>
