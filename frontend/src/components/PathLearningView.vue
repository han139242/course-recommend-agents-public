<script setup>
import { computed, ref } from 'vue'
import { X, ChevronRight } from '@lucide/vue'

const props = defineProps({
  pathNodes: {
    type: Array,
    default: () => [],
  },
  pathName: {
    type: String,
    default: '学习路径',
  },
  nodes: {
    type: Array,
    default: () => [],
  },
  progressMap: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['close', 'select-node'])

const nodeLookup = computed(() => {
  const map = new Map()
  props.nodes.forEach((node) => {
    const id = node.routeKnowledgeId || node.id
    if (id) {
      map.set(id, node)
    }
  })
  return map
})

const pathItems = computed(() => {
  return props.pathNodes.map((nodeId, index) => {
    const node = nodeLookup.value.get(nodeId)
    const progress = props.progressMap[nodeId] || {}
    const mastery = Number(progress.masteryScore || 0)

    return {
      id: nodeId,
      index: index + 1,
      name: node?.canonicalName || node?.name || nodeId,
      course: node?.courseTitle || node?.chapterTitle || '',
      mastery: mastery,
      status: mastery >= 80 ? 'mastered' : mastery >= 40 ? 'learning' : 'pending',
      statusLabel: mastery >= 80 ? '已掌握' : mastery >= 40 ? '学习中' : '待学习',
    }
  })
})

const overallProgress = computed(() => {
  const total = pathItems.value.length
  const completed = pathItems.value.filter((item) => item.mastery >= 80).length
  return { completed, total, percentage: total > 0 ? Math.round((completed / total) * 100) : 0 }
})

function handleNodeClick(nodeId) {
  emit('select-node', nodeId)
}
</script>

<template>
  <div class="path-learning-view">
    <div class="path-learning-header">
      <div class="path-learning-header__content">
        <h2>{{ pathName }}</h2>
        <div class="path-learning-stats">
          <span class="stat-item">共 {{ overallProgress.total }} 个知识点</span>
          <span class="stat-item">已掌握 {{ overallProgress.completed }} 个</span>
        </div>
      </div>
      <button type="button" class="icon-btn" title="关闭" @click="emit('close')">
        <X :size="20" aria-hidden="true" />
      </button>
    </div>

    <div class="path-learning-container">
      <!-- 左侧信息栏 -->
      <aside class="path-learning-sidebar">
        <div class="sidebar-section">
          <h3 class="sidebar-title">学习进度</h3>
          <div class="progress-stats">
            <div class="progress-circle">
              <svg viewBox="0 0 100 100">
                <defs>
                  <linearGradient id="progressGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" style="stop-color:#5eead4;stop-opacity:1" />
                    <stop offset="100%" style="stop-color:#22c55e;stop-opacity:1" />
                  </linearGradient>
                </defs>
                <circle cx="50" cy="50" r="45" class="progress-bg" />
                <circle
                  cx="50"
                  cy="50"
                  r="45"
                  class="progress-fill"
                  :style="{ strokeDashoffset: 283 - (283 * overallProgress.percentage) / 100 }"
                />
              </svg>
              <div class="progress-label">{{ overallProgress.percentage }}%</div>
            </div>
            <div class="progress-details">
              <div class="stat-row">
                <span class="stat-label">已完成</span>
                <span class="stat-value">{{ overallProgress.completed }}</span>
              </div>
              <div class="stat-row">
                <span class="stat-label">待学习</span>
                <span class="stat-value">{{ overallProgress.total - overallProgress.completed }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="sidebar-section">
          <h3 class="sidebar-title">学习统计</h3>
          <div class="stat-list">
            <div class="stat-item-card stat-item-card--mastered">
              <span class="stat-item-label">已掌握</span>
              <span class="stat-item-count">{{ pathItems.filter(i => i.status === 'mastered').length }}</span>
            </div>
            <div class="stat-item-card stat-item-card--learning">
              <span class="stat-item-label">学习中</span>
              <span class="stat-item-count">{{ pathItems.filter(i => i.status === 'learning').length }}</span>
            </div>
            <div class="stat-item-card stat-item-card--pending">
              <span class="stat-item-label">待学习</span>
              <span class="stat-item-count">{{ pathItems.filter(i => i.status === 'pending').length }}</span>
            </div>
          </div>
        </div>

        <div class="sidebar-section">
          <h3 class="sidebar-title">快捷筛选</h3>
          <div class="filter-buttons">
            <button type="button" class="filter-btn">全部</button>
            <button type="button" class="filter-btn">待学习</button>
            <button type="button" class="filter-btn">已掌握</button>
          </div>
        </div>
      </aside>

      <!-- 右侧卡片网格 -->
      <div class="path-learning-main">
        <div class="path-learning-grid">
          <article
            v-for="item in pathItems"
            :key="item.id"
            class="path-node-card"
            :class="`path-node-card--${item.status}`"
            @click="handleNodeClick(item.id)"
          >
            <div class="card-index">{{ item.index }}</div>
            <div class="card-content">
              <h4 class="card-title">{{ item.name }}</h4>
              <span class="card-course">{{ item.course }}</span>
            </div>
            <div class="card-footer">
              <div class="card-progress">
                <div class="card-progress-bar">
                  <div class="card-progress-fill" :style="{ width: `${item.mastery}%` }"></div>
                </div>
                <span class="card-progress-text">{{ Math.round(item.mastery) }}%</span>
              </div>
              <span class="card-badge" :class="`card-badge--${item.status}`">{{ item.statusLabel }}</span>
            </div>
          </article>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.path-learning-view {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 200;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
}

.path-learning-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 24px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border-subtle);
}

.path-learning-header__content h2 {
  font-size: 1.4rem;
  font-weight: 800;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.path-learning-stats {
  display: flex;
  gap: 16px;
}

.stat-item {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.path-learning-container {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 左侧边栏 */
.path-learning-sidebar {
  width: 280px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sidebar-title {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.progress-stats {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.progress-circle {
  position: relative;
  width: 120px;
  height: 120px;
}

.progress-circle svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.progress-bg {
  fill: none;
  stroke: rgba(255, 255, 255, 0.08);
  stroke-width: 8;
}

.progress-fill {
  fill: none;
  stroke: url(#progressGradient);
  stroke-width: 8;
  stroke-linecap: round;
  stroke-dasharray: 283;
  transition: stroke-dashoffset 0.5s ease;
}

.progress-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--accent-light);
}

.progress-details {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: var(--radius-md);
}

.stat-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.stat-value {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-primary);
}

.stat-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-item-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-radius: var(--radius-md);
  border-left: 3px solid;
}

.stat-item-card--mastered {
  background: rgba(34, 197, 94, 0.1);
  border-color: var(--success);
}

.stat-item-card--learning {
  background: rgba(56, 189, 248, 0.1);
  border-color: var(--info);
}

.stat-item-card--pending {
  background: rgba(148, 163, 184, 0.1);
  border-color: var(--text-muted);
}

.stat-item-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.stat-item-count {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.filter-buttons {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-btn {
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.045);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-btn:hover {
  background: var(--accent-glow);
  border-color: var(--accent);
  color: var(--accent-light);
}

/* 主内容区 */
.path-learning-main {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: linear-gradient(180deg, var(--bg-primary) 0%, var(--bg-secondary) 100%);
}

.path-learning-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  max-width: 1600px;
  margin: 0 auto;
}

/* 紧凑的卡片样式 */
.path-node-card {
  position: relative;
  padding: 20px;
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 160px;
}

.path-node-card:hover {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(45, 212, 191, 0.15);
}

.path-node-card--mastered {
  border-left: 3px solid var(--success);
}

.path-node-card--learning {
  border-left: 3px solid var(--info);
}

.path-node-card--pending {
  border-left: 3px solid var(--text-muted);
}

.card-index {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-glow);
  color: var(--accent-light);
  font-weight: 700;
  font-size: 0.85rem;
  border-radius: var(--radius-full);
}

.card-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 48px;
}

.card-title {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  line-height: 1.5;
}

.card-course {
  font-size: 0.8rem;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-progress {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-progress-bar {
  flex: 1;
  height: 5px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.card-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #5eead4, #22c55e);
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

.card-progress-text {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  min-width: 36px;
  text-align: right;
}

.card-badge {
  padding: 4px 10px;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: var(--radius-md);
  white-space: nowrap;
}

.card-badge--mastered {
  background: rgba(34, 197, 94, 0.15);
  color: var(--success);
}

.card-badge--learning {
  background: rgba(56, 189, 248, 0.15);
  color: var(--info);
}

.card-badge--pending {
  background: rgba(148, 163, 184, 0.15);
  color: var(--text-muted);
}
</style>
