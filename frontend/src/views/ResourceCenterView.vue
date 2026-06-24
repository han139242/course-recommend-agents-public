<script setup>
import {
  BookOpen,
  Bot,
  BrainCircuit,
  CheckCircle2,
  CircleStop,
  FileText,
  ImagePlus,
  PlayCircle,
  RefreshCcw,
  Save,
  Send,
  Sparkles,
  Wand2,
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { extractLearningProfile, fetchLearningProfile, fetchProfileTimeline, refreshLearningProfile, saveLearningProfile } from '../api/profile'
import {
  cancelResourceJob,
  compareResources,
  exportResourceJobMarkdown,
  fetchJobEvidencePackage,
  fetchJobReviewSummary,
  fetchResourceFeed,
  fetchResourceDemoScript,
  fetchResourceJob,
  fetchResourceMediaStatus,
  fetchResourceRecommendations,
  fetchResourcePresentationOutline,
  fetchResourceQualityReport,
  fetchResourceSafetyReport,
  fetchResources,
  generateResourceImage,
  generateResourceVideo,
  generateResourceVariant,
  generateResources,
  retryResourceJob,
  streamResourceJob,
  fetchStudyChecklist,
  submitResourceFeedback,
  updateResourceLearningStatus,
} from '../api/resources'
import ResourceContent from '../components/ResourceContent.vue'
import { APP_NAME } from '../constants/appInfo'

const DEFAULT_PROMPT = '我是大二人工智能学生，想做智能体项目，机器学习基础不太稳，希望多一些导图、代码案例和练习题。'
const RESOURCE_FORM_STORAGE_KEY = 'a3-resource-center-form-v2'

const route = useRoute()

const profilePrompt = ref(DEFAULT_PROMPT)
const profilePayload = ref(null)
const profileLoading = ref(false)
const profileSaving = ref(false)
const profileError = ref('')
const profileSavedNotice = ref('')
const profileTimeline = ref([])
const profileRecentSignals = ref([])
const profileRefreshInsights = ref([])
const profileTimelineLoading = ref(false)

const resourceForm = ref({
  knowledgeId: '',
  knowledgeName: '智能体与工具调用',
  courseTitle: '人工智能导论',
  target: '',
  resourceTypes: ['explanation_doc', 'mind_map', 'practice_questions', 'coding_case', 'reading_material', 'video_script'],
})
const resourceLoading = ref(false)
const resourceError = ref('')
const mediaPrompt = ref('')
const mediaLoading = ref('')
const mediaError = ref('')
const mediaNotice = ref('')
const attachedImages = ref([])
const regeneratingId = ref(null)
const activeJob = ref(null)
const generatedResources = ref([])
const resourceLibrary = ref([])
const libraryLoading = ref(false)
const recommendations = ref([])
const recommendationSummary = ref('')
const feedbackSummary = ref(null)
const resourceFeed = ref([])
const resourceFeedSummary = ref('')
const resourceFeedLoading = ref(false)
const recommendationsLoading = ref(false)
const libraryKeyword = ref('')
const libraryTypeFilter = ref('')
const libraryStatusFilter = ref('')
const librarySort = ref('latest')
const libraryScope = ref('current')
const feedbackDrafts = ref({})
const feedbackSaving = ref({})
const statusUpdating = ref({})
const feedbackNotice = ref('')
const feedbackError = ref('')
const routeContextNotice = ref('')
const safetyReports = ref({})
const safetyLoading = ref({})
const qualityReports = ref({})
const qualityLoading = ref({})
const mediaStatuses = ref({})
const mediaStatusLoading = ref({})
const compareSelected = ref([])
const compareLoading = ref(false)
const compareResult = ref(null)
const retryingJob = ref(false)
const cancelingJob = ref(false)
const variantGeneratingId = ref(null)
const reviewSummaryLoading = ref(false)
const reviewSummary = ref(null)
const reviewSummaryError = ref('')
const evidenceLoading = ref(false)
const evidencePackage = ref(null)
const evidenceError = ref('')
const checklistLoading = ref(false)
const studyChecklist = ref(null)
const checklistError = ref('')
const exportLoading = ref(false)
const exportPayload = ref(null)
const exportError = ref('')
const outlineLoading = ref(false)
const presentationOutline = ref(null)
const outlineError = ref('')
const demoScriptLoading = ref(false)
const demoScript = ref(null)
const demoScriptError = ref('')

const dimensionOrder = [
  'knowledge_foundation',
  'learning_goal',
  'learning_progress',
  'cognitive_style',
  'error_preference',
  'resource_preference',
]

const resourceTypeOptions = [
  { value: 'explanation_doc', label: '课程讲解文档', hint: '结构化知识讲解' },
  { value: 'mind_map', label: '知识点思维导图', hint: 'Mermaid 导图' },
  { value: 'practice_questions', label: '练习题目', hint: '含解析与易错提醒' },
  { value: 'coding_case', label: '代码实操案例', hint: '可落地项目片段' },
  { value: 'reading_material', label: '拓展阅读材料', hint: '补充资料与阅读路径' },
  { value: 'video_script', label: '短视频讲解脚本', hint: '可选加分项' },
]

const resourceTypeLabels = {
  ...Object.fromEntries(resourceTypeOptions.map((item) => [item.value, item.label])),
  generated_image: 'AI 生成配图',
  generated_video: 'AI 生成短视频',
}

const difficultyLabels = {
  easy: '偏易',
  fit: '适中',
  hard: '偏难',
}

const activeProfile = computed(() => profilePayload.value?.profile || null)

const orderedDimensions = computed(() => {
  const dimensions = activeProfile.value?.dimensions || []
  return [...dimensions].sort((left, right) => {
    const leftIndex = dimensionOrder.indexOf(left.key)
    const rightIndex = dimensionOrder.indexOf(right.key)
    return (leftIndex === -1 ? 99 : leftIndex) - (rightIndex === -1 ? 99 : rightIndex)
  })
})

const hasProfile = computed(() => Boolean(activeProfile.value?.dimensions?.length))

const stepProgress = computed(() => {
  if (!activeJob.value?.steps?.length) return 0
  const completed = activeJob.value.steps.filter((step) => step.status === 'completed').length
  return Math.round((completed / activeJob.value.steps.length) * 100)
})

const profileSummary = computed(() => activeProfile.value?.summary || '尚未生成画像')

const resourceContextLabel = computed(() => {
  const name = resourceForm.value.knowledgeName?.trim()
  const course = resourceForm.value.courseTitle?.trim()
  if (!name) return ''
  return course ? `${course} · ${name}` : name
})

const resourceLibraryFilter = computed(() => (libraryScope.value === 'current' ? resourceForm.value.knowledgeId.trim() : ''))

const currentKnowledgeResources = computed(() => {
  const knowledgeID = resourceLibraryFilter.value
  if (!knowledgeID) return resourceLibrary.value
  return resourceLibrary.value.filter((resource) => resource.knowledgeId === knowledgeID)
})

const canCompareResources = computed(() => compareSelected.value.length >= 2)

function dimensionValue(key) {
  return orderedDimensions.value.find((item) => item.key === key)?.value || ''
}

function applyProfileTarget(profile) {
  const goal = profile?.dimensions?.find((item) => item.key === 'learning_goal')?.value || ''
  if (!resourceForm.value.target && (goal || profile?.summary)) {
    resourceForm.value.target = goal || profile.summary
  }
}

function firstQueryValue(value) {
  return Array.isArray(value) ? value[0] || '' : value || ''
}

function applyRouteContext() {
  const knowledgeId = String(firstQueryValue(route.query.knowledgeId)).trim()
  const knowledgeName = String(firstQueryValue(route.query.knowledgeName)).trim()
  const courseTitle = String(firstQueryValue(route.query.courseTitle)).trim()
  const target = String(firstQueryValue(route.query.target)).trim()

  if (!knowledgeId && !knowledgeName && !courseTitle && !target) return

  resourceForm.value = {
    ...resourceForm.value,
    knowledgeId: knowledgeId || resourceForm.value.knowledgeId,
    knowledgeName: knowledgeName || resourceForm.value.knowledgeName,
    courseTitle: courseTitle || resourceForm.value.courseTitle,
    target: target || resourceForm.value.target,
  }
  routeContextNotice.value = '已从知识点页带入生成上下文，资源会自动关联当前知识点。'
}

function restoreResourcePreferences() {
  try {
    const raw = window.localStorage.getItem(RESOURCE_FORM_STORAGE_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    if (saved?.form) {
      resourceForm.value = {
        ...resourceForm.value,
        ...saved.form,
        resourceTypes: Array.isArray(saved.form.resourceTypes) && saved.form.resourceTypes.length
          ? saved.form.resourceTypes
          : resourceForm.value.resourceTypes,
      }
    }
    if (typeof saved?.mediaPrompt === 'string') {
      mediaPrompt.value = saved.mediaPrompt
    }
  } catch {
    window.localStorage.removeItem(RESOURCE_FORM_STORAGE_KEY)
  }
}

function persistResourcePreferences() {
  try {
    window.localStorage.setItem(RESOURCE_FORM_STORAGE_KEY, JSON.stringify({
      form: {
        ...resourceForm.value,
        resourceTypes: [...resourceForm.value.resourceTypes],
      },
      mediaPrompt: mediaPrompt.value,
    }))
  } catch {
    // Ignore storage quota/private-mode failures; generation still works.
  }
}

async function loadCurrentProfile() {
  profileError.value = ''
  try {
    const payload = await fetchLearningProfile()
    profilePayload.value = payload
    profileRefreshInsights.value = payload?.refreshInsights || []
    profileRecentSignals.value = payload?.recentSignals || []
    applyProfileTarget(payload?.profile)
  } catch (err) {
    profilePayload.value = { profile: null, hasCompletedProfile: false }
    profileError.value = err?.message || '学习画像读取失败。'
  }
}

async function loadProfileTimeline() {
  profileTimelineLoading.value = true
  try {
    const payload = await fetchProfileTimeline()
    profileTimeline.value = payload?.items || []
    profileRecentSignals.value = payload?.recentEvents || profileRecentSignals.value
  } catch {
    profileTimeline.value = []
  } finally {
    profileTimelineLoading.value = false
  }
}

async function handleRefreshProfile() {
  profileLoading.value = true
  profileError.value = ''
  profileSavedNotice.value = ''
  try {
    const payload = await refreshLearningProfile()
    profilePayload.value = payload
    profileRefreshInsights.value = payload?.refreshInsights || []
    profileRecentSignals.value = payload?.recentSignals || []
    profileSavedNotice.value = '已根据最近学习行为刷新画像。'
    await loadProfileTimeline()
  } catch (err) {
    profileError.value = err?.message || '画像刷新失败，请稍后重试。'
  } finally {
    profileLoading.value = false
  }
}

async function handleExtractProfile() {
  const message = profilePrompt.value.trim()
  if (!message) {
    profileError.value = '先写一段你的学习基础、目标和偏好。'
    return
  }

  profileLoading.value = true
  profileError.value = ''
  profileSavedNotice.value = ''
  try {
    const payload = await extractLearningProfile(message)
    profilePayload.value = payload
    profileRefreshInsights.value = payload?.refreshInsights || []
    resourceForm.value.target = ''
    applyProfileTarget(payload?.profile)
    await loadProfileTimeline()
  } catch (err) {
    profileError.value = err?.message || '画像生成失败，请稍后重试。'
  } finally {
    profileLoading.value = false
  }
}

async function handleSaveProfile() {
  if (!hasProfile.value) {
    profileError.value = '请先生成学习画像后再确认。'
    return
  }

  profileSaving.value = true
  profileError.value = ''
  profileSavedNotice.value = ''
  try {
    const payload = await saveLearningProfile({
      summary: activeProfile.value.summary,
      sourceText: profilePrompt.value.trim(),
      dimensions: activeProfile.value.dimensions,
    })
    profilePayload.value = payload
    profileSavedNotice.value = '学习画像已确认，资源生成会优先使用这些维度。'
    await loadProfileTimeline()
  } catch (err) {
    profileError.value = err?.message || '画像保存失败，请稍后重试。'
  } finally {
    profileSaving.value = false
  }
}

function toggleResourceType(type) {
  const current = new Set(resourceForm.value.resourceTypes)
  if (current.has(type)) {
    current.delete(type)
  } else {
    current.add(type)
  }
  resourceForm.value.resourceTypes = [...current]
}

function isResourceTypeSelected(type) {
  return resourceForm.value.resourceTypes.includes(type)
}

const pollToken = ref(0)
let pollTimer = null
let streamController = null

const jobHeadline = computed(() => {
  const job = activeJob.value
  if (!job) return ''
  if (job.status === 'completed') return job.summary || '资源包生成完成，可查看、导出与提交反馈'
  if (job.status === 'canceled') return job.retrySuggestion || '任务已停止，可调整参数后重新发起'
  if (job.status === 'failed') return job.failureReason || job.errorMessage || '资源生成失败，请稍后重试'
  return `正在执行：${job.currentAgent || '多智能体协同'}`
})

function stopPolling() {
  pollToken.value += 1
  stopStreaming()
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function stopStreaming() {
  if (streamController) {
    streamController.abort()
    streamController = null
  }
}

function clearJobArtifacts() {
  exportPayload.value = null
  presentationOutline.value = null
  demoScript.value = null
  reviewSummary.value = null
  reviewSummaryError.value = ''
  evidencePackage.value = null
  evidenceError.value = ''
  studyChecklist.value = null
  checklistError.value = ''
}

function waitForNextPoll(ms, token) {
  return new Promise((resolve) => {
    pollTimer = setTimeout(() => {
      pollTimer = null
      resolve(token === pollToken.value)
    }, ms)
  })
}

// pollJobUntilDone polls the generation job so the progress bar, agent steps and
// resource cards update live until the job completes, fails or times out.
async function pollJobUntilDone(jobId) {
  const token = ++pollToken.value
  const deadline = Date.now() + 180000
  while (Date.now() < deadline) {
    const stillCurrent = await waitForNextPoll(1400, token)
    if (!stillCurrent) return null
    let job
    try {
      job = await fetchResourceJob(jobId)
    } catch {
      continue
    }
    if (token !== pollToken.value) return null
    activeJob.value = job
    if (job?.resources?.length) generatedResources.value = job.resources
    if (['completed', 'failed', 'canceled'].includes(job?.status)) return job
  }
  return activeJob.value
}

// streamJobUntilDone prefers the backend SSE stream and falls back to polling
// when the browser/proxy interrupts streaming.
async function streamJobUntilDone(jobId) {
  stopStreaming()
  streamController = new AbortController()
  let finalJob = activeJob.value
  try {
    await streamResourceJob(jobId, {
      signal: streamController.signal,
      onEvent: (payload) => {
        if (!payload || payload.error || !payload.id) return
        activeJob.value = payload
        finalJob = payload
        if (payload.resources?.length) generatedResources.value = payload.resources
      },
    })
    return finalJob
  } catch {
    return pollJobUntilDone(jobId)
  } finally {
    streamController = null
  }
}

async function handleGenerateResources() {
  if (!resourceForm.value.resourceTypes.length) {
    resourceError.value = '至少选择一种资源类型。'
    return
  }

  const knowledgeName = resourceForm.value.knowledgeName.trim()
  if (!knowledgeName) {
    resourceError.value = '请填写知识点名称。'
    return
  }

  stopPolling()
  resourceLoading.value = true
  resourceError.value = ''
  generatedResources.value = []
  clearJobArtifacts()
  try {
    const payload = await generateResources({
      knowledgeId: resourceForm.value.knowledgeId.trim(),
      knowledgeName,
      courseTitle: resourceForm.value.courseTitle.trim(),
      target: resourceForm.value.target.trim() || profileSummary.value,
      resourceTypes: resourceForm.value.resourceTypes,
      images: attachedImages.value.map((item) => item.dataUrl),
    })
    activeJob.value = payload?.job || null
    generatedResources.value = payload?.resources || []

    if (activeJob.value?.id && activeJob.value.status !== 'completed') {
      const finalJob = await streamJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '资源生成失败，请稍后重试。'
      }
    }

    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
    await loadResourceFeed()
  } catch (err) {
    resourceError.value = err?.message || '资源生成失败，请稍后重试。'
  } finally {
    resourceLoading.value = false
  }
}

function buildMediaPayload() {
  return {
    knowledgeId: resourceForm.value.knowledgeId.trim(),
    knowledgeName: resourceForm.value.knowledgeName.trim(),
    courseTitle: resourceForm.value.courseTitle.trim(),
    target: resourceForm.value.target.trim() || profileSummary.value,
    prompt: mediaPrompt.value.trim(),
  }
}

async function handleGenerateMedia(kind) {
  const knowledgeName = resourceForm.value.knowledgeName.trim()
  if (!knowledgeName) {
    mediaError.value = '请先填写知识点名称。'
    return
  }
  mediaLoading.value = kind
  mediaError.value = ''
  mediaNotice.value = ''
  try {
    const payload = kind === 'video'
      ? await generateResourceVideo({ ...buildMediaPayload(), size: '1280x720', durationSec: 6 })
      : await generateResourceImage({ ...buildMediaPayload(), size: '1024x1024' })
    if (payload?.resource?.id) {
      generatedResources.value = [payload.resource, ...generatedResources.value]
      initFeedbackDrafts(generatedResources.value)
    }
    mediaNotice.value = kind === 'video'
      ? '真实文生视频任务已完成或已提交，结果已进入资源库。'
      : '真实文生图已生成，结果已进入资源库。'
    await loadResourceLibrary()
    await loadRecommendations()
    await loadResourceFeed()
  } catch (err) {
    mediaError.value = err?.message || (kind === 'video' ? '视频生成失败，请检查媒体模型配置。' : '图片生成失败，请检查媒体模型配置。')
  } finally {
    mediaLoading.value = ''
  }
}

function onSelectImages(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  for (const file of files) {
    if (attachedImages.value.length >= 3) break
    if (!file.type.startsWith('image/')) continue
    if (file.size > 4 * 1024 * 1024) {
      resourceError.value = '单张图片需小于 4MB。'
      continue
    }
    const reader = new FileReader()
    reader.onload = () => {
      attachedImages.value = [...attachedImages.value, { name: file.name, dataUrl: String(reader.result) }]
    }
    reader.readAsDataURL(file)
  }
}

function removeImage(index) {
  attachedImages.value = attachedImages.value.filter((_, position) => position !== index)
}

// handleRegenerate re-runs generation for a single resource type at an easier or
// harder level, reusing the same async job + progress polling.
async function handleRegenerate(resource, difficulty) {
  if (!resource?.resourceType || regeneratingId.value !== null || resourceLoading.value) return

  stopPolling()
  regeneratingId.value = resource.id
  resourceError.value = ''
  clearJobArtifacts()
  try {
    const payload = await generateResources({
      knowledgeId: resource.knowledgeId || resourceForm.value.knowledgeId.trim(),
      knowledgeName: resource.knowledgeName || resourceForm.value.knowledgeName.trim(),
      courseTitle: resource.courseTitle || resourceForm.value.courseTitle.trim(),
      target: resourceForm.value.target.trim() || profileSummary.value,
      resourceTypes: [resource.resourceType],
      difficulty,
    })
    activeJob.value = payload?.job || null
    generatedResources.value = payload?.resources || []

    if (activeJob.value?.id && activeJob.value.status !== 'completed') {
      const finalJob = await streamJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '重新生成失败，请稍后重试。'
      }
    }

    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
    await loadResourceFeed()
  } catch (err) {
    resourceError.value = err?.message || '重新生成失败，请稍后重试。'
  } finally {
    regeneratingId.value = null
  }
}

async function handleGenerateVariant(resource, mode) {
  if (!resource?.id || variantGeneratingId.value !== null || resourceLoading.value) return

  stopPolling()
  variantGeneratingId.value = resource.id
  resourceError.value = ''
  clearJobArtifacts()
  try {
    const payload = await generateResourceVariant(resource.id, {
      mode,
      difficulty: mode === 'deepen' ? 'harder' : mode === 'simplify' ? 'easier' : 'normal',
      target: resourceForm.value.target.trim() || profileSummary.value,
    })
    activeJob.value = payload?.job || null
    generatedResources.value = payload?.resources || []
    if (activeJob.value?.id && activeJob.value.status !== 'completed') {
      const finalJob = await streamJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '变体生成失败，请稍后重试。'
      }
    }
    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
    await loadResourceFeed()
  } catch (err) {
    resourceError.value = err?.message || '变体生成失败，请稍后重试。'
  } finally {
    variantGeneratingId.value = null
  }
}

async function handleRetryJob() {
  if (!activeJob.value?.id || retryingJob.value || resourceLoading.value) return

  stopPolling()
  retryingJob.value = true
  resourceError.value = ''
  generatedResources.value = []
  clearJobArtifacts()
  try {
    const payload = await retryResourceJob(activeJob.value.id, {
      target: resourceForm.value.target.trim() || activeJob.value.target || profileSummary.value,
      resourceTypes: resourceForm.value.resourceTypes,
      images: attachedImages.value.map((item) => item.dataUrl),
    })
    activeJob.value = payload?.job || null
    generatedResources.value = payload?.resources || []
    if (activeJob.value?.id && activeJob.value.status !== 'completed') {
      const finalJob = await streamJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '重试任务仍然失败，请检查模型配置或稍后再试。'
      }
    }
    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
    await loadResourceFeed()
  } catch (err) {
    resourceError.value = err?.message || '重试任务失败，请稍后再试。'
  } finally {
    retryingJob.value = false
  }
}

async function handleCancelJob() {
  if (!activeJob.value?.id || cancelingJob.value) return

  cancelingJob.value = true
  resourceError.value = ''
  try {
    const job = await cancelResourceJob(activeJob.value.id)
    activeJob.value = job
    if (job?.resources?.length) generatedResources.value = job.resources
    stopPolling()
  } catch (err) {
    resourceError.value = err?.message || '停止任务失败，请稍后重试。'
  } finally {
    cancelingJob.value = false
    resourceLoading.value = false
  }
}

async function handleLoadPresentationOutline() {
  if (!activeJob.value?.id) return

  outlineLoading.value = true
  outlineError.value = ''
  try {
    presentationOutline.value = await fetchResourcePresentationOutline(activeJob.value.id)
  } catch (err) {
    outlineError.value = err?.message || 'PPT 大纲生成失败，请稍后重试。'
  } finally {
    outlineLoading.value = false
  }
}

async function handleLoadDemoScript() {
  if (!activeJob.value?.id) return

  demoScriptLoading.value = true
  demoScriptError.value = ''
  try {
    demoScript.value = await fetchResourceDemoScript(activeJob.value.id)
  } catch (err) {
    demoScriptError.value = err?.message || '演示脚本生成失败，请稍后重试。'
  } finally {
    demoScriptLoading.value = false
  }
}

async function handleExportJob() {
  if (!activeJob.value?.id) return

  exportLoading.value = true
  exportError.value = ''
  try {
    exportPayload.value = await exportResourceJobMarkdown(activeJob.value.id)
  } catch (err) {
    exportError.value = err?.message || '资源包导出失败，请稍后重试。'
  } finally {
    exportLoading.value = false
  }
}

async function handleLoadReviewSummary() {
  if (!activeJob.value?.id) return

  reviewSummaryLoading.value = true
  reviewSummaryError.value = ''
  try {
    reviewSummary.value = await fetchJobReviewSummary(activeJob.value.id)
  } catch (err) {
    reviewSummaryError.value = err?.message || '任务评审摘要生成失败，请稍后重试。'
  } finally {
    reviewSummaryLoading.value = false
  }
}

async function handleLoadEvidencePackage() {
  if (!activeJob.value?.id) return

  evidenceLoading.value = true
  evidenceError.value = ''
  try {
    evidencePackage.value = await fetchJobEvidencePackage(activeJob.value.id)
  } catch (err) {
    evidenceError.value = err?.message || '证据包生成失败，请稍后重试。'
  } finally {
    evidenceLoading.value = false
  }
}

async function handleLoadStudyChecklist() {
  if (!activeJob.value?.id) return

  checklistLoading.value = true
  checklistError.value = ''
  try {
    studyChecklist.value = await fetchStudyChecklist(activeJob.value.id)
  } catch (err) {
    checklistError.value = err?.message || '学习清单生成失败，请稍后重试。'
  } finally {
    checklistLoading.value = false
  }
}

async function loadResourceLibrary() {
  libraryLoading.value = true
  try {
    const payload = await fetchResources({
      limit: 50,
      knowledgeId: resourceLibraryFilter.value,
      type: libraryTypeFilter.value,
      keyword: libraryKeyword.value.trim(),
      status: libraryStatusFilter.value,
      sort: librarySort.value,
    })
    resourceLibrary.value = payload?.list || []
  } catch {
    resourceLibrary.value = []
  } finally {
    libraryLoading.value = false
  }
}

async function loadRecommendations() {
  recommendationsLoading.value = true
  try {
    const payload = await fetchResourceRecommendations()
    recommendations.value = payload?.list || []
    recommendationSummary.value = payload?.summary || ''
    feedbackSummary.value = payload?.feedbackSummary || null
  } catch {
    recommendations.value = []
    recommendationSummary.value = ''
    feedbackSummary.value = null
  } finally {
    recommendationsLoading.value = false
  }
}

async function loadResourceFeed() {
  resourceFeedLoading.value = true
  try {
    const payload = await fetchResourceFeed()
    resourceFeed.value = payload?.items || []
    resourceFeedSummary.value = payload?.summary || ''
  } catch {
    resourceFeed.value = []
    resourceFeedSummary.value = ''
  } finally {
    resourceFeedLoading.value = false
  }
}

function initFeedbackDrafts(resources = []) {
  const nextDrafts = { ...feedbackDrafts.value }
  resources.forEach((resource) => {
    if (!resource?.id || nextDrafts[resource.id]) return
    nextDrafts[resource.id] = {
      rating: 4,
      difficulty: 'fit',
      helpful: true,
      completed: false,
      comment: '',
    }
  })
  feedbackDrafts.value = nextDrafts
}

async function handleSubmitFeedback(resource) {
  const draft = feedbackDrafts.value[resource.id]
  if (!draft) return

  feedbackSaving.value = { ...feedbackSaving.value, [resource.id]: true }
  feedbackNotice.value = ''
  feedbackError.value = ''
  try {
    const payload = await submitResourceFeedback(resource.id, draft)
    patchResourceLearningStatus(resource.id, {
      learningStatus: learningStatusFromFeedback(draft),
      actionText: payload?.nextAction || '',
    })
    feedbackNotice.value = payload?.nextAction || '反馈已记录，推荐策略已更新。'
    await Promise.all([loadRecommendations(), loadResourceFeed(), handleRefreshProfile()])
  } catch (err) {
    feedbackError.value = err?.message || '反馈提交失败，请稍后重试。'
  } finally {
    feedbackSaving.value = { ...feedbackSaving.value, [resource.id]: false }
  }
}

function patchResourceLearningStatus(resourceId, payload) {
  const applyStatus = (resource) => {
    if (resource.id !== resourceId) return resource
    return {
      ...resource,
      learningStatus: payload.learningStatus,
      actionText: payload.actionText,
    }
  }
  generatedResources.value = generatedResources.value.map(applyStatus)
  resourceLibrary.value = resourceLibrary.value.map(applyStatus)
}

async function handleUpdateLearningStatus(resource, status) {
  if (!resource?.id || statusUpdating.value[resource.id]) return

  statusUpdating.value = { ...statusUpdating.value, [resource.id]: status }
  feedbackNotice.value = ''
  feedbackError.value = ''
  try {
    const payload = await updateResourceLearningStatus(resource.id, { status })
    patchResourceLearningStatus(resource.id, payload)
    feedbackNotice.value = `已标记为「${learningStatusLabel(payload.learningStatus)}」。`
    await Promise.all([loadResourceFeed(), loadRecommendations()])
  } catch (err) {
    feedbackError.value = err?.message || '学习状态更新失败，请稍后重试。'
  } finally {
    statusUpdating.value = { ...statusUpdating.value, [resource.id]: '' }
  }
}

async function handleLoadSafetyReport(resource) {
  if (!resource?.id) return
  if (safetyReports.value[resource.id]) {
    safetyReports.value = { ...safetyReports.value, [resource.id]: null }
    return
  }

  safetyLoading.value = { ...safetyLoading.value, [resource.id]: true }
  try {
    const payload = await fetchResourceSafetyReport(resource.id)
    safetyReports.value = { ...safetyReports.value, [resource.id]: payload }
  } catch (err) {
    safetyReports.value = {
      ...safetyReports.value,
      [resource.id]: {
        status: 'attention_required',
        score: 0,
        checks: [],
        risks: [err?.message || '安全报告读取失败。'],
        suggestions: ['请稍后重试或人工复核资源内容。'],
      },
    }
  } finally {
    safetyLoading.value = { ...safetyLoading.value, [resource.id]: false }
  }
}

async function handleLoadQualityReport(resource) {
  if (!resource?.id) return
  if (qualityReports.value[resource.id]) {
    qualityReports.value = { ...qualityReports.value, [resource.id]: null }
    return
  }

  qualityLoading.value = { ...qualityLoading.value, [resource.id]: true }
  try {
    const payload = await fetchResourceQualityReport(resource.id)
    qualityReports.value = { ...qualityReports.value, [resource.id]: payload }
  } catch (err) {
    qualityReports.value = {
      ...qualityReports.value,
      [resource.id]: {
        score: 0,
        level: 'needs_improvement',
        metrics: [],
        strengths: [],
        improvements: [err?.message || '质量报告读取失败。'],
      },
    }
  } finally {
    qualityLoading.value = { ...qualityLoading.value, [resource.id]: false }
  }
}

async function handleLoadMediaStatus(resource) {
  if (!resource?.id || !isMediaResource(resource)) return
  mediaStatusLoading.value = { ...mediaStatusLoading.value, [resource.id]: true }
  try {
    const payload = await fetchResourceMediaStatus(resource.id)
    mediaStatuses.value = { ...mediaStatuses.value, [resource.id]: payload }
  } catch (err) {
    mediaStatuses.value = {
      ...mediaStatuses.value,
      [resource.id]: {
        status: 'unknown',
        ready: false,
        hint: err?.message || '媒体状态读取失败。',
      },
    }
  } finally {
    mediaStatusLoading.value = { ...mediaStatusLoading.value, [resource.id]: false }
  }
}

function toggleCompare(resource) {
  if (!resource?.id) return
  const id = Number(resource.id)
  if (compareSelected.value.includes(id)) {
    compareSelected.value = compareSelected.value.filter((item) => item !== id)
    return
  }
  if (compareSelected.value.length >= 6) {
    compareSelected.value = compareSelected.value.slice(1)
  }
  compareSelected.value = [...compareSelected.value, id]
  compareResult.value = null
}

async function handleCompareResources() {
  if (!canCompareResources.value) return
  compareLoading.value = true
  try {
    compareResult.value = await compareResources(compareSelected.value)
  } catch (err) {
    compareResult.value = {
      winnerId: 0,
      summary: err?.message || '资源对比失败，请稍后重试。',
      items: [],
    }
  } finally {
    compareLoading.value = false
  }
}

function applyRecommendation(recommendation) {
  resourceForm.value.knowledgeId = recommendation.knowledgeId || ''
  resourceForm.value.knowledgeName = recommendation.knowledgeName || resourceForm.value.knowledgeName
  resourceForm.value.courseTitle = recommendation.courseTitle || resourceForm.value.courseTitle
  resourceForm.value.target = recommendation.target || resourceForm.value.target
  resourceForm.value.resourceTypes = recommendation.resourceTypes?.length
    ? [...recommendation.resourceTypes]
    : resourceForm.value.resourceTypes
}

function formatPercent(confidence) {
  const value = Number(confidence || 0)
  return `${Math.round(value * 100)}%`
}

function formatDate(value) {
  if (!value) return '刚刚'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function typeLabel(type) {
  return resourceTypeLabels[type] || type
}

function difficultyLabel(value) {
  return difficultyLabels[value] || '适中'
}

function formatAverageRating(value) {
  const rating = Number(value || 0)
  return rating > 0 ? rating.toFixed(1) : '暂无'
}

function safetyStatusLabel(status) {
  if (status === 'passed') return '已通过'
  if (status === 'blocked') return '需阻断'
  return '需关注'
}

function qualityLevelLabel(level) {
  if (level === 'excellent') return '优秀'
  if (level === 'good') return '良好'
  if (level === 'usable') return '可用'
  return '需改进'
}

function learningStatusLabel(status) {
  if (status === 'completed') return '已完成'
  if (status === 'in_progress') return '学习中'
  if (status === 'needs_review') return '需复习'
  return '未开始'
}

function learningStatusFromFeedback(draft) {
  if (draft?.completed) return 'completed'
  if (draft?.rating <= 2 || !draft?.helpful || draft?.difficulty === 'hard') return 'needs_review'
  return 'in_progress'
}

function canCancelJob(job) {
  return job?.status === 'pending' || job?.status === 'running'
}

function isMediaResource(resource) {
  return resource?.format === 'image' || resource?.format === 'video' || resource?.resourceType === 'generated_image' || resource?.resourceType === 'generated_video'
}

onMounted(async () => {
  restoreResourcePreferences()
  applyRouteContext()
  await Promise.all([loadCurrentProfile(), loadProfileTimeline(), loadResourceLibrary(), loadRecommendations(), loadResourceFeed()])
})

watch(
  () => ({
    ...resourceForm.value,
    resourceTypes: [...resourceForm.value.resourceTypes],
    mediaPrompt: mediaPrompt.value,
  }),
  persistResourcePreferences,
  { deep: true },
)

watch(
  () => route.query,
  async () => {
    applyRouteContext()
    await loadResourceLibrary()
  },
)

watch(
  [libraryTypeFilter, libraryStatusFilter, librarySort, libraryScope],
  loadResourceLibrary,
)

onBeforeUnmount(stopPolling)
</script>

<template>
  <section class="resource-center student-workspace">
    <header class="resource-topbar">
      <RouterLink class="resource-brand" to="/">
        <span class="resource-brand__icon">
          <BrainCircuit :size="22" aria-hidden="true" />
        </span>
        <span>
          <strong>{{ APP_NAME }}</strong>
          <small>个性化资源智能体</small>
        </span>
      </RouterLink>

      <div class="resource-topbar__actions">
        <RouterLink class="ghost-link" to="/">
          <BookOpen :size="16" aria-hidden="true" />
          知识图谱
        </RouterLink>
        <button type="button" class="ghost-link" :disabled="libraryLoading" @click="loadResourceLibrary">
          <RefreshCcw :size="16" aria-hidden="true" />
          刷新资源
        </button>
      </div>
    </header>

    <main class="resource-shell">
      <section class="profile-panel">
        <div class="panel-heading">
          <span class="panel-kicker">Feature 1</span>
          <h1>对话式学生画像</h1>
        </div>

        <label class="field-block">
          <span>学习情况描述</span>
          <textarea
            v-model="profilePrompt"
            rows="7"
            placeholder="描述你的专业、基础、目标、学习进度、偏好和卡点"
          />
        </label>

        <div class="button-row">
          <button type="button" class="primary-action" :disabled="profileLoading" @click="handleExtractProfile">
            <Wand2 v-if="!profileLoading" :size="16" aria-hidden="true" />
            <RefreshCcw v-else :size="16" aria-hidden="true" />
            {{ profileLoading ? '生成中' : '生成画像' }}
          </button>
          <button type="button" class="secondary-action" :disabled="profileSaving || !hasProfile" @click="handleSaveProfile">
            <Save :size="16" aria-hidden="true" />
            确认画像
          </button>
          <button type="button" class="secondary-action" :disabled="profileLoading || !hasProfile" @click="handleRefreshProfile">
            <RefreshCcw :size="16" aria-hidden="true" />
            随学刷新
          </button>
        </div>

        <p v-if="profileError" class="inline-error">{{ profileError }}</p>
        <p v-if="profileSavedNotice" class="inline-success">{{ profileSavedNotice }}</p>

        <div class="profile-summary">
          <span>画像摘要</span>
          <strong>{{ profileSummary }}</strong>
          <small v-if="activeProfile?.updatedAt">更新于 {{ formatDate(activeProfile.updatedAt) }}</small>
        </div>

        <div v-if="orderedDimensions.length" class="dimension-grid">
          <article v-for="dimension in orderedDimensions" :key="dimension.key" class="dimension-card">
            <div class="dimension-card__head">
              <strong>{{ dimension.name }}</strong>
              <span>{{ formatPercent(dimension.confidence) }}</span>
            </div>
            <p>{{ dimension.value }}</p>
            <small>{{ dimension.evidence }}</small>
          </article>
        </div>
        <div v-else class="empty-state">
          <Bot :size="22" aria-hidden="true" />
          <p>生成画像后，这里会展示知识基础、目标、进度、认知风格、易错偏好和资源偏好。</p>
        </div>

        <section class="profile-timeline">
          <div class="panel-heading panel-heading--compact">
            <span class="panel-kicker">Timeline</span>
            <h2>画像更新时间线</h2>
          </div>
          <div v-if="profileRefreshInsights.length" class="insight-list">
            <p v-for="item in profileRefreshInsights" :key="item">{{ item }}</p>
          </div>
          <div v-if="profileTimelineLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>正在读取画像历史</p>
          </div>
          <div v-else-if="profileTimeline.length" class="timeline-list">
            <article v-for="item in profileTimeline.slice(0, 4)" :key="item.id" class="timeline-item">
              <span>{{ item.title }} · {{ formatDate(item.createdAt) }}</span>
              <strong>{{ item.summary }}</strong>
              <small>{{ item.source }}</small>
            </article>
          </div>
          <div v-else class="empty-state empty-state--small">
            <Bot :size="18" aria-hidden="true" />
            <p>确认或刷新画像后，这里会记录每一次变化。</p>
          </div>
        </section>
      </section>

      <section class="generation-panel">
        <div class="panel-heading">
          <span class="panel-kicker">Feature 2</span>
          <h1>多智能体资源生成</h1>
        </div>

        <p v-if="routeContextNotice" class="context-note">
          {{ routeContextNotice }}
          <strong v-if="resourceContextLabel">{{ resourceContextLabel }}</strong>
        </p>

        <div class="generation-grid">
          <label class="field-block">
            <span>知识点名称</span>
            <input v-model="resourceForm.knowledgeName" type="text" placeholder="例如：智能体与工具调用" />
          </label>
          <label class="field-block">
            <span>知识点 ID</span>
            <input v-model="resourceForm.knowledgeId" type="text" placeholder="可选，关联图谱节点" />
          </label>
          <label class="field-block">
            <span>课程</span>
            <input v-model="resourceForm.courseTitle" type="text" placeholder="例如：人工智能导论" />
          </label>
          <label class="field-block field-block--wide">
            <span>生成目标</span>
            <textarea
              v-model="resourceForm.target"
              rows="3"
              :placeholder="dimensionValue('learning_goal') || '说明这次希望补齐的学习目标'"
            />
          </label>
        </div>

        <div class="resource-types">
          <button
            v-for="option in resourceTypeOptions"
            :key="option.value"
            type="button"
            class="type-chip"
            :class="{ 'type-chip--active': isResourceTypeSelected(option.value) }"
            @click="toggleResourceType(option.value)"
          >
            <CheckCircle2 :size="16" aria-hidden="true" />
            <span>{{ option.label }}</span>
            <small>{{ option.hint }}</small>
          </button>
        </div>

        <div class="image-upload">
          <div class="image-upload__head">
            <span>多模态图片（可选）</span>
            <small>上传教材页 / 题目截图 / 手写笔记，VisionReaderAgent 会读图作为生成依据</small>
          </div>
          <div class="image-upload__list">
            <div v-for="(image, index) in attachedImages" :key="index" class="image-thumb">
              <img :src="image.dataUrl" :alt="image.name" />
              <button type="button" class="image-thumb__remove" aria-label="移除图片" @click="removeImage(index)">×</button>
            </div>
            <label v-if="attachedImages.length < 3" class="image-add">
              <input type="file" accept="image/*" multiple hidden @change="onSelectImages" />
              <ImagePlus :size="18" aria-hidden="true" />
              <span>添加图片</span>
            </label>
          </div>
        </div>

        <div class="button-row button-row--end">
          <label class="field-block field-block--wide media-prompt-field">
            <span>图片/视频创意提示（可选）</span>
            <textarea v-model="mediaPrompt" rows="2" placeholder="例如：生成一张展示智能体协作流程的课程插图" />
          </label>
          <button type="button" class="secondary-action" :disabled="!!mediaLoading" @click="handleGenerateMedia('image')">
            <ImagePlus :size="16" aria-hidden="true" />
            {{ mediaLoading === 'image' ? '生成配图中' : '生成教学配图' }}
          </button>
          <button type="button" class="secondary-action" :disabled="!!mediaLoading" @click="handleGenerateMedia('video')">
            <PlayCircle :size="16" aria-hidden="true" />
            {{ mediaLoading === 'video' ? '生成视频中' : '生成短视频' }}
          </button>
          <button type="button" class="primary-action" :disabled="resourceLoading" @click="handleGenerateResources">
            <Sparkles v-if="!resourceLoading" :size="16" aria-hidden="true" />
            <RefreshCcw v-else :size="16" aria-hidden="true" />
            {{ resourceLoading ? '生成资源包中' : '生成资源包' }}
          </button>
        </div>
        <p v-if="resourceError" class="inline-error">{{ resourceError }}</p>
        <p v-if="mediaError" class="inline-error">{{ mediaError }}</p>
        <p v-if="mediaNotice" class="inline-success">{{ mediaNotice }}</p>
        <p v-if="feedbackError" class="inline-error">{{ feedbackError }}</p>
        <p v-if="feedbackNotice" class="inline-success">{{ feedbackNotice }}</p>

        <section v-if="activeJob" class="agent-progress">
          <div class="agent-progress__head">
            <div>
              <span>任务 #{{ activeJob.id }}</span>
              <strong>{{ jobHeadline }}</strong>
            </div>
            <em>{{ activeJob.progress || stepProgress }}%</em>
          </div>
          <div class="progress-bar">
            <span :style="{ width: `${activeJob.progress || stepProgress}%` }" />
          </div>
          <div class="button-row button-row--end">
            <button v-if="canCancelJob(activeJob)" type="button" class="secondary-action" :disabled="cancelingJob" @click="handleCancelJob">
              <CircleStop :size="16" aria-hidden="true" />
              {{ cancelingJob ? '停止中' : '停止生成' }}
            </button>
            <button v-if="activeJob.status === 'failed'" type="button" class="secondary-action" :disabled="retryingJob" @click="handleRetryJob">
              <RefreshCcw :size="16" aria-hidden="true" />
              {{ retryingJob ? '重试中' : '重试任务' }}
            </button>
            <button v-if="activeJob.status === 'canceled'" type="button" class="secondary-action" :disabled="retryingJob" @click="handleRetryJob">
              <RefreshCcw :size="16" aria-hidden="true" />
              {{ retryingJob ? '重试中' : '重新生成' }}
            </button>
            <button type="button" class="secondary-action" :disabled="reviewSummaryLoading || activeJob.status !== 'completed'" @click="handleLoadReviewSummary">
              <CheckCircle2 :size="16" aria-hidden="true" />
              {{ reviewSummaryLoading ? '评审中' : '任务评审摘要' }}
            </button>
            <button type="button" class="secondary-action" :disabled="checklistLoading || activeJob.status !== 'completed'" @click="handleLoadStudyChecklist">
              <FileText :size="16" aria-hidden="true" />
              {{ checklistLoading ? '生成中' : '学习清单' }}
            </button>
            <button type="button" class="secondary-action" :disabled="evidenceLoading || activeJob.status !== 'completed'" @click="handleLoadEvidencePackage">
              <Sparkles :size="16" aria-hidden="true" />
              {{ evidenceLoading ? '打包中' : '证据包 JSON' }}
            </button>
            <button type="button" class="secondary-action" :disabled="outlineLoading || activeJob.status !== 'completed'" @click="handleLoadPresentationOutline">
              <Sparkles :size="16" aria-hidden="true" />
              {{ outlineLoading ? '生成中' : '生成 PPT 大纲' }}
            </button>
            <button type="button" class="secondary-action" :disabled="demoScriptLoading || activeJob.status !== 'completed'" @click="handleLoadDemoScript">
              <Send :size="16" aria-hidden="true" />
              {{ demoScriptLoading ? '生成中' : '生成演示脚本' }}
            </button>
            <button type="button" class="secondary-action" :disabled="exportLoading || activeJob.status !== 'completed'" @click="handleExportJob">
              <FileText :size="16" aria-hidden="true" />
              {{ exportLoading ? '导出中' : '导出 Markdown' }}
            </button>
          </div>
          <p v-if="outlineError" class="inline-error">{{ outlineError }}</p>
          <p v-if="demoScriptError" class="inline-error">{{ demoScriptError }}</p>
          <p v-if="exportError" class="inline-error">{{ exportError }}</p>
          <p v-if="reviewSummaryError" class="inline-error">{{ reviewSummaryError }}</p>
          <p v-if="evidenceError" class="inline-error">{{ evidenceError }}</p>
          <p v-if="checklistError" class="inline-error">{{ checklistError }}</p>
          <div v-if="activeJob.status === 'failed' && (activeJob.failureReason || activeJob.retrySuggestion)" class="failure-box">
            <strong>{{ activeJob.failureReason || '任务失败' }}</strong>
            <p>{{ activeJob.retrySuggestion || activeJob.errorMessage }}</p>
          </div>
          <div v-if="reviewSummary" class="review-box">
            <div class="review-box__head">
              <div>
                <strong>任务评审摘要</strong>
                <small>{{ reviewSummary.summary }}</small>
              </div>
              <span>{{ reviewSummary.averageQuality }} / {{ reviewSummary.averageSafety }}</span>
            </div>
            <div class="review-metrics">
              <article>
                <span>资源数</span>
                <strong>{{ reviewSummary.resourceCount }}</strong>
              </article>
              <article>
                <span>媒体就绪</span>
                <strong>{{ reviewSummary.mediaReadyCount }}</strong>
              </article>
              <article>
                <span>待跟踪</span>
                <strong>{{ reviewSummary.mediaPendingCount }}</strong>
              </article>
            </div>
            <div v-if="reviewSummary.suggestions?.length" class="safety-list">
              <strong>改进建议</strong>
              <p v-for="item in reviewSummary.suggestions" :key="item">{{ item }}</p>
            </div>
          </div>
          <div v-if="studyChecklist" class="checklist-box">
            <div class="review-box__head">
              <div>
                <strong>学习清单</strong>
                <small>{{ studyChecklist.summary }}</small>
              </div>
              <span>{{ studyChecklist.items?.length || 0 }} 项</span>
            </div>
            <article v-for="item in studyChecklist.items || []" :key="`${item.index}-${item.title}`">
              <span>{{ item.index }}</span>
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.action }}</p>
                <small>{{ item.expected }}</small>
              </div>
            </article>
          </div>
          <div v-if="presentationOutline" class="outline-box">
            <div class="outline-box__head">
              <div>
                <strong>{{ presentationOutline.title }}</strong>
                <small>{{ presentationOutline.subtitle }}</small>
              </div>
              <span>{{ presentationOutline.slides?.length || 0 }} 页</span>
            </div>
            <div class="outline-slides">
              <article v-for="slide in presentationOutline.slides || []" :key="slide.index" class="outline-slide">
                <span>Slide {{ slide.index }}</span>
                <strong>{{ slide.title }}</strong>
                <ul>
                  <li v-for="bullet in slide.bullets" :key="bullet">{{ bullet }}</li>
                </ul>
                <p>{{ slide.speakerNotes }}</p>
                <small>{{ slide.visualHint }}</small>
              </article>
            </div>
          </div>
          <div v-if="demoScript" class="outline-box">
            <div class="outline-box__head">
              <div>
                <strong>{{ demoScript.title }}</strong>
                <small>{{ demoScript.duration }} · 可直接用于比赛演示视频</small>
              </div>
              <span>{{ demoScript.scenes?.length || 0 }} 段</span>
            </div>
            <div class="outline-slides">
              <article v-for="scene in demoScript.scenes || []" :key="scene.index" class="outline-slide">
                <span>{{ scene.timeRange }}</span>
                <strong>{{ scene.title }}</strong>
                <ul>
                  <li v-for="action in scene.actions" :key="action">{{ action }}</li>
                </ul>
                <p>{{ scene.narration }}</p>
                <small>{{ (scene.proofPoints || []).join('；') }}</small>
              </article>
            </div>
          </div>
          <div v-if="exportPayload" class="export-box">
            <div class="export-box__head">
              <strong>{{ exportPayload.fileName }}</strong>
              <small>已生成可提交/可粘贴的资源包 Markdown</small>
            </div>
            <textarea readonly :value="exportPayload.content" rows="8" />
          </div>
          <div v-if="evidencePackage" class="export-box">
            <div class="export-box__head">
              <strong>{{ evidencePackage.fileName }}</strong>
              <small>已生成可提交的 JSON 证据包</small>
            </div>
            <textarea readonly :value="JSON.stringify(evidencePackage, null, 2)" rows="8" />
          </div>
          <div class="agent-steps">
            <article v-for="step in activeJob.steps || []" :key="step.id || step.agentKey" class="agent-step">
              <span class="agent-step__status" :class="`agent-step__status--${step.status}`" />
              <div>
                <strong>{{ step.agentName }}</strong>
                <p>{{ step.summary }}</p>
              </div>
            </article>
          </div>
        </section>

        <section v-if="generatedResources.length" class="resource-output">
          <div class="section-title">
            <FileText :size="18" aria-hidden="true" />
            <strong>本次生成结果</strong>
          </div>
          <div class="compare-toolbar">
            <span>已选择 {{ compareSelected.length }} 项用于质量对比</span>
            <button type="button" class="secondary-action" :disabled="!canCompareResources || compareLoading" @click="handleCompareResources">
              <Sparkles :size="16" aria-hidden="true" />
              {{ compareLoading ? '对比中' : '对比资源' }}
            </button>
          </div>
          <div v-if="compareResult" class="compare-result">
            <strong>{{ compareResult.summary }}</strong>
            <article v-for="item in compareResult.items || []" :key="item.resource.id">
              <span :class="{ 'compare-result__winner': item.resource.id === compareResult.winnerId }">
                {{ item.resource.title }}
              </span>
              <em>{{ item.quality.score }} 分 · {{ qualityLevelLabel(item.quality.level) }}</em>
              <p>{{ item.reason }}</p>
            </article>
          </div>
          <article v-for="resource in generatedResources" :key="resource.id" class="resource-card">
            <div class="resource-card__head">
              <span>{{ resource.typeLabel || typeLabel(resource.resourceType) }}</span>
              <strong>{{ resource.title }}</strong>
              <small>{{ resource.reason }}</small>
            </div>
            <ResourceContent :content="resource.content" :format="resource.format" />
            <div class="regen-actions">
              <span>难度不合适？让智能体重写或生成变体：</span>
              <button
                type="button"
                class="ghost-mini"
                :disabled="resourceLoading || regeneratingId !== null"
                @click="handleRegenerate(resource, 'easier')"
              >
                {{ regeneratingId === resource.id ? '重新生成中…' : '更简单一版' }}
              </button>
              <button
                type="button"
                class="ghost-mini"
                :disabled="resourceLoading || regeneratingId !== null"
                @click="handleRegenerate(resource, 'harder')"
              >
                更难一版
              </button>
              <button
                type="button"
                class="ghost-mini"
                :disabled="variantGeneratingId !== null || resourceLoading"
                @click="handleGenerateVariant(resource, 'example')"
              >
                {{ variantGeneratingId === resource.id ? '变体生成中…' : '补充例子' }}
              </button>
              <button
                type="button"
                class="ghost-mini"
                :disabled="variantGeneratingId !== null || resourceLoading"
                @click="handleGenerateVariant(resource, 'visual')"
              >
                可视化变体
              </button>
            </div>
            <div class="safety-actions">
              <RouterLink class="secondary-action" :to="{ name: 'resource-detail', params: { resourceId: resource.id } }">
                <FileText :size="16" aria-hidden="true" />
                详情页
              </RouterLink>
              <button type="button" class="secondary-action" @click="toggleCompare(resource)">
                <Sparkles :size="16" aria-hidden="true" />
                {{ compareSelected.includes(resource.id) ? '取消对比' : '加入对比' }}
              </button>
              <button
                v-if="isMediaResource(resource)"
                type="button"
                class="secondary-action"
                :disabled="mediaStatusLoading[resource.id]"
                @click="handleLoadMediaStatus(resource)"
              >
                <RefreshCcw :size="16" aria-hidden="true" />
                {{ mediaStatusLoading[resource.id] ? '刷新中' : '媒体状态' }}
              </button>
              <button type="button" class="secondary-action" :disabled="qualityLoading[resource.id]" @click="handleLoadQualityReport(resource)">
                <Sparkles :size="16" aria-hidden="true" />
                {{ qualityReports[resource.id] ? '收起质量报告' : qualityLoading[resource.id] ? '评分中' : '质量评分' }}
              </button>
              <button type="button" class="secondary-action" :disabled="safetyLoading[resource.id]" @click="handleLoadSafetyReport(resource)">
                <CheckCircle2 :size="16" aria-hidden="true" />
                {{ safetyReports[resource.id] ? '收起安全报告' : safetyLoading[resource.id] ? '审查中' : '安全审查' }}
              </button>
            </div>
            <div v-if="mediaStatuses[resource.id]" class="media-status-box">
              <strong>{{ mediaStatuses[resource.id].ready ? '媒体已就绪' : '媒体处理中' }}</strong>
              <span>{{ mediaStatuses[resource.id].status }}</span>
              <p>{{ mediaStatuses[resource.id].hint }}</p>
              <small v-if="mediaStatuses[resource.id].providerTaskId">任务号：{{ mediaStatuses[resource.id].providerTaskId }}</small>
            </div>
            <div v-if="qualityReports[resource.id]" class="quality-report">
              <div class="quality-report__head">
                <div>
                  <span>QualityJudgeAgent</span>
                  <strong>{{ qualityLevelLabel(qualityReports[resource.id].level) }}</strong>
                </div>
                <em>{{ qualityReports[resource.id].score }} 分</em>
              </div>
              <div class="quality-metrics">
                <article v-for="metric in qualityReports[resource.id].metrics || []" :key="metric.key">
                  <div>
                    <strong>{{ metric.label }}</strong>
                    <span>{{ metric.score }} 分</span>
                  </div>
                  <p>{{ metric.reason }}</p>
                </article>
              </div>
              <div class="safety-list">
                <strong>优势</strong>
                <p v-for="item in qualityReports[resource.id].strengths || []" :key="item">{{ item }}</p>
              </div>
              <div class="safety-list">
                <strong>改进项</strong>
                <p v-for="item in qualityReports[resource.id].improvements || []" :key="item">{{ item }}</p>
              </div>
            </div>
            <div v-if="safetyReports[resource.id]" class="safety-report">
              <div class="safety-report__head">
                <div>
                  <span>SafetyReviewerAgent</span>
                  <strong>{{ safetyStatusLabel(safetyReports[resource.id].status) }}</strong>
                </div>
                <em>{{ safetyReports[resource.id].score }} 分</em>
              </div>
              <div v-if="safetyReports[resource.id].checks?.length" class="safety-checks">
                <article v-for="check in safetyReports[resource.id].checks" :key="check.key" class="safety-check">
                  <span :class="`safety-check__dot safety-check__dot--${check.status}`" />
                  <div>
                    <strong>{{ check.name }}</strong>
                    <p>{{ check.conclusion }}</p>
                  </div>
                </article>
              </div>
              <div v-if="safetyReports[resource.id].risks?.length" class="safety-list">
                <strong>风险提示</strong>
                <p v-for="risk in safetyReports[resource.id].risks" :key="risk">{{ risk }}</p>
              </div>
              <div class="safety-list">
                <strong>审查建议</strong>
                <p v-for="suggestion in safetyReports[resource.id].suggestions || []" :key="suggestion">{{ suggestion }}</p>
              </div>
            </div>
            <div class="learning-status-panel">
              <div>
                <strong>学习状态</strong>
                <small>{{ resource.actionText || learningStatusLabel(resource.learningStatus) }}</small>
              </div>
              <div class="status-actions">
                <button
                  v-for="item in [
                    { value: 'pending', label: '未开始' },
                    { value: 'in_progress', label: '学习中' },
                    { value: 'completed', label: '已完成' },
                    { value: 'needs_review', label: '需复习' },
                  ]"
                  :key="item.value"
                  type="button"
                  class="ghost-mini"
                  :class="{ 'ghost-mini--active': (resource.learningStatus || 'pending') === item.value }"
                  :disabled="Boolean(statusUpdating[resource.id])"
                  @click="handleUpdateLearningStatus(resource, item.value)"
                >
                  {{ statusUpdating[resource.id] === item.value ? '更新中' : item.label }}
                </button>
              </div>
            </div>
            <div v-if="feedbackDrafts[resource.id]" class="feedback-box">
              <div class="feedback-box__head">
                <strong>学习反馈</strong>
                <small>用于更新学习效果评估和下一步推荐</small>
              </div>
              <div class="feedback-grid">
                <label class="field-block">
                  <span>评分</span>
                  <select v-model.number="feedbackDrafts[resource.id].rating">
                    <option v-for="score in [5, 4, 3, 2, 1]" :key="score" :value="score">
                      {{ score }} 分
                    </option>
                  </select>
                </label>
                <label class="field-block">
                  <span>难度</span>
                  <select v-model="feedbackDrafts[resource.id].difficulty">
                    <option value="easy">偏易</option>
                    <option value="fit">适中</option>
                    <option value="hard">偏难</option>
                  </select>
                </label>
                <label class="check-row">
                  <input v-model="feedbackDrafts[resource.id].helpful" type="checkbox" />
                  有帮助
                </label>
                <label class="check-row">
                  <input v-model="feedbackDrafts[resource.id].completed" type="checkbox" />
                  已完成学习
                </label>
                <label class="field-block feedback-grid__comment">
                  <span>备注</span>
                  <textarea
                    v-model="feedbackDrafts[resource.id].comment"
                    rows="2"
                    placeholder="记录仍然卡住的地方或希望下轮资源调整的方向"
                  />
                </label>
              </div>
              <div class="button-row button-row--end">
                <button
                  type="button"
                  class="secondary-action"
                  :disabled="feedbackSaving[resource.id]"
                  @click="handleSubmitFeedback(resource)"
                >
                  <Send :size="16" aria-hidden="true" />
                  {{ feedbackSaving[resource.id] ? '提交中' : '提交反馈' }}
                </button>
              </div>
            </div>
          </article>
        </section>
      </section>

      <aside class="library-panel">
        <section class="recommend-panel">
          <div class="panel-heading panel-heading--compact">
            <span class="panel-kicker">Feature 3</span>
            <h2>路径资源推送</h2>
          </div>

          <p v-if="resourceFeedSummary" class="recommend-summary">{{ resourceFeedSummary }}</p>

          <div v-if="resourceFeedLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>正在整理资源推送流</p>
          </div>
          <div v-else-if="resourceFeed.length" class="recommend-list">
            <RouterLink
              v-for="item in resourceFeed.slice(0, 5)"
              :key="item.id"
              class="recommend-item recommend-item--link"
              :to="{ name: 'resource-detail', params: { resourceId: item.resource.id } }"
            >
              <span>{{ item.stage }} · 优先级 {{ item.priority }}</span>
              <strong>{{ item.resource.title }}</strong>
              <small>{{ item.reason }}</small>
              <div class="recommend-item__types">
                <em>{{ item.resource.typeLabel || typeLabel(item.resource.resourceType) }}</em>
                <em>{{ item.actionText }}</em>
              </div>
            </RouterLink>
          </div>
          <div v-else class="empty-state empty-state--small">
            <Sparkles :size="18" aria-hidden="true" />
            <p>生成资源包后，系统会按学习路径推送今日资源。</p>
          </div>
        </section>

        <section class="recommend-panel">
          <div class="panel-heading panel-heading--compact">
            <span class="panel-kicker">Feature 4</span>
            <h2>下一步推荐</h2>
          </div>

          <div v-if="feedbackSummary" class="feedback-summary">
            <span>学习效果</span>
            <strong>{{ formatAverageRating(feedbackSummary.averageRating) }}</strong>
            <small>
              {{ feedbackSummary.total }} 条反馈 · {{ feedbackSummary.completedCount }} 次完成 ·
              偏难 {{ feedbackSummary.difficulty?.hard || 0 }} 次
            </small>
          </div>

          <p v-if="recommendationSummary" class="recommend-summary">{{ recommendationSummary }}</p>

          <div v-if="recommendationsLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>正在生成下一步推荐</p>
          </div>
          <div v-else-if="recommendations.length" class="recommend-list">
            <article v-for="item in recommendations" :key="item.id" class="recommend-item">
              <span>优先级 {{ item.priority }}</span>
              <strong>{{ item.title }}</strong>
              <small>{{ item.reason }}</small>
              <div class="recommend-item__types">
                <em v-for="type in item.resourceTypes" :key="type">{{ typeLabel(type) }}</em>
              </div>
              <button type="button" class="secondary-action" @click="applyRecommendation(item)">
                <Sparkles :size="15" aria-hidden="true" />
                {{ item.actionText }}
              </button>
            </article>
          </div>
          <div v-else class="empty-state empty-state--small">
            <Sparkles :size="18" aria-hidden="true" />
            <p>生成画像或提交反馈后，这里会给出下一步资源建议。</p>
          </div>
        </section>

        <div class="panel-heading panel-heading--compact">
          <span class="panel-kicker">Library</span>
          <h2>资源库</h2>
        </div>
        <div class="library-filters">
          <select v-model="libraryScope" aria-label="资源范围">
            <option value="current">当前知识点</option>
            <option value="all">全部资源</option>
          </select>
          <input
            v-model="libraryKeyword"
            type="search"
            placeholder="搜索标题/知识点/正文"
            @keyup.enter="loadResourceLibrary"
          />
          <select v-model="libraryTypeFilter" aria-label="资源类型">
            <option value="">全部类型</option>
            <option v-for="item in resourceTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
            <option value="generated_image">AI 生成配图</option>
            <option value="generated_video">AI 生成短视频</option>
          </select>
          <select v-model="libraryStatusFilter" aria-label="学习状态">
            <option value="">全部状态</option>
            <option value="pending">未开始</option>
            <option value="in_progress">学习中</option>
            <option value="completed">已完成</option>
            <option value="needs_review">需复习</option>
          </select>
          <select v-model="librarySort" aria-label="排序方式">
            <option value="latest">最新优先</option>
            <option value="oldest">最早优先</option>
            <option value="title">按标题</option>
          </select>
          <button type="button" class="secondary-action" :disabled="libraryLoading" @click="loadResourceLibrary">
            <RefreshCcw :size="15" aria-hidden="true" />
            刷新
          </button>
        </div>
        <p v-if="resourceLibraryFilter" class="recommend-summary">
          仅显示「{{ resourceForm.knowledgeName || resourceForm.knowledgeId }}」的资源
        </p>

        <div v-if="libraryLoading" class="empty-state empty-state--small">
          <RefreshCcw :size="18" aria-hidden="true" />
          <p>正在读取资源库</p>
        </div>
        <div v-else-if="!currentKnowledgeResources.length" class="empty-state empty-state--small">
          <FileText :size="18" aria-hidden="true" />
          <p>{{ resourceLibraryFilter ? '当前知识点还没有生成资源。' : '生成资源后会沉淀在这里。' }}</p>
        </div>
        <div v-else class="library-list">
          <RouterLink
            v-for="resource in currentKnowledgeResources"
            :key="resource.id"
            class="library-item library-item--link"
            :to="{ name: 'resource-detail', params: { resourceId: resource.id } }"
          >
            <span>{{ resource.typeLabel || typeLabel(resource.resourceType) }}</span>
            <strong>{{ resource.title }}</strong>
            <small>{{ resource.knowledgeName }} · {{ learningStatusLabel(resource.learningStatus) }} · {{ formatDate(resource.createdAt) }}</small>
          </RouterLink>
        </div>
      </aside>
    </main>
  </section>
</template>

<style scoped>
.resource-center {
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(39, 111, 191, 0.12), transparent 30rem),
    linear-gradient(180deg, #f7fbff 0%, #eef4f7 100%);
  color: #172033;
}

.resource-topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.16);
  background: rgba(247, 251, 255, 0.86);
  backdrop-filter: blur(16px);
}

.resource-brand,
.ghost-link,
.primary-action,
.secondary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 0;
  border-radius: 8px;
  font: inherit;
  text-decoration: none;
  cursor: pointer;
}

.resource-brand {
  color: #172033;
}

.resource-brand__icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  border-radius: 8px;
  color: #ffffff;
  background: #215f9a;
}

.resource-brand strong,
.resource-brand small {
  display: block;
}

.resource-brand small {
  margin-top: 0.1rem;
  color: #64748b;
}

.resource-topbar__actions,
.button-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
}

.button-row--end {
  justify-content: flex-end;
}

.ghost-link {
  min-height: 2.5rem;
  padding: 0 0.9rem;
  border: 1px solid rgba(90, 112, 132, 0.18);
  color: #24415f;
  background: #ffffff;
}

.ghost-link:disabled,
.primary-action:disabled,
.secondary-action:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.resource-shell {
  display: grid;
  grid-template-columns: minmax(20rem, 0.9fr) minmax(28rem, 1.6fr) minmax(17rem, 0.75fr);
  gap: 1rem;
  padding: 1rem;
}

.profile-panel,
.generation-panel,
.library-panel {
  min-width: 0;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 18px 45px rgba(31, 46, 67, 0.08);
}

.profile-panel,
.generation-panel {
  padding: 1rem;
}

.library-panel {
  align-self: start;
  padding: 0.9rem;
  position: sticky;
  top: 5rem;
}

.panel-heading {
  margin-bottom: 0.95rem;
}

.panel-heading h1,
.panel-heading h2 {
  margin: 0.15rem 0 0;
  font-size: clamp(1.25rem, 2vw, 1.7rem);
  letter-spacing: 0;
}

.panel-heading--compact h2 {
  font-size: 1.1rem;
}

.panel-kicker {
  color: #176b87;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

.field-block {
  display: grid;
  gap: 0.45rem;
}

.field-block span {
  color: #43536a;
  font-size: 0.86rem;
  font-weight: 700;
}

.field-block input,
.field-block select,
.field-block textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid rgba(90, 112, 132, 0.22);
  border-radius: 8px;
  padding: 0.75rem 0.8rem;
  color: #172033;
  background: #ffffff;
  font: inherit;
  line-height: 1.55;
  outline: none;
  resize: vertical;
}

.field-block input:focus,
.field-block select:focus,
.field-block textarea:focus {
  border-color: #2f80b7;
  box-shadow: 0 0 0 3px rgba(47, 128, 183, 0.14);
}

.field-block--wide {
  grid-column: 1 / -1;
}

.media-prompt-field {
  flex: 1 1 100%;
}

.primary-action,
.secondary-action {
  min-height: 2.55rem;
  padding: 0 1rem;
  font-weight: 800;
}

.primary-action {
  color: #ffffff;
  background: #215f9a;
}

.secondary-action {
  border: 1px solid rgba(90, 112, 132, 0.24);
  color: #24415f;
  background: #f8fbfd;
}

.inline-error,
.inline-success {
  margin: 0.75rem 0 0;
  border-radius: 8px;
  padding: 0.7rem 0.8rem;
  font-size: 0.9rem;
}

.inline-error {
  color: #8f1d1d;
  background: #fff0f0;
}

.inline-success {
  color: #16613f;
  background: #eaf8f0;
}

.profile-summary {
  display: grid;
  gap: 0.35rem;
  margin-top: 1rem;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.9rem;
  background: #f4f9fd;
}

.profile-summary span,
.profile-summary small,
.dimension-card small,
.resource-card small,
.library-item small {
  color: #64748b;
}

.profile-summary strong,
.dimension-card p,
.resource-card strong,
.library-item strong {
  overflow-wrap: anywhere;
}

.dimension-grid {
  display: grid;
  gap: 0.7rem;
  margin-top: 0.8rem;
}

.dimension-card {
  display: grid;
  gap: 0.45rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.8rem;
  background: #ffffff;
}

.dimension-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
}

.dimension-card__head span {
  flex: 0 0 auto;
  border-radius: 999px;
  padding: 0.15rem 0.45rem;
  color: #176b87;
  background: #e8f6fb;
  font-size: 0.78rem;
  font-weight: 800;
}

.dimension-card p {
  margin: 0;
  line-height: 1.55;
}

.profile-timeline {
  margin-top: 1rem;
}

.insight-list {
  display: grid;
  gap: 0.4rem;
  margin-bottom: 0.7rem;
}

.insight-list p {
  margin: 0;
  border-radius: 8px;
  padding: 0.55rem 0.65rem;
  color: #176b87;
  background: #e8f6fb;
  line-height: 1.45;
}

.timeline-list {
  display: grid;
  gap: 0.55rem;
}

.timeline-item {
  display: grid;
  gap: 0.25rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.7rem;
  background: #ffffff;
}

.timeline-item span,
.timeline-item small {
  color: #64748b;
  font-size: 0.78rem;
}

.timeline-item strong {
  color: #172033;
  line-height: 1.45;
}

.empty-state {
  display: grid;
  place-items: center;
  gap: 0.6rem;
  margin-top: 1rem;
  border: 1px dashed rgba(90, 112, 132, 0.3);
  border-radius: 8px;
  padding: 1.2rem;
  color: #64748b;
  text-align: center;
}

.empty-state p {
  margin: 0;
}

.empty-state--small {
  margin-top: 0;
  padding: 0.9rem;
}

.generation-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.8rem;
}

.resource-types {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;
  margin: 0.9rem 0;
}

.type-chip {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.15rem 0.45rem;
  align-items: center;
  min-height: 4rem;
  border: 1px solid rgba(90, 112, 132, 0.2);
  border-radius: 8px;
  padding: 0.7rem;
  color: #334155;
  background: #ffffff;
  text-align: left;
  cursor: pointer;
}

.type-chip svg {
  color: #94a3b8;
}

.type-chip span,
.type-chip small {
  min-width: 0;
  overflow-wrap: anywhere;
}

.type-chip span {
  font-weight: 800;
}

.type-chip small {
  grid-column: 2;
  color: #64748b;
}

.type-chip--active {
  border-color: rgba(33, 95, 154, 0.55);
  background: #eff8ff;
}

.type-chip--active svg {
  color: #215f9a;
}

.agent-progress,
.resource-output {
  margin-top: 1rem;
}

.agent-progress {
  border: 1px solid rgba(33, 95, 154, 0.18);
  border-radius: 8px;
  padding: 0.9rem;
  background: #f8fbfd;
}

.agent-progress__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.agent-progress__head span,
.agent-progress__head strong {
  display: block;
}

.agent-progress__head span {
  color: #64748b;
  font-size: 0.82rem;
}

.agent-progress__head em {
  color: #176b87;
  font-style: normal;
  font-weight: 900;
}

.progress-bar {
  height: 0.5rem;
  margin: 0.8rem 0;
  overflow: hidden;
  border-radius: 999px;
  background: #e2e8f0;
}

.progress-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #215f9a, #1f8a70);
}

.agent-steps {
  display: grid;
  gap: 0.55rem;
  margin-top: 0.75rem;
}

.agent-step {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.65rem;
  align-items: start;
  border-radius: 8px;
  padding: 0.65rem;
  background: #ffffff;
}

.agent-step p {
  margin: 0.2rem 0 0;
  color: #64748b;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.agent-step__status {
  width: 0.7rem;
  height: 0.7rem;
  margin-top: 0.28rem;
  border-radius: 999px;
  background: #94a3b8;
}

.agent-step__status--completed {
  background: #1f8a70;
}

.agent-step__status--running {
  background: #d97706;
}

.agent-step__status--canceled {
  background: #64748b;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  margin-bottom: 0.75rem;
}

.resource-card {
  overflow: hidden;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  background: #ffffff;
}

.resource-card + .resource-card {
  margin-top: 0.8rem;
}

.resource-card__head {
  display: grid;
  gap: 0.25rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.12);
  padding: 0.8rem;
}

.resource-card__head span,
.library-item span {
  color: #176b87;
  font-size: 0.78rem;
  font-weight: 900;
}

.resource-card pre {
  max-height: 28rem;
  margin: 0;
  overflow: auto;
  padding: 0.9rem;
  color: #1f2937;
  background: #fbfdff;
  font: 0.9rem/1.65 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  white-space: pre-wrap;
  word-break: break-word;
}

.export-box {
  display: grid;
  gap: 0.55rem;
  margin: 0.8rem 0;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  padding: 0.8rem;
  background: #ffffff;
}

.outline-box {
  display: grid;
  gap: 0.75rem;
  margin: 0.8rem 0;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.85rem;
  background: #f8fbfd;
}

.outline-box__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
}

.outline-box__head strong,
.outline-box__head small {
  display: block;
}

.outline-box__head small {
  margin-top: 0.2rem;
  color: #64748b;
}

.outline-box__head span {
  flex: 0 0 auto;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  color: #176b87;
  background: #e8f6fb;
  font-size: 0.78rem;
  font-weight: 900;
}

.outline-slides {
  display: grid;
  gap: 0.65rem;
}

.outline-slide {
  display: grid;
  gap: 0.4rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #ffffff;
}

.outline-slide span {
  color: #176b87;
  font-size: 0.76rem;
  font-weight: 900;
}

.outline-slide ul {
  margin: 0;
  padding-left: 1.1rem;
}

.outline-slide li {
  margin: 0.2rem 0;
  line-height: 1.45;
}

.outline-slide p,
.outline-slide small {
  margin: 0;
  color: #64748b;
  line-height: 1.45;
}

.export-box__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
}

.export-box__head small {
  color: #64748b;
}

.export-box textarea {
  width: 100%;
  min-height: 12rem;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  padding: 0.75rem;
  color: #1f2937;
  background: #fbfdff;
  font: 0.86rem/1.6 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  resize: vertical;
}

.safety-actions {
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid rgba(90, 112, 132, 0.12);
  padding: 0.75rem 0.9rem 0;
}

.image-upload {
  display: grid;
  gap: 0.6rem;
  margin: 0.4rem 0 0.9rem;
}

.image-upload__head {
  display: grid;
  gap: 0.15rem;
}

.image-upload__head span {
  color: #43536a;
  font-size: 0.86rem;
  font-weight: 700;
}

.image-upload__head small {
  color: #64748b;
}

.image-upload__list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.image-thumb {
  position: relative;
  width: 5.5rem;
  height: 5.5rem;
  border: 1px solid rgba(90, 112, 132, 0.2);
  border-radius: 8px;
  overflow: hidden;
}

.image-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-thumb__remove {
  position: absolute;
  top: 0.2rem;
  right: 0.2rem;
  width: 1.3rem;
  height: 1.3rem;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 999px;
  color: #ffffff;
  background: rgba(23, 32, 51, 0.72);
  font-size: 0.9rem;
  line-height: 1;
  cursor: pointer;
}

.image-add {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  width: 5.5rem;
  height: 5.5rem;
  border: 1px dashed rgba(90, 112, 132, 0.4);
  border-radius: 8px;
  color: #43536a;
  background: #f8fbfd;
  font-size: 0.8rem;
  cursor: pointer;
}

.image-add:hover {
  border-color: rgba(33, 95, 154, 0.55);
  color: #215f9a;
}

.regen-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 0.9rem 0;
}

.regen-actions span {
  color: #64748b;
  font-size: 0.82rem;
}

.ghost-mini {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  min-height: 2rem;
  padding: 0 0.7rem;
  border: 1px solid rgba(90, 112, 132, 0.24);
  border-radius: 8px;
  color: #24415f;
  background: #f8fbfd;
  font: inherit;
  font-size: 0.82rem;
  font-weight: 700;
  cursor: pointer;
}

.ghost-mini:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.ghost-mini--active {
  border-color: rgba(33, 95, 154, 0.55);
  color: #ffffff;
  background: #215f9a;
}

.compare-toolbar,
.media-status-box,
.quality-report,
.compare-result,
.review-box,
.checklist-box,
.failure-box,
.learning-status-panel {
  display: grid;
  gap: 0.75rem;
  margin: 0.75rem 0.9rem 0;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.85rem;
  background: #f8fbfd;
}

.compare-toolbar {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  margin: 0 0 0.75rem;
}

.compare-toolbar span,
.media-status-box p,
.media-status-box small,
.compare-result p {
  margin: 0;
  color: #64748b;
  line-height: 1.45;
}

.compare-result {
  margin: 0 0 0.75rem;
}

.review-box,
.checklist-box {
  margin: 0.8rem 0 0;
  background: #ffffff;
}

.failure-box {
  margin: 0.8rem 0 0;
  border-color: rgba(185, 28, 28, 0.18);
  color: #7f1d1d;
  background: #fff7f7;
}

.failure-box p {
  margin: 0;
  color: #8f1d1d;
  line-height: 1.45;
}

.learning-status-panel {
  grid-template-columns: minmax(0, 1fr);
  background: #ffffff;
}

.learning-status-panel strong,
.learning-status-panel small {
  display: block;
}

.learning-status-panel small {
  margin-top: 0.2rem;
  color: #64748b;
}

.status-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.review-box__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.review-box__head strong,
.review-box__head small {
  display: block;
}

.review-box__head small {
  margin-top: 0.2rem;
  color: #64748b;
  line-height: 1.45;
}

.review-box__head span {
  flex: 0 0 auto;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  color: #176b87;
  background: rgba(23, 107, 135, 0.1);
  font-weight: 900;
}

.review-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.6rem;
}

.review-metrics article,
.checklist-box article {
  border-radius: 8px;
  padding: 0.65rem;
  background: #f8fbfd;
}

.review-metrics span,
.checklist-box span {
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.review-metrics strong {
  display: block;
  margin-top: 0.15rem;
  color: #172033;
  font-size: 1.1rem;
}

.checklist-box article {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.6rem;
}

.checklist-box p,
.checklist-box small {
  margin: 0.2rem 0 0;
  color: #64748b;
  line-height: 1.45;
}

.compare-result article {
  border-radius: 8px;
  padding: 0.65rem;
  background: #ffffff;
}

.compare-result span,
.compare-result em {
  display: block;
}

.compare-result em {
  color: #176b87;
  font-style: normal;
  font-weight: 800;
}

.compare-result__winner {
  color: #1f6f5b;
  font-weight: 900;
}

.media-status-box {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
}

.media-status-box span {
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  color: #176b87;
  background: rgba(23, 107, 135, 0.1);
  font-size: 0.78rem;
  font-weight: 800;
}

.media-status-box p,
.media-status-box small {
  grid-column: 1 / -1;
}

.quality-report__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.quality-report__head span,
.quality-report__head strong {
  display: block;
}

.quality-report__head span {
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.quality-report__head em {
  color: #176b87;
  font-style: normal;
  font-weight: 900;
}

.quality-metrics {
  display: grid;
  gap: 0.55rem;
}

.quality-metrics article {
  border-radius: 8px;
  padding: 0.65rem;
  background: #ffffff;
}

.quality-metrics div {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
}

.quality-metrics span {
  color: #176b87;
  font-weight: 800;
}

.quality-metrics p {
  margin: 0.24rem 0 0;
  color: #64748b;
  line-height: 1.45;
}

.safety-report {
  display: grid;
  gap: 0.75rem;
  margin: 0.75rem 0.9rem 0;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.85rem;
  background: #f8fbfd;
}

.safety-report__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.safety-report__head span,
.safety-report__head strong {
  display: block;
}

.safety-report__head span {
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.safety-report__head em {
  color: #176b87;
  font-style: normal;
  font-weight: 900;
}

.safety-checks {
  display: grid;
  gap: 0.5rem;
}

.safety-check {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.55rem;
  align-items: start;
  border-radius: 8px;
  padding: 0.55rem;
  background: #ffffff;
}

.safety-check p,
.safety-list p {
  margin: 0.18rem 0 0;
  color: #64748b;
  line-height: 1.45;
}

.safety-check__dot {
  width: 0.7rem;
  height: 0.7rem;
  margin-top: 0.28rem;
  border-radius: 999px;
  background: #d97706;
}

.safety-check__dot--passed {
  background: #1f8a70;
}

.safety-check__dot--blocked {
  background: #b91c1c;
}

.safety-list {
  display: grid;
  gap: 0.35rem;
}

.feedback-box {
  display: grid;
  gap: 0.75rem;
  border-top: 1px solid rgba(90, 112, 132, 0.12);
  padding: 0.9rem;
  background: #ffffff;
}

.feedback-box__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
}

.feedback-box__head small,
.recommend-summary {
  color: #64748b;
}

.feedback-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.7rem;
}

.feedback-grid__comment {
  grid-column: 1 / -1;
}

.check-row {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  min-height: 2.55rem;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  padding: 0 0.7rem;
  color: #334155;
  background: #f8fbfd;
  font-weight: 700;
}

.check-row input {
  width: 1rem;
  height: 1rem;
  accent-color: #215f9a;
}

.recommend-panel {
  display: grid;
  gap: 0.75rem;
  margin-bottom: 1rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.12);
  padding-bottom: 1rem;
}

.feedback-summary {
  display: grid;
  gap: 0.2rem;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.75rem;
  background: #f4f9fd;
}

.feedback-summary span {
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.feedback-summary strong {
  color: #176b87;
  font-size: 1.45rem;
  line-height: 1;
}

.feedback-summary small {
  color: #64748b;
}

.recommend-summary {
  margin: 0;
  line-height: 1.45;
}

.recommend-list {
  display: grid;
  gap: 0.65rem;
}

.recommend-item {
  display: grid;
  gap: 0.45rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #ffffff;
}

.recommend-item--link {
  color: inherit;
  text-decoration: none;
}

.recommend-item--link:hover {
  border-color: rgba(33, 95, 154, 0.42);
  box-shadow: 0 10px 24px rgba(31, 46, 67, 0.08);
}

.recommend-item span {
  color: #176b87;
  font-size: 0.76rem;
  font-weight: 900;
}

.recommend-item small {
  color: #64748b;
  line-height: 1.45;
}

.recommend-item__types {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.recommend-item__types em {
  border-radius: 999px;
  padding: 0.15rem 0.45rem;
  color: #24415f;
  background: #eef6fb;
  font-size: 0.76rem;
  font-style: normal;
  font-weight: 800;
}

.library-list {
  display: grid;
  gap: 0.65rem;
}

.library-filters {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.library-filters input,
.library-filters select {
  min-width: 0;
  border: 1px solid rgba(90, 112, 132, 0.22);
  border-radius: 8px;
  padding: 0.58rem 0.65rem;
  color: #172033;
  background: #ffffff;
  font: inherit;
  font-size: 0.82rem;
  outline: none;
}

.library-filters input {
  grid-column: 1 / -1;
}

.library-filters button {
  min-height: 2.25rem;
  padding: 0 0.7rem;
}

.library-item {
  display: grid;
  gap: 0.25rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #ffffff;
}

.library-item--link {
  color: inherit;
  text-decoration: none;
}

.library-item--link:hover {
  border-color: rgba(33, 95, 154, 0.42);
}

@media (max-width: 1180px) {
  .resource-shell {
    grid-template-columns: minmax(18rem, 0.9fr) minmax(24rem, 1.4fr);
  }

  .library-panel {
    position: static;
    grid-column: 1 / -1;
  }
}

@media (max-width: 860px) {
  .resource-topbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .resource-shell,
  .generation-grid,
  .feedback-grid,
  .library-filters,
  .resource-types {
    grid-template-columns: 1fr;
  }

  .button-row--end {
    justify-content: flex-start;
  }
}
</style>
