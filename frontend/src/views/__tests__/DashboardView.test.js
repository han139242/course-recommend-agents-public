import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { vi } from 'vitest'

const routeMock = vi.hoisted(() => ({
  current: { name: 'dashboard', path: '/dashboard' },
}))

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
  useRoute: () => routeMock.current,
}))

import DashboardView from '../DashboardView.vue'
import { useUserStore } from '../../stores/userStore'

const mockCanvasContext = {
  setTransform: vi.fn(),
  clearRect: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  closePath: vi.fn(),
  fill: vi.fn(),
  stroke: vi.fn(),
  arc: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  fillText: vi.fn(),
  measureText: vi.fn((text = '') => ({ width: String(text).length * 12 })),
}

describe('DashboardView', () => {
  beforeEach(() => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.restoreAllMocks()
    routeMock.current = { name: 'dashboard', path: '/dashboard' }
    vi.spyOn(window.HTMLCanvasElement.prototype, 'getContext').mockReturnValue(mockCanvasContext)
  })

  function prepareStore() {
    const pinia = createPinia()
    setActivePinia(pinia)
    const userStore = useUserStore()

    userStore.user = {
      name: '测试学生',
      email: 'student@test.local',
    }
    userStore.progressList = []
    userStore.dashboardSummary = {
      recommendedPath: [
        {
          title: '梯度下降',
          detail: '建议优先理解梯度下降。',
          reasonLabel: '图谱关联',
          score: 0.95,
        },
      ],
      knowledgeCards: [
        { title: '梯度下降', domain: '图谱关联', progress: 95 },
      ],
      weeklyTrend: [],
    }
    userStore.competencyProfile = {
      profile: { name: '测试学生' },
      radarAxes: [],
      growthInsights: [],
      growthSummaryCards: [],
      heatmapDomains: [],
      heatmapRows: [],
    }

    userStore.loadProgress = vi.fn().mockResolvedValue(userStore.progressList)
    userStore.loadDashboard = vi.fn().mockResolvedValue(userStore.dashboardSummary)
    userStore.loadCompetency = vi.fn().mockResolvedValue(userStore.competencyProfile)
    return { pinia, userStore }
  }

  it('renders a student learning overview with fallback radar axes', async () => {
    const { pinia } = prepareStore()
    const wrapper = mount(DashboardView, {
      global: {
        plugins: [pinia],
      },
    })

    await flushPromises()

    expect(wrapper.find('canvas').exists()).toBe(true)
    expect(wrapper.text()).toContain('测试学生的学习总览')
    expect(wrapper.text()).toContain('能力分布')
    expect(wrapper.text()).toContain('机器学习81%')
    expect(wrapper.text()).toContain('下一步学习')
  })

  it('renders a distinct competency map view on the competency route', async () => {
    routeMock.current = { name: 'competency-map', path: '/competency' }
    const { pinia } = prepareStore()

    const wrapper = mount(DashboardView, {
      global: {
        plugins: [pinia],
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('测试学生的能力图谱')
    expect(wrapper.text()).toContain('能力雷达图')
    expect(wrapper.text()).toContain('能力补强路径')
    expect(wrapper.text()).toContain('平均能力')
    expect(wrapper.text()).not.toContain('测试学生的学习总览')
  })
})
