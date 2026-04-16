<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import AiAssistant from '../components/ai/AiAssistant.vue'
import GraphCanvas from '../components/graph/GraphCanvas.vue'
import RightPanel from '../components/detail/RightPanel.vue'
import SearchBar from '../components/search/SearchBar.vue'
import LeftPanel from '../components/sidebar/LeftPanel.vue'
import { buildGraphStateMap } from '../composables/useNodeFocus'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'

const graphStore = useGraphStore()
const userStore = useUserStore()
const router = useRouter()

const searchValue = ref('')
const leftCollapsed = ref(false)
const rightCollapsed = ref(false)
const aiOpen = ref(false)
const aiAutoPrompt = ref(null)
const searchBarRef = ref(null)
const graphCanvasRef = ref(null)
const recommendedPath = ref([])

const sceneMeta = computed(() => graphStore.graphData?.meta || { totalNodes: 0, totalEdges: 0, visibleNodes: 0, visibleEdges: 0, openCourses: 0 })

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
const knowledgeMentions = computed(() => graphStore.rawNodes.map((node) => node.routeKnowledgeId).filter(Boolean))

async function bootstrapPage() {
  await Promise.all([
    graphStore.loadFullGraph().catch((err) => console.warn('[Bootstrap] graph load failed:', err?.message)),
    userStore.loadProgress().catch(() => null),
    userStore.loadDashboard().catch(() => null),
    userStore.loadCompetency().catch(() => null),
  ])
}

async function handleSearchSelect(item) {
  searchValue.value = item.name
  await graphStore.revealNodeByRouteKnowledgeId(item.routeKnowledgeId)
  rightCollapsed.value = false
  await graphCanvasRef.value?.flashNode?.(graphStore.selectedNodeId)
}

async function handleNodeSelect(nodeId) {
  await graphStore.selectNode(nodeId)
  rightCollapsed.value = false
}

async function handleNodeDoubleClick(nodeId) {
  graphStore.toggleNodeExpansion(nodeId)
  await graphStore.selectNode(nodeId)
  rightCollapsed.value = false
  await graphCanvasRef.value?.centerElement?.(nodeId, 1.4, 220)
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
  const node = graphStore.focusCourseRepresentative(courseTitle)
  if (!node?.id) return
  await graphStore.selectNode(node.id)
  await graphCanvasRef.value?.focusElement?.(node.id, 180)
}

async function handleTreeNodeFocus(routeKnowledgeId) {
  await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId)
  rightCollapsed.value = false
  await graphCanvasRef.value?.flashNode?.(graphStore.selectedNodeId)
}

async function handleContextAction({ action, node }) {
  const routeKnowledgeId = node?.data?.routeKnowledgeId
  if (!routeKnowledgeId) return

  if (action === 'toggle-bookmark') {
    userStore.toggleBookmark(routeKnowledgeId)
    return
  }

  if (action === 'note') {
    await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId)
    rightCollapsed.value = false
    return
  }

  if (action === 'detail') {
    router.push(`/knowledge/${encodeURIComponent(routeKnowledgeId)}`)
    return
  }

  if (action === 'ai') {
    await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId)
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
  await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId)
  rightCollapsed.value = false
}

function handleCreatePath() {
  const name = window.prompt('输入路径名称', '我的学习路径')
  if (!name) return
  userStore.createCustomPath(name)
}

function handleShowPath(nodeIds) {
  graphStore.highlightPathNodes(nodeIds)
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
  await graphStore.revealNodeByRouteKnowledgeId(routeKnowledgeId)
  rightCollapsed.value = false
}

function handleOpenAi() {
  if (!graphStore.selectedRouteKnowledgeId) return
  aiAutoPrompt.value = {
    id: Date.now(),
    text: `请围绕“${graphStore.selectedRouteKnowledgeId}”做一段易懂讲解，并指出我下一步该学什么。`,
  }
  aiOpen.value = true
}

function handleSceneReset() {
  searchValue.value = ''
  graphStore.setCourseFilter('全部')
  graphStore.collapseAllCourses()
  graphStore.clearSelection()
  graphCanvasRef.value?.fitView?.()
}

watch(
  () => graphStore.selectedRouteKnowledgeId,
  async (routeKnowledgeId) => {
    if (!routeKnowledgeId) {
      recommendedPath.value = []
      return
    }

    await graphStore.ensureNodeDetail(routeKnowledgeId)
    const detail = graphStore.detailCache[routeKnowledgeId]
    const target = detail?.chapterTitle || detail?.canonicalName || detail?.name || ''
    const recommendation = await graphStore.ensureRecommendedPath(target, routeKnowledgeId, 'detail')
    recommendedPath.value = recommendation?.path || []
  },
  { immediate: true },
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
      graphCanvasRef.value?.fitView?.()
    }

    if (event.key === '0') {
      event.preventDefault()
      handleSceneReset()
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
          <span class="topbar__logo-icon">AI</span>
          <span class="topbar__logo-pulse"></span>
        </div>
        <div>
          <strong class="topbar__title">突破课程边界<span class="topbar__title-accent"> · 知识点推荐智能体</span></strong>
          <p>分层展开的 Neo4j 风格知识图谱，优先保证大图流畅、节点可读和逐层探索体验。</p>
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
        <RouterLink v-if="!userStore.isAuthenticated" class="secondary-btn" to="/login">登录</RouterLink>
        <RouterLink v-else class="profile-chip" to="/">
          <span class="profile-chip__dot"></span>
          <span>{{ userStore.user?.username || userStore.user?.email || '已登录' }}</span>
        </RouterLink>
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
            首屏只保留少量关键知识点，双击任意知识点才会像 Neo4j 一样展开它的关联节点；再次双击已经展开的节点，会把它展开出的关联节点收回去。
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
      />
    </div>

    <AiAssistant
      v-model="aiOpen"
      :detail="graphStore.selectedDetail"
      :is-authenticated="userStore.isAuthenticated"
      :auto-prompt="aiAutoPrompt"
      :knowledge-mentions="knowledgeMentions"
      @focus-knowledge="handleRelatedSelect"
    />

    <div class="mobile-dock">
      <button type="button" class="mobile-dock__btn" @click="leftCollapsed = !leftCollapsed">导航</button>
      <button type="button" class="mobile-dock__btn" @click="rightCollapsed = !rightCollapsed">详情</button>
      <button type="button" class="mobile-dock__btn mobile-dock__btn--primary" @click="aiOpen = !aiOpen">AI</button>
    </div>
  </section>
</template>
