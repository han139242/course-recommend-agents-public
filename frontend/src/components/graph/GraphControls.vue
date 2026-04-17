<script setup>
defineProps({
  zoom: {
    type: Number,
    default: 1,
  },
})

const emit = defineEmits(['fit', 'zoom-in', 'zoom-out', 'fullscreen'])
</script>

<template>
  <div class="graph-controls">
    <button type="button" class="graph-control-btn" @click="emit('zoom-out')">-</button>
    <button type="button" class="graph-control-btn graph-control-btn--value" @click="emit('fit')">
      {{ Math.round(zoom * 100) }}%
    </button>
    <button type="button" class="graph-control-btn" @click="emit('zoom-in')">+</button>
    <button type="button" class="graph-control-btn" @click="emit('fullscreen')">全屏</button>
  </div>
</template>
