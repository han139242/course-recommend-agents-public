<script setup>
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
])
</script>

<template>
  <aside class="left-panel" :class="{ 'left-panel--collapsed': collapsed }">
    <div class="panel-header">
      <strong>学习导航</strong>
      <button type="button" class="panel-toggle" @click="emit('toggle')">
        {{ collapsed ? '展开' : '折叠' }}
      </button>
    </div>

    <div v-if="!collapsed" class="panel-stack">
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
        @show-path="emit('show-path', $event)"
        @create-path="emit('create-path')"
        @reorder-path="emit('reorder-path', $event)"
        @remove-node="emit('remove-path-node', $event)"
        @focus-node="emit('focus-node', $event)"
        @start-path="emit('start-path', $event)"
      />

      <BookmarkPanel
        :bookmarks="bookmarks"
        :nodes="nodes"
        @select="emit('bookmark-select', $event)"
      />
    </div>
  </aside>
</template>
