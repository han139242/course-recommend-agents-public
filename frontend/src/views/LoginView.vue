<script setup>
import { reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { useUserStore } from '../stores/userStore'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const form = reactive({
  email: '',
  password: '',
})
const loading = ref(false)
const error = ref('')

function fillDemoStudent() {
  form.email = 'student@test.local'
  form.password = 'TestPass123!'
}

async function submit() {
  error.value = ''
  loading.value = true

  try {
    await userStore.login(form.email, form.password)
    const redirect = route.query.redirect ? decodeURIComponent(route.query.redirect) : '/'
    router.push(redirect)
  } catch (loginError) {
    error.value = loginError?.message || '登录失败，请检查用户名和密码。'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-view page-frame">
    <div class="auth-card page-card">
      <p class="eyebrow">欢迎回来</p>
      <h1>登录学习空间</h1>
      <p>登录后可以同步个人学习进度、收藏和 AI 推荐路径。</p>
      <p>当前测试学生账号：`student@test.local` / `TestPass123!`。</p>
      <button type="button" class="secondary-btn" @click="fillDemoStudent">填入测试学生账号</button>

      <label>
        邮箱
        <input v-model="form.email" type="email" placeholder="student@test.local" />
      </label>
      <label>
        密码
        <input v-model="form.password" type="password" placeholder="TestPass123!" />
      </label>

      <p v-if="error" class="form-error">{{ error }}</p>

      <button type="button" class="primary-btn" :disabled="loading" @click="submit">
        {{ loading ? '登录中...' : '登录' }}
      </button>

      <RouterLink class="link-button" to="/register">没有账号？去注册</RouterLink>
      <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
    </div>
  </section>
</template>
