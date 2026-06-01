<script setup>
import { computed } from 'vue'

import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../../utils/courseMeta'

const props = defineProps({
  bookmarks: {
    type: Array,
    default: () => [],
  },
  nodes: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['select'])

const bookmarkNodes = computed(() =>
  props.bookmarks
    .map((bookmark) => props.nodes.find((node) => node.routeKnowledgeId === bookmark))
    .filter(Boolean),
)

function getCourseTitle(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>收藏夹</strong>
      <span>{{ bookmarkNodes.length }} 个</span>
    </div>

    <div v-if="bookmarkNodes.length" class="bookmark-list">
      <button
        v-for="node in bookmarkNodes"
        :key="node.id"
        type="button"
        class="bookmark-item"
        @click="emit('select', node.routeKnowledgeId)"
      >
        <strong>{{ node.canonicalName || node.name }}</strong>
        <span>{{ getCourseTitle(node) }}</span>
      </button>
    </div>
    <p v-else class="muted-text">遇到重点知识点时，可以右键加入收藏，后续从这里快速回到学习位置。</p>
  </section>
</template>
