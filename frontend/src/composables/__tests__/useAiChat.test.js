import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const onActionsSpy = vi.fn()
const streamAssistantChatMock = vi.fn()

vi.mock('../../api/assistant', () => ({
  streamAssistantChat: (...args) => streamAssistantChatMock(...args),
}))

import { useAiChat } from '../useAiChat'

const TestHarness = defineComponent({
  setup() {
    const selectedDetail = ref(null)
    const isAuthenticated = ref(true)
    const { messages, send, sending } = useAiChat(selectedDetail, isAuthenticated, {
      onActions: onActionsSpy,
    })

    return {
      messages,
      send,
      sending,
    }
  },
  template: `
    <div>
      <button class="send" type="button" @click="send('帮我快速理解当前图谱')">send</button>
      <p class="content">{{ messages[messages.length - 1]?.content || '' }}</p>
      <span class="agent">{{ messages[messages.length - 1]?.agentLabel || '' }}</span>
      <span class="sending">{{ sending ? 'yes' : 'no' }}</span>
    </div>
  `,
})

describe('useAiChat', () => {
  it('reactively renders streamed assistant content into the latest message', async () => {
    onActionsSpy.mockReset()
    streamAssistantChatMock.mockImplementationOnce(async (_messages, handlers = {}) => {
      handlers.onMessage?.('先看课程簇')
      await nextTick()
      handlers.onMessage?.('，再展开核心节点。')
      handlers.onEvent?.({
        type: 'assistant_meta',
        agent: 'explorer',
        agentLabel: '知识探索助手',
        actions: [],
      })
    })

    const wrapper = mount(TestHarness)

    await wrapper.get('.send').trigger('click')
    await nextTick()
    await nextTick()

    expect(wrapper.get('.content').text()).toContain('先看课程簇，再展开核心节点。')
    expect(wrapper.get('.agent').text()).toBe('知识探索助手')
    expect(wrapper.get('.sending').text()).toBe('no')
  })

  it('renders a local fallback when the stream resolves without content', async () => {
    onActionsSpy.mockReset()
    streamAssistantChatMock.mockImplementationOnce(async () => {})

    const wrapper = mount(TestHarness)

    await wrapper.get('.send').trigger('click')
    await nextTick()
    await nextTick()

    expect(wrapper.get('.content').text()).not.toBe('')
    expect(wrapper.get('.content').text()).toContain('本地')
    expect(wrapper.get('.sending').text()).toBe('no')
  })

  it('does not execute graph actions for explanation-style prompts', async () => {
    onActionsSpy.mockReset()
    streamAssistantChatMock.mockImplementationOnce(async (_messages, handlers = {}) => {
      handlers.onMessage?.('已为你尝试重置当前图谱场景。\n\n先看课程簇，再展开核心节点。')
      handlers.onEvent?.({
        type: 'assistant_meta',
        agent: 'explorer',
        agentLabel: '知识探索助手',
        actions: [{ type: 'reset_graph_scene' }],
      })
    })

    const wrapper = mount(TestHarness)

    await wrapper.get('.send').trigger('click')
    await nextTick()
    await nextTick()

    expect(wrapper.get('.content').text()).toBe('先看课程簇，再展开核心节点。')
    expect(onActionsSpy).not.toHaveBeenCalled()
  })

  it('executes graph actions for natural course-switch prompts', async () => {
    onActionsSpy.mockReset()
    streamAssistantChatMock.mockImplementationOnce(async (_messages, handlers = {}) => {
      handlers.onMessage?.('我已经让前端按“强化学习”课程过滤图谱，并尽量定位到该课程的代表知识点。')
      handlers.onEvent?.({
        type: 'assistant_meta',
        agent: 'explorer',
        agentLabel: '知识探索助手',
        actions: [{ type: 'filter_course', courseTitle: '强化学习' }],
      })
    })

    const wrapper = mount(TestHarness)

    await wrapper.vm.send('帮我把图谱切到强化学习')
    await nextTick()
    await nextTick()

    expect(onActionsSpy).toHaveBeenCalledWith([{ type: 'filter_course', courseTitle: '强化学习' }])
    expect(wrapper.get('.content').text()).toContain('强化学习')
  })

  it('renders an action hint when the stream only returns actions', async () => {
    onActionsSpy.mockReset()
    streamAssistantChatMock.mockImplementationOnce(async (_messages, handlers = {}) => {
      handlers.onEvent?.({
        type: 'assistant_meta',
        agent: 'explorer',
        agentLabel: '知识探索助手',
        actions: [{ type: 'fit_graph_view' }],
      })
    })

    const wrapper = mount(TestHarness)

    await wrapper.vm.send('适配视图')
    await nextTick()
    await nextTick()

    expect(onActionsSpy).toHaveBeenCalledWith([{ type: 'fit_graph_view' }])
    expect(wrapper.get('.content').text()).toContain('适配')
  })
})
