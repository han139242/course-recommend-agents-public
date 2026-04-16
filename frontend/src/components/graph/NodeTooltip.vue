<script setup>
defineProps({
  visible: Boolean,
  x: Number,
  y: Number,
  node: Object,
  progress: Object,
})
</script>

<template>
  <div
    v-if="visible && node"
    class="node-tooltip"
    :style="{
      left: `${x}px`,
      top: `${y}px`,
    }"
  >
    <strong>{{ node.data?.label }}</strong>
    <span>{{ node.data?.chapterTitle || '其他课程' }}</span>
    <span>类型：{{ node.data?.entityType || 'concept' }}</span>
    <span>掌握度：{{ Math.round(progress?.masteryScore || node.data?.masteryScore || 0) }}%</span>
  </div>
</template>
