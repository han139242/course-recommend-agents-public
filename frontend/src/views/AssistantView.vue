<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

import { assistantPrompts, assistantReplies } from '../data/learning'

const route = useRoute()

const draft = ref((route.query.q ?? '').toString())
const selectedPrompt = ref(
  assistantPrompts.includes(draft.value) ? draft.value : assistantPrompts[0],
)

const currentReply = computed(() => {
  if (!draft.value.trim()) {
    return '你可以直接输入当前学习卡点，或者点击上方示例问题，我会根据你的目标、薄弱点和推荐路径给出建议。'
  }

  return (
    assistantReplies[draft.value.trim()] ??
    `已收到你的问题：“${draft.value.trim()}”。当前演示版会优先围绕薄弱知识点、先修依赖和下一步推荐给出建议，后续可以继续接入真实对话能力。`
  )
})

function usePrompt(prompt) {
  draft.value = prompt
  selectedPrompt.value = prompt
}
</script>

<template>
  <section class="assistant-page">
    <div class="detail-hero">
      <span class="section-tag">AI 助学室</span>
      <h1>跨课程学习助手</h1>
      <p>围绕知识点补弱、进阶学习和场景目标提供建议，帮助你快速找到下一步该学什么。</p>
    </div>

    <div class="assistant-layout">
      <article class="assistant-main-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">智能问答</span>
            <h2>先问问题，再决定学习路径</h2>
          </div>
        </div>

        <div class="assistant-dialogue">
          <div class="chat-bubble user">
            <span>我的问题</span>
            <strong>{{ draft || '暂未输入问题' }}</strong>
          </div>

          <div class="chat-bubble assistant">
            <span>AI 助学建议</span>
            <strong>{{ currentReply }}</strong>
          </div>
        </div>

        <div class="assistant-input-box">
          <textarea
            v-model="draft"
            class="assistant-textarea"
            rows="4"
            placeholder="输入你当前学不懂的知识点、目标方向，或想完成的项目场景"
          />
        </div>
      </article>

      <aside class="assistant-side">
        <div class="detail-side-card">
          <span class="section-tag">示例问题</span>
          <div class="assistant-quick-list">
            <button
              v-for="prompt in assistantPrompts"
              :key="prompt"
              class="assistant-quick-item"
              :class="{ active: selectedPrompt === prompt }"
              @click="usePrompt(prompt)"
            >
              {{ prompt }}
            </button>
          </div>
        </div>

        <div class="detail-side-card">
          <span class="section-tag">当前能力</span>
          <div class="assistant-mini-tags">
            <span>补弱优先</span>
            <span>数学基础待提升</span>
            <span>适合进入 CNN 主线</span>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
