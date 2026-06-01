import { flushPromises, mount } from '@vue/test-utils'
import { vi } from 'vitest'

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
}))

vi.mock('../../api/graph', () => ({
  fetchFrontierLibrary: vi.fn().mockResolvedValue({
    status: 'ready',
    message: 'ok',
    generatedAt: '2026-05-31T12:00:00Z',
    total: 1,
    items: [
      {
        paperId: 'p1',
        title: 'Retrieval-Augmented Transformer Agents',
        abstract: 'A paper about RAG agents and knowledge graphs.',
        venue: 'arXiv',
        year: 2026,
        url: 'https://arxiv.org/abs/2605.12345',
        source: 'arxiv',
        topics: [{ topicId: 't1', name: 'RAG Agent', reason: 'extends RAG' }],
        knowledgeRefs: [
          { knowledgeId: 'rag', name: '检索增强生成', course: '大模型应用', reason: 'uses RAG' },
          { knowledgeId: 'transformer', name: 'Transformer', course: '大模型基础', reason: 'uses transformer' },
        ],
      },
    ],
  }),
}))

import FrontierLibraryView from '../FrontierLibraryView.vue'
import { fetchFrontierLibrary } from '../../api/graph'

describe('FrontierLibraryView', () => {
  it('renders frontier papers and linked knowledge nodes', async () => {
    const wrapper = mount(FrontierLibraryView)
    await flushPromises()

    expect(fetchFrontierLibrary).toHaveBeenCalledWith({ limit: 50 })
    expect(wrapper.text()).toContain('AI 前沿资料')
    expect(wrapper.text()).toContain('Retrieval-Augmented Transformer Agents')
    expect(wrapper.text()).toContain('检索增强生成')
    expect(wrapper.text()).toContain('Transformer')
    expect(wrapper.find('a[href="/knowledge/rag"]').exists()).toBe(true)
    expect(wrapper.find('a[href="https://arxiv.org/abs/2605.12345"]').attributes('target')).toBe('_blank')
  })

  it('filters by search query', async () => {
    const wrapper = mount(FrontierLibraryView)
    await flushPromises()

    await wrapper.find('input[type="search"]').setValue('not-found')
    expect(wrapper.text()).not.toContain('Retrieval-Augmented Transformer Agents')

    await wrapper.find('input[type="search"]').setValue('knowledge graphs')
    expect(wrapper.text()).toContain('Retrieval-Augmented Transformer Agents')
  })
})
