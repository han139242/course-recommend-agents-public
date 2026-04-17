<script setup>
const props = defineProps({
  item: {
    type: Object,
    default: () => ({}),
  },
})
</script>

<template>
  <div class="bridge-view">
    <div class="bridge-view__rail">
      <span class="bridge-view__course">{{ item.sourceCourse || '当前课程' }}</span>
      <span class="bridge-view__arrow">→</span>
      <span class="bridge-view__course bridge-view__course--target">{{ item.targetCourse || '目标课程' }}</span>
    </div>
    <p class="bridge-view__copy">
      {{ item.detail || '系统检测到一个适合跨课程迁移的知识桥梁。' }}
    </p>
  </div>
</template>
