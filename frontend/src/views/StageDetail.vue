<script setup>
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { stageTabs } from '../data/learning'

const route = useRoute()

const currentStage = computed(
  () => stageTabs.find((stage) => stage.id === route.params.stageId) ?? stageTabs[0],
)
</script>

<template>
  <section class="detail-page">
    <div class="detail-hero">
      <span class="section-tag">学习阶段详情</span>
      <h1>{{ currentStage.title }}</h1>
      <p>{{ currentStage.intro }}</p>
    </div>

    <div class="detail-layout">
      <article class="detail-main">
        <div class="detail-block">
          <span class="panel-label">{{ currentStage.panelTitle }}</span>
          <h2>{{ currentStage.panelContent }}</h2>
        </div>

        <div class="detail-block">
          <span class="panel-label">{{ currentStage.feedbackLabel }}</span>
          <h3>{{ currentStage.feedbackContent }}</h3>
        </div>

        <div class="detail-block">
          <span class="panel-label">建议动作</span>
          <ul class="detail-list">
            <li v-for="tip in currentStage.tips" :key="tip">{{ tip }}</li>
          </ul>
        </div>

        <div class="detail-block">
          <span class="panel-label">这一阶段在解决什么</span>
          <h3>
            {{
              currentStage.id === 'learn'
                ? '帮助你先找到当前知识点和先修内容的关系，避免直接进入新知识时听不懂。'
                : currentStage.id === 'practice'
                  ? '通过跨课程练习把知识点真正练熟，而不是只停留在看懂概念。'
                  : currentStage.id === 'test'
                    ? '用短测判断知识链接是否真正建立，并把结果同步到后续推荐。'
                    : '根据掌握度、能力目标和应用方向，生成可执行的跨课程学习路径。'
            }}
          </h3>
        </div>
      </article>

      <aside class="detail-side">
        <div class="detail-side-card">
          <span class="section-tag">阶段导航</span>
          <div class="detail-links">
            <RouterLink
              v-for="stage in stageTabs"
              :key="stage.id"
              :to="`/flow/${stage.id}`"
              class="detail-link"
              :class="{ active: stage.id === currentStage.id }"
            >
              <strong>{{ stage.label }}</strong>
              <span>{{ stage.title }}</span>
            </RouterLink>
          </div>
        </div>

        <div class="detail-side-card">
          <span class="section-tag">继续操作</span>
          <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
        </div>
      </aside>
    </div>
  </section>
</template>
