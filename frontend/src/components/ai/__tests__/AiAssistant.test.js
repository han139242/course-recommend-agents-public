import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const sendSpy = vi.hoisted(() => vi.fn())

vi.mock('../../../composables/useAiChat', () => ({
  useAiChat() {
    return {
      messages: ref([
        {
          role: 'assistant',
          content: '你可以先看矩阵，再理解梯度下降。',
          agentLabel: '知识探索助手',
        },
      ]),
      quickPrompts: ref(['帮我解释矩阵', '推荐下一步']),
      sending: ref(false),
      send: sendSpy,
    }
  },
}))

import AiAssistant from '../AiAssistant.vue'

describe('AiAssistant', () => {
  it('renders a proactive hint bubble when the panel is closed', () => {
    const wrapper = mount(AiAssistant, {
      props: {
        modelValue: false,
        isAuthenticated: true,
        proactiveHints: [
          {
            reasonCode: 'gap_bridge',
            title: '先补前置链',
            detail: '你可以先回到矩阵，再继续往后学。',
          },
        ],
      },
    })

    expect(wrapper.find('.ai-proactive-bubble').exists()).toBe(true)
    expect(wrapper.find('.ai-proactive-bubble').text()).toContain('先补前置链')
  })

  it('shows learner profile and emits focus events from proactive hint cards', async () => {
    const wrapper = mount(AiAssistant, {
      props: {
        modelValue: true,
        isAuthenticated: true,
        learnerProfile: {
          style: 'systematic',
          label: '系统型学习者',
          summary: '更适合按层次逐步推进。',
        },
        proactiveHints: [
          {
            reasonCode: 'cross_course_transfer',
            title: '跨课程迁移',
            detail: '线性代数里的矩阵可以迁移到深度学习。',
            knowledgeId: '矩阵',
            severity: 'high',
          },
        ],
      },
    })

    expect(wrapper.find('.ai-learner-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('系统型学习者')
    expect(wrapper.text()).toContain('知识探索助手')
    expect(wrapper.text()).toContain('快捷提问')

    await wrapper.find('.ai-hint-card .inline-link--ai').trigger('click')

    expect(wrapper.emitted('focus-knowledge')).toEqual([['矩阵']])
  })

  it('hides the floating action button when the panel is open', () => {
    const wrapper = mount(AiAssistant, {
      props: {
        modelValue: true,
        isAuthenticated: true,
      },
    })

    expect(wrapper.find('.ai-assistant').classes()).toContain('ai-assistant--open')
    expect(wrapper.find('.ai-fab').exists()).toBe(true)
  })
})
