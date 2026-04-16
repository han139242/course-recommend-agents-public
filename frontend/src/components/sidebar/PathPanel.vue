<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  recommendedPath: {
    type: Array,
    default: () => [],
  },
  customPaths: {
    type: Array,
    default: () => [],
  },
  progressMap: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits([
  'show-path',
  'create-path',
  'remove-node',
  'reorder-path',
  'focus-node',
  'start-path',
])

const dragState = ref({
  pathId: '',
  fromIndex: -1,
})

const recommendedRouteIds = computed(() =>
  props.recommendedPath.map((item) => item.knowledgeId || item.title).filter(Boolean),
)

function getNodeMeta(routeKnowledgeId) {
  const progress = props.progressMap[String(routeKnowledgeId || '').trim()] || {}
  const mastery = Number(progress.masteryScore || 0)
  const answeredCount = Number(progress.answeredCount || 0)

  if (mastery >= 80) {
    return {
      label: '已掌握',
      tone: 'mastered',
    }
  }

  if (mastery < 30 && answeredCount > 0) {
    return {
      label: '薄弱点',
      tone: 'weak',
    }
  }

  if (mastery >= 40) {
    return {
      label: '学习中',
      tone: 'learning',
    }
  }

  return {
    label: '未学习',
    tone: 'idle',
  }
}

function getPathSummary(path) {
  const total = Array.isArray(path?.nodes) ? path.nodes.length : 0
  const completed = (path?.nodes || []).filter((nodeId) => getNodeMeta(nodeId).tone === 'mastered').length

  return {
    total,
    completed,
  }
}

function resetDragState() {
  dragState.value = {
    pathId: '',
    fromIndex: -1,
  }
}

function handleDragStart(pathId, fromIndex, event) {
  dragState.value = {
    pathId,
    fromIndex,
  }

  if (event?.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', `${pathId}:${fromIndex}`)
  }
}

function handleDrop(pathId, toIndex) {
  const { pathId: sourcePathId, fromIndex } = dragState.value
  if (!sourcePathId || sourcePathId !== pathId) {
    resetDragState()
    return
  }

  if (fromIndex < 0 || fromIndex === toIndex) {
    resetDragState()
    return
  }

  emit('reorder-path', {
    pathId,
    fromIndex,
    toIndex,
  })
  resetDragState()
}

function handleMove(pathId, fromIndex, delta) {
  emit('reorder-path', {
    pathId,
    fromIndex,
    toIndex: fromIndex + delta,
  })
}

function handleStartPath(nodeIds) {
  const routeIds = (nodeIds || []).filter(Boolean)
  if (!routeIds.length) return
  emit('start-path', routeIds)
}
</script>

<template>
  <section class="panel-card">
    <div class="panel-card__head">
      <strong>学习路径</strong>
      <button type="button" class="link-button" @click="emit('create-path')">新建</button>
    </div>

    <div class="path-block">
      <span class="panel-label">AI 推荐路径</span>

      <template v-if="recommendedRouteIds.length">
        <div class="path-chain">
          <span v-for="item in recommendedPath" :key="item.knowledgeId || item.title">
            {{ item.stage ? `${item.stage} · ` : '' }}{{ item.title }}
          </span>
        </div>

        <div class="path-block__actions">
          <button
            type="button"
            class="secondary-btn"
            @click="emit('show-path', recommendedRouteIds)"
          >
            在图谱中显示
          </button>
          <button type="button" class="primary-btn" @click="handleStartPath(recommendedRouteIds)">
            开始学习
          </button>
        </div>
      </template>

      <p v-else class="muted-text">选中知识点后，将自动尝试获取推荐路径。</p>
    </div>

    <div class="path-block">
      <span class="panel-label">我的路径</span>

      <div v-if="customPaths.length" class="panel-stack panel-stack--compact">
        <div v-for="path in customPaths" :key="path.id" class="custom-path-card">
          <div class="custom-path-card__head">
            <div class="path-card__meta">
              <strong>{{ path.name }}</strong>
              <span class="path-progress">
                {{ getPathSummary(path).completed }}/{{ getPathSummary(path).total || 0 }} 已掌握
              </span>
            </div>

            <div class="path-card__actions">
              <button type="button" class="link-button" @click="emit('show-path', path.nodes)">高亮</button>
              <button
                type="button"
                class="secondary-btn"
                :disabled="!path.nodes.length"
                @click="handleStartPath(path.nodes)"
              >
                开始学习
              </button>
            </div>
          </div>

          <div v-if="path.nodes.length" class="path-stack">
            <article
              v-for="(nodeId, index) in path.nodes"
              :key="`${path.id}-${nodeId}-${index}`"
              class="path-node"
              :class="{ 'path-node--dragging': dragState.pathId === path.id && dragState.fromIndex === index }"
              draggable="true"
              @dragstart="handleDragStart(path.id, index, $event)"
              @dragend="resetDragState"
              @dragover.prevent
              @drop="handleDrop(path.id, index)"
            >
              <button type="button" class="path-node__main" @click="emit('focus-node', nodeId)">
                <span class="path-node__handle" aria-hidden="true">⋮⋮</span>
                <span class="path-node__label">{{ nodeId }}</span>
                <span
                  class="path-node__status"
                  :class="`path-node__status--${getNodeMeta(nodeId).tone}`"
                >
                  {{ getNodeMeta(nodeId).label }}
                </span>
              </button>

              <div class="path-node__actions">
                <button
                  type="button"
                  class="path-action-btn"
                  title="上移"
                  :disabled="index === 0"
                  @click="handleMove(path.id, index, -1)"
                >
                  ↑
                </button>
                <button
                  type="button"
                  class="path-action-btn"
                  title="下移"
                  :disabled="index === path.nodes.length - 1"
                  @click="handleMove(path.id, index, 1)"
                >
                  ↓
                </button>
                <button
                  type="button"
                  class="path-action-btn path-action-btn--danger"
                  title="移除"
                  @click="emit('remove-node', { pathId: path.id, knowledgeId: nodeId })"
                >
                  ×
                </button>
              </div>
            </article>
          </div>

          <p v-else class="muted-text">右键图谱节点并选择“加入学习路径”后，会出现在这里。</p>
        </div>
      </div>

      <p v-else class="muted-text">还没有自定义路径，先从图谱里收藏几个关键节点吧。</p>
    </div>
  </section>
</template>
