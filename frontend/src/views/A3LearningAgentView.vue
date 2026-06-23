<script setup>
import {
  BookOpen,
  Bot,
  BrainCircuit,
  CheckCircle2,
  Compass,
  FileText,
  ImagePlus,
  PlayCircle,
  RefreshCcw,
  Save,
  Send,
  Sparkles,
  Wand2,
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import {
  extractLearningProfile,
  fetchLearningProfile,
  refreshLearningProfile,
  saveLearningProfile,
} from '../api/profile'
import {
  exportResourceJobMarkdown,
  fetchLearningReport,
  fetchResourceDemoScript,
  fetchResourceEffectSummary,
  fetchResourceJob,
  fetchResourceMediaStatus,
  fetchResourcePresentationOutline,
  fetchResourceQualityReport,
  fetchResourceSafetyReport,
  generateResourceImage,
  generateResourceVideo,
  generateResources,
  recordLearningEvent,
  retryResourceJob,
  streamResourceJob,
  submitResourceFeedback,
} from '../api/resources'
import { fetchLearningProgress } from '../api/learning'
import { adjustStudyPlan, fetchCurrentStudyPlan, generateStudyPlan, saveGoalProfile } from '../api/studyPlan'
import AiAssistant from '../components/ai/AiAssistant.vue'
import QuizPanel from '../components/QuizPanel.vue'
import ResourceContent from '../components/ResourceContent.vue'
import ScoreBreakdown from '../components/ScoreBreakdown.vue'
import { useSpeech } from '../composables/useSpeech'
import { APP_NAME } from '../constants/appInfo'

// 演示主线默认输入，方便评委一键体验完整闭环。
const DEFAULT_PROMPT =
  '我是大二人工智能专业的学生，想做一个智能体项目，机器学习基础不太稳，喜欢通过代码案例和思维导图学习，希望补齐智能体与工具调用相关知识。'

// 画像六维展示顺序，保证卡片稳定排列。
const dimensionOrder = [
  'knowledge_foundation',
  'learning_goal',
  'learning_progress',
  'cognitive_style',
  'error_preference',
  'resource_preference',
]

// 资源类型选项，覆盖赛题要求的多类个性化资源。
const resourceTypeOptions = [
  { value: 'explanation_doc', label: '课程讲解文档', hint: '结构化知识讲解' },
  { value: 'mind_map', label: '知识点思维导图', hint: 'Mermaid 导图' },
  { value: 'practice_questions', label: '练习题目', hint: '含解析与易错提醒' },
  { value: 'coding_case', label: '代码实操案例', hint: '可落地项目片段' },
  { value: 'reading_material', label: '拓展阅读材料', hint: '补充资料与阅读路径' },
]
const resourceTypeLabels = {
  ...Object.fromEntries(resourceTypeOptions.map((item) => [item.value, item.label])),
  generated_image: 'AI 生成配图',
  generated_video: 'AI 生成短视频',
}

// 画像状态
const profilePrompt = ref(DEFAULT_PROMPT)
const profilePayload = ref(null)
const profileLoading = ref(false)
const profileSaving = ref(false)
const profileError = ref('')
const profileNotice = ref('')

// 学习计划与推荐状态（同一份学习计划同时提供阶段任务与下一步推荐）
const studyPlan = ref(null)
const planLoading = ref(false)
const planError = ref('')
const planNotice = ref('')

// 资源生成与反馈状态
const resourceForm = ref({
  knowledgeId: '',
  knowledgeName: '智能体与工具调用',
  courseTitle: '人工智能导论',
  target: '',
  resourceTypes: ['explanation_doc', 'mind_map', 'practice_questions', 'coding_case', 'reading_material'],
})
const resourceLoading = ref(false)
const resourceError = ref('')
const mediaPrompt = ref('')
const mediaLoading = ref('')
const mediaError = ref('')
const mediaNotice = ref('')
const activeJob = ref(null)
const generatedResources = ref([])
const attachedImages = ref([])
const feedbackDrafts = ref({})
const feedbackSaving = ref({})
const feedbackNotice = ref('')
const feedbackError = ref('')

// 学习效果评估看板
const effectSummary = ref(null)
const effectLoading = ref(false)

// 安全审查（防幻觉可信展示）
const safetyReports = ref({})
const safetyLoading = ref({})
const qualityReports = ref({})
const qualityLoading = ref({})
const mediaStatuses = ref({})
const mediaStatusLoading = ref({})
const retryingJob = ref(false)

// 答辩物料导出
const outlineLoading = ref(false)
const presentationOutline = ref(null)
const outlineError = ref('')
const demoScriptLoading = ref(false)
const demoScript = ref(null)
const demoScriptError = ref('')
const exportLoading = ref(false)
const exportPayload = ref(null)
const exportError = ref('')

// 语音朗读（多模态输出，可对接科大讯飞 TTS）
const { supported: speechSupported, speakingId, speak } = useSpeech()

// AI 学习助教（智能辅导）
const assistantOpen = ref(false)

// 多轮追问式画像补全
const profileFollowUp = ref('')
const profileTurns = ref([])

// 学情学习报告
const reportLoading = ref(false)
const learningReport = ref(null)
const reportError = ref('')

// 知识点掌握度（学习效果可见）
const masteryMap = ref({})

// 资源收藏与学习笔记
const noteDrafts = ref({})
const noteSaving = ref({})
const noteNotice = ref('')

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
const profileSummary = computed(() => activeProfile.value?.summary || '尚未生成画像')

// assistantDetail 把当前知识点与画像打包给 AI 学习助教，使辅导聚焦本次学习上下文。
const assistantDetail = computed(() => {
  const name = resourceForm.value.knowledgeName?.trim()
  if (!name) return null
  return {
    routeKnowledgeId: resourceForm.value.knowledgeId || name,
    canonicalName: name,
    name,
    courseTitle: resourceForm.value.courseTitle,
    summary: dimensionValue('learning_goal') || profileSummary.value,
    description: profileSummary.value,
    prerequisites: [],
    masteryScore: currentMastery.value,
  }
})

// currentMastery 当前知识点的掌握度（0-100），答题后会更新。
const currentMastery = computed(() => {
  const id = resourceForm.value.knowledgeId?.trim()
  if (id && masteryMap.value[id] !== undefined) return Math.round(masteryMap.value[id])
  return null
})

const planStages = computed(() => studyPlan.value?.stages || [])
const nextSteps = computed(() => studyPlan.value?.nextSteps || [])
const hasPlan = computed(() => planStages.value.length > 0)
const hasResources = computed(() => generatedResources.value.length > 0)

// 当前所处步骤，用于顶部进度提示。
const currentStep = computed(() => {
  if (!hasProfile.value) return 1
  if (!hasPlan.value) return 3
  if (!hasResources.value) return 4
  return 5
})

function dimensionValue(key) {
  return orderedDimensions.value.find((item) => item.key === key)?.value || ''
}

function formatPercent(confidence) {
  const value = Number(confidence || 0)
  return `${Math.round(value * 100)}%`
}

function formatDate(value) {
  if (!value) return '刚刚'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function typeLabel(type) {
  return resourceTypeLabels[type] || type
}

// 学习行为事件的中文标签，让效果评估看板更易读。
const eventTypeLabels = {
  resource_generation_started: '发起资源生成',
  resource_generated: '生成资源',
  resource_viewed: '查看资源',
  resource_feedback_submitted: '提交反馈',
  resource_completed: '完成学习',
  image_generated: '生成配图',
  video_generated: '生成视频',
  practice_answered: '随堂练习作答',
  note_saved: '保存笔记',
  profile_refreshed: '画像刷新',
}

function eventTypeLabel(type) {
  return eventTypeLabels[type] || type
}

// applyProfileTarget 把学习目标维度回填到资源生成目标，减少重复输入。
function applyProfileTarget(profile) {
  const goal = profile?.dimensions?.find((item) => item.key === 'learning_goal')?.value || ''
  if (!resourceForm.value.target && (goal || profile?.summary)) {
    resourceForm.value.target = goal || profile.summary
  }
}

async function loadCurrentProfile() {
  try {
    const payload = await fetchLearningProfile()
    profilePayload.value = payload
    applyProfileTarget(payload?.profile)
  } catch (err) {
    profilePayload.value = { profile: null, hasCompletedProfile: false }
    profileError.value = err?.message || '学习画像读取失败。'
  }
}

async function loadCurrentPlan() {
  try {
    const payload = await fetchCurrentStudyPlan()
    if (payload?.plan) studyPlan.value = payload.plan
  } catch {
    // 首次进入工作台时通常还没有学习计划，忽略读取失败即可。
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
  profileNotice.value = ''
  try {
    const payload = await extractLearningProfile(message)
    profilePayload.value = payload
    resourceForm.value.target = ''
    applyProfileTarget(payload?.profile)
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
  profileNotice.value = ''
  try {
    const payload = await saveLearningProfile({
      summary: activeProfile.value.summary,
      sourceText: profilePrompt.value.trim(),
      dimensions: activeProfile.value.dimensions,
    })
    profilePayload.value = payload
    profileNotice.value = '学习画像已确认，后续计划与资源会优先引用这些维度。'
  } catch (err) {
    profileError.value = err?.message || '画像保存失败，请稍后重试。'
  } finally {
    profileSaving.value = false
  }
}

async function handleRefreshProfile() {
  profileLoading.value = true
  profileError.value = ''
  profileNotice.value = ''
  try {
    const payload = await refreshLearningProfile()
    profilePayload.value = payload
    profileNotice.value = '已根据最近学习行为刷新画像。'
  } catch (err) {
    profileError.value = err?.message || '画像刷新失败，请稍后重试。'
  } finally {
    profileLoading.value = false
  }
}

// deriveGoalProfile 从自然语言画像推断出学习计划所需的目标画像，
// 使学生无需再填写传统表单即可一键生成三阶段学习计划。
function deriveGoalProfile() {
  const foundationText = dimensionValue('knowledge_foundation')
  const styleText = `${dimensionValue('cognitive_style')} ${dimensionValue('resource_preference')}`
  const goalText = dimensionValue('learning_goal') || profileSummary.value || profilePrompt.value.trim()

  let foundationLevel = 'intermediate'
  if (/薄弱|不稳|不牢|入门|零基础|较差|beginner/i.test(foundationText)) foundationLevel = 'beginner'
  else if (/扎实|熟练|较好|深入|advanced|精通/i.test(foundationText)) foundationLevel = 'advanced'

  let learningPreference = 'balanced'
  if (/项目|实战|代码|动手|工程/i.test(styleText)) learningPreference = 'project_first'
  else if (/考试|应试|刷题/i.test(styleText)) learningPreference = 'exam_first'
  else if (/理论|原理|推导/i.test(styleText)) learningPreference = 'theory_first'

  return {
    gradeLevel: 'sophomore',
    major: '人工智能',
    goalType: 'project_building',
    goalText: goalText || '系统补齐人工智能核心知识',
    targetDirection: dimensionValue('learning_goal') || resourceForm.value.knowledgeName,
    learnedTopics: [],
    foundationLevel,
    weeklyHours: 6,
    learningPreference,
  }
}

// handleGeneratePlan 生成三阶段学习计划。若尚未保存目标画像，
// 会先用自然语言画像推断并保存一份，再触发计划生成，保证一键可用。
async function handleGeneratePlan() {
  planLoading.value = true
  planError.value = ''
  planNotice.value = ''
  try {
    let payload
    try {
      payload = await generateStudyPlan()
    } catch {
      // 多数情况是尚未设置目标画像，自动从自然语言画像推断并补齐后重试一次。
      await saveGoalProfile(deriveGoalProfile())
      payload = await generateStudyPlan()
    }
    studyPlan.value = payload?.plan || null
    planNotice.value = '已生成基础、核心、应用三阶段学习计划与下一步推荐。'
  } catch (err) {
    planError.value = err?.message || '学习计划生成失败，请先确认画像后再试。'
  } finally {
    planLoading.value = false
  }
}

// handleAdjustPlan 根据最新掌握度与练习表现重新规划学习计划（自适应闭环）。
async function handleAdjustPlan() {
  planLoading.value = true
  planError.value = ''
  planNotice.value = ''
  try {
    const payload = await adjustStudyPlan()
    studyPlan.value = payload?.plan || studyPlan.value
    planNotice.value = '已根据最新掌握度与练习表现重新规划学习计划。'
  } catch (err) {
    planError.value = err?.message || '重新规划失败，请稍后重试。'
  } finally {
    planLoading.value = false
  }
}

// selectNextStep 把推荐知识点带入资源生成上下文。
function selectNextStep(item) {
  resourceForm.value.knowledgeId = item.knowledgeId || ''
  resourceForm.value.knowledgeName = item.title || resourceForm.value.knowledgeName
  if (item.sourceCourse) resourceForm.value.courseTitle = item.sourceCourse
  if (!resourceForm.value.target) resourceForm.value.target = item.detail || item.title || ''
  resourceError.value = ''
  feedbackNotice.value = `已选择「${item.title}」作为资源生成知识点。`
}

function toggleResourceType(type) {
  const current = new Set(resourceForm.value.resourceTypes)
  if (current.has(type)) current.delete(type)
  else current.add(type)
  resourceForm.value.resourceTypes = [...current]
}

function isResourceTypeSelected(type) {
  return resourceForm.value.resourceTypes.includes(type)
}

const jobHeadline = computed(() => {
  const job = activeJob.value
  if (!job) return ''
  if (job.status === 'completed') return job.summary || '资源包生成完成，可查看并提交反馈'
  if (job.status === 'failed') return job.errorMessage || '资源生成失败，请稍后重试'
  return `正在执行：${job.currentAgent || '多智能体协同'}`
})

const stepProgress = computed(() => {
  if (!activeJob.value?.steps?.length) return 0
  const completed = activeJob.value.steps.filter((step) => step.status === 'completed').length
  return Math.round((completed / activeJob.value.steps.length) * 100)
})

// 轮询令牌，保证每次生成只跟踪最新任务。
const pollToken = ref(0)
let pollTimer = null

function stopPolling() {
  pollToken.value += 1
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function waitForNextPoll(ms, token) {
  return new Promise((resolve) => {
    pollTimer = setTimeout(() => {
      pollTimer = null
      resolve(token === pollToken.value)
    }, ms)
  })
}

// pollJobUntilDone 轮询异步生成任务，让进度条与智能体步骤实时刷新。
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
    if (job?.status === 'completed' || job?.status === 'failed') return job
  }
  return activeJob.value
}

// SSE 订阅控制器，便于在重新生成或离开页面时中断旧的流。
let streamController = null

function stopStreaming() {
  if (streamController) {
    streamController.abort()
    streamController = null
  }
}

// streamJobUntilDone 优先用 SSE 订阅任务进度；若 SSE 不可用则回退到轮询。
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
    // SSE 中断或不被支持时回退到轮询，保证进度仍能刷新。
    return pollJobUntilDone(jobId)
  } finally {
    streamController = null
  }
}

function initFeedbackDrafts(resources = []) {
  const nextDrafts = { ...feedbackDrafts.value }
  resources.forEach((resource) => {
    if (!resource?.id || nextDrafts[resource.id]) return
    nextDrafts[resource.id] = { rating: 4, difficulty: 'fit', helpful: true, completed: false, comment: '' }
  })
  feedbackDrafts.value = nextDrafts
}

// onSelectImages 读取学生上传的图片为 data URL，作为多模态生成依据。
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

async function loadEffectSummary() {
  effectLoading.value = true
  try {
    effectSummary.value = await fetchResourceEffectSummary()
  } catch {
    effectSummary.value = null
  } finally {
    effectLoading.value = false
  }
}

// loadMastery 读取学习进度，构建知识点掌握度映射，用于展示答题后的掌握度变化。
async function loadMastery() {
  try {
    const payload = await fetchLearningProgress()
    const map = {}
    for (const item of payload?.items || []) {
      if (item.knowledgeId) map[item.knowledgeId] = item.masteryScore
    }
    masteryMap.value = map
  } catch {
    // 未登录或暂无进度时忽略。
  }
}

// handleFollowUpProfile 多轮追问式补全画像：把补充说明并入已有描述后重新抽取，
// 体现“对话式、摒弃繁琐表单”的画像构建方式。
async function handleFollowUpProfile() {
  const extra = profileFollowUp.value.trim()
  if (!extra) return
  profileTurns.value = [...profileTurns.value, extra]
  const combined = [profilePrompt.value.trim(), ...profileTurns.value.map((t) => `补充：${t}`)]
    .filter(Boolean)
    .join('\n')
  profileLoading.value = true
  profileError.value = ''
  profileNotice.value = ''
  try {
    const payload = await extractLearningProfile(combined)
    profilePayload.value = payload
    applyProfileTarget(payload?.profile)
    profileFollowUp.value = ''
    profileNotice.value = '已结合补充说明更新画像。'
  } catch (err) {
    profileError.value = err?.message || '补充更新失败，请稍后重试。'
  } finally {
    profileLoading.value = false
  }
}

// handleLoadReport 生成可分享/可提交的 Markdown 学情学习报告。
async function handleLoadReport() {
  reportLoading.value = true
  reportError.value = ''
  try {
    learningReport.value = await fetchLearningReport()
  } catch (err) {
    reportError.value = err?.message || '学习报告生成失败，请稍后重试。'
  } finally {
    reportLoading.value = false
  }
}

// handleSaveNote 保存资源学习笔记/收藏，写入 note_saved 行为事件并刷新效果。
async function handleSaveNote(resource) {
  const note = (noteDrafts.value[resource.id] || '').trim()
  if (!note) return
  noteSaving.value = { ...noteSaving.value, [resource.id]: true }
  noteNotice.value = ''
  try {
    await recordLearningEvent({
      eventType: 'note_saved',
      knowledgeId: resource.knowledgeId || resourceForm.value.knowledgeId,
      knowledgeName: resource.knowledgeName || resourceForm.value.knowledgeName,
      resourceId: resource.id,
      resourceType: resource.resourceType,
      payload: { note },
    })
    noteNotice.value = '笔记已保存，将纳入学习行为与效果评估。'
    await loadEffectSummary()
  } catch (err) {
    noteNotice.value = err?.message || '笔记保存失败，请稍后重试。'
  } finally {
    noteSaving.value = { ...noteSaving.value, [resource.id]: false }
  }
}

async function handleGenerateResources() {
  if (!resourceForm.value.resourceTypes.length) {
    resourceError.value = '至少选择一种资源类型。'
    return
  }
  const knowledgeName = resourceForm.value.knowledgeName.trim()
  if (!knowledgeName) {
    resourceError.value = '请填写或从推荐中选择知识点。'
    return
  }

  stopPolling()
  stopStreaming()
  resourceLoading.value = true
  resourceError.value = ''
  generatedResources.value = []
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
    await loadEffectSummary()
  } catch (err) {
    resourceError.value = err?.message || '资源生成失败，请稍后重试。'
  } finally {
    resourceLoading.value = false
  }
}

async function handleRetryJob() {
  if (!activeJob.value?.id || retryingJob.value || resourceLoading.value) return

  stopPolling()
  stopStreaming()
  retryingJob.value = true
  resourceError.value = ''
  generatedResources.value = []
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
    await loadEffectSummary()
  } catch (err) {
    resourceError.value = err?.message || '重试任务失败，请稍后再试。'
  } finally {
    retryingJob.value = false
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
    mediaError.value = '请先填写或选择知识点。'
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
    await loadEffectSummary()
  } catch (err) {
    mediaError.value = err?.message || (kind === 'video' ? '视频生成失败，请检查媒体模型配置。' : '图片生成失败，请检查媒体模型配置。')
  } finally {
    mediaLoading.value = ''
  }
}

async function handleSubmitFeedback(resource) {
  const draft = feedbackDrafts.value[resource.id]
  if (!draft) return
  feedbackSaving.value = { ...feedbackSaving.value, [resource.id]: true }
  feedbackNotice.value = ''
  feedbackError.value = ''
  try {
    const payload = await submitResourceFeedback(resource.id, draft)
    feedbackNotice.value = payload?.nextAction || '反馈已记录，画像与推荐策略已更新。'
    // 反馈后刷新画像与学习效果评估，体现“随学随新”闭环。
    await Promise.all([handleRefreshProfile(), loadEffectSummary()])
  } catch (err) {
    feedbackError.value = err?.message || '反馈提交失败，请稍后重试。'
  } finally {
    feedbackSaving.value = { ...feedbackSaving.value, [resource.id]: false }
  }
}

// onQuizCompleted 答题后刷新画像、学习效果评估与掌握度，体现测评闭环。
async function onQuizCompleted() {
  await Promise.all([handleRefreshProfile(), loadEffectSummary(), loadMastery()])
}

// handleLoadSafetyReport 拉取/收起单个资源的安全与防幻觉审查报告。
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
    qualityReports.value = { ...qualityReports.value, [resource.id]: await fetchResourceQualityReport(resource.id) }
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
    mediaStatuses.value = { ...mediaStatuses.value, [resource.id]: await fetchResourceMediaStatus(resource.id) }
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

function isMediaResource(resource) {
  return resource?.format === 'image' || resource?.format === 'video' || resource?.resourceType === 'generated_image' || resource?.resourceType === 'generated_video'
}

// speakResource 朗读资源正文；再次点击同一资源会停止朗读。
function speakResource(resource) {
  speak(resource.content, `resource-${resource.id}`)
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

onMounted(async () => {
  await Promise.all([loadCurrentProfile(), loadCurrentPlan(), loadEffectSummary(), loadMastery()])
})

onBeforeUnmount(() => {
  stopPolling()
  stopStreaming()
})
</script>

<template>
  <section class="a3-workbench">
    <header class="a3-topbar">
      <RouterLink class="a3-brand" to="/">
        <span class="a3-brand__icon">
          <BrainCircuit :size="22" aria-hidden="true" />
        </span>
        <span>
          <strong>{{ APP_NAME }}</strong>
          <small>A3 学习智能体工作台</small>
        </span>
      </RouterLink>

      <nav class="a3-steps" aria-label="演示主线步骤">
        <span
          v-for="step in [
            { id: 1, label: '画像' },
            { id: 3, label: '计划/推荐' },
            { id: 4, label: '资源' },
            { id: 5, label: '反馈' },
          ]"
          :key="step.id"
          class="a3-step"
          :class="{ 'a3-step--active': currentStep >= step.id }"
        >
          {{ step.label }}
        </span>
      </nav>

      <div class="a3-topbar__actions">
        <button type="button" class="ghost-link" @click="assistantOpen = true">
          <Bot :size="16" aria-hidden="true" />
          AI 学习助教
        </button>
        <RouterLink class="ghost-link" to="/">
          <BookOpen :size="16" aria-hidden="true" />
          知识图谱
        </RouterLink>
        <RouterLink class="ghost-link" to="/resources">
          <Sparkles :size="16" aria-hidden="true" />
          资源中心
        </RouterLink>
      </div>
    </header>

    <main class="a3-shell">
      <!-- 步骤 1+2：对话式画像与六维展示 -->
      <section class="a3-card">
        <div class="card-heading">
          <span class="card-badge">1</span>
          <div>
            <span class="card-kicker">Profile</span>
            <h1>对话式学生画像</h1>
          </div>
        </div>

        <label class="field-block">
          <span>用一段话描述你的专业、基础、目标、进度和偏好</span>
          <textarea v-model="profilePrompt" rows="5" placeholder="例如：我是大二人工智能学生，机器学习基础不稳，想做智能体项目……" />
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
        <p v-if="profileNotice" class="inline-success">{{ profileNotice }}</p>

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
          <p>生成画像后，这里会展示知识基础、学习目标、学习进度、认知风格、易错偏好和资源偏好。</p>
        </div>

        <div v-if="hasProfile" class="followup-row">
          <label class="field-block">
            <span>多轮追问补充（对话式补全画像）</span>
            <textarea
              v-model="profileFollowUp"
              rows="2"
              placeholder="例如：我其实更想先补线性代数；平时每周只有 3 小时；偏好看视频讲解"
            />
          </label>
          <button type="button" class="secondary-action" :disabled="profileLoading || !profileFollowUp.trim()" @click="handleFollowUpProfile">
            <Send :size="16" aria-hidden="true" />
            追加并更新画像
          </button>
        </div>
      </section>

      <!-- 步骤 3+4：学习计划与下一步推荐 -->
      <section class="a3-card">
        <div class="card-heading">
          <span class="card-badge">2</span>
          <div>
            <span class="card-kicker">Plan & Recommend</span>
            <h1>学习计划与下一步推荐</h1>
          </div>
        </div>

        <div class="button-row">
          <button type="button" class="primary-action" :disabled="planLoading || !hasProfile" @click="handleGeneratePlan">
            <Compass v-if="!planLoading" :size="16" aria-hidden="true" />
            <RefreshCcw v-else :size="16" aria-hidden="true" />
            {{ planLoading ? '生成中' : '生成学习计划' }}
          </button>
          <button v-if="hasPlan" type="button" class="secondary-action" :disabled="planLoading" @click="handleAdjustPlan">
            <RefreshCcw :size="16" aria-hidden="true" />
            按最新掌握度重新规划
          </button>
          <small v-if="!hasProfile" class="hint-text">先确认画像后再生成计划。</small>
        </div>

        <p v-if="planError" class="inline-error">{{ planError }}</p>
        <p v-if="planNotice" class="inline-success">{{ planNotice }}</p>

        <div v-if="hasPlan" class="plan-stages">
          <article v-for="stage in planStages" :key="stage.key" class="plan-stage">
            <div class="plan-stage__head">
              <strong>{{ stage.title }}</strong>
              <small>{{ stage.summary }}</small>
            </div>
            <ul class="plan-stage__items">
              <li v-for="item in stage.items || []" :key="item.knowledgeId || item.title">
                <span>{{ item.title }}</span>
                <em>{{ item.reasonLabel }}</em>
              </li>
            </ul>
          </article>
        </div>

        <div v-if="nextSteps.length" class="next-steps">
          <div class="section-title">
            <Sparkles :size="18" aria-hidden="true" />
            <strong>下一步推荐知识点</strong>
          </div>
          <article v-for="item in nextSteps" :key="item.knowledgeId || item.title" class="next-step">
            <div class="next-step__main">
              <div>
                <strong>{{ item.title }}</strong>
                <small>{{ item.detail || item.reasonLabel }}</small>
              </div>
              <button type="button" class="secondary-action" @click="selectNextStep(item)">
                <PlayCircle :size="15" aria-hidden="true" />
                用它生成资源
              </button>
            </div>
            <ScoreBreakdown :breakdown="item.scoringBreakdown" />
          </article>
        </div>
        <div v-else-if="!hasPlan" class="empty-state">
          <Compass :size="22" aria-hidden="true" />
          <p>生成学习计划后，这里会按基础 / 核心 / 应用三阶段展示任务，并给出带推荐依据的下一步知识点。</p>
        </div>
      </section>

      <!-- 步骤 5：资源生成与反馈 -->
      <section class="a3-card">
        <div class="card-heading">
          <span class="card-badge">3</span>
          <div>
            <span class="card-kicker">Resource & Feedback</span>
            <h1>个性化资源生成与学习反馈</h1>
          </div>
        </div>

        <div class="generation-grid">
          <label class="field-block">
            <span>知识点名称</span>
            <input v-model="resourceForm.knowledgeName" type="text" placeholder="可从上方推荐选择，或手动输入" />
          </label>
          <label class="field-block">
            <span>课程</span>
            <input v-model="resourceForm.courseTitle" type="text" placeholder="例如：人工智能导论" />
          </label>
          <label class="field-block field-block--wide">
            <span>生成目标</span>
            <textarea v-model="resourceForm.target" rows="2" :placeholder="dimensionValue('learning_goal') || '说明这次希望补齐的学习目标'" />
          </label>
        </div>

        <p v-if="currentMastery !== null" class="mastery-badge">
          当前知识点掌握度：<strong>{{ currentMastery }}%</strong>
          <span>完成随堂练习后会实时更新</span>
        </p>

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
            <textarea v-model="mediaPrompt" rows="2" placeholder="例如：用可视化流程展示智能体如何调用工具完成任务" />
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
              <span>任务 #{{ activeJob.id }} · 实时流式进度</span>
              <strong>{{ jobHeadline }}</strong>
            </div>
            <em>{{ activeJob.progress || stepProgress }}%</em>
          </div>
          <div class="progress-bar">
            <span :style="{ width: `${activeJob.progress || stepProgress}%` }" />
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

          <div class="button-row button-row--end export-actions">
            <button v-if="activeJob.status === 'failed'" type="button" class="secondary-action" :disabled="retryingJob" @click="handleRetryJob">
              <RefreshCcw :size="16" aria-hidden="true" />
              {{ retryingJob ? '重试中' : '重试任务' }}
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
              <small>可提交/可粘贴的资源包 Markdown</small>
            </div>
            <textarea readonly :value="exportPayload.content" rows="8" />
          </div>
        </section>

        <section v-if="hasResources" class="resource-output">
          <div class="section-title">
            <FileText :size="18" aria-hidden="true" />
            <strong>本次生成结果</strong>
          </div>
          <article v-for="resource in generatedResources" :key="resource.id" class="resource-card">
            <div class="resource-card__head">
              <span>{{ resource.typeLabel || typeLabel(resource.resourceType) }}</span>
              <strong>{{ resource.title }}</strong>
              <small>{{ resource.reason }}</small>
            </div>
            <ResourceContent :content="resource.content" :format="resource.format" />

            <div v-if="resource.evidence?.length" class="evidence-row">
              <span class="evidence-row__label">依据来源</span>
              <em v-for="item in resource.evidence" :key="item.label">{{ item.label }}：{{ item.value }}</em>
            </div>

            <div class="resource-card__actions">
              <button v-if="speechSupported" type="button" class="ghost-mini" @click="speakResource(resource)">
                <PlayCircle :size="15" aria-hidden="true" />
                {{ speakingId === `resource-${resource.id}` ? '停止朗读' : '语音朗读' }}
              </button>
              <button
                v-if="isMediaResource(resource)"
                type="button"
                class="ghost-mini"
                :disabled="mediaStatusLoading[resource.id]"
                @click="handleLoadMediaStatus(resource)"
              >
                <RefreshCcw :size="15" aria-hidden="true" />
                {{ mediaStatusLoading[resource.id] ? '刷新中' : '媒体状态' }}
              </button>
              <button type="button" class="ghost-mini" :disabled="qualityLoading[resource.id]" @click="handleLoadQualityReport(resource)">
                <Sparkles :size="15" aria-hidden="true" />
                {{ qualityReports[resource.id] ? '收起质量' : qualityLoading[resource.id] ? '评分中' : '质量评分' }}
              </button>
              <button type="button" class="ghost-mini" :disabled="safetyLoading[resource.id]" @click="handleLoadSafetyReport(resource)">
                <CheckCircle2 :size="15" aria-hidden="true" />
                {{ safetyReports[resource.id] ? '收起安全报告' : safetyLoading[resource.id] ? '审查中' : '安全审查' }}
              </button>
            </div>

            <div v-if="mediaStatuses[resource.id]" class="media-status-box">
              <strong>{{ mediaStatuses[resource.id].ready ? '媒体已就绪' : '媒体处理中' }}</strong>
              <span>{{ mediaStatuses[resource.id].status }}</span>
              <p>{{ mediaStatuses[resource.id].hint }}</p>
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

            <div class="note-box">
              <label class="field-block">
                <span>学习笔记 / 收藏</span>
                <textarea
                  v-model="noteDrafts[resource.id]"
                  rows="2"
                  placeholder="记录这份资源的重点或你的理解，保存后会纳入学习行为与效果评估"
                />
              </label>
              <div class="button-row button-row--end">
                <button type="button" class="ghost-mini" :disabled="noteSaving[resource.id] || !(noteDrafts[resource.id] || '').trim()" @click="handleSaveNote(resource)">
                  <Save :size="15" aria-hidden="true" />
                  {{ noteSaving[resource.id] ? '保存中' : '保存笔记' }}
                </button>
              </div>
            </div>

            <div v-if="feedbackDrafts[resource.id]" class="feedback-box">
              <div class="feedback-box__head">
                <strong>学习反馈</strong>
                <small>用于更新学习效果评估与下一步推荐</small>
              </div>
              <div class="feedback-grid">
                <label class="field-block">
                  <span>评分</span>
                  <select v-model.number="feedbackDrafts[resource.id].rating">
                    <option v-for="score in [5, 4, 3, 2, 1]" :key="score" :value="score">{{ score }} 分</option>
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
                  <textarea v-model="feedbackDrafts[resource.id].comment" rows="2" placeholder="记录仍然卡住的地方或希望下轮资源调整的方向" />
                </label>
              </div>
              <div class="button-row button-row--end">
                <button type="button" class="secondary-action" :disabled="feedbackSaving[resource.id]" @click="handleSubmitFeedback(resource)">
                  <Send :size="16" aria-hidden="true" />
                  {{ feedbackSaving[resource.id] ? '提交中' : '提交反馈' }}
                </button>
              </div>
            </div>
          </article>
        </section>
        <div v-else class="empty-state">
          <FileText :size="22" aria-hidden="true" />
          <p>选择知识点并生成资源包后，这里会展示多智能体协作步骤、资源正文与学习反馈入口。</p>
        </div>

        <QuizPanel
          class="a3-quiz"
          :knowledge-id="resourceForm.knowledgeId"
          :knowledge-name="resourceForm.knowledgeName"
          :course-title="resourceForm.courseTitle"
          @completed="onQuizCompleted"
        />
      </section>

      <!-- 步骤 6：学习效果评估看板 -->
      <section class="a3-card">
        <div class="card-heading card-heading--with-action">
          <div class="card-heading__main">
            <span class="card-badge">4</span>
            <div>
              <span class="card-kicker">Effect</span>
              <h1>学习效果评估</h1>
            </div>
          </div>
          <button type="button" class="secondary-action" :disabled="reportLoading" @click="handleLoadReport">
            <FileText :size="16" aria-hidden="true" />
            {{ reportLoading ? '生成中' : '生成学情报告' }}
          </button>
        </div>

        <p v-if="reportError" class="inline-error">{{ reportError }}</p>
        <div v-if="learningReport" class="export-box">
          <div class="export-box__head">
            <strong>{{ learningReport.fileName }}</strong>
            <small>可复制/提交的 Markdown 学情报告</small>
          </div>
          <textarea readonly :value="learningReport.content" rows="8" />
        </div>

        <div v-if="effectLoading" class="empty-state">
          <RefreshCcw :size="22" aria-hidden="true" />
          <p>正在统计学习效果…</p>
        </div>
        <template v-else-if="effectSummary">
          <div class="effect-metrics">
            <article v-for="metric in effectSummary.metrics" :key="metric.key" class="effect-metric">
              <span>{{ metric.label }}</span>
              <strong>{{ metric.value }}</strong>
              <small>{{ metric.hint }}</small>
            </article>
          </div>

          <div v-if="effectSummary.difficulty" class="effect-difficulty">
            <span>难度分布</span>
            <em>偏易 {{ effectSummary.difficulty.easy || 0 }}</em>
            <em>适中 {{ effectSummary.difficulty.fit || 0 }}</em>
            <em>偏难 {{ effectSummary.difficulty.hard || 0 }}</em>
          </div>

          <div v-if="effectSummary.highlights?.length" class="effect-highlights">
            <p v-for="line in effectSummary.highlights" :key="line">{{ line }}</p>
          </div>

          <div v-if="effectSummary.recentEvents?.length" class="effect-events">
            <div class="section-title">
              <Bot :size="18" aria-hidden="true" />
              <strong>近期学习行为</strong>
            </div>
            <article v-for="event in effectSummary.recentEvents.slice(0, 5)" :key="event.id" class="effect-event">
              <span>{{ formatDate(event.occurredAt) }}</span>
              <strong>{{ eventTypeLabel(event.eventType) }}</strong>
              <small>{{ event.knowledgeName || event.resourceType }}</small>
            </article>
          </div>
        </template>
        <div v-else class="empty-state">
          <Sparkles :size="22" aria-hidden="true" />
          <p>生成资源并提交学习反馈后，这里会展示学习效果评估与随学随新的行为信号。</p>
        </div>
      </section>
    </main>

    <AiAssistant v-model="assistantOpen" :detail="assistantDetail" :is-authenticated="true" />
  </section>
</template>

<style scoped>
.a3-workbench {
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(39, 111, 191, 0.12), transparent 30rem),
    linear-gradient(180deg, #f7fbff 0%, #eef4f7 100%);
  color: #172033;
}

.a3-topbar {
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

.a3-brand,
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

.a3-brand {
  color: #172033;
}

.a3-brand__icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  border-radius: 8px;
  color: #ffffff;
  background: #215f9a;
}

.a3-brand strong,
.a3-brand small {
  display: block;
}

.a3-brand small {
  margin-top: 0.1rem;
  color: #64748b;
}

.a3-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.a3-step {
  border-radius: 999px;
  padding: 0.25rem 0.75rem;
  border: 1px solid rgba(90, 112, 132, 0.2);
  color: #64748b;
  background: #ffffff;
  font-size: 0.82rem;
  font-weight: 700;
}

.a3-step--active {
  border-color: rgba(33, 95, 154, 0.55);
  color: #215f9a;
  background: #eff8ff;
}

.a3-topbar__actions,
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

.a3-shell {
  display: grid;
  gap: 1rem;
  max-width: 64rem;
  margin: 0 auto;
  padding: 1rem;
}

.a3-card {
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 18px 45px rgba(31, 46, 67, 0.08);
  padding: 1.1rem;
}

.card-heading {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.card-badge {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  border-radius: 999px;
  color: #ffffff;
  background: #215f9a;
  font-weight: 900;
}

.card-heading h1 {
  margin: 0.15rem 0 0;
  font-size: clamp(1.2rem, 2vw, 1.55rem);
}

.card-kicker {
  color: #176b87;
  font-size: 0.78rem;
  font-weight: 800;
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
  padding: 0.7rem 0.8rem;
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

.hint-text {
  color: #64748b;
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
.dimension-card small {
  color: #64748b;
}

.profile-summary strong,
.dimension-card p {
  overflow-wrap: anywhere;
}

.dimension-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
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

.plan-stages {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
  gap: 0.8rem;
  margin-top: 0.8rem;
}

.plan-stage {
  display: grid;
  gap: 0.55rem;
  align-content: start;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.85rem;
  background: #ffffff;
}

.plan-stage__head strong {
  display: block;
}

.plan-stage__head small {
  color: #64748b;
}

.plan-stage__items {
  display: grid;
  gap: 0.45rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.plan-stage__items li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  border-radius: 6px;
  padding: 0.45rem 0.55rem;
  background: #f4f9fd;
}

.plan-stage__items span {
  overflow-wrap: anywhere;
}

.plan-stage__items em {
  flex: 0 0 auto;
  color: #176b87;
  font-style: normal;
  font-size: 0.76rem;
  font-weight: 800;
}

.next-steps {
  margin-top: 1rem;
  display: grid;
  gap: 0.7rem;
}

.next-step {
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.8rem;
  background: #f8fbfd;
}

.next-step__main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.next-step__main strong {
  display: block;
  overflow-wrap: anywhere;
}

.next-step__main small {
  color: #64748b;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  margin-bottom: 0.75rem;
}

.generation-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.8rem;
}

.resource-types {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
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

.effect-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(8.5rem, 1fr));
  gap: 0.7rem;
}

.effect-metric {
  display: grid;
  gap: 0.2rem;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.8rem;
  background: #f4f9fd;
}

.effect-metric span {
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.effect-metric strong {
  color: #176b87;
  font-size: 1.5rem;
  line-height: 1;
}

.effect-metric small {
  color: #64748b;
  line-height: 1.4;
}

.effect-difficulty {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.9rem;
}

.effect-difficulty span {
  color: #43536a;
  font-size: 0.86rem;
  font-weight: 700;
}

.effect-difficulty em {
  border-radius: 999px;
  padding: 0.15rem 0.55rem;
  color: #24415f;
  background: #eef6fb;
  font-size: 0.78rem;
  font-style: normal;
  font-weight: 800;
}

.effect-highlights {
  display: grid;
  gap: 0.4rem;
  margin-top: 0.9rem;
}

.effect-highlights p {
  margin: 0;
  border-radius: 8px;
  padding: 0.55rem 0.65rem;
  color: #176b87;
  background: #e8f6fb;
  line-height: 1.45;
}

.effect-events {
  margin-top: 1rem;
}

.effect-event {
  display: grid;
  gap: 0.2rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.6rem 0.7rem;
  background: #ffffff;
}

.effect-event + .effect-event {
  margin-top: 0.5rem;
}

.effect-event span {
  color: #64748b;
  font-size: 0.76rem;
}

.effect-event small {
  color: #64748b;
}

.a3-quiz {
  margin-top: 1rem;
}

.export-actions {
  margin-top: 0.8rem;
}

.outline-box {
  display: grid;
  gap: 0.75rem;
  margin: 0.8rem 0 0;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.85rem;
  background: #ffffff;
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
  background: #f8fbfd;
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

.export-box {
  display: grid;
  gap: 0.55rem;
  margin: 0.8rem 0 0;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  padding: 0.8rem;
  background: #ffffff;
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
  min-height: 11rem;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  padding: 0.75rem;
  color: #1f2937;
  background: #fbfdff;
  font: 0.84rem/1.6 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  resize: vertical;
}

.evidence-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
  padding: 0.7rem 0.9rem 0;
}

.evidence-row__label {
  color: #43536a;
  font-size: 0.8rem;
  font-weight: 800;
}

.evidence-row em {
  border-radius: 999px;
  padding: 0.15rem 0.5rem;
  color: #24415f;
  background: #eef6fb;
  font-size: 0.76rem;
  font-style: normal;
  font-weight: 700;
}

.resource-card__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 0.75rem 0.9rem 0;
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

.media-status-box,
.quality-report {
  display: grid;
  gap: 0.75rem;
  margin: 0.75rem 0.9rem 0;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.85rem;
  background: #f8fbfd;
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

.media-status-box p {
  grid-column: 1 / -1;
  margin: 0;
  color: #64748b;
  line-height: 1.45;
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

.safety-report__head span {
  display: block;
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 800;
}

.safety-report__head strong {
  display: block;
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

.agent-progress {
  margin-top: 1rem;
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

.agent-progress__head span {
  display: block;
  color: #64748b;
  font-size: 0.82rem;
}

.agent-progress__head strong {
  display: block;
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

.resource-output {
  margin-top: 1rem;
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

.resource-card__head span {
  color: #176b87;
  font-size: 0.78rem;
  font-weight: 900;
}

.resource-card__head small {
  color: #64748b;
}

.resource-card__head strong {
  overflow-wrap: anywhere;
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

.feedback-box__head small {
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

@media (max-width: 720px) {
  .a3-topbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .generation-grid,
  .feedback-grid {
    grid-template-columns: 1fr;
  }

  .next-step__main {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
