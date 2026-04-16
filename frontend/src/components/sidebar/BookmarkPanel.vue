<script setup>
import { computed } from 'vue'

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
        <span>{{ node.chapterTitle || '其他课程' }}</span>
      </button>
    </div>
    <p v-else class="muted-text">右键节点后，可以把重点知识点加入收藏。</p>
  </section>
</template>
