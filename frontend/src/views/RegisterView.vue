<script setup>
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { useUserStore } from '../stores/userStore'

const router = useRouter()
const userStore = useUserStore()
const form = reactive({
  email: '',
  password: '',
})
const loading = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
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
  <section class="auth-view page-frame">
    <div class="auth-card page-card">
      <p class="eyebrow">创建账号</p>
      <h1>注册学生身份</h1>
      <p>如果后端暂未开放自助注册，这里会给出清晰提示。</p>

      <label>
        邮箱
        <input v-model="form.email" type="email" placeholder="new_student@school.edu" />
      </label>
      <label>
        密码
        <input v-model="form.password" type="password" placeholder="至少 6 位密码" />
        <span class="muted-text">密码至少 6 位，建议包含数字和字母</span>
      </label>

      <p v-if="error" class="form-error">{{ error }}</p>

      <button type="button" class="primary-btn" :disabled="loading" @click="submit">
        {{ loading ? '提交中...' : '注册' }}
      </button>

      <RouterLink class="link-button" to="/login">已有账号？去登录</RouterLink>
      <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
    </div>
  </section>
</template>
