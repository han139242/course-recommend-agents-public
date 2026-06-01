<script setup>
import { computed, ref } from 'vue'

import ContextMenu from './ContextMenu.vue'
import GraphControls from './GraphControls.vue'
import NodeTooltip from './NodeTooltip.vue'
import { useGraph } from '../../composables/useGraph'
import { useUserStore } from '../../stores/userStore'

const props = defineProps({
  graphData: {
    type: Object,
    required: true,
  },
  stateMap: {
    type: Object,
    required: true,
  },
  highlightPath: {
    type: Array,
    default: () => [],
  },
  selectedNodeId: {
    type: String,
    default: null,
  },
  loading: Boolean,
  error: String,
})

const emit = defineEmits([
  'select-node',
  'node-double-click',
  'hub-click',
  'hub-double-click',
  'clear-selection',
  'context-action',
])

const userStore = useUserStore()
const tooltip = ref({ visible: false, x: 0, y: 0, node: null, progress: null })
const contextMenu = ref({ visible: false, x: 0, y: 0, node: null })
const legendOpen = ref(false)

function findGraphNode(nodeId) {
  return props.graphData.nodes.find((item) => item.id === nodeId) || null
}

function isKnowledgeNode(node) {
  return Boolean(node?.data?.routeKnowledgeId) && !node?.data?.isHub && !node?.data?.synthetic
}

function getRelativePoint(event) {
  const bounds = containerRef.value?.getBoundingClientRect()
  if (!bounds) return { x: 0, y: 0 }
  const clientX = event?.client?.x ?? event?.clientX ?? bounds.left
  const clientY = event?.client?.y ?? event?.clientY ?? bounds.top
  return {
    x: clientX - bounds.left + 12,
    y: clientY - bounds.top + 12,
  }
}

function handleNodeEnter(event) {
  const node = findGraphNode(event.target.id)
  if (!isKnowledgeNode(node)) {
    tooltip.value.visible = false
    return
  }
  const point = getRelativePoint(event)
  tooltip.value = {
    visible: true,
    x: point.x,
    y: point.y,
    node,
    progress: userStore.getProgressMeta(node?.data?.routeKnowledgeId),
  }
}

function handleNodeLeave() {
  tooltip.value.visible = false
}

function handleNodeContextMenu(event) {
  event.preventDefault?.()
  const node = findGraphNode(event.target.id)
  if (!isKnowledgeNode(node)) return
  const point = getRelativePoint(event)
  contextMenu.value = {
    visible: true,
    x: point.x,
    y: point.y,
    node,
  }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function handleContextAction(action) {
  emit('context-action', {
    action,
    node: contextMenu.value.node,
  })
  closeContextMenu()
}

async function toggleFullscreen() {
  if (!containerRef.value) return
  if (document.fullscreenElement) {
    await document.exitFullscreen()
    return
  }
  await containerRef.value.requestFullscreen()
}

const {
  containerRef,
  graphMountRef,
  renderError,
  zoom,
  fitView,
  centerGraph,
  centerElement,
  queueViewportFocus,
  zoomIn,
  zoomOut,
  focusElement,
  softFollowElement,
  cancelViewportAutomation,
  flashNode,
  getNodeSnapshot,
  getNodeSnapshots,
  getViewportCenterWorld,
  syncNow,
} = useGraph({
  data: computed(() => props.graphData),
  stateMap: computed(() => props.stateMap),
  selectedNodeId: computed(() => props.selectedNodeId),
  onNodeClick: (event) => {
    closeContextMenu()
    const node = findGraphNode(event.target.id)
    if (!isKnowledgeNode(node)) return
    emit('select-node', event.target.id)
  },
  onNodeDoubleClick: (event) => {
    closeContextMenu()
    const node = findGraphNode(event.target.id)
    if (!isKnowledgeNode(node)) return
    emit('node-double-click', event.target.id)
  },
  onHubClick: (event) => {
    closeContextMenu()
    tooltip.value.visible = false
    emit('hub-click', event.target.id)
  },
  onHubDoubleClick: (event) => {
    closeContextMenu()
    tooltip.value.visible = false
    emit('hub-double-click', event.target.id)
  },
  onCanvasClick: () => {
    closeContextMenu()
    tooltip.value.visible = false
    emit('clear-selection')
  },
  onNodeEnter: handleNodeEnter,
  onNodeLeave: handleNodeLeave,
  onNodeContextMenu: handleNodeContextMenu,
})

defineExpose({
  fitView,
  centerGraph,
  centerElement,
  queueViewportFocus,
  flashNode,
  closeContextMenu,
  focusElement,
  softFollowElement,
  cancelViewportAutomation,
  getNodeSnapshot,
  getNodeSnapshots,
  getViewportCenterWorld,
  syncNow,
})
</script>

<template>
  <div ref="containerRef" class="graph-canvas-shell">
    <div v-if="loading" class="graph-canvas-status">
      <div class="graph-canvas-status__inner">
        <div class="status-spinner"></div>
        <strong>知识地图加载中</strong>
        <span>正在同步课程目录和知识点关系。</span>
      </div>
    </div>
    <div v-else-if="error || renderError" class="graph-canvas-status graph-canvas-status--error">
      <div class="graph-canvas-status__inner">
        <strong>{{ renderError ? '图谱渲染失败' : '数据加载失败' }}</strong>
        <p>{{ renderError || error }}</p>
        <p class="muted-text">请确认后端服务、图数据库和浏览器 Canvas 能正常使用。</p>
      </div>
    </div>

    <canvas ref="graphMountRef" class="graph-canvas-stage"></canvas>

    <GraphControls
      :zoom="zoom"
      @fit="fitView"
      @zoom-in="zoomIn"
      @zoom-out="zoomOut"
      @fullscreen="toggleFullscreen"
    />

    <div class="graph-overlay-stack">
      <button
        type="button"
        class="graph-legend-toggle"
        :aria-expanded="legendOpen ? 'true' : 'false'"
        @click="legendOpen = !legendOpen"
      >
        <span class="graph-legend-toggle__title">图例</span>
        <span class="graph-legend-toggle__meta">{{ legendOpen ? '收起' : '展开' }}</span>
      </button>

      <div v-if="legendOpen" class="graph-legend">
        <div class="graph-legend__group">
          <span class="graph-legend__title">学习状态</span>
          <span class="graph-legend__item"><i class="graph-dot graph-dot--mastered"></i>已掌握</span>
          <span class="graph-legend__item"><i class="graph-dot graph-dot--learning"></i>学习中</span>
          <span class="graph-legend__item"><i class="graph-dot graph-dot--weak"></i>待巩固</span>
        </div>
        <div class="graph-legend__group">
          <span class="graph-legend__title">操作</span>
          <span class="graph-legend__item">单击查看详情</span>
          <span class="graph-legend__item">双击展开关联节点</span>
          <span class="graph-legend__item">右键加入路径或收藏</span>
        </div>
      </div>
    </div>

    <NodeTooltip
      :visible="tooltip.visible"
      :x="tooltip.x"
      :y="tooltip.y"
      :node="tooltip.node"
      :progress="tooltip.progress"
    />

    <ContextMenu
      :visible="contextMenu.visible"
      :x="contextMenu.x"
      :y="contextMenu.y"
      :node="contextMenu.node"
      :bookmarked="userStore.bookmarks.includes(contextMenu.node?.data?.routeKnowledgeId)"
      @action="handleContextAction"
      @close="closeContextMenu"
    />
  </div>
</template>
