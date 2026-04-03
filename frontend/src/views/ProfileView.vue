<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { authState, clearSession, fetchCompetencyProfile, refreshCurrentUser } from '../auth'

const router = useRouter()
const loading = ref(true)
const profileSnapshot = ref(null)

const mergedProfile = computed(() => {
  const profile = profileSnapshot.value?.profile
  return {
    name: authState.user?.name || profile?.name || '学习者',
    role: authState.user?.role || profile?.role || '人工智能专业学生',
    target: authState.user?.target ? `目标方向：${authState.user.target}` : profile?.target || '目标方向：待设置',
    summary: profile?.summary || '正在同步你的学习画像...',
    stats: profile?.stats || [],
    goals: profile?.goals || [],
    recentRecords: profile?.recentRecords || [],
  }
})

function logout() {
  clearSession()
  router.push('/')
}

onMounted(async () => {
  try {
    await refreshCurrentUser()
    profileSnapshot.value = await fetchCompetencyProfile()
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="profile-page">
    <div class="detail-hero">
      <span class="section-tag">个人学习档案</span>
      <h1>{{ mergedProfile.name }}</h1>
      <p>{{ mergedProfile.role }} · {{ mergedProfile.target }}</p>
    </div>

    <div class="profile-layout">
      <article class="profile-main-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">学习概况</span>
            <h2>当前学习状态</h2>
          </div>
        </div>

        <p class="profile-summary">
          {{ loading ? '正在同步你的账号信息...' : mergedProfile.summary }}
        </p>

        <div class="profile-account-box">
          <div>
            <span class="section-tag">账号信息</span>
            <p class="profile-account-text">邮箱：{{ authState.user?.email }}</p>
          </div>
          <button class="profile-logout-btn" @click="logout">退出登录</button>
        </div>

        <div class="profile-stats-grid">
          <article v-for="item in mergedProfile.stats" :key="item.label" class="stat-card">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </article>
        </div>
      </article>

      <aside class="profile-side">
        <div class="detail-side-card">
          <span class="section-tag">当前目标</span>
          <ul class="detail-list">
            <li v-for="goal in mergedProfile.goals" :key="goal">{{ goal }}</li>
          </ul>
        </div>

        <div class="detail-side-card">
          <span class="section-tag">最近记录</span>
          <ul class="detail-list">
            <li v-for="record in mergedProfile.recentRecords" :key="record">{{ record }}</li>
          </ul>
        </div>
      </aside>
    </div>
  </section>
</template>
