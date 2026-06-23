<script setup>
import { ArrowLeft, CheckCircle2, FileText, RefreshCcw, Send, ShieldCheck, Sparkles } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import {
  fetchResource,
  fetchResourceMediaStatus,
  fetchResourceProvenance,
  fetchResourceQualityReport,
  fetchResourceSafetyReport,
  recordLearningEvent,
  submitResourceFeedback,
} from '../api/resources'
import ResourceContent from '../components/ResourceContent.vue'

const route = useRoute()

const resource = ref(null)
const loading = ref(false)
const error = ref('')
const safetyReport = ref(null)
const safetyLoading = ref(false)
const qualityReport = ref(null)
const qualityLoading = ref(false)
const mediaStatus = ref(null)
const mediaStatusLoading = ref(false)
const provenance = ref(null)
const provenanceLoading = ref(false)
const feedbackSaving = ref(false)
const feedbackNotice = ref('')
const feedbackError = ref('')
const feedbackDraft = ref({
  rating: 4,
  difficulty: 'fit',
  helpful: true,
  completed: false,
  comment: '',
})

const resourceId = computed(() => Number(route.params.resourceId || 0))
const isVideoScript = computed(() => resource.value?.resourceType === 'video_script')
const isMediaResource = computed(() => ['image', 'video'].includes(resource.value?.format) || ['generated_image', 'generated_video'].includes(resource.value?.resourceType))
const evidenceList = computed(() => resource.value?.evidence || [])

const storyboardCards = computed(() => {
  if (!isVideoScript.value) return []
  const lines = String(resource.value?.content || '').split(/\r?\n/)
  const cards = []
  let current = null
  lines.forEach((line) => {
    const heading = line.match(/^##\s+(.+)$/)
    if (heading) {
      current = { title: heading[1].trim(), body: [] }
      cards.push(current)
      return
    }
    if (current && line.trim()) current.body.push(line.replace(/^[-*]\s*/, '').trim())
  })
  return cards.slice(0, 6)
})

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

async function loadResource() {
  if (!resourceId.value) return
  loading.value = true
  error.value = ''
  try {
    resource.value = await fetchResource(resourceId.value)
    recordLearningEvent({
      eventType: 'resource_viewed',
      resourceId: resourceId.value,
      payload: {
        title: resource.value?.title,
        typeLabel: resource.value?.typeLabel,
      },
    }).catch(() => null)
  } catch (err) {
    error.value = err?.message || '资源读取失败。'
  } finally {
    loading.value = false
  }
}

async function loadSafetyReport() {
  if (!resourceId.value) return
  safetyLoading.value = true
  try {
    safetyReport.value = await fetchResourceSafetyReport(resourceId.value)
  } catch (err) {
    safetyReport.value = {
      status: 'attention_required',
      score: 0,
      checks: [],
      risks: [err?.message || '安全审查读取失败。'],
      suggestions: ['请稍后重试或进行人工复核。'],
      reviewerName: 'SafetyReviewerAgent',
    }
  } finally {
    safetyLoading.value = false
  }
}

async function loadQualityReport() {
  if (!resourceId.value) return
  qualityLoading.value = true
  try {
    qualityReport.value = await fetchResourceQualityReport(resourceId.value)
  } catch (err) {
    qualityReport.value = {
      score: 0,
      level: 'needs_improvement',
      metrics: [],
      strengths: [],
      improvements: [err?.message || '质量报告读取失败。'],
    }
  } finally {
    qualityLoading.value = false
  }
}

async function loadMediaStatus() {
  if (!resourceId.value || !isMediaResource.value) return
  mediaStatusLoading.value = true
  try {
    mediaStatus.value = await fetchResourceMediaStatus(resourceId.value)
  } catch (err) {
    mediaStatus.value = {
      status: 'unknown',
      ready: false,
      hint: err?.message || '媒体状态读取失败。',
    }
  } finally {
    mediaStatusLoading.value = false
  }
}

async function loadProvenance() {
  if (!resourceId.value) return
  provenanceLoading.value = true
  try {
    provenance.value = await fetchResourceProvenance(resourceId.value)
  } catch (err) {
    provenance.value = {
      summary: err?.message || '生成追溯读取失败。',
      steps: [],
      events: [],
    }
  } finally {
    provenanceLoading.value = false
  }
}

async function handleSubmitFeedback() {
  if (!resourceId.value) return
  feedbackSaving.value = true
  feedbackNotice.value = ''
  feedbackError.value = ''
  try {
    const payload = await submitResourceFeedback(resourceId.value, feedbackDraft.value)
    feedbackNotice.value = payload?.nextAction || '反馈已记录，画像和资源推荐会据此更新。'
  } catch (err) {
    feedbackError.value = err?.message || '反馈提交失败。'
  } finally {
    feedbackSaving.value = false
  }
}

onMounted(async () => {
  await loadResource()
  await Promise.all([loadSafetyReport(), loadQualityReport(), loadMediaStatus(), loadProvenance()])
})
</script>

<template>
  <section class="resource-detail">
    <header class="resource-detail__topbar">
      <RouterLink class="ghost-link" to="/resources">
        <ArrowLeft :size="16" aria-hidden="true" />
        资源中心
      </RouterLink>
      <RouterLink class="ghost-link" to="/">
        <Sparkles :size="16" aria-hidden="true" />
        知识图谱
      </RouterLink>
    </header>

    <main class="resource-detail__shell">
      <section class="resource-main">
        <div v-if="loading" class="empty-state">
          <RefreshCcw :size="20" aria-hidden="true" />
          <p>正在读取资源</p>
        </div>
        <div v-else-if="error" class="inline-error">{{ error }}</div>
        <template v-else-if="resource">
          <header class="resource-hero">
            <span>{{ resource.typeLabel || resource.resourceType }}</span>
            <h1>{{ resource.title }}</h1>
            <p>{{ resource.reason }}</p>
          </header>

          <section v-if="storyboardCards.length" class="storyboard-panel">
            <div class="section-heading">
              <Sparkles :size="18" aria-hidden="true" />
              <strong>多模态分镜预览</strong>
            </div>
            <div class="storyboard-grid">
              <article v-for="(card, index) in storyboardCards" :key="card.title" class="story-card">
                <span>镜头 {{ index + 1 }}</span>
                <strong>{{ card.title }}</strong>
                <p>{{ card.body.slice(0, 2).join(' ') }}</p>
              </article>
            </div>
          </section>

          <article class="content-card">
            <div class="section-heading">
              <FileText :size="18" aria-hidden="true" />
              <strong>资源正文</strong>
            </div>
            <ResourceContent :content="resource.content" :format="resource.format" />
          </article>
        </template>
      </section>

      <aside class="resource-side">
        <section class="side-panel">
          <div class="section-heading">
            <ShieldCheck :size="18" aria-hidden="true" />
            <strong>依据来源</strong>
          </div>
          <div v-if="evidenceList.length" class="evidence-list">
            <article v-for="item in evidenceList" :key="`${item.label}-${item.value}`">
              <span>{{ item.label }}</span>
              <p>{{ item.value }}</p>
            </article>
          </div>
        </section>

        <section class="side-panel">
          <div class="section-heading">
            <Sparkles :size="18" aria-hidden="true" />
            <strong>生成追溯</strong>
          </div>
          <div v-if="provenanceLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>读取追溯链</p>
          </div>
          <div v-else-if="provenance" class="provenance-box">
            <p>{{ provenance.summary }}</p>
            <article v-for="step in provenance.steps || []" :key="step.id || step.agentKey">
              <strong>{{ step.agentName }}</strong>
              <small>{{ step.summary }}</small>
            </article>
            <article v-for="event in provenance.events || []" :key="event.id">
              <strong>{{ event.eventType }}</strong>
              <small>{{ event.knowledgeName || event.resourceType || '学习行为信号' }}</small>
            </article>
          </div>
        </section>

        <section class="side-panel">
          <div class="section-heading">
            <CheckCircle2 :size="18" aria-hidden="true" />
            <strong>安全审查</strong>
          </div>
          <div v-if="safetyLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>审查中</p>
          </div>
          <div v-else-if="safetyReport" class="safety-box">
            <div class="safety-box__score">
              <span>{{ safetyReport.reviewerName || 'SafetyReviewerAgent' }}</span>
              <strong>{{ safetyStatusLabel(safetyReport.status) }}</strong>
              <em>{{ safetyReport.score }} 分</em>
            </div>
            <article v-for="check in safetyReport.checks || []" :key="check.key" class="safety-check">
              <strong>{{ check.name }}</strong>
              <p>{{ check.conclusion }}</p>
            </article>
            <p v-for="risk in safetyReport.risks || []" :key="risk" class="inline-error">{{ risk }}</p>
            <p v-for="suggestion in safetyReport.suggestions || []" :key="suggestion" class="safety-suggestion">
              {{ suggestion }}
            </p>
          </div>
        </section>

        <section class="side-panel">
          <div class="section-heading">
            <Sparkles :size="18" aria-hidden="true" />
            <strong>质量评分</strong>
          </div>
          <div v-if="qualityLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>评分中</p>
          </div>
          <div v-else-if="qualityReport" class="quality-box">
            <div class="safety-box__score">
              <span>QualityJudgeAgent</span>
              <strong>{{ qualityLevelLabel(qualityReport.level) }}</strong>
              <em>{{ qualityReport.score }} 分</em>
            </div>
            <article v-for="metric in qualityReport.metrics || []" :key="metric.key" class="quality-metric">
              <strong>{{ metric.label }} · {{ metric.score }} 分</strong>
              <p>{{ metric.reason }}</p>
            </article>
            <p v-for="item in qualityReport.improvements || []" :key="item" class="safety-suggestion">
              {{ item }}
            </p>
          </div>
        </section>

        <section v-if="isMediaResource" class="side-panel">
          <div class="section-heading">
            <RefreshCcw :size="18" aria-hidden="true" />
            <strong>媒体状态</strong>
          </div>
          <div v-if="mediaStatusLoading" class="empty-state empty-state--small">
            <RefreshCcw :size="18" aria-hidden="true" />
            <p>刷新中</p>
          </div>
          <div v-else-if="mediaStatus" class="media-status-box">
            <strong>{{ mediaStatus.ready ? '媒体已就绪' : '媒体处理中' }}</strong>
            <span>{{ mediaStatus.status }}</span>
            <p>{{ mediaStatus.hint }}</p>
            <small v-if="mediaStatus.providerTaskId">任务号：{{ mediaStatus.providerTaskId }}</small>
          </div>
          <button type="button" class="secondary-action" :disabled="mediaStatusLoading" @click="loadMediaStatus">
            <RefreshCcw :size="16" aria-hidden="true" />
            {{ mediaStatusLoading ? '刷新中' : '刷新状态' }}
          </button>
        </section>

        <section class="side-panel">
          <div class="section-heading">
            <Send :size="18" aria-hidden="true" />
            <strong>学习反馈</strong>
          </div>
          <div class="feedback-form">
            <label>
              <span>评分</span>
              <select v-model.number="feedbackDraft.rating">
                <option v-for="score in [5, 4, 3, 2, 1]" :key="score" :value="score">{{ score }} 分</option>
              </select>
            </label>
            <label>
              <span>难度</span>
              <select v-model="feedbackDraft.difficulty">
                <option value="easy">偏易</option>
                <option value="fit">适中</option>
                <option value="hard">偏难</option>
              </select>
            </label>
            <label class="check-row">
              <input v-model="feedbackDraft.helpful" type="checkbox" />
              有帮助
            </label>
            <label class="check-row">
              <input v-model="feedbackDraft.completed" type="checkbox" />
              已完成学习
            </label>
            <label class="feedback-form__wide">
              <span>备注</span>
              <textarea v-model="feedbackDraft.comment" rows="3" placeholder="记录仍然卡住的地方" />
            </label>
            <button type="button" class="primary-action" :disabled="feedbackSaving" @click="handleSubmitFeedback">
              <Send :size="16" aria-hidden="true" />
              {{ feedbackSaving ? '提交中' : '提交反馈' }}
            </button>
          </div>
          <p v-if="feedbackNotice" class="inline-success">{{ feedbackNotice }}</p>
          <p v-if="feedbackError" class="inline-error">{{ feedbackError }}</p>
        </section>
      </aside>
    </main>
  </section>
</template>

<style scoped>
.resource-detail {
  min-height: 100vh;
  background: linear-gradient(180deg, #f7fbff 0%, #eef4f7 100%);
  color: #172033;
}

.resource-detail__topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.16);
  background: rgba(247, 251, 255, 0.9);
  backdrop-filter: blur(14px);
}

.resource-detail__shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(18rem, 0.34fr);
  gap: 1rem;
  padding: 1rem;
}

.resource-main,
.resource-side {
  display: grid;
  gap: 1rem;
  align-self: start;
}

.resource-hero,
.content-card,
.side-panel,
.storyboard-panel {
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 18px 45px rgba(31, 46, 67, 0.08);
}

.resource-hero {
  padding: 1.1rem;
}

.resource-hero span {
  color: #176b87;
  font-size: 0.8rem;
  font-weight: 900;
}

.resource-hero h1 {
  margin: 0.25rem 0;
  font-size: clamp(1.5rem, 3vw, 2.2rem);
  letter-spacing: 0;
}

.resource-hero p,
.story-card p,
.evidence-list p,
.safety-check p,
.safety-suggestion {
  margin: 0;
  color: #64748b;
  line-height: 1.55;
}

.section-heading {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.85rem 1rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.12);
}

.storyboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
  gap: 0.75rem;
  padding: 0.9rem;
}

.story-card,
.evidence-list article,
.safety-check {
  display: grid;
  gap: 0.35rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #f8fbfd;
}

.story-card span,
.evidence-list span {
  color: #176b87;
  font-size: 0.76rem;
  font-weight: 900;
}

.side-panel {
  padding-bottom: 0.9rem;
}

.evidence-list,
.safety-box,
.quality-box,
.media-status-box,
.provenance-box,
.feedback-form {
  display: grid;
  gap: 0.7rem;
  padding: 0.9rem;
}

.provenance-box p {
  margin: 0;
  color: #43536a;
  line-height: 1.5;
}

.provenance-box article {
  display: grid;
  gap: 0.25rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.65rem;
  background: #f8fbfd;
}

.provenance-box small {
  color: #64748b;
  line-height: 1.45;
}

.quality-metric {
  display: grid;
  gap: 0.25rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.7rem;
  background: #f8fbfd;
}

.quality-metric p,
.media-status-box p,
.media-status-box small {
  margin: 0;
  color: #64748b;
  line-height: 1.45;
}

.media-status-box span {
  width: fit-content;
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  color: #176b87;
  background: rgba(23, 107, 135, 0.1);
  font-size: 0.78rem;
  font-weight: 800;
}

.safety-box__score {
  display: grid;
  gap: 0.2rem;
  border-radius: 8px;
  padding: 0.8rem;
  color: #ffffff;
  background: #215f9a;
}

.safety-box__score span,
.safety-box__score em {
  opacity: 0.86;
  font-style: normal;
}

.feedback-form label {
  display: grid;
  gap: 0.4rem;
}

.feedback-form span {
  color: #43536a;
  font-size: 0.86rem;
  font-weight: 700;
}

.feedback-form select,
.feedback-form textarea {
  width: 100%;
  border: 1px solid rgba(90, 112, 132, 0.22);
  border-radius: 8px;
  padding: 0.7rem;
  font: inherit;
}

.check-row {
  display: flex !important;
  align-items: center;
  gap: 0.45rem;
}

.primary-action,
.secondary-action,
.ghost-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  min-height: 2.45rem;
  border: 0;
  border-radius: 8px;
  padding: 0 0.9rem;
  font: inherit;
  font-weight: 800;
  text-decoration: none;
}

.primary-action {
  color: #ffffff;
  background: #215f9a;
}

.secondary-action {
  border: 1px solid rgba(90, 112, 132, 0.22);
  color: #24415f;
  background: #f8fbfd;
}

.ghost-link {
  border: 1px solid rgba(90, 112, 132, 0.18);
  color: #24415f;
  background: #ffffff;
}

.inline-error,
.inline-success,
.empty-state {
  margin: 0.8rem;
  border-radius: 8px;
  padding: 0.75rem;
}

.inline-error {
  color: #8f1d1d;
  background: #fff0f0;
}

.inline-success {
  color: #16613f;
  background: #eaf8f0;
}

.empty-state {
  display: grid;
  place-items: center;
  gap: 0.5rem;
  border: 1px dashed rgba(90, 112, 132, 0.3);
  color: #64748b;
  text-align: center;
}

@media (max-width: 920px) {
  .resource-detail__shell {
    grid-template-columns: 1fr;
  }
}
</style>
