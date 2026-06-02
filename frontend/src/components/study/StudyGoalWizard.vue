<script setup>
import { computed, reactive, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  initialProfile: {
    type: Object,
    default: null,
  },
  draftProfile: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['close', 'submit', 'update:modelValue'])

const form = reactive({
  gradeLevel: 'sophomore',
  major: '',
  goalType: 'project_building',
  goalText: '',
  targetDirection: '',
  learnedTopicsText: '',
  foundationLevel: 'intermediate',
  weeklyHours: 6,
  learningPreference: 'balanced',
})

const open = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const canSubmit = computed(() => form.gradeLevel && form.goalType && form.goalText.trim() && form.foundationLevel && form.learningPreference)

function applyProfile(profile = {}) {
  form.gradeLevel = profile.gradeLevel || 'sophomore'
  form.major = profile.major || ''
  form.goalType = profile.goalType || 'project_building'
  form.goalText = profile.goalText || ''
  form.targetDirection = profile.targetDirection || ''
  form.learnedTopicsText = Array.isArray(profile.learnedTopics) ? profile.learnedTopics.join('、') : ''
  form.foundationLevel = profile.foundationLevel || 'intermediate'
  form.weeklyHours = Number(profile.weeklyHours || 6)
  form.learningPreference = profile.learningPreference || 'balanced'
}

watch(
  () => [props.initialProfile, props.draftProfile, props.modelValue],
  () => {
    if (!props.modelValue) return
    applyProfile(props.draftProfile || props.initialProfile || {})
  },
  { immediate: true },
)

function splitTopics(value) {
  return String(value || '')
    .split(/[、,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    gradeLevel: form.gradeLevel,
    major: form.major.trim(),
    goalType: form.goalType,
    goalText: form.goalText.trim(),
    targetDirection: form.targetDirection.trim(),
    learnedTopics: splitTopics(form.learnedTopicsText),
    foundationLevel: form.foundationLevel,
    weeklyHours: Number(form.weeklyHours || 6),
    learningPreference: form.learningPreference,
  })
}
</script>

<template>
  <div v-if="open" class="study-goal-modal" role="dialog" aria-modal="true" aria-labelledby="study-goal-title">
    <div class="study-goal-modal__panel">
      <header class="study-goal-modal__head">
        <div>
          <span class="eyebrow">Learning Goal</span>
          <h2 id="study-goal-title">制定你的学习目标</h2>
        </div>
        <button type="button" class="icon-btn" title="关闭" @click="open = false; emit('close')">×</button>
      </header>

      <div class="study-goal-grid">
        <label>
          <span>你现在是</span>
          <select v-model="form.gradeLevel">
            <option value="freshman">大一</option>
            <option value="sophomore">大二</option>
            <option value="junior">大三</option>
            <option value="senior">大四</option>
            <option value="postgraduate">研究生</option>
            <option value="other">其他</option>
          </select>
        </label>

        <label>
          <span>目标类型</span>
          <select v-model="form.goalType">
            <option value="course_learning">课程学习</option>
            <option value="career_preparation">就业准备</option>
            <option value="project_building">项目实践</option>
          </select>
        </label>

        <label>
          <span>专业或方向</span>
          <input v-model="form.major" type="text" placeholder="例如：人工智能、软件工程" />
        </label>

        <label>
          <span>目标方向</span>
          <input v-model="form.targetDirection" type="text" placeholder="例如：智能体、大模型应用、算法工程师" />
        </label>

        <label class="study-goal-grid__wide">
          <span>我想完成什么</span>
          <textarea v-model="form.goalText" rows="3" placeholder="例如：我是大二学生，想做一个智能体项目，但机器学习基础不太稳。" />
        </label>

        <label class="study-goal-grid__wide">
          <span>已经学过什么</span>
          <input v-model="form.learnedTopicsText" type="text" placeholder="例如：Python、线性代数、机器学习" />
        </label>

        <label>
          <span>基础水平</span>
          <select v-model="form.foundationLevel">
            <option value="beginner">基础偏弱</option>
            <option value="intermediate">有一些基础</option>
            <option value="advanced">基础较好</option>
          </select>
        </label>

        <label>
          <span>每周学习时间</span>
          <input v-model.number="form.weeklyHours" type="number" min="1" max="80" />
        </label>

        <label>
          <span>学习偏好</span>
          <select v-model="form.learningPreference">
            <option value="balanced">均衡推进</option>
            <option value="theory_first">先讲原理</option>
            <option value="project_first">先做项目</option>
            <option value="exam_first">面向考试</option>
          </select>
        </label>
      </div>

      <footer class="study-goal-modal__actions">
        <button type="button" class="secondary-btn" @click="open = false; emit('close')">稍后再说</button>
        <button type="button" class="primary-btn" :disabled="!canSubmit" @click="submit">保存并生成计划</button>
      </footer>
    </div>
  </div>
</template>
