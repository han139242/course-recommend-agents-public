<script setup>
import { RouterLink } from 'vue-router'
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
            <h2>当前阶段不开放自主注册</h2>
          </div>
        </div>

        <div class="auth-form">
          <p class="auth-side-copy">
            根据当前 API 契约，学生账号由教师或管理员统一导入创建，前端不再直接提交注册请求。
          </p>
          <p class="auth-side-copy">
            如果你需要体验学生端页面，请让教师先分配账号，或使用已有测试账号直接登录。
          </p>
          <RouterLink to="/login" class="auth-submit">返回登录</RouterLink>
        </div>
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
