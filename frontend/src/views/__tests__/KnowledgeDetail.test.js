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
    template: '<a><slot /></a>',
  },
}))

vi.mock('../../composables/useKatex', () => ({
  renderWithKatex: (content) => content,
}))

import KnowledgeDetail from '../KnowledgeDetail.vue'
import { useGraphStore } from '../../stores/graphStore'
import { useUserStore } from '../../stores/userStore'

describe('KnowledgeDetail', () => {
  it('renders frontier digest fields returned by the briefing API', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const graphStore = useGraphStore()
    const userStore = useUserStore()

    const detail = {
      routeKnowledgeId: 'gradient-descent',
      canonicalName: '梯度下降',
      chapterTitle: '机器学习',
      entityType: 'concept',
      summary: '知识点说明',
      prerequisites: ['线性回归'],
      scenarios: ['优化训练'],
      recommendReason: '建议结合优化器一起学习。',
    }
    const frontier = {
      status: 'ready',
      digestSummary: '梯度下降近期在自适应优化方向上仍是核心基础。',
      topics: [
        {
          topicId: 'adaptive-optimization',
          name: '自适应优化',
          reason: '与梯度下降直接相关。',
        },
      ],
      papers: [
        {
          paperId: 'paper-1',
          title: 'A Survey on Adaptive Optimization',
          venue: 'NeurIPS',
          year: 2025,
          url: 'https://example.com/paper-1',
        },
      ],
      generatedAt: '2026-04-01T00:00:00Z',
    }

    graphStore.catalogNodes = [detail]
    graphStore.detailCache = {
      'gradient-descent': detail,
    }
    graphStore.frontierCache = {
      'gradient-descent': frontier,
    }
    graphStore.ensureNodeDetail = vi.fn().mockResolvedValue(detail)
    graphStore.ensureFrontier = vi.fn().mockResolvedValue(frontier)
    graphStore.ensureRecommendedPath = vi.fn().mockResolvedValue({ path: [] })
    graphStore.loadCatalog = vi.fn().mockResolvedValue()

    userStore.notes = {}
    userStore.bookmarks = []
    userStore.getProgressMeta = vi.fn(() => ({ masteryScore: 64 }))
    userStore.toggleBookmark = vi.fn()
    userStore.markMastered = vi.fn()
    userStore.saveNote = vi.fn()

    const wrapper = mount(KnowledgeDetail, {
      global: {
        plugins: [pinia],
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('梯度下降近期在自适应优化方向上仍是核心基础。')
    expect(wrapper.text()).toContain('自适应优化')
    expect(wrapper.text()).toContain('A Survey on Adaptive Optimization')
    expect(graphStore.ensureFrontier).toHaveBeenCalledWith('gradient-descent')
    expect(graphStore.ensureRecommendedPath).toHaveBeenCalledWith(detail.canonicalName, 'gradient-descent', 'detail')
  })
})
