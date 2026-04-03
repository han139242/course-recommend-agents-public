<script setup>
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { register } from '../auth'

const router = useRouter()

const submitting = ref(false)
const errorMessage = ref('')

const form = reactive({
  name: '',
  email: '',
  password: '',
  role: '人工智能专业学生',
  target: '',
})

async function handleSubmit() {
  submitting.value = true
  errorMessage.value = ''

  try {
    await register(form)
    router.push('/profile')
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
      <span class="section-tag">账号注册</span>
      <h1>创建你的学习账号</h1>
      <p>把基础身份信息和学习目标录入系统后，推荐路径会更贴合你的能力阶段与发展方向。</p>
    </div>

    <div class="auth-layout">
      <article class="auth-card">
        <div class="section-head compact">
          <div>
            <span class="section-tag">注册</span>
            <h2>开始建立个人学习档案</h2>
          </div>
        </div>

        <form class="auth-form" @submit.prevent="handleSubmit">
          <label class="auth-field">
            <span>姓名</span>
            <input v-model.trim="form.name" type="text" placeholder="请输入姓名" autocomplete="name" />
          </label>

          <label class="auth-field">
            <span>邮箱</span>
            <input v-model.trim="form.email" type="email" placeholder="请输入邮箱" autocomplete="email" />
          </label>

          <label class="auth-field">
            <span>密码</span>
            <input
              v-model="form.password"
              type="password"
              placeholder="至少 6 位密码"
              autocomplete="new-password"
            />
          </label>

          <label class="auth-field">
            <span>身份角色</span>
            <input v-model.trim="form.role" type="text" placeholder="如：人工智能专业学生" />
          </label>

          <label class="auth-field">
            <span>目标方向</span>
            <input v-model.trim="form.target" type="text" placeholder="如：算法工程师 / 智能体应用开发" />
          </label>

          <p v-if="errorMessage" class="auth-error">{{ errorMessage }}</p>

          <button class="auth-submit" type="submit" :disabled="submitting">
            {{ submitting ? '注册中...' : '注册并进入学习空间' }}
          </button>
        </form>
      </article>

      <aside class="auth-side">
        <div class="detail-side-card">
          <span class="section-tag">已有账号</span>
          <h3>已经注册过了？</h3>
          <p class="auth-side-copy">直接登录即可继续使用原来的学习档案和推荐记录。</p>
          <RouterLink to="/login" class="auth-secondary-link">去登录</RouterLink>
        </div>
      </aside>
    </div>
  </section>
</template>
