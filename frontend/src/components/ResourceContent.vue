<script setup>
import { computed } from 'vue'

import { renderMarkdown, renderMindMap } from '../composables/useMarkdown'

const props = defineProps({
  content: { type: String, default: '' },
  format: { type: String, default: 'markdown' },
})

const isMindMap = computed(
  () => props.format === 'mermaid' || /^\s*mindmap\b/i.test(props.content || ''),
)

const html = computed(() =>
  isMindMap.value ? renderMindMap(props.content) : renderMarkdown(props.content),
)
</script>

<template>
  <!-- Output is built from escaped text with a fixed tag set (see useMarkdown), so v-html is safe. -->
  <div
    class="resource-content"
    :class="{ 'resource-content--mindmap': isMindMap }"
    v-html="html"
  />
</template>

<style scoped>
.resource-content {
  max-height: 34rem;
  overflow: auto;
  padding: 1rem 1.1rem;
  color: #1f2937;
  background: #fbfdff;
  line-height: 1.7;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.resource-content :deep(h1),
.resource-content :deep(h2),
.resource-content :deep(h3),
.resource-content :deep(h4) {
  margin: 1.1rem 0 0.6rem;
  line-height: 1.4;
  color: #14233b;
}

.resource-content :deep(h1) {
  font-size: 1.3rem;
}

.resource-content :deep(h2) {
  font-size: 1.15rem;
  padding-bottom: 0.3rem;
  border-bottom: 1px solid rgba(90, 112, 132, 0.16);
}

.resource-content :deep(h3) {
  font-size: 1.04rem;
}

.resource-content :deep(h4) {
  font-size: 0.96rem;
}

.resource-content :deep(p) {
  margin: 0.55rem 0;
}

.resource-content :deep(ul),
.resource-content :deep(ol) {
  margin: 0.5rem 0;
  padding-left: 1.35rem;
}

.resource-content :deep(li) {
  margin: 0.28rem 0;
}

.resource-content :deep(a) {
  color: #1f6fb2;
  text-decoration: underline;
}

.resource-content :deep(strong) {
  color: #14233b;
}

.resource-content :deep(code) {
  padding: 0.1rem 0.35rem;
  border-radius: 6px;
  background: #eef3f8;
  color: #b4344f;
  font: 0.86em/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

.resource-content :deep(.code-block) {
  margin: 0.7rem 0;
  padding: 0.85rem 1rem;
  overflow: auto;
  border: 1px solid rgba(90, 112, 132, 0.16);
  border-radius: 10px;
  background: #0f1c2e;
}

.resource-content :deep(.code-block code) {
  padding: 0;
  background: transparent;
  color: #e6edf6;
  font: 0.86rem/1.6 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

.resource-content :deep(blockquote) {
  margin: 0.7rem 0;
  padding: 0.5rem 0.9rem;
  border-left: 3px solid #2f80b7;
  border-radius: 0 8px 8px 0;
  background: #eef6fb;
  color: #3b4a60;
}

.resource-content :deep(.md-table) {
  width: 100%;
  margin: 0.7rem 0;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.resource-content :deep(.md-table th),
.resource-content :deep(.md-table td) {
  padding: 0.5rem 0.7rem;
  border: 1px solid rgba(90, 112, 132, 0.2);
  text-align: left;
}

.resource-content :deep(.md-table th) {
  background: #eef3f8;
  color: #14233b;
}

.resource-content :deep(hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid rgba(90, 112, 132, 0.18);
}

/* Mind map: indented tree with connector accents. */
.resource-content--mindmap {
  background: #f7fbfe;
}

.resource-content :deep(.mindmap-tree ul) {
  margin: 0;
  padding-left: 1.1rem;
  list-style: none;
}

.resource-content :deep(.mindmap-tree > ul) {
  padding-left: 0;
}

.resource-content :deep(.mindmap-tree li) {
  position: relative;
  margin: 0.3rem 0;
  padding-left: 0.9rem;
}

.resource-content :deep(.mindmap-tree li::before) {
  content: "";
  position: absolute;
  left: 0;
  top: 0.85rem;
  width: 0.6rem;
  height: 1px;
  background: rgba(47, 128, 183, 0.5);
}

.resource-content :deep(.mindmap-tree > ul > li) {
  padding-left: 0;
}

.resource-content :deep(.mindmap-tree > ul > li::before) {
  display: none;
}

.resource-content :deep(.mindmap-node) {
  display: inline-block;
  padding: 0.28rem 0.65rem;
  border: 1px solid rgba(33, 95, 154, 0.22);
  border-radius: 999px;
  background: #ffffff;
  color: #21405f;
  font-weight: 700;
}

.resource-content :deep(.mindmap-tree > ul > li > .mindmap-node) {
  color: #ffffff;
  background: #215f9a;
  border-color: #215f9a;
}
</style>
