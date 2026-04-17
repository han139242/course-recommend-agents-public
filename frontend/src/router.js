import { createRouter, createWebHistory } from 'vue-router'

import { useUserStore } from './stores/userStore'

const GraphHome = () => import('./views/GraphHome.vue')
const KnowledgeDetail = () => import('./views/KnowledgeDetail.vue')
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
    document.title = `${to.meta.title} | 突破课程边界 · 知识点推荐智能体`
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
