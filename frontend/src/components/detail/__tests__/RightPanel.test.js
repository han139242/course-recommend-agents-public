import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'

import { useGraphStore } from '../../../stores/graphStore'
import { useUserStore } from '../../../stores/userStore'
import RightPanel from '../RightPanel.vue'

describe('RightPanel', () => {
  it('renders bridge recommendations for cross-course transfer items', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    const graphStore = useGraphStore()

    userStore.bookmarks = []
    graphStore.catalogNodes = [
      {
        id: 'B',
        routeKnowledgeId: 'B',
        name: '神经网络',
        canonicalName: '神经网络',
        chapterTitle: '深度学习',
        entityType: 'concept',
      },
    ]

    const wrapper = mount(RightPanel, {
      global: {
        plugins: [pinia],
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
      props: {
        collapsed: false,
        detail: {
          routeKnowledgeId: 'A',
          canonicalName: '矩阵',
          chapterTitle: '线性代数',
          entityType: 'concept',
          prerequisites: [],
          related: [],
        },
        progress: {
          masteryScore: 62,
        },
        nextNodes: [],
        recommendedPath: [
          {
            knowledgeId: 'B',
            title: '神经网络',
            reasonCode: 'cross_course_transfer',
            reasonLabel: '跨课程迁移',
            stage: '跨课程',
            sourceCourse: '线性代数',
            targetCourse: '深度学习',
            masteryScore: 35,
            detail: '矩阵运算会直接迁移到神经网络的参数更新。',
          },
        ],
      },
    })

    expect(wrapper.find('.bridge-view').exists()).toBe(true)

    await wrapper.find('.recommend-card__title').trigger('click')

    expect(wrapper.emitted('select-related')).toEqual([['B']])
  })

  it('normalizes legacy course labels in cross-course recommendations', () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    const graphStore = useGraphStore()

    userStore.bookmarks = []
    graphStore.catalogNodes = [
      {
        id: 'B',
        routeKnowledgeId: 'B',
        name: 'BERT',
        canonicalName: 'BERT',
        chapterTitle: '自然语言处理',
        entityType: 'concept',
      },
    ]

    const wrapper = mount(RightPanel, {
      global: {
        plugins: [pinia],
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
      props: {
        collapsed: false,
        detail: {
          routeKnowledgeId: 'A',
          canonicalName: '知识表示',
          chapterTitle: '人工智能导论',
          entityType: 'concept',
          prerequisites: [],
          related: [],
        },
        progress: {
          masteryScore: 55,
        },
        nextNodes: [],
        recommendedPath: [
          {
            knowledgeId: 'B',
            title: 'BERT',
            reasonCode: 'cross_course_transfer',
            reasonLabel: '跨课程迁移',
            sourceCourse: '人工智能基础知识骨架',
            targetCourse: '自然语言处理及其应用',
            masteryScore: 30,
          },
        ],
      },
    })

    expect(wrapper.text()).toContain('人工智能导论')
    expect(wrapper.text()).toContain('自然语言处理')
  })
})
