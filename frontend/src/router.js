import { createRouter, createWebHistory } from 'vue-router'
import { authState, bootstrapAuth } from './auth'

import AssistantView from './views/AssistantView.vue'
import GraphView from './views/GraphView.vue'
import GrowthView from './views/GrowthView.vue'
import LearnHome from './views/LearnHome.vue'
import KnowledgeDetail from './views/KnowledgeDetail.vue'
import LoginView from './views/LoginView.vue'
import PlaceholderPage from './views/PlaceholderPage.vue'
import ProfileView from './views/ProfileView.vue'
import RegisterView from './views/RegisterView.vue'
import StageDetail from './views/StageDetail.vue'

const routes = [
  {
    path: '/',
    name: 'learn-home',
    component: LearnHome,
    meta: {
      title: '知识学习',
    },
  },
  {
    path: '/flow/:stageId',
    name: 'stage-detail',
    component: StageDetail,
    meta: {
      title: '学练测荐详情',
    },
  },
  {
    path: '/knowledge/:knowledgeId',
    name: 'knowledge-detail',
    component: KnowledgeDetail,
    meta: {
      title: '知识点详情',
    },
  },
  {
    path: '/profile',
    name: 'profile',
    component: ProfileView,
    meta: {
      title: '个人学习档案',
      requiresAuth: true,
    },
  },
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: {
      title: '登录',
      guestOnly: true,
    },
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterView,
    meta: {
      title: '注册',
      guestOnly: true,
    },
  },
  {
    path: '/assistant',
    name: 'assistant',
    component: AssistantView,
    meta: {
      title: 'AI 助学室',
    },
  },
  {
    path: '/graph',
    name: 'graph',
    component: GraphView,
    meta: {
      title: '知识图谱',
    },
  },
  {
    path: '/question-graph',
    name: 'question-graph',
    component: PlaceholderPage,
    meta: {
      title: '问题图谱',
      description: '问题图谱页面正在开发中，后续会支持错因聚类、薄弱点追踪和问题溯源。',
    },
  },
  {
    path: '/tasks',
    name: 'tasks',
    component: PlaceholderPage,
    meta: {
      title: '任务·作业·考试',
      description: '任务、作业与考试模块正在开发中，后续会支持跨课程任务编排、项目实践和测评管理。',
    },
  },
  {
    path: '/growth',
    name: 'growth',
    component: GrowthView,
    meta: {
      title: '成长分析',
    },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  await bootstrapAuth()

  if (to.meta?.title) {
    document.title = `${to.meta.title} | 课程推荐智能体`
  }

  if (to.meta?.requiresAuth && !authState.token) {
    return {
      name: 'login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (to.meta?.guestOnly && authState.token) {
    return {
      name: 'learn-home',
    }
  }

  return true
})

export default router
