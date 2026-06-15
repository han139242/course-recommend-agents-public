<script setup>
import { computed } from 'vue'
import { Bot, Compass, Highlighter, PlayCircle, SlidersHorizontal } from '@lucide/vue'

const props = defineProps({
  authenticated: Boolean,
  profile: {
    type: Object,
    default: null,
  },
  plan: {
    type: Object,
    default: null,
  },
  loading: Boolean,
})

const emit = defineEmits(['ask-ai', 'edit-goal', 'generate', 'highlight-path', 'start'])

const hasProfile = computed(() => Boolean(props.profile))
const hasPlan = computed(() => Boolean(props.plan))
const nextStep = computed(() => props.plan?.nextSteps?.[0] || null)
const routeIds = computed(() =>
  (props.plan?.nextSteps || props.plan?.stages?.flatMap((stage) => stage.items || []) || [])
    .map((item) => item.knowledgeId || item.title)
    .filter(Boolean),
)

const titleText = computed(() => {
  if (!props.authenticated) return '先说出你想完成什么'
  if (!hasProfile.value) return '先定目标，再看图谱'
  if (!hasPlan.value) return '目标已保存，生成路线'
  return props.plan?.summary || '目标学习计划'
})

const helperText = computed(() => {
  if (!props.authenticated) return '登录后保存目标画像'
  if (!hasProfile.value) return '从“下一步能做什么”开始'
  if (!hasPlan.value) return '生成基础、核心、应用三阶段'
  if (nextStep.value) return `下一步：${nextStep.value.title}`
  return '按目标路线推进'
})

function getStageItems(stage) {
  return (stage?.items || []).slice(0, 3)
}

function stageRouteIds(stage) {
  return getStageItems(stage)
    .map((item) => item.knowledgeId || item.title)
    .filter(Boolean)
}

function stageSummary(stage) {
  const items = getStageItems(stage)
  if (!items.length) return '待生成'

  const count = items.length
  const total = (stage?.items || []).length

  // 显示知识点数量
  if (total > count) {
    return `${count} 个知识点（共 ${total} 个）`
  }
  return `${count} 个知识点`
}
</script>

<template>
  <section class="study-plan-card" aria-label="目标学习计划">
    <div class="study-plan-card__main">
      <span class="study-plan-card__eyebrow">Start</span>
      <div class="study-plan-card__copy">
        <strong>{{ titleText }}</strong>
        <span>{{ helperText }}</span>
      </div>
    </div>

    <div v-if="hasPlan" class="study-plan-card__stages" aria-label="三阶段摘要">
      <button
        v-for="stage in plan.stages || []"
        :key="stage.key"
        type="button"
        class="study-plan-stage"
        @click="emit('highlight-path', stageRouteIds(stage))"
      >
        <strong>{{ stage.title }}</strong>
        <span>{{ stageSummary(stage) }}</span>
      </button>
    </div>

    <div class="study-plan-card__actions">
      <button v-if="authenticated && !hasProfile" type="button" class="primary-btn" @click="emit('edit-goal')">
        <Compass :size="16" aria-hidden="true" />
        制定目标
      </button>
      <button v-else-if="authenticated && !hasPlan" type="button" class="primary-btn" :disabled="loading" @click="emit('generate')">
        <PlayCircle :size="16" aria-hidden="true" />
        生成计划
      </button>
      <button v-else-if="authenticated && hasPlan" type="button" class="primary-btn" :disabled="!routeIds.length" @click="emit('start', routeIds)">
        <PlayCircle :size="16" aria-hidden="true" />
        开始
      </button>
      <button v-if="hasPlan" type="button" class="secondary-btn" :disabled="!routeIds.length" @click="emit('highlight-path', routeIds)">
        <Highlighter :size="16" aria-hidden="true" />
        高亮
      </button>
      <button v-if="authenticated && hasProfile" type="button" class="secondary-btn" @click="emit('edit-goal')">
        <SlidersHorizontal :size="16" aria-hidden="true" />
        调整
      </button>
      <button type="button" class="secondary-btn" @click="emit('ask-ai')">
        <Bot :size="16" aria-hidden="true" />
        告诉 AI
      </button>
    </div>
  </section>
</template>
