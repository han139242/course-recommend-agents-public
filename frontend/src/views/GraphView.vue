<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import { authState, fetchGraphSubgraph, fetchKnowledgeDetail, fetchLearningProgress } from '../auth'
import { graphEdges, graphNodes, knowledgeDetails } from '../data/learning'
import { edgeSvgCoords, layoutNeoSubgraph, resolveKnowledgeRouteId } from '../utils/neoSubgraphLayout'
import { buildKnowledgeLookupCandidates, resolveKnowledgeCatalogEntry } from '../utils/knowledgeLookup'
import { buildDisplayDomains } from '../utils/learningAnalytics'
import { buildKnowledgeGraphContribution, getKnowledgeMasteryProfile } from '../utils/knowledgeExperience'

const loading = ref(true)
const errorMessage = ref('')
const selectedNodeId = ref('')
const graphData = ref({
  root: null,
  nodes: [],
  edges: [],
})
const selectedKnowledge = ref(null)
const progressItems = ref([])
const usingLocalFallback = ref(false)

const relationRank = {
  prerequisite: 0,
  related: 1,
  topic: 2,
  next: 3,
  application: 4,
}

const nodeMap = computed(() =>
  Object.fromEntries((graphData.value.nodes || []).map((node) => [node.id, node])),
)

const selectedNode = computed(
  () => nodeMap.value[selectedNodeId.value] ?? graphData.value.root ?? graphData.value.nodes[0] ?? null,
)

const selectedName = computed(
  () => selectedKnowledge.value?.name || selectedNode.value?.canonicalName || selectedNode.value?.name || '知识点',
)

const selectedRouteKnowledgeId = computed(() => resolveKnowledgeRouteId(selectedNode.value))
const hasKnowledgeDetail = computed(() => Boolean(selectedRouteKnowledgeId.value))

const layoutPack = computed(() =>
  layoutNeoSubgraph(graphData.value, selectedNodeId.value || graphData.value.root?.id),
)

const positionedNodes = computed(() => layoutPack.value.positionedNodes || [])
const positionedNodeMap = computed(() => layoutPack.value.nodeMap || {})
const visibleEdges = computed(() => layoutPack.value.edges || [])

const displayDomains = computed(() => buildDisplayDomains(progressItems.value))
const localPointMap = computed(() =>
  Object.fromEntries(
    displayDomains.value.flatMap((domain) =>
      domain.points.map((point) => [point.id, { ...point, domainName: domain.name, domainId: domain.id }]),
    ),
  ),
)
const progressMap = computed(() =>
  Object.fromEntries(
    (progressItems.value || []).map((item) => [resolveKnowledgeRouteId({ knowledgeId: item.knowledgeId, canonicalName: item.canonicalName, name: item.canonicalName }), item]),
  ),
)
const selectedLocalPoint = computed(
  () => localPointMap.value[selectedRouteKnowledgeId.value] || localPointMap.value[selectedNodeId.value] || null,
)

const selectedProgress = computed(
  () => progressMap.value[selectedRouteKnowledgeId.value] || progressMap.value[selectedNodeId.value] || null,
)

const selectedMasteryScore = computed(() =>
  Math.round(Math.max(0, Math.min(100, Number(selectedProgress.value?.masteryScore ?? selectedLocalPoint.value?.mastery ?? 0)))),
)

const selectedMasteryLabel = computed(() => (
  authState.token ? `${selectedMasteryScore.value}%` : '登录后可见'
))
const selectedMasteryProfile = computed(() => getKnowledgeMasteryProfile(selectedLocalPoint.value || selectedKnowledge.value || {}, selectedMasteryScore.value))
const graphContribution = computed(() => buildKnowledgeGraphContribution(selectedLocalPoint.value || selectedKnowledge.value || {}, selectedKnowledge.value))

const selectedStudyStatus = computed(() => {
  if (!authState.token) return '游客模式'
  if (!selectedProgress.value && !selectedLocalPoint.value) return '暂无作答记录'
  if (selectedMasteryScore.value >= 80) return '已掌握'
  if (!selectedProgress.value && selectedMasteryScore.value > 0) return '建议优先学习'
  return '学习中'
})

const selectedDescription = computed(() => {
  if (selectedKnowledge.value?.summary) {
    return selectedKnowledge.value.summary
  }
  if (selectedNode.value?.description) {
    return selectedNode.value.description
  }
  return '这里会把当前知识点和它周围的先修、关联、延展节点连起来，帮助你直接决定下一步学什么。'
})

function localPointForId(id = '') {
  return localPointMap.value[String(id || '').trim()] || null
}

function buildLocalGraphData(rootId = 'knowledge-graph') {
  const normalizedRoot = graphNodes.find((node) => node.id === rootId) ? rootId : 'knowledge-graph'
  const neighborIds = new Set([normalizedRoot])

  for (const edge of graphEdges) {
    if (edge.from === normalizedRoot || edge.to === normalizedRoot) {
      neighborIds.add(edge.from)
      neighborIds.add(edge.to)
    }
  }

  const mappedNodes = graphNodes
    .filter((node) => neighborIds.has(node.id))
    .map((node) => {
      const point = localPointForId(node.id)
      return {
        id: node.id,
        routeKnowledgeId: node.id,
        name: point?.name || node.label,
        canonicalName: point?.name || node.label,
        entityType: node.group === '行业实践' ? 'application' : 'concept',
        description: point?.description || knowledgeDetails[node.id]?.summary || '',
        chapterTitle: point?.domainName || node.group,
        sourcePath: 'local-learning-map',
      }
    })

  const mappedEdges = graphEdges
    .filter((edge) => neighborIds.has(edge.from) && neighborIds.has(edge.to))
    .map((edge, index) => ({
      relationId: `local-${index}-${edge.from}-${edge.to}`,
      relationType:
        edge.label === '先修'
          ? 'PREREQUISITE_OF'
          : edge.label === '应用'
            ? 'APPLIES_TO'
            : edge.label === '训练基础'
              ? 'PREREQUISITE_OF'
              : 'RELATED_TO',
      sourceId: edge.from,
      targetId: edge.to,
      sourceName: mappedNodes.find((node) => node.id === edge.from)?.name || edge.from,
      targetName: mappedNodes.find((node) => node.id === edge.to)?.name || edge.to,
    }))

  const rootNode = mappedNodes.find((node) => node.id === normalizedRoot) || mappedNodes[0] || null
  return {
    root: rootNode,
    nodes: mappedNodes,
    edges: mappedEdges,
  }
}

function entityTypeLabel(type = '') {
  if (type === 'algorithm') return '算法'
  if (type === 'method') return '方法'
  if (type === 'concept') return '概念'
  if (type === 'application') return '应用'
  if (type === 'topic') return '主题'
  return '知识点'
}

function relationInfoForEdge(edge, focusId = selectedNodeId.value) {
  if (!edge) {
    return { label: '关联', group: 'related' }
  }

  const incoming = edge.targetId === focusId

  if (edge.relationType === 'PREREQUISITE_OF') {
    return incoming
      ? { label: '先修', group: 'prerequisite' }
      : { label: '后续', group: 'next' }
  }

  if (edge.relationType === 'RELATED_TO') {
    return { label: '关联', group: 'related' }
  }

  if (edge.relationType === 'SUBTOPIC_OF') {
    return incoming
      ? { label: '下属主题', group: 'topic' }
      : { label: '所属主题', group: 'topic' }
  }

  return {
    label: edge.relationType ? edge.relationType.replaceAll('_', ' ') : '关联',
    group: 'related',
  }
}

const edgeGroupMap = computed(() => {
  const map = {}

  for (const edge of graphData.value.edges || []) {
    if (edge.sourceId !== selectedNodeId.value && edge.targetId !== selectedNodeId.value) {
      continue
    }

    const neighborId = edge.sourceId === selectedNodeId.value ? edge.targetId : edge.sourceId
    if (!neighborId || map[neighborId]) {
      continue
    }

    map[neighborId] = relationInfoForEdge(edge)
  }

  return map
})

function nodeForLabel(label = '') {
  const trimmed = String(label || '').trim()
  if (!trimmed) return null

  return (
    graphData.value.nodes.find((node) =>
      [
        node.id,
        node.routeKnowledgeId,
        node.name,
        node.canonicalName,
      ].some((value) => String(value || '').trim() === trimmed),
    ) || null
  )
}

function routeForLabel(label = '') {
  const trimmed = String(label || '').trim()
  if (!trimmed) return ''

  const matched = nodeForLabel(trimmed)
  const routeId = resolveKnowledgeRouteId(
    matched || {
      id: trimmed,
      routeKnowledgeId: trimmed,
      name: trimmed,
      canonicalName: trimmed,
    },
  )

  return routeId ? `/knowledge/${routeId}` : ''
}

function makeGuideItem(label, fallbackHint = '') {
  const trimmed = String(label || '').trim()
  if (!trimmed) return null

  const node = nodeForLabel(trimmed)
  const route = routeForLabel(trimmed)

  return {
    label: node?.canonicalName || node?.name || trimmed,
    hint: fallbackHint || node?.description || `${entityTypeLabel(node?.entityType)}节点`,
    route,
    focusId: node?.id || '',
  }
}

function dedupeGuideItems(items) {
  const seen = new Set()
  return items.filter(Boolean).filter((item) => {
    const key = `${item.label}|${item.route}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

const adjacentNodes = computed(() =>
  positionedNodes.value
    .filter((node) => node.id !== selectedNodeId.value)
    .map((node) => {
      const relation = edgeGroupMap.value[node.id] || {
        label: node.entityType === 'application' ? '应用' : '关联',
        group: node.entityType === 'application' ? 'application' : 'related',
      }

      return {
        ...node,
        relationLabel: relation.label,
        relationGroup: relation.group,
        entityLabel: entityTypeLabel(node.entityType || node.group),
      }
    })
    .sort((left, right) => {
      const leftRank = relationRank[left.relationGroup] ?? 99
      const rightRank = relationRank[right.relationGroup] ?? 99
      if (leftRank !== rightRank) return leftRank - rightRank
      return left.label.localeCompare(right.label, 'zh-CN')
    }),
)

const prerequisiteItems = computed(() =>
  dedupeGuideItems([
    ...(selectedKnowledge.value?.prerequisites || []).map((label) =>
      makeGuideItem(label, '建议先补这个先修点'),
    ),
    ...adjacentNodes.value
      .filter((node) => node.relationGroup === 'prerequisite')
      .map((node) => makeGuideItem(node.label, '这是当前知识点的先修内容')),
  ]),
)

const relatedItems = computed(() =>
  dedupeGuideItems([
    ...(selectedKnowledge.value?.related || []).map((label) =>
      makeGuideItem(label, '建议和当前知识点一起理解'),
    ),
    ...adjacentNodes.value
      .filter((node) => node.relationGroup === 'related')
      .map((node) => makeGuideItem(node.label, '可以用来一起对照理解')),
  ]),
)

const extensionItems = computed(() =>
  dedupeGuideItems([
    ...(selectedKnowledge.value?.parentTopics || []).map((label) =>
      makeGuideItem(label, '它能帮你看清这个知识点属于哪条主题主线'),
    ),
    ...adjacentNodes.value
      .filter((node) => node.relationGroup === 'topic' || node.relationGroup === 'next' || node.entityType === 'application')
      .map((node) =>
        makeGuideItem(
          node.label,
          node.relationGroup === 'topic'
            ? '这是当前知识点所在的主题主线'
            : '学完后可以继续延展到这里',
        ),
      ),
  ]),
)

const summaryCards = computed(() => [
  { label: '当前状态', value: selectedStudyStatus.value },
  { label: '掌握度', value: selectedMasteryLabel.value },
  { label: '先修项', value: String(prerequisiteItems.value.length) },
  { label: '可切换节点', value: String(adjacentNodes.value.length) },
])

const primaryAction = computed(() => {
  if (prerequisiteItems.value.length && selectedMasteryScore.value < 60) {
    const first = prerequisiteItems.value[0]
    return {
      title: '先补先修内容',
      detail: `当前更建议先补 ${first.label}，把基础打通后再回来学 ${selectedName.value}。`,
      route: first.route,
      cta: '先去补这个知识点',
    }
  }

  if (relatedItems.value.length) {
    const first = relatedItems.value[0]
    return {
      title: '一起对照理解',
      detail: `可以把 ${selectedName.value} 和 ${first.label} 放在一起看，理解会更连贯。`,
      route: first.route,
      cta: '查看关联知识点',
    }
  }

  if (extensionItems.value.length) {
    const first = extensionItems.value[0]
    return {
      title: '继续往后延展',
      detail: `学完当前内容后，可以顺着 ${first.label} 继续往后推进。`,
      route: first.route,
      cta: '继续下一步',
    }
  }

  if (hasKnowledgeDetail.value) {
    return {
      title: '先看知识点详情',
      detail: '进入详情页查看学习摘要、AI 问答和更完整的知识说明。',
      route: `/knowledge/${selectedRouteKnowledgeId.value}`,
      cta: '查看知识点详情',
    }
  }

  return {
    title: '从当前节点开始',
    detail: '当前子图信息较少，可以先从这个节点本身开始了解。',
    route: '',
    cta: '',
  }
})

function nodeStyle(node) {
  return {
    left: `${node.x}%`,
    top: `${node.y}%`,
  }
}

function edgePosition(edge) {
  return edgeSvgCoords(edge, positionedNodeMap.value)
}

function nodeSubtitle(node) {
  if (node.id === selectedNodeId.value) return '当前中心'

  const relation = edgeGroupMap.value[node.id]
  if (relation) {
    return `${relation.label} · ${entityTypeLabel(node.entityType || node.group)}`
  }

  return entityTypeLabel(node.entityType || node.group)
}

async function loadKnowledgeSnapshot(node) {
  const routeKnowledgeId = resolveKnowledgeRouteId(node)
  const localPoint = localPointForId(routeKnowledgeId)
  const localDetail = routeKnowledgeId ? knowledgeDetails[routeKnowledgeId] || null : null

  if (!routeKnowledgeId) {
    selectedKnowledge.value = null
    return
  }

  if (localPoint || usingLocalFallback.value) {
    selectedKnowledge.value = localDetail
      ? {
          routeKnowledgeId,
          id: routeKnowledgeId,
          name: localPoint?.name || localDetail.title,
          chapterTitle: localPoint?.domainName || localDetail.domain,
          summary: localDetail.summary,
          description: localPoint?.description || localDetail.summary,
          prerequisites: localDetail.prerequisites || [],
          related: localDetail.related || [],
          parentTopics: [localPoint?.domainName || localDetail.domain].filter(Boolean),
        }
      : (localPoint
          ? {
              routeKnowledgeId,
              id: routeKnowledgeId,
              name: localPoint.name,
              chapterTitle: localPoint.domainName,
              summary: localPoint.description,
              description: localPoint.description,
              prerequisites: [],
              related: [],
              parentTopics: [localPoint.domainName],
            }
          : null)
    return
  }

  selectedKnowledge.value = await fetchKnowledgeDetail(routeKnowledgeId).catch(() => (
    localDetail
      ? {
          routeKnowledgeId,
          id: routeKnowledgeId,
          name: localPoint?.name || localDetail.title,
          chapterTitle: localPoint?.domainName || localDetail.domain,
          summary: localDetail.summary,
          description: localPoint?.description || localDetail.summary,
          prerequisites: localDetail.prerequisites || [],
          related: localDetail.related || [],
          parentTopics: [localPoint?.domainName || localDetail.domain].filter(Boolean),
        }
      : (localPoint
          ? {
              routeKnowledgeId,
              id: routeKnowledgeId,
              name: localPoint.name,
              chapterTitle: localPoint.domainName,
              summary: localPoint.description,
              description: localPoint.description,
              prerequisites: [],
              related: [],
              parentTopics: [localPoint.domainName],
            }
          : null)
  ))
}

async function loadGraph(root = '') {
  loading.value = true
  errorMessage.value = ''
  usingLocalFallback.value = false

  try {
    const preferredRoot = String(root || '').trim()
    if (!preferredRoot) {
      const localData = buildLocalGraphData('knowledge-graph')
      graphData.value = localData
      selectedNodeId.value = localData.root?.id || localData.nodes?.[0]?.id || ''
      usingLocalFallback.value = true
      await loadKnowledgeSnapshot(localData.root || localData.nodes?.[0] || null)
      return
    }

    if (graphNodes.some((node) => node.id === preferredRoot)) {
      const localData = buildLocalGraphData(preferredRoot)
      graphData.value = localData
      selectedNodeId.value = localData.root?.id || localData.nodes?.[0]?.id || ''
      usingLocalFallback.value = true
      await loadKnowledgeSnapshot(localData.root || localData.nodes?.[0] || null)
      return
    }

    const localPoint = resolveKnowledgeCatalogEntry(preferredRoot)
    const candidates = buildKnowledgeLookupCandidates({
      knowledgeId: preferredRoot,
      currentPoint: localPoint ? { id: localPoint.id, name: localPoint.name } : null,
      detailSnapshot: selectedKnowledge.value,
    })

    for (const candidate of candidates) {
      const data = await fetchGraphSubgraph(candidate, 12).catch(() => null)
      if (!data?.nodes?.length) {
        continue
      }

      if (!data.root && data.nodes?.length) {
        data.root = data.nodes.find((node) => node.id === candidate) || data.nodes[0]
      }

      graphData.value = data
      selectedNodeId.value = data.root?.id || data.nodes?.[0]?.id || ''
      await loadKnowledgeSnapshot(data.root || data.nodes?.[0] || null)
      return
    }

    const localData = buildLocalGraphData(localPoint?.id || preferredRoot)
    graphData.value = localData
    selectedNodeId.value = localData.root?.id || localData.nodes?.[0]?.id || ''
    usingLocalFallback.value = true
    await loadKnowledgeSnapshot(localData.root || localData.nodes?.[0] || null)
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

async function loadProgress() {
  if (!authState.token) {
    progressItems.value = []
    return
  }

  try {
    const data = await fetchLearningProgress()
    progressItems.value = data.list || []
  } catch {
    progressItems.value = []
  }
}

async function focusNode(id) {
  if (!id) return
  await loadGraph(id)
}

onMounted(() => {
  loadProgress()
  loadGraph()
})

watch(() => authState.token, () => {
  loadProgress()
})
</script>

<template>
  <section class="graph-page">
    <div class="detail-hero">
      <span class="section-tag">学习路径</span>
      <h1>把知识图谱变成可执行的学习路径</h1>
      <p>优先展示真正能指导学习动作的路径结构；图数据库里暂时没收录的部分，就退回到课程主线继续给你可用导航。</p>
    </div>

    <div class="graph-summary-grid">
      <article v-for="card in summaryCards" :key="card.label" class="stat-card">
        <span>{{ card.label }}</span>
        <strong>{{ card.value }}</strong>
      </article>
    </div>

    <div class="graph-layout">
      <article class="graph-canvas-card graph-workbench">
        <div class="section-head compact">
          <div>
            <span class="section-tag">学习导航台</span>
            <h2>{{ selectedName }}</h2>
          </div>
          <p>{{ usingLocalFallback ? '当前使用课程主线路径兜底，先保证导航可用，再补图谱扩展关系。' : '先看最需要动作的部分，再用下面的关系速览辅助定位。' }}</p>
        </div>

        <div class="graph-focus-panel">
          <div class="graph-focus-copy">
            <div class="graph-focus-tags">
              <span class="graph-pill graph-pill--neutral">{{ entityTypeLabel(selectedNode?.entityType) }}</span>
              <span v-if="selectedKnowledge?.chapterTitle || selectedNode?.chapterTitle" class="graph-pill graph-pill--soft">
                {{ selectedKnowledge?.chapterTitle || selectedNode?.chapterTitle }}
              </span>
            </div>
            <p class="graph-focus-desc">{{ selectedDescription }}</p>
          </div>

          <div class="graph-focus-stats">
            <div class="graph-focus-stat">
              <span>学习状态</span>
              <strong>{{ selectedMasteryProfile.stageLabel }}</strong>
            </div>
            <div class="graph-focus-stat">
              <span>掌握度</span>
              <strong>{{ selectedMasteryLabel }}</strong>
            </div>
            <div class="graph-focus-stat">
              <span>连接节点</span>
              <strong>{{ adjacentNodes.length }}</strong>
            </div>
          </div>
        </div>

        <div class="graph-guide-grid">
          <article class="graph-guide-card">
            <div class="graph-guide-head">
              <span class="section-tag">先学什么</span>
              <h3>先修内容</h3>
            </div>
            <p v-if="prerequisiteItems.length === 0" class="graph-guide-empty">当前子图里还没有明确的先修项。</p>
            <div v-else class="graph-link-list">
              <div v-for="item in prerequisiteItems" :key="`pre-${item.label}`" class="graph-link-item">
                <div class="graph-link-main">
                  <RouterLink v-if="item.route" :to="item.route" class="graph-link-title">{{ item.label }}</RouterLink>
                  <strong v-else class="graph-link-title">{{ item.label }}</strong>
                  <span>{{ item.hint }}</span>
                </div>
                <button
                  v-if="item.focusId && item.focusId !== selectedNodeId"
                  class="graph-chip-btn"
                  @click="focusNode(item.focusId)"
                >
                  聚焦
                </button>
              </div>
            </div>
          </article>

          <article class="graph-guide-card">
            <div class="graph-guide-head">
              <span class="section-tag">一起理解</span>
              <h3>关联知识</h3>
            </div>
            <p v-if="relatedItems.length === 0" class="graph-guide-empty">当前没有抓到特别明确的关联节点。</p>
            <div v-else class="graph-link-list">
              <div v-for="item in relatedItems" :key="`rel-${item.label}`" class="graph-link-item">
                <div class="graph-link-main">
                  <RouterLink v-if="item.route" :to="item.route" class="graph-link-title">{{ item.label }}</RouterLink>
                  <strong v-else class="graph-link-title">{{ item.label }}</strong>
                  <span>{{ item.hint }}</span>
                </div>
                <button
                  v-if="item.focusId && item.focusId !== selectedNodeId"
                  class="graph-chip-btn"
                  @click="focusNode(item.focusId)"
                >
                  聚焦
                </button>
              </div>
            </div>
          </article>

          <article class="graph-guide-card">
            <div class="graph-guide-head">
              <span class="section-tag">学完去哪</span>
              <h3>延展方向</h3>
            </div>
            <p v-if="extensionItems.length === 0" class="graph-guide-empty">当前没有更远一步的延展信息。</p>
            <div v-else class="graph-link-list">
              <div v-for="item in extensionItems" :key="`ext-${item.label}`" class="graph-link-item">
                <div class="graph-link-main">
                  <RouterLink v-if="item.route" :to="item.route" class="graph-link-title">{{ item.label }}</RouterLink>
                  <strong v-else class="graph-link-title">{{ item.label }}</strong>
                  <span>{{ item.hint }}</span>
                </div>
                <button
                  v-if="item.focusId && item.focusId !== selectedNodeId"
                  class="graph-chip-btn"
                  @click="focusNode(item.focusId)"
                >
                  聚焦
                </button>
              </div>
            </div>
          </article>
        </div>

        <div class="graph-canvas-shell">
          <div class="graph-canvas-top">
            <div>
              <span class="section-tag">关系速览</span>
              <h3>点击节点快速切换中心</h3>
            </div>
            <p>{{ usingLocalFallback ? '这部分当前展示课程主线关系，避免默认跳到一个不相关的图谱节点。' : '关系图现在只负责辅助定位，不再承载主要信息。' }}</p>
          </div>

          <div class="graph-canvas graph-canvas--compact">
            <p v-if="loading" class="auth-side-copy">正在加载学习路径...</p>
            <p v-else-if="errorMessage" class="auth-error">{{ errorMessage }}</p>
            <p v-else-if="positionedNodes.length === 0" class="auth-side-copy">当前没有可展示的子图节点。</p>

            <template v-else>
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
                <span>{{ nodeSubtitle(node) }}</span>
              </button>
            </template>
          </div>
        </div>
      </article>

      <aside class="graph-side">
        <div class="detail-side-card graph-side-card">
          <span class="section-tag">图谱作用</span>
          <h3>{{ graphContribution.title }}</h3>
          <p class="graph-side-copy">{{ graphContribution.summary }}</p>
          <div class="graph-legend graph-legend--stack">
            <div v-for="item in graphContribution.items" :key="item.label">
              <strong>{{ item.label }}</strong>
              <span>{{ item.text }}</span>
            </div>
          </div>
        </div>

        <div class="detail-side-card graph-side-card">
          <span class="section-tag">建议动作</span>
          <h3>{{ primaryAction.title }}</h3>
          <p class="graph-side-copy">{{ primaryAction.detail }}</p>
          <div class="graph-side-actions">
            <RouterLink v-if="primaryAction.route" :to="primaryAction.route" class="back-home-link">
              {{ primaryAction.cta }}
            </RouterLink>
            <RouterLink v-if="hasKnowledgeDetail" :to="`/knowledge/${selectedRouteKnowledgeId}`" class="auth-secondary-link">
              查看详情页
            </RouterLink>
          </div>
        </div>

        <div class="detail-side-card graph-side-card">
          <span class="section-tag">{{ usingLocalFallback ? '当前路径节点' : '当前子图节点' }}</span>
          <div class="graph-node-list">
            <button
              v-for="node in adjacentNodes"
              :key="`node-${node.id}`"
              class="graph-node-list-item"
              @click="focusNode(node.id)"
            >
              <div>
                <strong>{{ node.label }}</strong>
                <p>{{ node.relationLabel }} · {{ node.entityLabel }}</p>
              </div>
              <span>聚焦</span>
            </button>
          </div>
        </div>

        <div class="detail-side-card graph-side-card">
          <span class="section-tag">怎么使用</span>
          <div class="graph-legend graph-legend--stack">
            <div><strong>1.</strong><span>先看“先修内容”，确认当前是不是卡在基础没打通。</span></div>
            <div><strong>2.</strong><span>再看“关联知识”，把容易混淆或需要一起理解的点串起来。</span></div>
            <div><strong>3.</strong><span>最后看“延展方向”，决定学完当前内容后下一步去哪里。</span></div>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
