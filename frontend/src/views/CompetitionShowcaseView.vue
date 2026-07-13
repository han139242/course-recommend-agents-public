<script setup>
import {
  Award,
  BrainCircuit,
  CheckCircle2,
  Gauge,
  GitBranch,
  Layers3,
  MonitorCheck,
  Moon,
  Network,
  Presentation,
  Route,
  ShieldCheck,
  Sparkles,
  Sun,
  Target,
} from '@lucide/vue'
import { RouterLink } from 'vue-router'

import { APP_NAME } from '../constants/appInfo'
import { useThemeStore } from '../stores/themeStore'

const themeStore = useThemeStore()

const scoreCards = [
  {
    key: '创意',
    score: '20%',
    icon: Sparkles,
    title: '跨课程知识边界的个性化学习智能体',
    detail: '把课程目录、知识图谱、学习进度、练习反馈和 AI 助教组合成一个可闭环的推荐系统，而不是单点问答工具。',
  },
  {
    key: '技术',
    score: '20%-30%',
    icon: Network,
    title: 'Go + Vue + Neo4j + Redis 的工程化全栈架构',
    detail: '后端按模块化垂直切片组织，图谱检索、路径推荐、学习画像、流式助教和评测能力都有清晰边界。',
  },
  {
    key: '应用',
    score: '20%',
    icon: Route,
    title: '学生能直接完成“看图谱-学路径-做练习-问助教”',
    detail: '首页、知识点详情、学习总览和 AI 助教围绕真实学习任务设计，评委现场可以连续操作验证。',
  },
  {
    key: '设计',
    score: '20%-30%',
    icon: MonitorCheck,
    title: '暗色教学工作台 + 可视化知识地图',
    detail: '界面突出课程导航、薄弱点、下一步学习和进度反馈，兼顾桌面演示和移动端窄屏稳定性。',
  },
  {
    key: '效果',
    score: '20%',
    icon: Gauge,
    title: '可运行、可测试、可复现的比赛演示环境',
    detail: '提供一键启动脚本、健康检查、自动化测试和浏览器截图自测，减少现场环境风险。',
  },
]

const architectureLayers = [
  {
    icon: Layers3,
    title: '前端体验层',
    items: ['学生学习工作台', '课程知识地图', '知识点详情', '学习总览', '评委展示中心'],
  },
  {
    icon: BrainCircuit,
    title: '智能服务层',
    items: ['统一学习路径推荐', 'AI 助教流式问答', '知识卡片生成', '练习反馈解释', '学习画像分析'],
  },
  {
    icon: GitBranch,
    title: '知识图谱层',
    items: ['Neo4j 知识点关系', '先修路径', '邻居扩展', '课程分组', '图谱可视化布局'],
  },
  {
    icon: ShieldCheck,
    title: '工程支撑层',
    items: ['Go 单端口托管', 'MySQL/Redis 状态存储', 'Docker 依赖编排', '前端生产构建', '自动化测试'],
  },
]

const demoSteps = [
  {
    time: '0:00 - 1:00',
    title: '打开作品展示中心',
    detail: '先说明项目如何对齐 A 系列评分标准，并展示一键启动、测试和健康检查能力。',
    route: '/showcase',
  },
  {
    time: '1:00 - 2:20',
    title: '进入课程知识地图',
    detail: '展示跨课程知识点、先修关系、当前掌握度和 AI 助教入口。',
    route: '/',
  },
  {
    time: '2:20 - 3:40',
    title: '打开知识点详情',
    detail: '展示前置知识、典型场景、练习测验、推荐路径、笔记和助教追问。',
    route: '/knowledge/gradient_descent',
  },
  {
    time: '3:40 - 5:00',
    title: '进入学习总览',
    detail: '展示总体掌握度、待巩固内容、本周节奏和下一步学习建议。',
    route: '/dashboard',
  },
]

const evidenceItems = [
  '固定访问入口：http://127.0.0.1:8080/',
  '一键演示脚本：scripts/start-demo.ps1',
  '前端测试：17 个测试文件、66 个用例覆盖核心交互和数据转换',
  '生产构建：frontend 构建产物自动输出到 backend/web/dist',
  '健康检查：/healthz 与 /readyz 可验证后端、MySQL、Redis、Neo4j 状态',
]
</script>

<template>
  <section class="showcase-view page-frame">
    <header class="showcase-hero page-card">
      <div class="showcase-hero__copy">
        <span class="eyebrow">Competition Showcase</span>
        <h1>{{ APP_NAME }}</h1>
        <p>
          面向 C4 网络技术挑战赛 A 系列作品评分标准，把创新性、技术深度、应用闭环、界面设计和运行效果集中展示给评委。
        </p>
        <div class="showcase-hero__actions">
          <button
            type="button"
            class="theme-toggle-btn"
            :title="themeStore.theme === 'light' ? '切换夜间模式' : '切换白天模式'"
            @click="themeStore.toggleTheme"
          >
            <Moon v-if="themeStore.theme === 'light'" :size="15" aria-hidden="true" />
            <Sun v-else :size="15" aria-hidden="true" />
            <span>{{ themeStore.theme === 'light' ? '夜间' : '白天' }}</span>
          </button>
          <RouterLink class="primary-btn" to="/">
            <Presentation :size="16" aria-hidden="true" />
            开始现场演示
          </RouterLink>
          <RouterLink class="secondary-btn" to="/dashboard">查看学习总览</RouterLink>
        </div>
      </div>
      <div class="showcase-hero__panel">
        <Award :size="36" aria-hidden="true" />
        <strong>目标定位</strong>
        <span>国家一等奖答辩版本</span>
        <p>先证明系统完整可运行，再展示推荐逻辑、图谱智能和教学落地价值。</p>
      </div>
    </header>

    <section class="showcase-section">
      <div class="showcase-section__head">
        <span class="eyebrow">Score Alignment</span>
        <h2>A 系列评分点对齐</h2>
      </div>
      <div class="score-alignment-grid">
        <article v-for="card in scoreCards" :key="card.key" class="score-alignment-card">
          <div class="score-alignment-card__top">
            <component :is="card.icon" :size="22" aria-hidden="true" />
            <span>{{ card.key }}</span>
            <strong>{{ card.score }}</strong>
          </div>
          <h3>{{ card.title }}</h3>
          <p>{{ card.detail }}</p>
        </article>
      </div>
    </section>

    <section class="showcase-section">
      <div class="showcase-section__head">
        <span class="eyebrow">Architecture</span>
        <h2>作品技术架构</h2>
      </div>
      <div class="architecture-grid">
        <article v-for="layer in architectureLayers" :key="layer.title" class="architecture-card">
          <component :is="layer.icon" :size="22" aria-hidden="true" />
          <h3>{{ layer.title }}</h3>
          <ul>
            <li v-for="item in layer.items" :key="item">
              <CheckCircle2 :size="14" aria-hidden="true" />
              <span>{{ item }}</span>
            </li>
          </ul>
        </article>
      </div>
    </section>

    <section class="showcase-section showcase-section--split">
      <div>
        <div class="showcase-section__head">
          <span class="eyebrow">Demo Route</span>
          <h2>5 分钟答辩演示路径</h2>
        </div>
        <div class="demo-route-list">
          <RouterLink v-for="step in demoSteps" :key="step.time" class="demo-route-card" :to="step.route">
            <span>{{ step.time }}</span>
            <strong>{{ step.title }}</strong>
            <p>{{ step.detail }}</p>
          </RouterLink>
        </div>
      </div>

      <aside class="showcase-evidence page-card">
        <Target :size="24" aria-hidden="true" />
        <h2>现场验收证据</h2>
        <ul>
          <li v-for="item in evidenceItems" :key="item">{{ item }}</li>
        </ul>
      </aside>
    </section>
  </section>
</template>
