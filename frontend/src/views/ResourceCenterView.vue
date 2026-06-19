<script setup>
import {
  BookOpen,
  Bot,
  BrainCircuit,
  CheckCircle2,
  FileText,
  ImagePlus,
  RefreshCcw,
  Save,
  Send,
  Sparkles,
  Wand2,
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { extractLearningProfile, fetchLearningProfile, saveLearningProfile } from '../api/profile'
import {
  exportResourceJobMarkdown,
  fetchResourceJob,
  fetchResourceRecommendations,
  fetchResourcePresentationOutline,
  fetchResourceSafetyReport,
  fetchResources,
  generateResources,
  submitResourceFeedback,
} from '../api/resources'
import ResourceContent from '../components/ResourceContent.vue'
import { APP_NAME } from '../constants/appInfo'

const DEFAULT_PROMPT = '我是大二人工智能学生，想做智能体项目，机器学习基础不太稳，希望多一些导图、代码案例和练习题。'

const route = useRoute()

const profilePrompt = ref(DEFAULT_PROMPT)
const profilePayload = ref(null)
const profileLoading = ref(false)
const profileSaving = ref(false)
const profileError = ref('')
const profileSavedNotice = ref('')

const resourceForm = ref({
  knowledgeId: '',
  knowledgeName: '智能体与工具调用',
  courseTitle: '人工智能导论',
  target: '',
  resourceTypes: ['explanation_doc', 'mind_map', 'practice_questions', 'coding_case', 'reading_material', 'video_script'],
})
const resourceLoading = ref(false)
const resourceError = ref('')
const attachedImages = ref([])
const regeneratingId = ref(null)
const activeJob = ref(null)
const generatedResources = ref([])
const resourceLibrary = ref([])
const libraryLoading = ref(false)
const recommendations = ref([])
const recommendationSummary = ref('')
const feedbackSummary = ref(null)
const recommendationsLoading = ref(false)
const feedbackDrafts = ref({})
const feedbackSaving = ref({})
const feedbackNotice = ref('')
const feedbackError = ref('')
const routeContextNotice = ref('')
const safetyReports = ref({})
const safetyLoading = ref({})
const exportLoading = ref(false)
const exportPayload = ref(null)
const exportError = ref('')
const outlineLoading = ref(false)
const presentationOutline = ref(null)
const outlineError = ref('')

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

const resourceTypeLabels = Object.fromEntries(resourceTypeOptions.map((item) => [item.value, item.label]))

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

const resourceLibraryFilter = computed(() => resourceForm.value.knowledgeId.trim())

const currentKnowledgeResources = computed(() => {
  const knowledgeID = resourceForm.value.knowledgeId.trim()
  if (!knowledgeID) return resourceLibrary.value
  return resourceLibrary.value.filter((resource) => resource.knowledgeId === knowledgeID)
})

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

async function loadCurrentProfile() {
  profileError.value = ''
  try {
    const payload = await fetchLearningProfile()
    profilePayload.value = payload
    applyProfileTarget(payload?.profile)
  } catch (err) {
    profilePayload.value = { profile: null, hasCompletedProfile: false }
    profileError.value = err?.message || '学习画像读取失败。'
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
  profileSavedNotice.value = ''
  try {
    const payload = await saveLearningProfile({
      summary: activeProfile.value.summary,
      sourceText: profilePrompt.value.trim(),
      dimensions: activeProfile.value.dimensions,
    })
    profilePayload.value = payload
    profileSavedNotice.value = '学习画像已确认，资源生成会优先使用这些维度。'
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

const jobHeadline = computed(() => {
  const job = activeJob.value
  if (!job) return ''
  if (job.status === 'completed') return job.summary || '资源包生成完成，可查看、导出与提交反馈'
  if (job.status === 'failed') return job.errorMessage || '资源生成失败，请稍后重试'
  return `正在执行：${job.currentAgent || '多智能体协同'}`
})

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
    if (job?.status === 'completed' || job?.status === 'failed') return job
  }
  return activeJob.value
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
  exportPayload.value = null
  presentationOutline.value = null
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
      const finalJob = await pollJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '资源生成失败，请稍后重试。'
      }
    }

    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
  } catch (err) {
    resourceError.value = err?.message || '资源生成失败，请稍后重试。'
  } finally {
    resourceLoading.value = false
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
  exportPayload.value = null
  presentationOutline.value = null
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
      const finalJob = await pollJobUntilDone(activeJob.value.id)
      if (finalJob?.status === 'failed') {
        resourceError.value = finalJob.errorMessage || '重新生成失败，请稍后重试。'
      }
    }

    initFeedbackDrafts(generatedResources.value)
    await loadResourceLibrary()
    await loadRecommendations()
  } catch (err) {
    resourceError.value = err?.message || '重新生成失败，请稍后重试。'
  } finally {
    regeneratingId.value = null
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

async function loadResourceLibrary() {
  libraryLoading.value = true
  try {
    const payload = await fetchResources({ limit: 30, knowledgeId: resourceLibraryFilter.value })
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
    feedbackNotice.value = payload?.nextAction || '反馈已记录，推荐策略已更新。'
    await loadRecommendations()
  } catch (err) {
    feedbackError.value = err?.message || '反馈提交失败，请稍后重试。'
  } finally {
    feedbackSaving.value = { ...feedbackSaving.value, [resource.id]: false }
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

onMounted(async () => {
  applyRouteContext()
  await Promise.all([loadCurrentProfile(), loadResourceLibrary(), loadRecommendations()])
})

watch(
  () => route.query,
  async () => {
    applyRouteContext()
    await loadResourceLibrary()
  },
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
          <button type="button" class="primary-action" :disabled="resourceLoading" @click="handleGenerateResources">
            <Sparkles v-if="!resourceLoading" :size="16" aria-hidden="true" />
            <RefreshCcw v-else :size="16" aria-hidden="true" />
            {{ resourceLoading ? '生成资源包中' : '生成资源包' }}
          </button>
        </div>
        <p v-if="resourceError" class="inline-error">{{ resourceError }}</p>
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
            <button type="button" class="secondary-action" :disabled="outlineLoading || activeJob.status !== 'completed'" @click="handleLoadPresentationOutline">
              <Sparkles :size="16" aria-hidden="true" />
              {{ outlineLoading ? '生成中' : '生成 PPT 大纲' }}
            </button>
            <button type="button" class="secondary-action" :disabled="exportLoading || activeJob.status !== 'completed'" @click="handleExportJob">
              <FileText :size="16" aria-hidden="true" />
              {{ exportLoading ? '导出中' : '导出 Markdown' }}
            </button>
          </div>
          <p v-if="outlineError" class="inline-error">{{ outlineError }}</p>
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
          <div v-if="exportPayload" class="export-box">
            <div class="export-box__head">
              <strong>{{ exportPayload.fileName }}</strong>
              <small>已生成可提交/可粘贴的资源包 Markdown</small>
            </div>
            <textarea readonly :value="exportPayload.content" rows="8" />
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
          <article v-for="resource in generatedResources" :key="resource.id" class="resource-card">
            <div class="resource-card__head">
              <span>{{ resource.typeLabel || typeLabel(resource.resourceType) }}</span>
              <strong>{{ resource.title }}</strong>
              <small>{{ resource.reason }}</small>
            </div>
            <ResourceContent :content="resource.content" :format="resource.format" />
            <div class="regen-actions">
              <span>难度不合适？让智能体重写：</span>
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
            </div>
            <div class="safety-actions">
              <button type="button" class="secondary-action" :disabled="safetyLoading[resource.id]" @click="handleLoadSafetyReport(resource)">
                <CheckCircle2 :size="16" aria-hidden="true" />
                {{ safetyReports[resource.id] ? '收起安全报告' : safetyLoading[resource.id] ? '审查中' : '安全审查' }}
              </button>
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
          <article v-for="resource in currentKnowledgeResources" :key="resource.id" class="library-item">
            <span>{{ resource.typeLabel || typeLabel(resource.resourceType) }}</span>
            <strong>{{ resource.title }}</strong>
            <small>{{ resource.knowledgeName }} · {{ formatDate(resource.createdAt) }}</small>
          </article>
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

.library-item {
  display: grid;
  gap: 0.25rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #ffffff;
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
  .resource-types {
    grid-template-columns: 1fr;
  }

  .button-row--end {
    justify-content: flex-start;
  }
}
</style>
