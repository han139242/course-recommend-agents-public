<script setup>
import {
  BookOpen,
  Bot,
  BrainCircuit,
  CircleUserRound,
  GraduationCap,
  Newspaper,
  LayoutDashboard,
  LogIn,
  Maximize2,
  RefreshCcw,
} from '@lucide/vue'
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import SearchBar from '../components/search/SearchBar.vue'
import LeftPanel from '../components/sidebar/LeftPanel.vue'
import StudyGoalWizard from '../components/study/StudyGoalWizard.vue'
import StudyPlanCard from '../components/study/StudyPlanCard.vue'
import { buildGraphStateMap } from '../composables/useNodeFocus'
import { APP_NAME } from '../constants/appInfo'
import { fetchCurrentStudyPlan, fetchGoalProfile, generateStudyPlan, saveGoalProfile } from '../api/studyPlan'
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
const goalWizardOpen = ref(false)
const pendingGoalDraft = ref(null)
const studyProfile = ref(null)
const studyPlan = ref(null)
const studyPlanLoading = ref(false)
const recommendationMeta = ref({
  path: [],
  insights: [],
  learnerProfile: null,
})
let recommendationRequestId = 0

const sceneMeta = computed(() => graphStore.graphData?.meta || { totalNodes: 0, visibleNodes: 0, visibleEdges: 0, rootCount: 0, expandedCount: 0 })

const learningStats = computed(() => {
  let mastered = 0
  let learning = 0
  let weak = 0

  graphStore.rawNodes.forEach((node) => {
    const progress = userStore.getProgressMeta(node.routeKnowledgeId)
    const mastery = Number(progress.masteryScore || 0)
    const answeredCount = Number(progress.answeredCount || 0)
    if (mastery >= 80) mastered += 1
    else if (mastery >= 40) learning += 1
    if (mastery < 30 && answeredCount > 0) weak += 1
  })

  return {
    total: graphStore.rawNodes.length,
    mastered,
    learning,
    weak,
    courseCount: Math.max((graphStore.courseOptions || []).length - 1, 0),
  }
})

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
  await loadStudyPlanContext()
}

async function loadStudyPlanContext() {
  if (!userStore.isAuthenticated) {
    studyProfile.value = null
    studyPlan.value = null
    return
  }
  try {
    const [profilePayload, planPayload] = await Promise.all([
      fetchGoalProfile().catch(() => null),
      fetchCurrentStudyPlan().catch(() => null),
    ])
    studyProfile.value = profilePayload?.profile || planPayload?.profile || null
    studyPlan.value = planPayload?.plan || null
  } catch {
    studyProfile.value = null
    studyPlan.value = null
  }
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
    if (options.flash) await graphCanvasRef.value?.flashNode?.(node.id)
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
    queueExpandedClusterFocus(node.id, newChildIds, { zoomLevel: 1.08, duration: 220, lockDuration: 820 })
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
        await revealAndFocus(resolved.routeKnowledgeId, { flash: action.flash !== false, duration: 220 })
        rightCollapsed.value = false
      }
      continue
    }

    if (type === 'propose_goal_profile') {
      handleProposedGoalProfile(action.payload || {})
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
      if (resolved?.routeKnowledgeId) router.push(`/knowledge/${encodeURIComponent(resolved.routeKnowledgeId)}`)
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

    if (type === 'reset_graph_scene') await handleSceneReset()
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
  graphCanvasRef.value?.softFollowElement?.(nodeId, { duration: 180 })
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
    queueExpandedClusterFocus(nodeId, changedNodeIds, { zoomLevel: 1.08, duration: 240, lockDuration: 920 })
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
  if (courseTitle) graphStore.setCourseFilter(courseTitle)
}

function handleHubDoubleClick(hubId) {
  const courseTitle = extractCourseTitle(hubId)
  if (courseTitle) graphStore.setCourseFilter(courseTitle)
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
      text: `请用学生能听懂的方式讲解“${routeKnowledgeId}”，并告诉我下一步该学什么。`,
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

function handleRenamePath({ pathId, name }) {
  if (!pathId || !name) return
  userStore.renamePath(pathId, name)
}

function handleReorderPath({ pathId, fromIndex, toIndex }) {
  if (!pathId) return
  const path = userStore.customPaths.find((item) => item.id === pathId)
  if (!path || !Number.isInteger(fromIndex) || !Number.isInteger(toIndex)) return
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

function handleAskGoalAi() {
  aiAutoPrompt.value = {
    id: Date.now(),
    text: '我想制定一个学习目标。请先问我年级、想完成什么、学过什么，然后帮我整理成学习画像。',
  }
  aiOpen.value = true
}

async function handleSaveGoalProfile(payload) {
  if (!userStore.isAuthenticated) {
    router.push({ name: 'login', query: { redirect: '/' } })
    return
  }
  studyPlanLoading.value = true
  try {
    const profilePayload = await saveGoalProfile(payload)
    studyProfile.value = profilePayload?.profile || null
    const planPayload = await generateStudyPlan()
    studyPlan.value = planPayload?.plan || null
    pendingGoalDraft.value = null
    goalWizardOpen.value = false
    if (studyPlan.value?.nextSteps?.length) {
      await handleShowPath(studyPlan.value.nextSteps.map((item) => item.knowledgeId || item.title))
    }
  } finally {
    studyPlanLoading.value = false
  }
}

async function handleGenerateStudyPlan() {
  studyPlanLoading.value = true
  try {
    const payload = await generateStudyPlan()
    studyProfile.value = payload?.profile || studyProfile.value
    studyPlan.value = payload?.plan || null
  } finally {
    studyPlanLoading.value = false
  }
}

function handleEditGoal(profile = null) {
  pendingGoalDraft.value = profile || null
  goalWizardOpen.value = true
}

function handleProposedGoalProfile(payload) {
  pendingGoalDraft.value = payload || null
  goalWizardOpen.value = true
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
      ? `请围绕“${graphStore.selectedRouteKnowledgeId}”讲解核心概念，并给出一个练习建议。`
      : '请帮我从当前课程图谱里选一个适合开始学习的知识点。',
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
    await loadStudyPlanContext()
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
  }

  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  if (onKeydown) window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <section class="graph-home student-workspace">
    <header class="topbar student-topbar">
      <div class="topbar__brand">
        <div class="topbar__logo">
          <GraduationCap :size="22" aria-hidden="true" />
        </div>
        <div>
          <strong class="topbar__title">{{ APP_NAME }}</strong>
          <p>从课程目录、知识地图和 AI 助教开始你的下一步学习。</p>
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
          :title="themeStore.theme === 'light' ? '切换夜间模式' : '切换白天模式'"
          @click="themeStore.toggleTheme"
        >
          <span v-if="themeStore.theme === 'light'">夜间</span>
          <span v-else>白天</span>
        </button>
        <RouterLink v-if="userStore.isAuthenticated" class="secondary-btn" to="/dashboard">
          <LayoutDashboard :size="16" aria-hidden="true" />
          学习总览
        </RouterLink>
        <RouterLink v-if="userStore.isAuthenticated" class="secondary-btn" to="/competency">
          <BrainCircuit :size="16" aria-hidden="true" />
          能力图谱
        </RouterLink>
        <RouterLink class="secondary-btn" to="/frontier">
          <Newspaper :size="16" aria-hidden="true" />
          前沿资料
        </RouterLink>
        <RouterLink v-if="!userStore.isAuthenticated" class="secondary-btn" to="/login">
          <LogIn :size="16" aria-hidden="true" />
          登录
        </RouterLink>
        <RouterLink v-else class="profile-chip" to="/dashboard">
          <CircleUserRound :size="16" aria-hidden="true" />
          <span>{{ userStore.user?.username || userStore.user?.email || '我的学习' }}</span>
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
        @rename-path="handleRenamePath"
        @start-path="handleStartPath"
      />

      <main class="graph-stage">
        <StudyPlanCard
          :authenticated="userStore.isAuthenticated"
          :profile="studyProfile"
          :plan="studyPlan"
          :loading="studyPlanLoading"
          @ask-ai="handleAskGoalAi"
          @edit-goal="handleEditGoal(studyProfile)"
          @generate="handleGenerateStudyPlan"
          @highlight-path="handleShowPath"
          @start="handleStartPath"
        />

        <section class="graph-learning-card">
          <div class="graph-learning-card__head">
            <div class="graph-learning-card__title">
              <span class="eyebrow">Knowledge Map</span>
              <h1>课程知识地图</h1>
              <p>单击查看知识点，双击展开关联，右键加入学习路径或收藏。</p>
            </div>
            <div class="graph-learning-card__meta" aria-label="图谱概览">
              <span :title="graphStore.activeCourseFilter === '全部' ? '全部课程' : graphStore.activeCourseFilter">
                {{ graphStore.activeCourseFilter === '全部' ? '全部课程' : graphStore.activeCourseFilter }}
              </span>
              <span>{{ learningStats.courseCount }} 门课</span>
              <span>{{ sceneMeta.visibleNodes }} 点</span>
              <span>{{ learningStats.mastered }} 已掌握</span>
            </div>
            <div class="graph-scene-bar__actions">
              <button type="button" class="graph-scene-btn" @click="graphCanvasRef?.fitView?.()">
                <Maximize2 :size="15" aria-hidden="true" />
                适配
              </button>
              <button type="button" class="graph-scene-btn" @click="handleSceneReset">
                <RefreshCcw :size="15" aria-hidden="true" />
                重置
              </button>
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

          <div v-if="quizCelebrationActive" class="graph-celebrations">
            <div class="graph-burst">
              <span class="graph-burst__label">掌握通关：{{ quizCelebrationNodeName }}</span>
              <div class="graph-burst__star graph-burst__star--1">★</div>
              <div class="graph-burst__star graph-burst__star--2">★</div>
              <div class="graph-burst__star graph-burst__star--3">★</div>
              <div class="graph-burst__star graph-burst__star--4">★</div>
            </div>
          </div>
        </section>
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

    <StudyGoalWizard
      v-model="goalWizardOpen"
      :initial-profile="studyProfile"
      :draft-profile="pendingGoalDraft"
      @submit="handleSaveGoalProfile"
      @close="pendingGoalDraft = null"
    />

    <div class="mobile-dock">
      <button type="button" class="mobile-dock__btn" @click="leftCollapsed = !leftCollapsed">
        <BookOpen :size="16" aria-hidden="true" />
        导航
      </button>
      <button type="button" class="mobile-dock__btn" @click="rightCollapsed = !rightCollapsed">详情</button>
      <button type="button" class="mobile-dock__btn mobile-dock__btn--primary" @click="aiOpen = !aiOpen">
        <Bot :size="16" aria-hidden="true" />
        助教
      </button>
    </div>
  </section>
</template>
