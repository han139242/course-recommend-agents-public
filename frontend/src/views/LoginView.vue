<script setup>
import { reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { login } from '../auth'

const route = useRoute()
const router = useRouter()

const submitting = ref(false)
const errorMessage = ref('')

const form = reactive({
  email: '',
  password: '',
})

async function handleSubmit() {
  submitting.value = true
  errorMessage.value = ''

  try {
    await login(form)
    router.push(String(route.query.redirect || '/profile'))
  } catch (error) {
    errorMessage.value = error.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="auth-page">
    <div class="detail-hero">
      <span class="section-tag">账号登录</span>
      <h1>登录你的学习账号</h1>
      <p>登录后可保存学习记录、查看个人档案，并让推荐结果和你的真实学习目标保持一致。</p>
    </div>

    <div class="auth-layout">
      <article class="auth-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">登录</span>
            <h2>欢迎回来</h2>
          </div>
        </div>

        <form class="auth-form" @submit.prevent="handleSubmit">
          <label class="auth-field">
            <span>邮箱</span>
            <input v-model.trim="form.email" type="email" placeholder="请输入邮箱" autocomplete="email" />
          </label>

          <label class="auth-field">
            <span>密码</span>
            <input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
            />
          </label>

          <p v-if="errorMessage" class="auth-error">{{ errorMessage }}</p>

          <button class="auth-submit" type="submit" :disabled="submitting">
            {{ submitting ? '登录中...' : '登录并进入学习空间' }}
          </button>
        </form>
      </article>

      <aside class="auth-side">
        <div class="detail-side-card">
          <span class="section-tag">新用户</span>
          <h3>还没有账号？</h3>
          <p class="auth-side-copy">注册后就能把姓名、目标方向和学习档案保存到数据库里。</p>
          <RouterLink to="/register" class="auth-secondary-link">去注册</RouterLink>
        </div>
      </aside>
    </div>
  </section>
</template>
