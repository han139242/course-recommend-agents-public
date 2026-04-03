<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import {
  assistantPrompts,
  curriculumFilters,
  stageTabs,
  valueCards,
} from '../data/learning'
import { fetchDashboardSummary } from '../auth'

const showStats = ref(false)
const loading = ref(true)
const errorMessage = ref('')
const dashboard = ref({
  quickStats: [],
  growthInsights: [],
  knowledgeCards: [],
  recommendedPath: [],
  profile: null,
})

const quickStats = computed(() => dashboard.value.quickStats || [])
const growthInsights = computed(() => dashboard.value.growthInsights || [])
const knowledgeCards = computed(() => dashboard.value.knowledgeCards || [])
const recommendedPath = computed(() => dashboard.value.recommendedPath || [])

async function loadDashboard() {
  loading.value = true
  errorMessage.value = ''

  try {
    dashboard.value = await fetchDashboardSummary()
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <section class="hero-banner">
    <div class="banner-copy">
      <span class="eyebrow">AI 知识推荐智能体</span>
      <h1>突破课程边界的人工智能专业知识图谱学习空间</h1>
      <p>
        从知识点出发组织学习，不再被课程顺序和学期边界限制。哪里不会，就从哪里开始，学完立刻知道下一步。
      </p>
      <p v-if="dashboard.profile?.summary" class="auth-side-copy">{{ dashboard.profile.summary }}</p>
    </div>

    <div class="banner-stats-compact">
      <button class="stats-trigger" @click="showStats = !showStats">
        <div>
          <span>我的学习数据</span>
          <strong>查看本周进展</strong>
        </div>
        <span class="stats-trigger-icon">{{ showStats ? '收起' : '展开' }}</span>
      </button>

      <div v-if="showStats" class="stats-popover">
        <article v-for="item in quickStats" :key="item.label" class="stat-card">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </article>
      </div>
    </div>
  </section>

  <p v-if="loading" class="auth-side-copy">正在加载学习空间数据...</p>
  <p v-else-if="errorMessage" class="auth-error">{{ errorMessage }}</p>

  <section class="dashboard-grid">
    <article class="assistant-card">
      <div class="card-toolbar">
        <div class="mini-tabs">
          <span class="mini-tab">AI 学伴</span>
          <span class="mini-tab active">智能体</span>
        </div>
        <RouterLink to="/assistant" class="toolbar-link">进入</RouterLink>
      </div>

      <div class="assistant-head">
        <div>
          <h2>Hi，我是你的跨课程学习助手</h2>
          <p>我会根据你的薄弱点、目标方向和兴趣偏好，帮你找到更合适的补弱、进阶和场景化学习路径。</p>
        </div>
        <div class="assistant-avatar">AI</div>
      </div>

      <div class="prompt-list">
        <RouterLink
          v-for="prompt in assistantPrompts"
          :key="prompt"
          :to="{ path: '/assistant', query: { q: prompt } }"
          class="prompt-item prompt-link"
        >
          {{ prompt }}
        </RouterLink>
      </div>

      <div class="mode-chips">
        <span>智慧测评专家</span>
        <span>个性化学习规划师</span>
        <span>基础薄弱导航</span>
      </div>

      <div class="chat-input">
        <span>请输入问题，或点击示例问题快速开始</span>
        <RouterLink to="/assistant" class="voice-btn">◎</RouterLink>
      </div>
    </article>

    <article class="ability-card">
      <div class="card-toolbar">
        <div class="mini-tabs dark">
          <span class="mini-tab active">成长摘要</span>
          <span class="mini-tab">能力趋势</span>
        </div>
        <RouterLink to="/growth" class="toolbar-link">查看</RouterLink>
      </div>

      <div class="ability-header">
        <h3>当前学习概况</h3>
        <p>首页只保留最关键的成长信号，更完整的能力画像和热力图放在成长分析页查看。</p>
      </div>

      <div class="ability-summary-list">
        <div v-for="item in growthInsights" :key="item.label" class="ability-summary-item">
          <div class="analysis-top">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}%</strong>
          </div>
          <div class="progress-track soft compact">
            <div class="progress-fill" :style="{ width: `${item.value}%` }"></div>
          </div>
        </div>
      </div>

      <RouterLink to="/growth" class="ability-cta">
        进入成长分析查看完整雷达图和知识点热力图
      </RouterLink>
    </article>
  </section>

  <section class="value-strip">
    <article v-for="card in valueCards" :key="card.title" class="value-pill-card">
      <span class="section-tag">{{ card.title }}</span>
      <p>{{ card.description }}</p>
    </article>
  </section>

  <section class="curriculum-section">
    <div class="section-head">
      <div>
        <span class="section-tag">知识点学习空间</span>
        <h2>从知识点出发组织跨课程学习</h2>
      </div>
      <div class="search-box">搜索知识点</div>
    </div>
    <div class="filter-row">
      <span class="filter-pill active">图谱目录</span>
      <span v-for="item in curriculumFilters" :key="item" class="filter-pill">{{ item }}</span>
    </div>

    <div class="loop-section">
      <div class="section-head compact">
        <div>
          <span class="section-tag">学练测荐</span>
          <h2>选择一个阶段继续学习</h2>
        </div>
        <p>每个阶段单独成页，帮助你在清晰的学习路径里完成导学、练习、测评和推荐。</p>
      </div>

      <div class="loop-tabs">
        <RouterLink
          v-for="stage in stageTabs"
          :key="stage.id"
          :to="`/flow/${stage.id}`"
          class="tab-btn tab-link"
        >
          <span>{{ stage.label }}</span>
          {{ stage.title }}
        </RouterLink>
      </div>

      <div class="focus-card">
        <div class="focus-row">
          <span class="panel-label">推荐学习顺序</span>
          <span class="status-pill">可跳转</span>
        </div>
        <h3>先补弱点，再做练习，再通过短测验证，最后进入跨课程推荐路径。</h3>
        <div class="focus-meta">
          <span>核心目标</span>
          <strong>让学习围绕知识点掌握推进，而不是被课程章节顺序绑住。</strong>
        </div>
      </div>
    </div>

    <div class="card-grid">
      <RouterLink v-for="card in knowledgeCards" :key="card.id" :to="card.route" class="knowledge-card">
        <div class="knowledge-top">
          <span class="knowledge-type">{{ card.type }}</span>
          <span class="progress-circle">{{ card.progress }}%</span>
        </div>
        <h3>{{ card.title }}</h3>
        <p>{{ card.domain }} · {{ card.tag }}</p>
        <div class="progress-track">
          <div class="progress-fill" :style="{ width: `${card.progress}%` }"></div>
        </div>
      </RouterLink>
    </div>

    <article class="path-card homepage-path-card">
      <div class="section-head compact">
        <div>
          <span class="section-tag">跨课程推荐</span>
          <h2>下一步学习路径</h2>
        </div>
        <p>系统会把补弱推荐、进阶推荐和场景推荐组织成一条可执行路径。</p>
      </div>

      <div class="path-list">
        <RouterLink
          v-for="item in recommendedPath"
          :key="item.stage"
          :to="item.route"
          class="path-item path-link"
        >
          <div class="path-stage">{{ item.stage }}</div>
          <div class="path-content">
            <h3>{{ item.title }}</h3>
            <p>{{ item.detail }}</p>
          </div>
        </RouterLink>
      </div>
    </article>
  </section>
</template>
