import { createRouter, createWebHistory } from 'vue-router'

import { APP_NAME } from './constants/appInfo'
import { useUserStore } from './stores/userStore'

const GraphHome = () => import('./views/GraphHome.vue')
const KnowledgeDetail = () => import('./views/KnowledgeDetail.vue')
const DashboardView = () => import('./views/DashboardView.vue')
const ResourceCenterView = () => import('./views/ResourceCenterView.vue')
const CompetitionShowcaseView = () => import('./views/CompetitionShowcaseView.vue')
const EvaluationView = () => import('./views/EvaluationView.vue')
const FrontierLibraryView = () => import('./views/FrontierLibraryView.vue')
const LoginView = () => import('./views/LoginView.vue')
const RegisterView = () => import('./views/RegisterView.vue')

const routes = [
  {
    path: '/',
    name: 'graph-home',
    component: GraphHome,
    meta: {
      title: '知识图谱',
      fullscreen: true,
    },
  },
  {
    path: '/knowledge/:knowledgeId',
    name: 'knowledge-detail',
    component: KnowledgeDetail,
    meta: {
      title: '知识详情',
    },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardView,
    meta: {
      title: '学习仪表盘',
      requiresAuth: true,
    },
  },
  {
    path: '/resources',
    name: 'resource-center',
    component: ResourceCenterView,
    meta: {
      title: '个性化资源智能体',
      requiresAuth: true,
    },
  },
  {
    path: '/competency',
    name: 'competency-map',
    component: DashboardView,
    meta: {
      title: '能力图谱',
      requiresAuth: true,
    },
  },
  {
    path: '/showcase',
    name: 'competition-showcase',
    component: CompetitionShowcaseView,
    meta: {
      title: '作品展示中心',
    },
  },
  {
    path: '/evaluation',
    name: 'evaluation',
    component: EvaluationView,
    meta: {
      title: '教研评测',
    },
  },
  {
    path: '/frontier',
    name: 'frontier-library',
    component: FrontierLibraryView,
    meta: {
      title: '前沿资料',
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
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()
  await userStore.bootstrap()

  if (to.meta?.title) {
    document.title = `${to.meta.title} | ${APP_NAME}`
  }

  if (to.meta?.requiresAuth && !userStore.isAuthenticated) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta?.guestOnly && userStore.isAuthenticated) {
    return { name: 'graph-home' }
  }

  return true
})

export default router
