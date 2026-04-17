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
})
