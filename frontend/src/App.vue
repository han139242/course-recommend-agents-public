<script setup>
import { computed } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'

import { authState, clearSession } from './auth'
import { navTabs } from './data/learning'

const router = useRouter()
const profileRoute = computed(() => (authState.user ? '/profile' : '/login'))
const profileLabel = computed(() => authState.user?.name || '登录 / 注册')

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }

  router.push('/')
}

function logout() {
  clearSession()
  router.push('/')
}
</script>

<template>
  <div class="page-shell">
    <header class="app-header">
      <div class="banner-top">
        <button class="back-btn" aria-label="返回" @click="goBack">←</button>
        <div class="header-actions">
        <RouterLink :to="profileRoute" class="profile-pill">
          <div class="avatar-dot"></div>
          <span>{{ profileLabel }}</span>
        </RouterLink>
        <button v-if="authState.user" class="header-logout" @click="logout">退出</button>
        </div>
      </div>

      <nav class="top-tabs app-nav">
        <RouterLink
          v-for="tab in navTabs"
          :key="tab.to"
          :to="tab.to"
          class="top-tab"
          exact-active-class="active"
        >
          {{ tab.label }}
        </RouterLink>
      </nav>
    </header>

    <RouterView />
  </div>
</template>
