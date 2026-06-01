<script setup>
import { getCoursePalette } from '../../utils/colorMap'
import { normalizeCourseTitle } from '../../utils/courseMeta'

const props = defineProps({
  item: {
    type: Object,
    default: () => ({}),
  },
})

function getCourseLabel(value, fallback) {
  return normalizeCourseTitle(value, fallback)
}

function getDotStyle(courseTitle) {
  const palette = getCoursePalette(courseTitle)
  return { background: palette.primary }
}

function getChipStyle(courseTitle) {
  const palette = getCoursePalette(courseTitle)
  return { background: palette.light, color: palette.dark, borderColor: palette.ring }
}
</script>

<template>
  <div class="bridge-view">
    <div class="bridge-view__rail">
      <span class="bridge-view__course" :style="getChipStyle(item.sourceCourse)">
        <span class="bridge-view__dot" :style="getDotStyle(item.sourceCourse)"></span>
        {{ getCourseLabel(item.sourceCourse, '当前课程') }}
      </span>
      <span class="bridge-view__arrow">→</span>
      <span class="bridge-view__course bridge-view__course--target" :style="getChipStyle(item.targetCourse)">
        <span class="bridge-view__dot" :style="getDotStyle(item.targetCourse)"></span>
        {{ getCourseLabel(item.targetCourse, '目标课程') }}
      </span>
    </div>
    <p class="bridge-view__copy">
      {{ item.detail || '系统发现这里可以衔接到另一门课程，适合作为下一步学习桥梁。' }}
    </p>
  </div>
</template>
