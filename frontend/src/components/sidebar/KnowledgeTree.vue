<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  courseGroups: {
    type: Array,
    default: () => [],
  },
  nodes: {
    type: Array,
    default: () => [],
  },
  progressMap: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['focus-course', 'focus-node'])
const mode = ref('course')

const typeLabelMap = {
  concept: '概念',
  method: '方法',
  algorithm: '算法',
  application: '应用',
  topic: '主题',
}

const entityGroups = computed(() => {
  const map = {}

  props.nodes.forEach((node) => {
    const entityType = String(node.entityType || 'concept').trim() || 'concept'
    if (!map[entityType]) {
      map[entityType] = {
        key: entityType,
        title: entityType,
        total: 0,
        mastered: 0,
        firstRouteKnowledgeId: node.routeKnowledgeId,
      }
    }

    map[entityType].total += 1
    if (Number(props.progressMap[node.routeKnowledgeId]?.masteryScore || 0) >= 80) {
      map[entityType].mastered += 1
    }
  })

  return Object.values(map).sort((left, right) => right.total - left.total)
})
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>知识体系</strong>
      <div class="mini-switch">
        <button type="button" :class="{ active: mode === 'course' }" @click="mode = 'course'">按学科</button>
        <button type="button" :class="{ active: mode === 'type' }" @click="mode = 'type'">按类型</button>
      </div>
    </div>

    <template v-if="mode === 'course'">
      <button
        v-for="group in courseGroups"
        :key="group.title"
        type="button"
        class="tree-item"
        @click="emit('focus-course', group.title)"
      >
        <div class="tree-item__meta">
          <strong>{{ group.title }}</strong>
          <span>{{ group.mastered }}/{{ group.total }}</span>
        </div>
        <div class="progress-track">
          <span
            class="progress-fill"
            :style="{ width: `${group.total ? (group.mastered / group.total) * 100 : 0}%` }"
          ></span>
        </div>
      </button>
    </template>

    <template v-else>
      <button
        v-for="group in entityGroups"
        :key="group.key"
        type="button"
        class="tree-item"
        @click="emit('focus-node', group.firstRouteKnowledgeId)"
      >
        <div class="tree-item__meta">
          <strong>{{ typeLabelMap[group.title] || group.title }}</strong>
          <span>{{ group.mastered }}/{{ group.total }}</span>
        </div>
        <div class="progress-track">
          <span
            class="progress-fill"
            :style="{ width: `${group.total ? (group.mastered / group.total) * 100 : 0}%` }"
          ></span>
        </div>
      </button>
    </template>
  </section>
</template>
