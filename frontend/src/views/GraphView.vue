<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchGraphSubgraph, fetchKnowledgeDetail } from '../auth'
import { edgeSvgCoords, layoutNeoSubgraph, resolveKnowledgeRouteId } from '../utils/neoSubgraphLayout'

const loading = ref(true)
const errorMessage = ref('')
const selectedNodeId = ref('')
const graphData = ref({
  root: null,
  nodes: [],
  edges: [],
})
const selectedKnowledge = ref(null)

const nodeMap = computed(() =>
  Object.fromEntries(graphData.value.nodes.map((node) => [node.id, node])),
)

const selectedNode = computed(
  () => nodeMap.value[selectedNodeId.value] ?? graphData.value.root ?? graphData.value.nodes[0] ?? null,
)

const selectedDescription = computed(() => {
  if (selectedKnowledge.value?.summary) {
    return selectedKnowledge.value.summary
  }
  if (selectedNode.value?.description) {
    return selectedNode.value.description
  }
  return '当前知识点用于展示课内知识的真实图谱关系。'
})

const hasKnowledgeDetail = computed(() => Boolean(selectedKnowledge.value?.id))

const selectedRouteKnowledgeId = computed(() => resolveKnowledgeRouteId(selectedNode.value))

const layoutPack = computed(() =>
  layoutNeoSubgraph(graphData.value, selectedNodeId.value || graphData.value.root?.id),
)

const positionedNodes = computed(() => layoutPack.value.positionedNodes)

const positionedNodeMap = computed(() => layoutPack.value.nodeMap)

const visibleEdges = computed(() => layoutPack.value.edges)

function nodeStyle(node) {
  return {
    left: `${node.x}%`,
    top: `${node.y}%`,
  }
}

function edgePosition(edge) {
  return edgeSvgCoords(edge, positionedNodeMap.value)
}

async function loadGraph(root = '') {
  loading.value = true
  errorMessage.value = ''

  try {
    const data = await fetchGraphSubgraph(root, 12)
    if (!data.root && data.nodes?.length) {
      data.root = root
        ? data.nodes.find((n) => n.id === root) || data.nodes[0]
        : data.nodes[0]
    }
    graphData.value = data
    selectedNodeId.value = data.root?.id || data.nodes?.[0]?.id || ''
    const kid = resolveKnowledgeRouteId(data.root || data.nodes?.[0])
    selectedKnowledge.value = kid ? await fetchKnowledgeDetail(kid).catch(() => null) : null
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

async function focusNode(id) {
  if (!id) {
    return
  }
  await loadGraph(id)
}

onMounted(() => {
  loadGraph()
})
</script>

<template>
  <section class="graph-page">
    <div class="detail-hero">
      <span class="section-tag">知识图谱</span>
      <h1>跨课程知识点关系图</h1>
      <p>用简化关系图展示先修依赖、能力延展和场景连接，帮助用户看懂“为什么推荐这个知识点”。</p>
    </div>

    <div class="graph-layout">
      <article class="graph-canvas-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">关系可视化</span>
            <h2>点击知识点查看关联</h2>
          </div>
          <p>当前图谱已切换为真实 Neo4j 子图查询，点击节点可重新聚焦并查看关联关系。</p>
        </div>

        <div class="graph-canvas">
          <p v-if="loading" class="auth-side-copy">正在加载知识图谱...</p>
          <p v-else-if="errorMessage" class="auth-error">{{ errorMessage }}</p>
          <svg viewBox="0 0 100 100" preserveAspectRatio="none" class="graph-svg" aria-hidden="true">
            <line
              v-for="edge in visibleEdges"
              :key="edge.relationId || `${edge.sourceId}-${edge.targetId}-${edge.relationType}`"
              :x1="edgePosition(edge).x1"
              :y1="edgePosition(edge).y1"
              :x2="edgePosition(edge).x2"
              :y2="edgePosition(edge).y2"
              class="graph-edge"
            />
          </svg>

          <button
            v-for="node in positionedNodes"
            :key="node.id"
            class="graph-node"
            :class="[node.level, { active: node.id === selectedNodeId }]"
            :style="nodeStyle(node)"
            @click="focusNode(node.id)"
          >
            <strong>{{ node.label }}</strong>
            <span>{{ node.group }}</span>
          </button>
        </div>
      </article>

      <aside class="graph-side">
        <div class="detail-side-card">
          <span class="section-tag">当前焦点</span>
          <h2>{{ selectedNode?.label || selectedNode?.canonicalName || '知识点' }}</h2>
          <p>{{ selectedDescription }}</p>
          <div class="detail-mini-list">
            <div>
              <span>当前状态</span>
              <strong>{{ selectedKnowledge?.masteryText ?? '图谱关系节点' }}</strong>
            </div>
            <div>
              <span>推荐原因</span>
              <strong>
                {{
                  selectedKnowledge?.recommendReason || '该节点用于解释当前知识点与其他课内知识的真实图谱关系。'
                }}
              </strong>
            </div>
          </div>
          <RouterLink v-if="hasKnowledgeDetail" :to="`/knowledge/${selectedRouteKnowledgeId}`" class="back-home-link">
            查看知识点详情
          </RouterLink>
        </div>

        <div class="detail-side-card">
          <span class="section-tag">关系说明</span>
          <div class="graph-legend">
            <div><span class="legend-dot base"></span><strong>基础概念 / 方法</strong></div>
            <div><span class="legend-dot focus"></span><strong>当前核心节点</strong></div>
            <div><span class="legend-dot core"></span><strong>主题节点</strong></div>
            <div><span class="legend-dot project"></span><strong>应用节点</strong></div>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
