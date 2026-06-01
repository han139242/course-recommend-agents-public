<script setup>
import { getEntityLabel } from '../../utils/colorMap'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../../utils/courseMeta'

defineProps({
  visible: Boolean,
  x: Number,
  y: Number,
  node: Object,
  progress: Object,
})

function getCourseTitle(node) {
  return resolveNodeCourseTitle(node?.data, DEFAULT_COURSE_TITLE)
}
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
    <span>{{ getCourseTitle(node) }}</span>
    <span>类型：{{ getEntityLabel(node.data?.entityType) }}</span>
    <span>掌握度：{{ Math.round(progress?.masteryScore || node.data?.masteryScore || 0) }}%</span>
  </div>
</template>
