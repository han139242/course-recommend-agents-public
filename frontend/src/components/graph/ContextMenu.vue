<script setup>
const props = defineProps({
  visible: Boolean,
  x: Number,
  y: Number,
  node: Object,
  bookmarked: Boolean,
})

const emit = defineEmits(['close', 'action'])

const actions = [
  { key: 'add-to-path', label: '加入学习路径', icon: '📌' },
  { key: 'toggle-bookmark', label: '收藏 / 取消收藏', icon: '⭐' },
  { key: 'note', label: '记笔记', icon: '📝' },
  { key: 'detail', label: '进入详细学习', icon: '📖' },
  { key: 'ai', label: 'AI 讲解这个知识点', icon: '🤖' },
  { key: 'chain', label: '查看前置依赖链', icon: '🔗' },
  { key: 'mastered', label: '标记为已掌握', icon: '✅' },
]

function handleAction(key) {
  emit('action', key)
}
</script>

<template>
  <div
    v-if="visible"
    class="context-menu"
    :style="{
      left: `${x}px`,
      top: `${y}px`,
    }"
  >
    <button
      v-for="action in actions"
      :key="action.key"
      type="button"
      class="context-menu__item"
      @click="handleAction(action.key)"
    >
      <span>{{ action.icon }}</span>
      <span>
        {{ action.key === 'toggle-bookmark' && bookmarked ? '取消收藏' : action.label }}
      </span>
    </button>
  </div>
</template>
