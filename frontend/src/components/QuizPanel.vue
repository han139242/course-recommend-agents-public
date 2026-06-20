<script setup>
import { CheckCircle2, RefreshCcw, Send, Sparkles } from '@lucide/vue'
import { computed, ref } from 'vue'

import { generateQuiz, recordLearningEvent } from '../api/resources'
import { recordKnowledgePractice } from '../api/learning'

const props = defineProps({
  knowledgeId: { type: String, default: '' },
  knowledgeName: { type: String, default: '' },
  courseTitle: { type: String, default: '' },
})

const emit = defineEmits(['completed'])

const quiz = ref(null)
const loading = ref(false)
const error = ref('')
const answers = ref({})
const submitted = ref(false)
const result = ref(null)

const questions = computed(() => quiz.value?.questions || [])
const hasQuiz = computed(() => questions.value.length > 0)
const allAnswered = computed(() => questions.value.length > 0 && questions.value.every((_, index) => answers.value[index] !== undefined))

async function handleGenerate() {
  const knowledgeName = (props.knowledgeName || '').trim()
  if (!knowledgeName) {
    error.value = '请先选择或填写知识点。'
    return
  }
  loading.value = true
  error.value = ''
  submitted.value = false
  result.value = null
  answers.value = {}
  try {
    quiz.value = await generateQuiz({
      knowledgeId: props.knowledgeId,
      knowledgeName,
      courseTitle: props.courseTitle,
      count: 3,
    })
  } catch (err) {
    error.value = err?.message || '练习题生成失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}

function selectOption(questionIndex, optionIndex) {
  if (submitted.value) return
  answers.value = { ...answers.value, [questionIndex]: optionIndex }
}

function optionClass(questionIndex, optionIndex) {
  if (!submitted.value) {
    return { 'quiz-option--picked': answers.value[questionIndex] === optionIndex }
  }
  const correct = questions.value[questionIndex].answer
  if (optionIndex === correct) return { 'quiz-option--correct': true }
  if (answers.value[questionIndex] === optionIndex) return { 'quiz-option--wrong': true }
  return {}
}

// handleSubmit 本地判分，并把每题结果以 practice_answered 事件回写，
// 驱动画像“易错点偏好”与学习效果评估更新，形成测评闭环。
async function handleSubmit() {
  if (!allAnswered.value || submitted.value) return
  let correct = 0
  questions.value.forEach((question, index) => {
    if (answers.value[index] === question.answer) correct += 1
  })
  submitted.value = true
  result.value = { total: questions.value.length, correct }

  // 逐题回写学习行为事件（驱动画像易错点）与知识点掌握度（驱动能力雷达/推荐）。
  await Promise.allSettled([
    ...questions.value.map((question, index) =>
      recordLearningEvent({
        eventType: 'practice_answered',
        knowledgeId: props.knowledgeId,
        knowledgeName: props.knowledgeName,
        resourceType: 'practice_questions',
        payload: { correct: answers.value[index] === question.answer },
      }),
    ),
    ...(props.knowledgeId
      ? questions.value.map((question, index) =>
          recordKnowledgePractice({
            knowledgeId: props.knowledgeId,
            correct: answers.value[index] === question.answer,
          }),
        )
      : []),
  ])
  emit('completed', result.value)
}

function reset() {
  quiz.value = null
  answers.value = {}
  submitted.value = false
  result.value = null
  error.value = ''
}

defineExpose({ reset })
</script>

<template>
  <section class="quiz-panel">
    <div class="quiz-panel__head">
      <div>
        <strong>随堂练习</strong>
        <small>用 3 道单选题自测，答题结果会回写画像与学习效果评估</small>
      </div>
      <button type="button" class="quiz-btn quiz-btn--primary" :disabled="loading" @click="handleGenerate">
        <Sparkles v-if="!loading" :size="15" aria-hidden="true" />
        <RefreshCcw v-else :size="15" aria-hidden="true" />
        {{ loading ? '出题中' : hasQuiz ? '换一组题' : '生成练习' }}
      </button>
    </div>

    <p v-if="error" class="quiz-error">{{ error }}</p>

    <div v-if="hasQuiz" class="quiz-list">
      <article v-for="(question, index) in questions" :key="index" class="quiz-item">
        <p class="quiz-stem">{{ index + 1 }}. {{ question.stem }}</p>
        <div class="quiz-options">
          <button
            v-for="(option, optionIndex) in question.options"
            :key="optionIndex"
            type="button"
            class="quiz-option"
            :class="optionClass(index, optionIndex)"
            :disabled="submitted"
            @click="selectOption(index, optionIndex)"
          >
            <span class="quiz-option__mark">{{ String.fromCharCode(65 + optionIndex) }}</span>
            <span>{{ option }}</span>
          </button>
        </div>
        <p v-if="submitted" class="quiz-explain">
          <CheckCircle2 :size="14" aria-hidden="true" />
          正确答案：{{ String.fromCharCode(65 + question.answer) }}。{{ question.explanation }}
        </p>
      </article>
    </div>

    <div v-if="hasQuiz" class="quiz-foot">
      <p v-if="submitted && result" class="quiz-score">
        本次得分：{{ result.correct }} / {{ result.total }}
        <span>{{ result.correct === result.total ? '全部答对，可以推进下一知识点。' : '错题已记录，画像会强化对应易错点。' }}</span>
      </p>
      <button v-else type="button" class="quiz-btn quiz-btn--primary" :disabled="!allAnswered" @click="handleSubmit">
        <Send :size="15" aria-hidden="true" />
        提交答案
      </button>
    </div>
  </section>
</template>

<style scoped>
.quiz-panel {
  display: grid;
  gap: 0.75rem;
  border: 1px solid rgba(33, 95, 154, 0.16);
  border-radius: 8px;
  padding: 0.9rem;
  background: #f8fbfd;
}

.quiz-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.quiz-panel__head strong {
  display: block;
}

.quiz-panel__head small {
  color: #64748b;
}

.quiz-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  min-height: 2.2rem;
  padding: 0 0.85rem;
  border: 0;
  border-radius: 8px;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.quiz-btn--primary {
  color: #ffffff;
  background: #215f9a;
}

.quiz-btn:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.quiz-error {
  margin: 0;
  border-radius: 8px;
  padding: 0.6rem 0.7rem;
  color: #8f1d1d;
  background: #fff0f0;
}

.quiz-list {
  display: grid;
  gap: 0.8rem;
}

.quiz-item {
  display: grid;
  gap: 0.5rem;
  border: 1px solid rgba(90, 112, 132, 0.14);
  border-radius: 8px;
  padding: 0.75rem;
  background: #ffffff;
}

.quiz-stem {
  margin: 0;
  font-weight: 700;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.quiz-options {
  display: grid;
  gap: 0.45rem;
}

.quiz-option {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  width: 100%;
  border: 1px solid rgba(90, 112, 132, 0.2);
  border-radius: 8px;
  padding: 0.55rem 0.65rem;
  color: #334155;
  background: #ffffff;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.quiz-option:disabled {
  cursor: default;
}

.quiz-option__mark {
  display: grid;
  place-items: center;
  width: 1.4rem;
  height: 1.4rem;
  flex: 0 0 auto;
  border-radius: 999px;
  background: #eef6fb;
  color: #176b87;
  font-size: 0.8rem;
  font-weight: 800;
}

.quiz-option--picked {
  border-color: rgba(33, 95, 154, 0.55);
  background: #eff8ff;
}

.quiz-option--correct {
  border-color: rgba(31, 138, 112, 0.6);
  background: #eaf8f0;
}

.quiz-option--wrong {
  border-color: rgba(185, 28, 28, 0.5);
  background: #fff0f0;
}

.quiz-explain {
  display: flex;
  align-items: flex-start;
  gap: 0.35rem;
  margin: 0;
  color: #176b87;
  font-size: 0.85rem;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.quiz-foot {
  display: flex;
  justify-content: flex-end;
}

.quiz-score {
  margin: 0;
  font-weight: 800;
  color: #176b87;
}

.quiz-score span {
  display: block;
  margin-top: 0.2rem;
  color: #64748b;
  font-weight: 600;
}
</style>
