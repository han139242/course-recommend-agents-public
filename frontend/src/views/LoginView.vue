<script setup>
import { LogIn, Map } from '@lucide/vue'
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

function validateForm() {
  if (!form.email.trim()) return '请输入邮箱地址'
  if (!form.password) return '请输入密码'
  if (form.password.length < 6) return '密码至少需要 6 位'
  return ''
}

async function submit() {
  error.value = ''
  const validationError = validateForm()
  if (validationError) {
    error.value = validationError
    return
  }
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

function handleKeydown(e) {
  if (e.key === 'Enter') submit()
}
</script>

<template>
  <section class="auth-view page-frame" @keydown="handleKeydown">
    <div class="auth-shell">
      <aside class="auth-aside">
        <span class="eyebrow">AI Learning</span>
        <h1>回到你的学习路径</h1>
        <p>登录后同步进度、收藏、笔记和个性化推荐。</p>
      </aside>

      <div class="auth-card page-card">
        <p class="eyebrow">欢迎回来</p>
        <h2>登录学习空间</h2>
        <p class="muted-text">测试学生账号：<code>student@test.local</code> / <code>TestPass123!</code></p>
        <button type="button" class="secondary-btn" @click="fillDemoStudent">填入测试账号</button>

        <label>
          邮箱
          <input
            v-model="form.email"
            type="email"
            placeholder="student@test.local"
            autocomplete="email"
            :aria-invalid="!!error"
            :disabled="loading"
          />
        </label>
        <label>
          密码
          <input
            v-model="form.password"
            type="password"
            placeholder="TestPass123!"
            autocomplete="current-password"
            :aria-invalid="!!error"
            :disabled="loading"
            @keydown.enter="submit"
          />
        </label>

        <p v-if="error" class="form-error" role="alert">{{ error }}</p>

        <button type="button" class="primary-btn" :disabled="loading" @click="submit">
          <LogIn :size="16" aria-hidden="true" />
          {{ loading ? '登录中...' : '登录' }}
        </button>

        <RouterLink class="link-button" to="/register">没有账号？去注册</RouterLink>
        <RouterLink class="secondary-btn" to="/">
          <Map :size="16" aria-hidden="true" />
          返回知识地图
        </RouterLink>
      </div>
    </div>
  </section>
</template>
