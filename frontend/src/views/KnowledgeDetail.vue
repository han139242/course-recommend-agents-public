<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { domains } from '../data/courseData'
import { knowledgeDetails, graphNodes, graphEdges, growthRadarAxes } from '../data/learning'
import { fetchCompetencyProfile, fetchGraphSubgraph, fetchKnowledgeDetail } from '../auth'
import { edgeSvgCoords, layoutNeoSubgraph, resolveKnowledgeRouteId } from '../utils/neoSubgraphLayout'
import { useAiChat, generateLearningContent, imageFileToDataUrl } from '../composables/useAiChat'
import { renderLatex } from '../composables/useKatex'
import 'katex/dist/katex.min.css'

/** 与 Vite 代理调用的 DeepSeek 模型一致，仅用于界面说明（不含密钥） */
const aiContentModelLabel =
  import.meta.env.VITE_AI_MODEL_LABEL || 'DeepSeek Chat（deepseek-chat）'

const route = useRoute()
const knowledgeId = computed(() => String(route.params.knowledgeId || ''))

// ─── Data Resolution ───

const currentPoint = computed(() => {
  for (const d of domains) {
    const p = d.points.find((pt) => pt.id === knowledgeId.value)
    if (p) return { ...p, domain: d.name, domainId: d.id }
  }
  return null
})

const detailSnapshot = ref(null)
const competencySnapshot = ref(null)

const detailData = computed(() => detailSnapshot.value || knowledgeDetails[knowledgeId.value] || null)

const currentMastery = computed(() => {
  const score = Number(detailData.value?.masteryScore)
  return Number.isFinite(score) ? score : (currentPoint.value?.mastery ?? 0)
})

const displayDescription = computed(() => {
  return detailData.value?.description || currentPoint.value?.description || ''
})

const prerequisitePoints = computed(() => {
  if (!detailData.value?.prerequisites) return []
  return detailData.value.prerequisites.map((name) => {
    for (const d of domains) {
      const p = d.points.find((pt) => pt.name === name || pt.name.includes(name))
      if (p) return { ...p, domain: d.name }
    }
    return { name, mastery: 0, domain: '未收录' }
  })
})

const nextSteps = computed(() => {
  if (!currentPoint.value) return []
  const currentDomain = domains.find((d) => d.id === currentPoint.value.domainId)
  if (!currentDomain) return []
  return currentDomain.points
    .filter((p) => p.id !== knowledgeId.value && p.mastery < 60)
    .slice(0, 3)
})

// ─── Left Column: Local Subgraph ───

const subgraphNodes = computed(() => {
  const id = knowledgeId.value
  const neighborIds = new Set()
  neighborIds.add(id)
  for (const e of graphEdges) {
    if (e.from === id) neighborIds.add(e.to)
    if (e.to === id) neighborIds.add(e.from)
  }
  // also add 2-hop neighbors
  const firstHop = [...neighborIds]
  for (const nid of firstHop) {
    for (const e of graphEdges) {
      if (e.from === nid) neighborIds.add(e.to)
      if (e.to === nid) neighborIds.add(e.from)
    }
  }
  return graphNodes.filter((n) => neighborIds.has(n.id))
})

const subgraphEdges = computed(() => {
  const nodeIds = new Set(subgraphNodes.value.map((n) => n.id))
  return graphEdges.filter((e) => nodeIds.has(e.from) && nodeIds.has(e.to))
})

/** 演示用 graphNodes 只含少量节点；未收录的知识点无法画出子图 */
const domainPeersForNav = computed(() => {
  if (!currentPoint.value) return []
  const d = domains.find((x) => x.id === currentPoint.value.domainId)
  if (!d) return []
  return d.points.filter((p) => p.id !== knowledgeId.value)
})

// ─── Neo4j 子图（优先）───

const neoGraphData = ref(null)
const neoLoading = ref(false)
const neoError = ref('')

const neoLayout = computed(() => {
  if (!neoGraphData.value?.nodes?.length) return null
  return layoutNeoSubgraph(neoGraphData.value, knowledgeId.value)
})

async function loadNeoSubgraph() {
  const id = knowledgeId.value
  if (!id) return
  neoLoading.value = true
  neoError.value = ''
  neoGraphData.value = null
  try {
    const data = await fetchGraphSubgraph(id, 14)
    if (!data.root && data.nodes?.length) {
      data.root = data.nodes.find((n) => n.id === id) || data.nodes[0]
    }
    neoGraphData.value = data
  } catch (e) {
    neoError.value = e.message || '子图加载失败'
    neoGraphData.value = null
  } finally {
    neoLoading.value = false
  }
}

function neoEdgeCoords(edge) {
  const m = neoLayout.value?.nodeMap || {}
  return edgeSvgCoords(edge, m)
}

function sgX(pct) { return (pct / 100) * 220 + 10 }
function sgY(pct) { return (pct / 100) * 180 + 10 }

// ─── Middle Column: AI-Generated Content ───

const aiContent = ref('')
const contentLoading = ref(false)
const contentError = ref('')

async function loadContent() {
  const pt = currentPoint.value
  const detail = detailData.value
  if (!pt) return

  contentLoading.value = true
  contentError.value = ''
  aiContent.value = ''

  try {
    aiContent.value = await generateLearningContent({
      id: pt.id,
      name: pt.name,
      domain: pt.domain,
      levelLabel: pt.levelLabel,
      description: displayDescription.value,
      mastery: currentMastery.value,
      prerequisites: detail?.prerequisites || [],
      related: detail?.related || [],
      scenarios: detail?.scenarios || [],
    })
  } catch (err) {
    contentError.value = err.message || '生成学习内容失败'
  } finally {
    contentLoading.value = false
  }
}

// ─── Right Column: AI Chat ───

const chatRef = ref(null)
const draft = ref('')
const fileInput = ref(null)
const pendingImage = ref('')
let chatInstance = null
const chatMessages = ref([])
const chatLoading = ref(false)

function initChat() {
  const ctx = currentPoint.value
  const detail = detailData.value
  chatInstance = useAiChat({
    knowledgeContext: {
      name: ctx?.name || knowledgeId.value,
      domain: ctx?.domain || '',
      levelLabel: ctx?.levelLabel || '',
      description: detail?.description || ctx?.description || detail?.summary || '',
      mastery: currentMastery.value,
      prerequisites: detail?.prerequisites || [],
      related: detail?.related || [],
      scenarios: detail?.scenarios || [],
    },
  })
  chatMessages.value = chatInstance.messages.value
  chatLoading.value = chatInstance.loading.value
}

const suggestedQuestions = computed(() => {
  const name = currentPoint.value?.name || '这个知识点'
  return [
    `${name}的核心思想是什么？`,
    `和先修知识有什么关系？`,
    `实际中怎么用？`,
  ]
})

async function handleSend(text) {
  const input = typeof text === 'string' ? text : draft.value
  if (!input.trim() && !pendingImage.value) return
  const img = pendingImage.value
  draft.value = ''
  pendingImage.value = ''
  await chatInstance.send(input, { imageDataUrl: img || undefined })
  chatMessages.value = chatInstance.messages.value
  chatLoading.value = chatInstance.loading.value
  await nextTick()
  if (chatRef.value) chatRef.value.scrollTop = chatRef.value.scrollHeight
}

watch(() => chatInstance?.messages?.value, (v) => { if (v) chatMessages.value = v }, { deep: true })
watch(() => chatInstance?.loading?.value, (v) => { if (v != null) chatLoading.value = v })

// ─── Lifecycle ───

async function loadKnowledgeDetail() {
  if (!knowledgeId.value) return
  try {
    detailSnapshot.value = await fetchKnowledgeDetail(knowledgeId.value)
  } catch {
    detailSnapshot.value = knowledgeDetails[knowledgeId.value] || null
  }
}

async function loadCompetencySnapshot() {
  try {
    competencySnapshot.value = await fetchCompetencyProfile()
  } catch {
    competencySnapshot.value = null
  }
}

watch(knowledgeId, async () => {
  await loadKnowledgeDetail()
  await loadCompetencySnapshot()
  initChat()
  loadContent()
  loadNeoSubgraph()
  draft.value = ''
  pendingImage.value = ''
}, { immediate: true })

function triggerImagePick() {
  fileInput.value?.click()
}

async function onImageSelected(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file || !file.type.startsWith('image/')) return
  if (file.size > 6 * 1024 * 1024) {
    window.alert('图片请小于 6MB')
    return
  }
  try {
    pendingImage.value = await imageFileToDataUrl(file)
  } catch (err) {
    window.alert(err.message || '图片处理失败')
  }
}

function clearPendingImage() {
  pendingImage.value = ''
}

// ─── Radar Chart for Ability Profile ───

const radarAxes = computed(() => competencySnapshot.value?.radarAxes?.length
  ? competencySnapshot.value.radarAxes
  : growthRadarAxes)
const radarCenter = { x: 90, y: 90 }
const radarRadius = 70

function radarPoint(index, value, total) {
  const angle = (Math.PI * 2 * index) / total - Math.PI / 2
  const r = (value / 100) * radarRadius
  return { x: radarCenter.x + r * Math.cos(angle), y: radarCenter.y + r * Math.sin(angle) }
}

const radarPolygon = computed(() => {
  return radarAxes.value
    .map((a, i) => {
      const p = radarPoint(i, a.value, radarAxes.value.length)
      return `${p.x},${p.y}`
    })
    .join(' ')
})

const radarLabelPositions = computed(() => {
  return radarAxes.value.map((a, i) => {
    const angle = (Math.PI * 2 * i) / radarAxes.value.length - Math.PI / 2
    const r = radarRadius + 18
    return {
      name: a.name,
      value: a.value,
      x: radarCenter.x + r * Math.cos(angle),
      y: radarCenter.y + r * Math.sin(angle),
    }
  })
})

// ─── Helpers ───

function renderMarkdown(md) {
  if (!md) return ''
  let html = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/^## (.+)$/gm, '<h3 class="ai-section-title">$1</h3>')
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/^- (.+)$/gm, '<li>$1</li>')
    .replace(/(<li>.*<\/li>\n?)+/g, (m) => `<ul>${m}</ul>`)
    .replace(/\n{2,}/g, '</p><p>')
    .replace(/\n/g, '<br>')
    .replace(/^/, '<p>')
    .replace(/$/, '</p>')
  return renderLatex(html)
}

function formatChat(text) {
  if (!text) return ''
  let html = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
  return renderLatex(html)
}
</script>

<template>
  <section class="kd-page" v-if="currentPoint">
    <div class="kd-three-col">

      <!-- ═══ LEFT: Graph Navigation + Ability Profile ═══ -->
      <aside class="kd-left">
        <!-- Ability Radar -->
        <div class="kd-card">
          <span class="section-tag">我的能力画像</span>
          <div class="kd-radar-wrap">
            <svg viewBox="0 0 180 180" class="kd-radar-svg">
              <polygon
                v-for="ring in [100, 75, 50, 25]"
                :key="ring"
                :points="radarAxes.map((_, i) => {
                  const p = radarPoint(i, ring, radarAxes.length)
                  return `${p.x},${p.y}`
                }).join(' ')"
                class="kd-radar-ring"
              />
              <line
                v-for="(_, i) in radarAxes"
                :key="'axis' + i"
                :x1="radarCenter.x"
                :y1="radarCenter.y"
                :x2="radarPoint(i, 100, radarAxes.length).x"
                :y2="radarPoint(i, 100, radarAxes.length).y"
                class="kd-radar-axis"
              />
              <polygon :points="radarPolygon" class="kd-radar-area" />
              <circle
                v-for="(a, i) in radarAxes"
                :key="'dot' + i"
                :cx="radarPoint(i, a.value, radarAxes.length).x"
                :cy="radarPoint(i, a.value, radarAxes.length).y"
                r="3"
                class="kd-radar-dot"
              />
              <text
                v-for="lp in radarLabelPositions"
                :key="lp.name"
                :x="lp.x"
                :y="lp.y"
                text-anchor="middle"
                dominant-baseline="central"
                class="kd-radar-label"
              >{{ lp.name }} {{ lp.value }}%</text>
            </svg>
          </div>
        </div>

        <div class="kd-card">
          <span class="section-tag">知识图谱导航</span>
          <p v-if="neoLoading" class="kd-neo-hint">正在从 Neo4j 加载子图…</p>
          <p v-else-if="neoError" class="kd-neo-err">{{ neoError }}</p>

          <!-- Neo4j 子图（与 /graph 页一致的数据源） -->
          <div v-if="!neoLoading && neoLayout?.positionedNodes?.length" class="kd-graph-wrap kd-graph-wrap--neo">
            <svg viewBox="0 0 100 100" class="kd-graph-svg kd-graph-svg--neo" preserveAspectRatio="xMidYMid meet">
              <line
                v-for="edge in neoLayout.edges"
                :key="edge.relationId || `${edge.sourceId}-${edge.targetId}`"
                :x1="neoEdgeCoords(edge).x1"
                :y1="neoEdgeCoords(edge).y1"
                :x2="neoEdgeCoords(edge).x2"
                :y2="neoEdgeCoords(edge).y2"
                class="kd-graph-edge"
              />
              <g v-for="n in neoLayout.positionedNodes" :key="n.id">
                <RouterLink :to="'/knowledge/' + resolveKnowledgeRouteId(n)">
                  <circle
                    :cx="n.x"
                    :cy="n.y"
                    :r="n.id === knowledgeId ? 6 : 4.5"
                    :class="['kd-graph-node-circle', n.level, { current: n.id === knowledgeId }]"
                  />
                  <text
                    :x="n.x"
                    :y="n.y + (n.id === knowledgeId ? 9 : 8)"
                    text-anchor="middle"
                    :class="['kd-graph-node-label', 'kd-graph-node-label--neo', { current: n.id === knowledgeId }]"
                  >{{ n.label }}</text>
                </RouterLink>
              </g>
            </svg>
          </div>

          <!-- 演示子图（Neo4j 无数据或失败时） -->
          <div v-else-if="!neoLoading && subgraphNodes.length" class="kd-graph-wrap">
            <svg viewBox="0 0 240 200" class="kd-graph-svg">
              <line
                v-for="(e, i) in subgraphEdges"
                :key="'e' + i"
                :x1="sgX(graphNodes.find(n => n.id === e.from)?.x || 0)"
                :y1="sgY(graphNodes.find(n => n.id === e.from)?.y || 0)"
                :x2="sgX(graphNodes.find(n => n.id === e.to)?.x || 0)"
                :y2="sgY(graphNodes.find(n => n.id === e.to)?.y || 0)"
                class="kd-graph-edge"
              />
              <g v-for="n in subgraphNodes" :key="n.id">
                <RouterLink :to="'/knowledge/' + n.id">
                  <circle
                    :cx="sgX(n.x)"
                    :cy="sgY(n.y)"
                    :r="n.id === knowledgeId ? 14 : 10"
                    :class="['kd-graph-node-circle', n.level, { current: n.id === knowledgeId }]"
                  />
                  <text
                    :x="sgX(n.x)"
                    :y="sgY(n.y) + (n.id === knowledgeId ? 24 : 20)"
                    text-anchor="middle"
                    :class="['kd-graph-node-label', { current: n.id === knowledgeId }]"
                  >{{ n.label }}</text>
                </RouterLink>
              </g>
            </svg>
          </div>

          <div v-else-if="!neoLoading" class="kd-graph-fallback">
            <p class="kd-graph-fallback-note">
              <template v-if="neoError">Neo4j 子图暂不可用，已切换为演示数据或同域跳转。请确认后端与 Neo4j 已启动。</template>
              <template v-else>当前知识点在演示子图中无节点，且 Neo4j 未返回关联数据。</template>
            </p>
            <p class="kd-graph-fallback-sub">同域其它知识点（可跳转）：</p>
            <div class="kd-peer-links">
              <RouterLink
                v-for="p in domainPeersForNav"
                :key="p.id"
                :to="p.route"
                class="kd-peer-link"
              >{{ p.name }}</RouterLink>
            </div>
          </div>
        </div>

        <div class="kd-card">
          <span class="section-tag">当前知识点</span>
          <div class="kd-pos-info">
            <div class="kd-pos-row">
              <span class="kd-pos-label">所属域</span>
              <span class="kd-pos-value">{{ currentPoint.domain }}</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">等级</span>
              <span :class="['point-level', currentPoint.level]">{{ currentPoint.levelLabel }}</span>
            </div>
            <div class="kd-pos-row">
              <span class="kd-pos-label">掌握度</span>
              <span class="kd-pos-value kd-mastery">{{ currentMastery }}%</span>
            </div>
            <div class="kd-pos-bar">
              <div class="kd-pos-bar-fill" :style="{ width: currentMastery + '%' }"></div>
            </div>
          </div>
        </div>

        <div class="kd-card" v-if="prerequisitePoints.length">
          <span class="section-tag">先修状态</span>
          <div class="kd-prereq-list">
            <div
              v-for="p in prerequisitePoints"
              :key="p.name"
              class="kd-prereq-item"
              :class="{ weak: p.mastery != null && p.mastery < 50 }"
            >
              <span class="kd-prereq-name">{{ p.name }}</span>
              <span v-if="p.mastery != null" :class="['kd-prereq-pct', { low: p.mastery < 50 }]">{{ p.mastery }}%</span>
            </div>
          </div>
        </div>

        <div class="kd-card" v-if="nextSteps.length">
          <span class="section-tag">推荐下一步</span>
          <div class="kd-next-list">
            <RouterLink v-for="s in nextSteps" :key="s.id" :to="s.route" class="kd-next-item">
              <span>{{ s.name }}</span>
              <span class="kd-next-mastery">{{ s.mastery }}%</span>
            </RouterLink>
          </div>
        </div>

        <RouterLink to="/" class="kd-back-link">← 返回首页</RouterLink>
      </aside>

      <!-- ═══ MIDDLE: AI-Generated Content ═══ -->
      <main class="kd-center">
        <div class="kd-center-header">
          <div class="kd-header-top">
            <span :class="['point-level', currentPoint.level]">{{ currentPoint.levelLabel }}</span>
            <span v-if="currentPoint.recommend" class="point-rec">{{ currentPoint.recommend }}</span>
            <span class="kd-domain-badge">{{ currentPoint.domain }}</span>
          </div>
          <h1 class="kd-title">{{ currentPoint.name }}</h1>
          <p class="kd-desc">{{ displayDescription }}</p>
        </div>

        <div class="kd-ai-content">
          <div class="kd-ai-content-head">
            <div class="kd-ai-head-text">
              <span class="section-tag">AI 生成学习内容</span>
              <p class="kd-ai-model-note">
                由 <strong>{{ aiContentModelLabel }}</strong> 根据当前知识点、先修与掌握度实时生成摘要，非预置讲义。
              </p>
            </div>
            <button v-if="!contentLoading" class="kd-refresh-btn" @click="loadContent">重新生成</button>
          </div>

          <div v-if="contentLoading" class="kd-content-loading">
            <div class="kd-spinner"></div>
            <span>AI 正在根据你的掌握情况生成个性化学习内容...</span>
          </div>

          <div v-else-if="contentError" class="kd-content-error">
            <p>{{ contentError }}</p>
            <button class="kd-retry-btn" @click="loadContent">重试</button>
          </div>

          <div v-else class="kd-content-body" v-html="renderMarkdown(aiContent)"></div>
        </div>
      </main>

      <!-- ═══ RIGHT: AI Chat ═══ -->
      <aside class="kd-right">
        <div class="kd-chat-panel">
          <div class="kd-chat-head">
            <span class="section-tag">AI 学伴</span>
          </div>

          <div class="kd-chat-suggest" v-if="chatMessages.length === 0">
            <p class="kd-chat-hint">关于「{{ currentPoint.name }}」，你想了解什么？</p>
            <button
              v-for="q in suggestedQuestions"
              :key="q"
              class="kd-suggest-btn"
              @click="handleSend(q)"
            >{{ q }}</button>
          </div>

          <div class="kd-chat-messages" ref="chatRef">
            <div v-for="(msg, idx) in chatMessages" :key="idx" :class="['kd-msg', msg.role]">
              <div class="kd-msg-avatar">{{ msg.role === 'user' ? '我' : 'AI' }}</div>
              <div class="kd-msg-body">
                <div class="kd-msg-content">
                  <img v-if="msg.role === 'user' && msg.imageUrl" :src="msg.imageUrl" class="kd-msg-image" alt="" />
                  <div v-html="formatChat(msg.content)"></div>
                </div>
              </div>
            </div>
            <div v-if="chatLoading && chatMessages.length > 0 && !chatMessages[chatMessages.length - 1]?.content" class="kd-msg assistant">
              <div class="kd-msg-avatar">AI</div>
              <div class="kd-msg-body"><div class="kd-msg-content kd-typing">思考中...</div></div>
            </div>
          </div>

          <div v-if="pendingImage" class="kd-image-preview">
            <img :src="pendingImage" alt="待发送" />
            <button type="button" class="kd-image-remove" @click="clearPendingImage" aria-label="移除图片">×</button>
          </div>
          <div class="kd-chat-input">
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              capture="environment"
              class="kd-file-input"
              @change="onImageSelected"
            />
            <button
              type="button"
              class="kd-attach-btn"
              title="拍照/上传图片"
              :disabled="chatLoading"
              @click="triggerImagePick"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
            </button>
            <input
              v-model="draft"
              type="text"
              placeholder="输入问题，可配图..."
              @keydown.enter.prevent="handleSend()"
              :disabled="chatLoading"
            />
            <button
              class="kd-send-btn"
              @click="handleSend()"
              :disabled="chatLoading || (!draft.trim() && !pendingImage)"
            >发送</button>
          </div>
        </div>
      </aside>

    </div>
  </section>

  <section class="kd-page" v-else>
    <div class="kd-fallback">
      <h2>未找到该知识点</h2>
      <p>请返回首页选择一个知识点开始学习。</p>
      <RouterLink to="/" class="back-home-link">返回知识学习首页</RouterLink>
    </div>
  </section>
</template>
