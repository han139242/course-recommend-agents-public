import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { vi } from 'vitest'

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: {
      knowledgeId: 'gradient-descent',
    },
  }),
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
}))

vi.mock('../../composables/useKatex', () => ({
  renderWithKatex: (content) => content,
}))

vi.mock('../../api/learning', () => ({
  fetchStudentQuestions: vi.fn().mockResolvedValue({ list: [] }),
  submitStudentAnswer: vi.fn().mockResolvedValue({
    correct: true,
    correctAnswer: 'A',
    explanation: '回答正确。',
  }),
}))

import KnowledgeDetail from '../KnowledgeDetail.vue'
import { useGraphStore } from '../../stores/graphStore'
import { useUserStore } from '../../stores/userStore'

describe('KnowledgeDetail', () => {
  function mountWithDetail(detailOverrides = {}, frontierOverride) {
    const pinia = createPinia()
    setActivePinia(pinia)

    const graphStore = useGraphStore()
    const userStore = useUserStore()

    const detail = {
      routeKnowledgeId: 'gradient-descent',
      canonicalName: '梯度下降',
      chapterTitle: '机器学习',
      entityType: 'concept',
      summary: '梯度下降用于沿着损失函数下降方向更新参数。',
      prerequisites: ['导数', '损失函数'],
      scenarios: ['模型训练', '参数优化'],
      recommendReason: '建议结合损失函数和学习率一起理解。',
      ...detailOverrides,
    }

    graphStore.catalogNodes = [detail]
    graphStore.detailCache = {
      'gradient-descent': detail,
    }
    graphStore.ensureNodeDetail = vi.fn().mockResolvedValue(detail)
    graphStore.ensureFrontier = frontierOverride !== undefined
      ? vi.fn().mockResolvedValue(frontierOverride)
      : vi.fn().mockResolvedValue({
      status: 'ready',
      digestSummary: '梯度下降作为优化领域的基础算法，正与自适应学习率、二阶优化等前沿方向深度融合。',
      topics: [
        { topicId: 't1', name: '自适应优化器', reason: '从 Adam 到 Sophia，自适应方法持续推动训练效率提升。' },
        { topicId: 't2', name: '联邦优化', reason: '数据不出本地的分布式场景下，梯度压缩和通信优化是热点。' },
      ],
      papers: [
        { paperId: 'p1', title: 'Sophia: A Scalable Stochastic Second-order Optimizer', venue: 'ICLR', year: 2024, url: 'https://arxiv.org/abs/2305.14342' },
        { paperId: 'p2', title: 'Communication-Efficient Federated Bilevel Optimization', venue: 'NeurIPS', year: 2025, url: 'https://arxiv.org/abs/2410.12345' },
      ],
    })
    graphStore.ensureRecommendedPath = vi.fn().mockResolvedValue({
      path: [
        {
          knowledgeId: 'loss-function',
          title: '损失函数',
          score: 0.84,
          route: '/knowledge/loss-function',
        },
      ],
    })
    graphStore.loadCatalog = vi.fn().mockResolvedValue()

    userStore.notes = {}
    userStore.bookmarks = []
    userStore.getProgressMeta = vi.fn(() => ({ masteryScore: 64, answeredCount: 3 }))
    userStore.toggleBookmark = vi.fn()
    userStore.markMastered = vi.fn()
    userStore.saveNote = vi.fn()

    const wrapper = mount(KnowledgeDetail, {
      global: {
        plugins: [pinia],
      },
    })

    return { wrapper, graphStore, userStore, detail }
  }

  it('renders student-facing knowledge detail with frontier research sections', async () => {
    const { wrapper, graphStore, detail } = mountWithDetail()

    await flushPromises()

    expect(wrapper.text()).toContain('知识点说明')
    expect(wrapper.text()).toContain('梯度下降用于沿着损失函数下降方向更新参数。')
    expect(wrapper.text()).toContain('练习测验')
    expect(wrapper.text()).toContain('AI 学习助教')
    expect(wrapper.text()).toContain('推荐路径')
    expect(wrapper.text()).toContain('前沿扩展')
    expect(wrapper.text()).toContain('自适应优化器')
    expect(wrapper.text()).toContain('Sophia: A Scalable Stochastic Second-order Optimizer')
    expect(graphStore.ensureFrontier).toHaveBeenCalledWith('gradient-descent')
    expect(graphStore.ensureRecommendedPath).toHaveBeenCalledWith(detail.canonicalName, 'gradient-descent', 'detail')
  })

  it('shows frontier papers as external links with view-original tags', async () => {
    const { wrapper } = mountWithDetail()
    await flushPromises()

    const links = wrapper.findAll('.frontier-paper-link')
    expect(links.length).toBe(2)
    expect(links[0].attributes('href')).toBe('https://arxiv.org/abs/2305.14342')
    expect(links[0].attributes('target')).toBe('_blank')
    expect(links[0].text()).toContain('查看原文')
  })

  it('shows empty state when frontier briefing is not ready', async () => {
    const { wrapper } = mountWithDetail({}, null)
    await flushPromises()
    expect(wrapper.text()).toContain('暂未收录前沿扩展内容')
  })

  it('renders showcase quiz questions and reveals explanation after submission', async () => {
    const { wrapper } = mountWithDetail()

    await flushPromises()

    expect(wrapper.text()).toContain('关于”梯度下降”，下列哪项最能概括它的核心含义？')

    await wrapper.find('.quiz-option').trigger('click')
    await wrapper.find('.quiz-item__footer .primary-btn').trigger('click')

    expect(wrapper.text()).toContain('回答正确')
    expect(wrapper.text()).toContain('因此应先把握它在当前课程体系中的核心定位。')
  })

  it('keeps notes and mastery actions available on the side panel', async () => {
    const { wrapper, userStore } = mountWithDetail()

    await flushPromises()

    await wrapper.find('textarea').setValue('学习率过大可能震荡。')
    await wrapper.find('.detail-side__card:last-child .primary-btn').trigger('click')

    expect(userStore.saveNote).toHaveBeenCalledWith('gradient-descent', '学习率过大可能震荡。')
  })
})
