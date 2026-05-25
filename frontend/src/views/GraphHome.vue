<script setup>
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import SearchBar from '../components/search/SearchBar.vue'
import LeftPanel from '../components/sidebar/LeftPanel.vue'
import { buildGraphStateMap } from '../composables/useNodeFocus'
import { APP_NAME } from '../constants/appInfo'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'
import { useThemeStore } from '../stores/themeStore'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../utils/courseMeta'

const GraphCanvas = defineAsyncComponent(() => import('../components/graph/GraphCanvas.vue'))
const RightPanel = defineAsyncComponent(() => import('../components/detail/RightPanel.vue'))
const AiAssistant = defineAsyncComponent(() => import('../components/ai/AiAssistant.vue'))

const graphStore = useGraphStore()
const userStore = useUserStore()
const themeStore = useThemeStore()
const router = useRouter()

const searchValue = ref('')
const leftCollapsed = ref(false)
const rightCollapsed = ref(false)
const aiOpen = ref(false)
const aiAutoPrompt = ref(null)
const searchBarRef = ref(null)
const graphCanvasRef = ref(null)
const recommendedPath = ref([])
const recommendationMeta = ref({
  path: [],
  insights: [],
  learnerProfile: null,
})
let recommendationRequestId = 0

const sceneMeta = computed(() => graphStore.graphData?.meta || { totalNodes: 0, visibleNodes: 0, visibleEdges: 0, rootCount: 0, expandedCount: 0 })

const nextLearningNodes = computed(() =>
  graphStore.getNextLearningOptions(graphStore.selectedRouteKnowledgeId).map((routeKnowledgeId) => {
    const node = graphStore.routeNodeMap.get(routeKnowledgeId)
    return {
      routeKnowledgeId,
      label: node?.canonicalName || node?.name || routeKnowledgeId,
    }
  }),
)

const searchResults = computed(() => graphStore.searchNodes(searchValue.value))

const stateMap = computed(() =>
  buildGraphStateMap({
    nodes: graphStore.graphData.nodes,
    edges: graphStore.graphData.edges,
    focusedNodeId: graphStore.focusedNodeId,
    selectedNodeId: graphStore.selectedNodeId,
    highlightFilter: graphStore.highlightFilter,
    highlightPath: graphStore.highlightPath,
    courseFilter: graphStore.activeCourseFilter,
    progressMap: userStore.progressMap,
    bookmarks: userStore.bookmarks,
  }),
)

const selectedProgress = computed(() => userStore.getProgressMeta(graphStore.selectedRouteKnowledgeId))

function upsertKnowledgeLookup(lookup, node) {
  const routeKnowledgeId = String(node?.routeKnowledgeId || '').trim()
  if (!routeKnowledgeId) return

  const label = String(node?.canonicalName || node?.name || routeKnowledgeId).trim() || routeKnowledgeId
  lookup[routeKnowledgeId] = {
    label,
    canonicalName: String(node?.canonicalName || '').trim(),
    name: String(node?.name || '').trim(),
  }
}

const aiKnowledgeLookup = computed(() => {
  const lookup = {}
  graphStore.rawNodes.forEach((node) => upsertKnowledgeLookup(lookup, node))
  Object.values(graphStore.detailCache || {}).forEach((detail) => upsertKnowledgeLookup(lookup, detail))
  return lookup
})

const knowledgeMentions = computed(() => Object.keys(aiKnowledgeLookup.value))

function getCourseLabel(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}

function getRecommendationTarget(node) {
  const preferred = String(node?.canonicalName || node?.name || '').trim()
  return preferred || getCourseLabel(node)
}

function applyRecommendationState(payload) {
  recommendationMeta.value = payload || {
    path: [],
    insights: [],
    learnerProfile: null,
  }
  recommendedPath.value = payload?.path || []
}

async function refreshRecommendationContext(routeKnowledgeId) {
  const requestId = ++recommendationRequestId

  if (routeKnowledgeId) {
    await graphStore.ensureNodeDetail(routeKnowledgeId)
    const detail = graphStore.detailCache[routeKnowledgeId]
    const target = getRecommendationTarget(detail)
    const recommendation = await graphStore.ensureRecommendedPath(target, routeKnowledgeId, 'detail')
    if (requestId !== recommendationRequestId) return
    applyRecommendationState(recommendation)
    return
  }

  const recommendation = await graphStore.ensureRecommendedPath('', '', 'assistant')
  if (requestId !== recommendationRequestId) return
  applyRecommendationState(recommendation)
}

async function bootstrapPage() {
  await Promise.all([
    graphStore.initGraph().catch((err) => console.warn('[Bootstrap] graph init failed:', err?.message)),
    userStore.loadProgress().catch(() => null),
    userStore.loadDashboard().catch(() => null),
    userStore.loadCompetency().catch(() => null),
  ])
}

function getRevealOptions() {
  return {
    anchorPosition: graphCanvasRef.value?.getViewportCenterWorld?.() || null,
    viewportCenterWorld: graphCanvasRef.value?.getViewportCenterWorld?.() || null,
    occupiedNodes: graphCanvasRef.value?.getNodeSnapshots?.() || [],
  }
}

function queueExpandedClusterFocus(nodeId, childIds = [], options = {}) {
  graphCanvasRef.value?.queueViewportFocus?.(nodeId, {
    zoomLevel: options.zoomLevel ?? 1.06,
    duration: options.duration ?? 220,
    lockDuration: options.lockDuration ?? 880,
    relatedNodeIds: childIds,
    mode: childIds.length ? 'keep-visible' : 'center',
  })
}

async function revealAndFocus(routeKnowledgeId, options = {}) {
  const node = await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId, getRevealOptions())
  if (node?.id) {
    await graphCanvasRef.value?.focusElement?.(node.id, options.duration ?? 200)
    if (options.flash) {
      await graphCanvasRef.value?.flashNode?.(node.id)
    }
  }
  return node
}

function resolveAssistantTarget(target) {
  const normalized = String(target || '').trim()
  if (!normalized) return null

  const direct = graphStore.routeNodeMap.get(normalized)
  if (direct?.routeKnowledgeId) return direct

  const candidate = graphStore.searchNodes(normalized)[0]
  if (candidate?.routeKnowledgeId) {
    return graphStore.routeNodeMap.get(candidate.routeKnowledgeId) || candidate
  }

  return null
}

async function expandKnowledgeFromAssistant(target) {
  const resolved = resolveAssistantTarget(target)
  if (!resolved?.routeKnowledgeId) return

  const node = await revealAndFocus(resolved.routeKnowledgeId, { flash: true, duration: 220 })
  if (!node?.id) return

  if (!graphStore.isNodeExpanded(node.id)) {
    const anchorSnapshot = graphCanvasRef.value?.getNodeSnapshot?.(node.id)
    const occupiedNodes = graphCanvasRef.value?.getNodeSnapshots?.() || []
    const newChildIds = await graphStore.expandNode(node.id, {
      anchorPosition: anchorSnapshot,
      viewportCenterWorld: graphCanvasRef.value?.getViewportCenterWorld?.() || null,
      occupiedNodes,
    })
    queueExpandedClusterFocus(node.id, newChildIds, {
      zoomLevel: 1.08,
      duration: 220,
      lockDuration: 820,
    })
  }

  await graphStore.selectNode(node.id)
  rightCollapsed.value = false
}

async function handleAssistantActions(actions = []) {
  for (const action of actions) {
    const type = String(action?.type || '').trim()
    if (!type) continue

    if (type === 'focus_knowledge') {
      const resolved = resolveAssistantTarget(action.target)
      if (resolved?.routeKnowledgeId) {
        await revealAndFocus(resolved.routeKnowledgeId, {
          flash: action.flash !== false,
          duration: 220,
        })
        rightCollapsed.value = false
      }
      continue
    }

    if (type === 'expand_knowledge') {
      await expandKnowledgeFromAssistant(action.target)
      continue
    }

    if (type === 'highlight_prerequisite_chain') {
      const resolved = resolveAssistantTarget(action.target)
      if (resolved?.routeKnowledgeId) {
        await revealAndFocus(resolved.routeKnowledgeId, { flash: true, duration: 200 })
        await graphStore.ensureNodeDetail(resolved.routeKnowledgeId)
        const path = graphStore.getPrerequisiteRouteChain(resolved.routeKnowledgeId)
        graphStore.highlightPathNodes(path)
        rightCollapsed.value = false
      }
      continue
    }

    if (type === 'open_knowledge_detail') {
      const resolved = resolveAssistantTarget(action.target)
      if (resolved?.routeKnowledgeId) {
        router.push(`/knowledge/${encodeURIComponent(resolved.routeKnowledgeId)}`)
      }
      continue
    }

    if (type === 'filter_course') {
      const courseTitle = String(action.courseTitle || action.target || '').trim()
      if (!courseTitle) continue
      graphStore.setCourseFilter(courseTitle)
      const node = await graphStore.focusCourseRepresentative(courseTitle, getRevealOptions())
      if (node?.id) {
        await graphStore.selectNode(node.id)
        await graphCanvasRef.value?.focusElement?.(node.id, 180)
      }
      continue
    }

    if (type === 'fit_graph_view') {
      await graphCanvasRef.value?.fitView?.()
      continue
    }

    if (type === 'reset_graph_scene') {
      await handleSceneReset()
    }
  }
}

async function handleSearchSelect(item) {
  searchValue.value = item.name
  await revealAndFocus(item.routeKnowledgeId, { flash: true, duration: 220 })
  rightCollapsed.value = false
}

async function handleNodeSelect(nodeId) {
  await graphStore.selectNode(nodeId)
  rightCollapsed.value = false
  graphCanvasRef.value?.softFollowElement?.(nodeId, {
    duration: 180,
  })
}

async function handleNodeDoubleClick(nodeId) {
  const wasExpanded = graphStore.isNodeExpanded(nodeId)
  const anchorSnapshot = graphCanvasRef.value?.getNodeSnapshot?.(nodeId)
  const occupiedNodes = graphCanvasRef.value?.getNodeSnapshots?.() || []
  const changedNodeIds = await graphStore.toggleNodeExpansion(nodeId, {
    anchorPosition: anchorSnapshot,
    viewportCenterWorld: graphCanvasRef.value?.getViewportCenterWorld?.() || null,
    occupiedNodes,
  })
  if (!wasExpanded) {
    queueExpandedClusterFocus(nodeId, changedNodeIds, {
      zoomLevel: 1.08,
      duration: 240,
      lockDuration: 920,
    })
  }
  await graphStore.selectNode(nodeId)
  rightCollapsed.value = false
}

function handleClearSelection() {
  graphStore.clearSelection()
}

function extractCourseTitle(hubId) {
  return String(hubId || '').replace(/^hub:/, '')
}

function handleHubClick(hubId) {
  const courseTitle = extractCourseTitle(hubId)
  if (!courseTitle) return
  graphStore.setCourseFilter(courseTitle)
}

function handleHubDoubleClick(hubId) {
  const courseTitle = extractCourseTitle(hubId)
  if (!courseTitle) return
  graphStore.setCourseFilter(courseTitle)
}

async function handleCourseFocus(courseTitle) {
  graphStore.setCourseFilter(courseTitle)
  const node = await graphStore.focusCourseRepresentative(courseTitle, getRevealOptions())
  if (!node?.id) return
  await graphStore.selectNode(node.id)
  await graphCanvasRef.value?.focusElement?.(node.id, 180)
}

async function handleTreeNodeFocus(routeKnowledgeId) {
  await revealAndFocus(routeKnowledgeId, { flash: true })
  rightCollapsed.value = false
}

async function handleContextAction({ action, node }) {
  const routeKnowledgeId = node?.data?.routeKnowledgeId
  if (!routeKnowledgeId) return

  if (action === 'toggle-bookmark') {
    userStore.toggleBookmark(routeKnowledgeId)
    return
  }

  if (action === 'note') {
    await revealAndFocus(routeKnowledgeId)
    rightCollapsed.value = false
    return
  }

  if (action === 'detail') {
    router.push(`/knowledge/${encodeURIComponent(routeKnowledgeId)}`)
    return
  }

  if (action === 'ai') {
    await revealAndFocus(routeKnowledgeId)
    rightCollapsed.value = false
    aiAutoPrompt.value = {
      id: Date.now(),
      text: `请帮我解释知识点“${routeKnowledgeId}”，并结合前置知识给出学习建议。`,
    }
    aiOpen.value = true
    return
  }

  if (action === 'chain') {
    await graphStore.ensureNodeDetail(routeKnowledgeId)
    const path = graphStore.getPrerequisiteRouteChain(routeKnowledgeId)
    graphStore.highlightPathNodes(path)
    return
  }

  if (action === 'mastered') {
    userStore.markMastered(routeKnowledgeId)
    return
  }

  if (action === 'add-to-path') {
    const path = userStore.customPaths[0] || userStore.createCustomPath('我的学习路径')
    userStore.addToPath(path.id, routeKnowledgeId)
  }
}

function handleToggleLeft() {
  leftCollapsed.value = !leftCollapsed.value
}

function handleToggleRight() {
  rightCollapsed.value = !rightCollapsed.value
}

async function handleBookmarkSelect(routeKnowledgeId) {
  await revealAndFocus(routeKnowledgeId, { flash: true })
  rightCollapsed.value = false
}

function handleCreatePath() {
  const name = window.prompt('输入路径名称', '我的学习路径')
  if (!name) return
  userStore.createCustomPath(name)
}

async function handleShowPath(nodeIds) {
  const routeIds = [...new Set((nodeIds || []).filter(Boolean))]
  if (!routeIds.length) return

  for (const routeKnowledgeId of routeIds) {
    await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId, getRevealOptions())
  }

  graphStore.highlightPathNodes(routeIds)
  rightCollapsed.value = false
  await nextTick()
  await graphCanvasRef.value?.fitView?.(240)
}

function handleRemovePathNode({ pathId, knowledgeId }) {
  if (!pathId || !knowledgeId) return
  userStore.removeFromPath(pathId, knowledgeId)
}

function handleReorderPath({ pathId, fromIndex, toIndex }) {
  if (!pathId) return
  if (!Number.isInteger(fromIndex) || !Number.isInteger(toIndex)) return

  const path = userStore.customPaths.find((item) => item.id === pathId)
  if (!path) return
  if (toIndex < 0 || toIndex >= path.nodes.length || fromIndex === toIndex) return

  userStore.reorderPath(pathId, fromIndex, toIndex)
}

function handleStartPath(nodeIds) {
  const routeIds = (nodeIds || []).filter(Boolean)
  if (!routeIds.length) return

  graphStore.highlightPathNodes(routeIds)
  const targetRouteKnowledgeId =
    routeIds.find((item) => Number(userStore.getProgressMeta(item).masteryScore || 0) < 80) || routeIds[0]
  router.push(`/knowledge/${encodeURIComponent(targetRouteKnowledgeId)}`)
}

async function handleRelatedSelect(routeKnowledgeId) {
  await revealAndFocus(routeKnowledgeId)
  rightCollapsed.value = false
}

const aiAssistantRef = ref(null)
const quizCelebrationActive = ref(false)
const quizCelebrationNodeName = ref('')

function handleOpenAi() {
  aiAutoPrompt.value = {
    id: Date.now(),
    text: graphStore.selectedRouteKnowledgeId
      ? `请围绕“${graphStore.selectedRouteKnowledgeId}”做一段易懂讲解，并指出我下一步该学什么。`
      : '请先概览当前知识图谱，并给我一个适合现在开始的学习建议。',
  }
  aiOpen.value = true
}

function handleOpenAiWithPrompt(promptText) {
  aiAutoPrompt.value = {
    id: Date.now(),
    text: promptText
  }
  aiOpen.value = true
}

function handleQuizFailed(payload) {
  if (aiAssistantRef.value?.triggerStruggleHint) {
    aiAssistantRef.value.triggerStruggleHint(payload.name || payload.nodeId)
  }
}

function handleQuizMastered(payload) {
  quizCelebrationNodeName.value = payload.name || payload.nodeId
  quizCelebrationActive.value = true
  setTimeout(() => {
    quizCelebrationActive.value = false
    quizCelebrationNodeName.value = ''
  }, 3000)
}

async function handleSceneReset() {
  searchValue.value = ''
  graphStore.setCourseFilter('全部')
  graphCanvasRef.value?.cancelViewportAutomation?.()
  graphStore.clearSelection()
  await graphStore.loadSeeds(14, { silent: true })
  await nextTick()
  await new Promise((resolve) => window.requestAnimationFrame(resolve))
  await graphCanvasRef.value?.syncNow?.({ flushPendingFocus: false })
  await graphCanvasRef.value?.fitView?.(0)
}

watch(
  () => graphStore.selectedRouteKnowledgeId,
  async (routeKnowledgeId) => {
    await refreshRecommendationContext(routeKnowledgeId)
  },
  { immediate: true },
)

watch(
  () => userStore.isAuthenticated,
  async () => {
    await refreshRecommendationContext(graphStore.selectedRouteKnowledgeId)
  },
)

let onKeydown = null

onMounted(() => {
  bootstrapPage()

  const isMobile = window.innerWidth < 768
  leftCollapsed.value = isMobile
  rightCollapsed.value = isMobile

  onKeydown = (event) => {
    if ((event.ctrlKey && event.key.toLowerCase() === 'k') || event.key === '/') {
      event.preventDefault()
      searchBarRef.value?.focusInput?.()
    }

    if (event.key === 'Escape') {
      graphStore.clearSelection()
      graphCanvasRef.value?.closeContextMenu?.()
      leftCollapsed.value = isMobile
      rightCollapsed.value = isMobile
      aiOpen.value = false
    }

    if (event.key === ' ') {
      event.preventDefault()
      Promise.resolve(graphCanvasRef.value?.fitView?.()).catch(() => null)
    }

    if (event.key === '0') {
      event.preventDefault()
      handleSceneReset().catch(() => null)
    }

    if (event.key === '[') {
      leftCollapsed.value = !leftCollapsed.value
    }

    if (event.key === ']') {
      rightCollapsed.value = !rightCollapsed.value
    }
  }

  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  if (onKeydown) {
    window.removeEventListener('keydown', onKeydown)
  }
})
</script>

<template>
  <section class="graph-home">
    <header class="topbar">
      <div class="topbar__brand">
        <div class="topbar__logo">
          <span class="topbar__logo-icon">智</span>
          <span class="topbar__logo-pulse"></span>
        </div>
        <div>
          <strong class="topbar__title">{{ APP_NAME }}</strong>
          <p class="topbar__subtitle">跨课程 AI 知识图谱推荐系统</p>
        </div>
      </div>

      <SearchBar
        ref="searchBarRef"
        v-model="searchValue"
        :suggestions="searchResults"
        :courses="graphStore.courseOptions"
        :course-filter="graphStore.activeCourseFilter"
        @select="handleSearchSelect"
        @update:course-filter="graphStore.setCourseFilter"
      />

      <div class="topbar__actions">
        <button
          type="button"
          class="theme-toggle-btn"
          title="切换白天/夜间模式"
          @click="themeStore.toggleTheme"
        >
          <span v-if="themeStore.theme === 'light'">🌙 夜间</span>
          <span v-else>☀️ 白天</span>
        </button>
        <RouterLink v-if="!userStore.isAuthenticated" class="secondary-btn" to="/login">登录</RouterLink>
        <template v-else>
          <RouterLink class="secondary-btn" to="/dashboard">能力图谱</RouterLink>
          <RouterLink class="profile-chip" to="/">
            <span class="profile-chip__dot"></span>
            <span>{{ userStore.user?.username || userStore.user?.email || '已登录' }}</span>
          </RouterLink>
        </template>
      </div>
    </header>

    <div class="graph-layout">
      <LeftPanel
        :collapsed="leftCollapsed"
        :course-groups="graphStore.courseGroups"
        :nodes="graphStore.rawNodes"
        :progress-map="userStore.progressMap"
        :bookmarks="userStore.bookmarks"
        :custom-paths="userStore.customPaths"
        :recommended-path="recommendedPath"
        :is-authenticated="userStore.isAuthenticated"
        @toggle="handleToggleLeft"
        @focus-course="handleCourseFocus"
        @focus-node="handleTreeNodeFocus"
        @highlight-status="graphStore.highlightByStatus"
        @show-path="handleShowPath"
        @bookmark-select="handleBookmarkSelect"
        @create-path="handleCreatePath"
        @reorder-path="handleReorderPath"
        @remove-path-node="handleRemovePathNode"
        @start-path="handleStartPath"
      />

      <main class="graph-stage">
        <div class="graph-stage__hint">
          <span>
            首屏只保留少量关键知识点，双击任意知识点会按层级展开它的关联节点；再次双击已经展开的节点，会把它展开出的关联节点收回去。
          </span>
          <span>
            鼠标滚轮缩放，左键选中知识点，双击展开或收回关联网络，右键打开学习菜单，<kbd>Space</kbd> 适配视图，<kbd>Ctrl+K</kbd> 搜索，<kbd>0</kbd> 重置。
          </span>
        </div>

        <div class="graph-scene-bar">
          <div class="graph-scene-bar__group">
            <span class="graph-scene-bar__label">Scene</span>
            <strong>{{ graphStore.activeCourseFilter === '全部' ? '全局总览' : graphStore.activeCourseFilter }}</strong>
            <span>
              {{ sceneMeta.totalNodes }} 个知识点 · 首屏种子 {{ sceneMeta.rootCount }} 个 · 当前可见 {{ sceneMeta.visibleNodes }} 个 · 可见关系 {{ sceneMeta.visibleEdges }} 条 · 已展开节点 {{ sceneMeta.expandedCount }} 个
            </span>
          </div>

          <div class="graph-scene-bar__actions">
            <button type="button" class="graph-scene-btn" @click="graphCanvasRef?.fitView?.()">适配视图</button>
            <button type="button" class="graph-scene-btn" @click="handleSceneReset">重置场景</button>
          </div>
        </div>

        <GraphCanvas
          ref="graphCanvasRef"
          :graph-data="graphStore.graphData"
          :state-map="stateMap"
          :highlight-path="graphStore.highlightPath"
          :selected-node-id="graphStore.selectedNodeId"
          :loading="graphStore.loading"
          :error="graphStore.error"
          @select-node="handleNodeSelect"
          @node-double-click="handleNodeDoubleClick"
          @hub-click="handleHubClick"
          @hub-double-click="handleHubDoubleClick"
          @clear-selection="handleClearSelection"
          @context-action="handleContextAction"
        />
        
        <!-- Celebrations Overlay -->
        <div v-if="quizCelebrationActive" class="graph-celebrations">
          <div class="graph-burst">
            <span class="graph-burst__label">✨ 掌握通关：{{ quizCelebrationNodeName }} ✨</span>
            <div class="graph-burst__star graph-burst__star--1">★</div>
            <div class="graph-burst__star graph-burst__star--2">★</div>
            <div class="graph-burst__star graph-burst__star--3">★</div>
            <div class="graph-burst__star graph-burst__star--4">★</div>
          </div>
        </div>
      </main>

      <RightPanel
        :collapsed="rightCollapsed"
        :detail="graphStore.selectedDetail"
        :progress="selectedProgress"
        :next-nodes="nextLearningNodes"
        :recommended-path="recommendedPath"
        @toggle="handleToggleRight"
        @select-related="handleRelatedSelect"
        @open-ai="handleOpenAi"
        @show-recommend-path="handleShowPath"
        @open-ai-with-prompt="handleOpenAiWithPrompt"
        @quiz-mastered="handleQuizMastered"
        @quiz-failed="handleQuizFailed"
      />
    </div>

    <AiAssistant
      ref="aiAssistantRef"
      v-model="aiOpen"
      :detail="graphStore.selectedDetail"
      :is-authenticated="userStore.isAuthenticated"
      :auto-prompt="aiAutoPrompt"
      :knowledge-mentions="knowledgeMentions"
      :knowledge-lookup="aiKnowledgeLookup"
      :recommended-path="recommendedPath"
      :proactive-hints="recommendationMeta.insights || []"
      :learner-profile="recommendationMeta.learnerProfile || null"
      @focus-knowledge="handleRelatedSelect"
      @show-recommend-path="handleShowPath"
      @run-actions="handleAssistantActions"
    />

    <div class="mobile-dock">
      <button type="button" class="mobile-dock__btn" @click="leftCollapsed = !leftCollapsed">导航</button>
      <button type="button" class="mobile-dock__btn" @click="rightCollapsed = !rightCollapsed">详情</button>
      <button type="button" class="mobile-dock__btn mobile-dock__btn--primary" @click="aiOpen = !aiOpen">助手</button>
    </div>
  </section>
</template>
