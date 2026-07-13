import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'

import CompetitionShowcaseView from '../CompetitionShowcaseView.vue'

describe('CompetitionShowcaseView', () => {
  it('renders the C4 A-series score alignment and demo evidence', () => {
    const wrapper = mount(CompetitionShowcaseView, {
      global: {
        plugins: [createPinia()],
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('A 系列评分点对齐')
    expect(wrapper.text()).toContain('创意')
    expect(wrapper.text()).toContain('技术')
    expect(wrapper.text()).toContain('应用')
    expect(wrapper.text()).toContain('设计')
    expect(wrapper.text()).toContain('效果')
    expect(wrapper.text()).toContain('5 分钟答辩演示路径')
    expect(wrapper.text()).toContain('scripts/start-demo.ps1')
    expect(wrapper.text()).toContain('17 个测试文件、66 个用例')
  })
})
