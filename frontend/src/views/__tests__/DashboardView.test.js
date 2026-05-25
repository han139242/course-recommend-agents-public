import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { vi } from 'vitest'

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
}

describe('DashboardView', () => {
  beforeEach(() => {
    const pinia = createPinia()
    setActivePinia(pinia)
    vi.restoreAllMocks()
    vi.spyOn(window.HTMLCanvasElement.prototype, 'getContext').mockReturnValue(mockCanvasContext)
  })

  it('renders radar canvas from fallback signals when competency axes are empty', async () => {
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
          detail: '建议优先理解 梯度下降，当前系统里还没有你的稳定学习记录。',
          reasonLabel: '图谱关联',
          score: 0.95,
        },
        {
          title: 'AdaGrad',
          detail: '建议优先理解 AdaGrad，当前系统里还没有你的稳定学习记录。',
          reasonLabel: '图谱关联',
          score: 0.87,
        },
        {
          title: 'RMSProp',
          detail: '建议优先理解 RMSProp，当前系统里还没有你的稳定学习记录。',
          reasonLabel: '图谱关联',
          score: 0.79,
        },
      ],
      knowledgeCards: [
        { title: '梯度下降', domain: '图谱关联', progress: 95 },
        { title: 'AdaGrad', domain: '图谱关联', progress: 87 },
        { title: 'RMSProp', domain: '图谱关联', progress: 79 },
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

    const wrapper = mount(DashboardView, {
      global: {
        plugins: [pinia],
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('canvas').exists()).toBe(true)
    expect(wrapper.text()).toContain('6 个能力维度已激活')
    expect(wrapper.text()).not.toContain('能力图谱还在生成')
  })
})
