import { mount } from '@vue/test-utils'

import StudyPlanCard from '../StudyPlanCard.vue'

describe('StudyPlanCard', () => {
  it('shows goal entry when authenticated user has no profile', () => {
    const wrapper = mount(StudyPlanCard, {
      props: {
        authenticated: true,
      },
    })

    expect(wrapper.text()).toContain('先定目标')
    expect(wrapper.text()).toContain('制定目标')
  })

  it('shows generate action when profile exists without plan', () => {
    const wrapper = mount(StudyPlanCard, {
      props: {
        authenticated: true,
        profile: { goalText: '想做智能体项目' },
      },
    })

    expect(wrapper.text()).toContain('目标已保存')
    expect(wrapper.text()).toContain('生成计划')
  })

  it('emits start and highlight events for generated plan', async () => {
    const wrapper = mount(StudyPlanCard, {
      props: {
        authenticated: true,
        profile: { goalText: '想做智能体项目' },
        plan: {
          summary: '智能体三阶段路线',
          nextSteps: [{ title: '线性代数', knowledgeId: 'linear-algebra' }],
          stages: [
            { key: 'foundation', title: '基础补齐', summary: '先补基础', items: [{ title: '线性代数', knowledgeId: 'linear-algebra' }] },
          ],
        },
      },
    })

    expect(wrapper.text()).toContain('智能体三阶段路线')
    await wrapper.find('.primary-btn').trigger('click')
    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual(['linear-algebra'])

    const secondaryButtons = wrapper.findAll('.secondary-btn')
    await secondaryButtons[0].trigger('click')
    expect(wrapper.emitted('highlight-path')?.[0]?.[0]).toEqual(['linear-algebra'])
  })
})
