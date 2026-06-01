<script setup>
import { Search } from '@lucide/vue'
import { computed, ref, watch } from 'vue'

import { ALL_COURSE_TITLE } from '../../utils/courseMeta'

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  suggestions: {
    type: Array,
    default: () => [],
  },
  courses: {
    type: Array,
    default: () => [ALL_COURSE_TITLE],
  },
  courseFilter: {
    type: String,
    default: ALL_COURSE_TITLE,
  },
})

const emit = defineEmits(['update:modelValue', 'select', 'update:courseFilter'])
const inputRef = ref(null)
const focused = ref(false)
let blurTimer = null

const showSuggestions = computed(() => focused.value && props.modelValue && props.suggestions.length)

function focusInput() {
  inputRef.value?.focus()
}

function handleKeydown(event) {
  if (event.key === 'Enter' && props.suggestions[0]) {
    emit('select', props.suggestions[0])
    focused.value = false
  }
}

function handleBlur() {
  blurTimer = window.setTimeout(() => {
    focused.value = false
  }, 120)
}

function handleSuggestionMouseDown(item) {
  if (blurTimer) {
    window.clearTimeout(blurTimer)
  }
  emit('select', item)
  focused.value = false
}

watch(
  () => props.modelValue,
  (value) => {
    if (!value) focused.value = false
  },
)

defineExpose({ focusInput })
</script>

<template>
  <div class="search-bar-wrap">
    <div class="search-bar">
      <Search class="search-bar__icon" :size="18" aria-hidden="true" />
      <input
        ref="inputRef"
        :value="modelValue"
        type="text"
        placeholder="搜索课程、知识点或主题"
        @focus="focused = true"
        @blur="handleBlur"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown="handleKeydown"
      />
      <div class="search-bar__select-wrap">
        <select class="search-bar__select" :value="courseFilter" @change="emit('update:courseFilter', $event.target.value)">
          <option v-for="course in courses" :key="course" :value="course">{{ course }}</option>
        </select>
      </div>
    </div>

    <div v-if="showSuggestions" class="search-suggestions">
      <button
        v-for="item in suggestions"
        :key="item.id"
        type="button"
        class="search-suggestion"
        @mousedown.prevent="handleSuggestionMouseDown(item)"
      >
        <strong>{{ item.name }}</strong>
        <span>{{ item.chapterTitle }}</span>
      </button>
    </div>
  </div>
</template>
