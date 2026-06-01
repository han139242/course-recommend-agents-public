<script setup>
import { Map, UserPlus } from '@lucide/vue'
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { useUserStore } from '../stores/userStore'

const router = useRouter()
const userStore = useUserStore()
const form = reactive({
  email: '',
  password: '',
  confirmPassword: '',
})
const loading = ref(false)
const error = ref('')

function validateForm() {
  if (!form.email.trim()) return '请输入邮箱地址'
  if (!form.password) return '请输入密码'
  if (form.password.length < 6) return '密码至少需要 6 位'
  if (!/[a-zA-Z]/.test(form.password) || !/[0-9]/.test(form.password)) {
    return '密码需同时包含字母和数字'
  }
  if (form.password !== form.confirmPassword) return '两次输入的密码不一致'
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
    await userStore.register(form.email, form.password, 'student')
    router.push('/dashboard')
  } catch (registerError) {
    error.value = registerError?.message || '注册暂时不可用，请联系管理员创建账号。'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-view page-frame" @keydown.enter="submit">
    <div class="auth-shell">
      <aside class="auth-aside">
        <span class="eyebrow">Student Account</span>
        <h1>创建学习账号</h1>
        <p>用于保存学习记录、推荐路径和个人笔记。</p>
      </aside>

      <div class="auth-card page-card">
        <p class="eyebrow">创建账号</p>
        <h2>注册学生身份</h2>
        <p class="muted-text">如果后端暂未开放自助注册，这里会给出清晰提示。</p>

        <label>
          邮箱
          <input
            v-model="form.email"
            type="email"
            placeholder="new_student@school.edu"
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
            placeholder="至少 6 位，包含字母和数字"
            autocomplete="new-password"
            :aria-invalid="!!error"
            :disabled="loading"
          />
          <span class="muted-text">至少 6 位，需同时包含字母和数字</span>
        </label>
        <label>
          确认密码
          <input
            v-model="form.confirmPassword"
            type="password"
            placeholder="再次输入密码"
            autocomplete="new-password"
            :aria-invalid="!!error"
            :disabled="loading"
          />
        </label>

        <p v-if="error" class="form-error" role="alert">{{ error }}</p>

        <button type="button" class="primary-btn" :disabled="loading" @click="submit">
          <UserPlus :size="16" aria-hidden="true" />
          {{ loading ? '提交中...' : '注册' }}
        </button>

        <RouterLink class="link-button" to="/login">已有账号？去登录</RouterLink>
        <RouterLink class="secondary-btn" to="/">
          <Map :size="16" aria-hidden="true" />
          返回知识地图
        </RouterLink>
      </div>
    </div>
  </section>
</template>
