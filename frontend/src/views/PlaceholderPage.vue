<script setup>
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()

const title = computed(() => route.meta.title ?? '页面')
const description = computed(
  () => route.meta.description ?? '该功能正在开发中，后续会补充完整内容。',
)
</script>

<template>
  <section class="placeholder-page">
    <div class="placeholder-card">
      <span class="section-tag">正在开发</span>
      <h1>{{ title }}</h1>
      <p>{{ description }}</p>
      <p class="placeholder-tip">当前你可以先返回首页，从知识点学习、阶段学习和推荐路径开始体验。</p>
      <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
    </div>
  </section>
</template>
