<script setup>
import {
  BookOpen,
  Bot,
  BrainCircuit,
  CheckCircle2,
  FileText,
  RefreshCcw,
  Save,
  Send,
  Sparkles,
  Wand2,
} from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { extractLearningProfile, fetchLearningProfile, saveLearningProfile } from '../api/profile'
import { fetchResources, generateResources } from '../api/resources'
import { APP_NAME } from '../constants/appInfo'

const DEFAULT_PROMPT = '我是大二人工智能学生，想做智能体项目，机器学习基础不太稳，希望多一些导图、代码案例和练习题。'

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
  resourceTypes: ['explanation_doc', 'mind_map', 'practice_questions', 'coding_case', 'reading_material'],
})
const resourceLoading = ref(false)
const resourceError = ref('')
const activeJob = ref(null)
const generatedResources = ref([])
const resourceLibrary = ref([])
const libraryLoading = ref(false)

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

function dimensionValue(key) {
  return orderedDimensions.value.find((item) => item.key === key)?.value || ''
}

function applyProfileTarget(profile) {
  const goal = profile?.dimensions?.find((item) => item.key === 'learning_goal')?.value || ''
  if (!resourceForm.value.target && (goal || profile?.summary)) {
    resourceForm.value.target = goal || profile.summary
  }
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

  resourceLoading.value = true
  resourceError.value = ''
  try {
    const payload = await generateResources({
      knowledgeId: resourceForm.value.knowledgeId.trim(),
      knowledgeName,
      courseTitle: resourceForm.value.courseTitle.trim(),
      target: resourceForm.value.target.trim() || profileSummary.value,
      resourceTypes: resourceForm.value.resourceTypes,
    })
    activeJob.value = payload?.job || null
    generatedResources.value = payload?.resources || []
    await loadResourceLibrary()
  } catch (err) {
    resourceError.value = err?.message || '资源生成失败，请稍后重试。'
  } finally {
    resourceLoading.value = false
  }
}

async function loadResourceLibrary() {
  libraryLoading.value = true
  try {
    const payload = await fetchResources({ limit: 30 })
    resourceLibrary.value = payload?.list || []
  } catch {
    resourceLibrary.value = []
  } finally {
    libraryLoading.value = false
  }
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

onMounted(async () => {
  await Promise.all([loadCurrentProfile(), loadResourceLibrary()])
})
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

        <div class="button-row button-row--end">
          <button type="button" class="primary-action" :disabled="resourceLoading" @click="handleGenerateResources">
            <Sparkles v-if="!resourceLoading" :size="16" aria-hidden="true" />
            <RefreshCcw v-else :size="16" aria-hidden="true" />
            {{ resourceLoading ? '生成资源包中' : '生成资源包' }}
          </button>
        </div>
        <p v-if="resourceError" class="inline-error">{{ resourceError }}</p>

        <section v-if="activeJob" class="agent-progress">
          <div class="agent-progress__head">
            <div>
              <span>任务 #{{ activeJob.id }}</span>
              <strong>{{ activeJob.summary || '资源包生成任务已完成' }}</strong>
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
            <pre>{{ resource.content }}</pre>
          </article>
        </section>
      </section>

      <aside class="library-panel">
        <div class="panel-heading panel-heading--compact">
          <span class="panel-kicker">Library</span>
          <h2>资源库</h2>
        </div>

        <div v-if="libraryLoading" class="empty-state empty-state--small">
          <RefreshCcw :size="18" aria-hidden="true" />
          <p>正在读取资源库</p>
        </div>
        <div v-else-if="!resourceLibrary.length" class="empty-state empty-state--small">
          <FileText :size="18" aria-hidden="true" />
          <p>生成资源后会沉淀在这里。</p>
        </div>
        <div v-else class="library-list">
          <article v-for="resource in resourceLibrary" :key="resource.id" class="library-item">
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
  .resource-types {
    grid-template-columns: 1fr;
  }

  .button-row--end {
    justify-content: flex-start;
  }
}
</style>
