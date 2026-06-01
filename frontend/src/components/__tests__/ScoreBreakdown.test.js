import { mount } from '@vue/test-utils'

import ScoreBreakdown from '../ScoreBreakdown.vue'

describe('ScoreBreakdown', () => {
  it('renders Chinese factor labels and clamps unsafe values', async () => {
    const wrapper = mount(ScoreBreakdown, {
      props: {
        breakdown: {
          overall: 1.2,
          weakness: 0.76,
          prerequisiteReadiness: 0.62,
          targetRelevance: 0.9,
          forgettingRisk: Number.NaN,
          crossCourseTransfer: -0.5,
          frontierRelevance: 2,
          masteredPenalty: 0.25,
        },
      },
    })

    expect(wrapper.find('.score-breakdown__overall').text()).toBe('100')
    expect(wrapper.find('button').attributes('aria-expanded')).toBe('false')

    await wrapper.find('button').trigger('click')

    expect(wrapper.find('button').attributes('aria-expanded')).toBe('true')
    expect(wrapper.text()).toContain('薄弱程度')
    expect(wrapper.text()).toContain('前置基础')
    expect(wrapper.text()).toContain('目标匹配')
    expect(wrapper.text()).toContain('复习价值')
    expect(wrapper.text()).toContain('知识迁移')
    expect(wrapper.text()).toContain('扩展关联')
    expect(wrapper.text()).toContain('熟练度修正')

    const values = wrapper.findAll('.sf-bar__value').map((node) => node.text())
    expect(values).toEqual(['76%', '62%', '90%', '0%', '0%', '100%', '25%'])
  })

  it('does not render an empty shell without breakdown data', () => {
    const wrapper = mount(ScoreBreakdown)

    expect(wrapper.find('.score-breakdown').exists()).toBe(false)
  })
})
