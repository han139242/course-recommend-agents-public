<script setup>
import { computed } from 'vue'

import { renderMarkdown, renderMindMap } from '../composables/useMarkdown'

const props = defineProps({
  content: { type: String, default: '' },
  format: { type: String, default: 'markdown' },
})

const mediaPayload = computed(() => {
  if (!['image', 'video'].includes(props.format)) return null
  const raw = String(props.content || '').trim()
  if (!raw) return {}
  if (/^https?:\/\//i.test(raw) || raw.startsWith('data:')) return { url: raw }
  try {
    return JSON.parse(raw)
  } catch {
    return { prompt: raw }
  }
})

const mediaSrc = computed(() => {
  const payload = mediaPayload.value || {}
  const direct = payload.url || payload.videoUrl || payload.imageUrl
  if (direct) return direct
  const b64 = payload.b64Json || payload.b64_json || payload.base64
  if (!b64) return ''
  return `data:${payload.mimeType || payload.mime_type || (props.format === 'video' ? 'video/mp4' : 'image/png')};base64,${b64}`
})

const isMindMap = computed(
  () => props.format === 'mermaid' || /^\s*mindmap\b/i.test(props.content || ''),
)

const html = computed(() =>
  isMindMap.value ? renderMindMap(props.content) : renderMarkdown(props.content),
)
</script>

<template>
  <div v-if="format === 'image'" class="resource-media resource-media--image">
    <img v-if="mediaSrc" :src="mediaSrc" :alt="mediaPayload?.prompt || 'AI 生成配图'" />
    <p v-else class="resource-media__pending">
      媒体生成任务已提交：{{ mediaPayload?.providerTaskId || mediaPayload?.status || '等待提供商返回结果' }}
    </p>
    <details v-if="mediaPayload?.prompt" class="resource-media__prompt">
      <summary>生成提示词</summary>
      <p>{{ mediaPayload.prompt }}</p>
    </details>
  </div>
  <div v-else-if="format === 'video'" class="resource-media resource-media--video">
    <video v-if="mediaSrc" :src="mediaSrc" controls playsinline />
    <p v-else class="resource-media__pending">
      视频生成任务已提交：{{ mediaPayload?.providerTaskId || mediaPayload?.status || '等待提供商返回结果' }}
    </p>
    <details v-if="mediaPayload?.prompt" class="resource-media__prompt">
      <summary>生成提示词</summary>
      <p>{{ mediaPayload.prompt }}</p>
    </details>
  </div>
  <!-- Output is built from escaped text with a fixed tag set (see useMarkdown), so v-html is safe. -->
  <div
    v-else
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
  color: var(--text-primary);
  background: var(--bg-primary);
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
  color: var(--text-primary);
}

.resource-content :deep(h1) {
  font-size: 1.3rem;
}

.resource-content :deep(h2) {
  font-size: 1.15rem;
  padding-bottom: 0.3rem;
  border-bottom: 1px solid var(--border-subtle);
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
  color: var(--accent-light);
  text-decoration: underline;
}

.resource-content :deep(strong) {
  color: var(--text-primary);
}

.resource-content :deep(code) {
  padding: 0.1rem 0.35rem;
  border-radius: 6px;
  background: var(--bg-hover);
  color: var(--accent-light);
  font: 0.86em/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

.resource-content :deep(.code-block) {
  margin: 0.7rem 0;
  padding: 0.85rem 1rem;
  overflow: auto;
  border: 1px solid var(--border-medium);
  border-radius: 10px;
  background: #070b12;
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
  background: var(--accent-glow);
  color: var(--text-secondary);
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
  border: 1px solid var(--border-medium);
  text-align: left;
}

.resource-content :deep(.md-table th) {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.resource-content :deep(hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid var(--border-subtle);
}

/* Mind map: indented tree with connector accents. */
.resource-content--mindmap {
  background: var(--bg-primary);
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
  border: 1px solid var(--border-medium);
  border-radius: 999px;
  background: var(--bg-panel);
  color: var(--text-primary);
  font-weight: 700;
}

.resource-content :deep(.mindmap-tree > ul > li > .mindmap-node) {
  color: #07110f;
  background: var(--accent);
  border-color: var(--accent);
}

.resource-media {
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--bg-primary);
}

.resource-media--image {
  background-color: var(--bg-primary);
  background-image:
    linear-gradient(45deg, var(--border-subtle) 25%, transparent 25%),
    linear-gradient(-45deg, var(--border-subtle) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, var(--border-subtle) 75%),
    linear-gradient(-45deg, transparent 75%, var(--border-subtle) 75%);
  background-position: 0 0, 0 8px, 8px -8px, -8px 0;
  background-size: 16px 16px;
}

.resource-media img,
.resource-media video {
  width: 100%;
  max-height: 34rem;
  border-radius: 8px;
  object-fit: contain;
  background: #0f1c2e;
}

.resource-media img {
  background: color-mix(in srgb, var(--bg-secondary) 92%, transparent);
}

.resource-media__pending {
  margin: 0;
  border: 1px dashed var(--border-medium);
  border-radius: 8px;
  padding: 0.85rem;
  color: var(--text-secondary);
  background: var(--accent-glow);
}

.resource-media__prompt {
  border-radius: 8px;
  padding: 0.7rem 0.85rem;
  background: var(--bg-hover);
  color: var(--text-secondary);
}

.resource-media__prompt summary {
  cursor: pointer;
  font-weight: 700;
  color: var(--text-primary);
}

.resource-media__prompt p {
  margin: 0.55rem 0 0;
  line-height: 1.6;
}
</style>
