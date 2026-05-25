<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import { useUserStore } from '../stores/userStore'
import { getCoursePalette } from '../utils/colorMap'

const userStore = useUserStore()
const radarCanvas = ref(null)
const ready = ref(false)

const DEMO_PROGRESS_ITEMS = [
  {
    knowledgeId: 'gradient_descent',
    routeKnowledgeId: 'gradient_descent',
    canonicalName: '梯度下降',
    name: '梯度下降',
    chapterTitle: '机器学习基础',
    masteryScore: 88,
    answeredCount: 12,
    correctRate: 0.92,
    status: 'mastered',
    masteryText: '已掌握',
  },
  {
    knowledgeId: 'loss_function',
    routeKnowledgeId: 'loss_function',
    canonicalName: '损失函数',
    name: '损失函数',
    chapterTitle: '机器学习基础',
    masteryScore: 82,
    answeredCount: 9,
    correctRate: 0.89,
    status: 'mastered',
    masteryText: '已掌握',
  },
  {
    knowledgeId: 'matrix_vector',
    routeKnowledgeId: 'matrix_vector',
    canonicalName: '矩阵与向量运算',
    name: '矩阵与向量运算',
    chapterTitle: '数学基础',
    masteryScore: 76,
    answeredCount: 8,
    correctRate: 0.81,
    status: 'learning',
    masteryText: '学习中',
  },
  {
    knowledgeId: 'model_evaluation',
    routeKnowledgeId: 'model_evaluation',
    canonicalName: '模型评估',
    name: '模型评估',
    chapterTitle: '机器学习基础',
    masteryScore: 64,
    answeredCount: 7,
    correctRate: 0.74,
    status: 'learning',
    masteryText: '学习中',
  },
  {
    knowledgeId: 'knowledge_graph',
    routeKnowledgeId: 'knowledge_graph',
    canonicalName: '知识图谱',
    name: '知识图谱',
    chapterTitle: '知识工程',
    masteryScore: 58,
    answeredCount: 6,
    correctRate: 0.67,
    status: 'learning',
    masteryText: '学习中',
  },
  {
    knowledgeId: 'backpropagation',
    routeKnowledgeId: 'backpropagation',
    canonicalName: '反向传播',
    name: '反向传播',
    chapterTitle: '深度学习',
    masteryScore: 47,
    answeredCount: 5,
    correctRate: 0.6,
    status: 'learning',
    masteryText: '学习中',
  },
  {
    knowledgeId: 'attention_mechanism',
    routeKnowledgeId: 'attention_mechanism',
    canonicalName: '注意力机制',
    name: '注意力机制',
    chapterTitle: '大模型基础',
    masteryScore: 43,
    answeredCount: 4,
    correctRate: 0.58,
    status: 'learning',
    masteryText: '学习中',
  },
  {
    knowledgeId: 'overfitting',
    routeKnowledgeId: 'overfitting',
    canonicalName: '过拟合与正则化',
    name: '过拟合与正则化',
    chapterTitle: '模型泛化',
    masteryScore: 28,
    answeredCount: 5,
    correctRate: 0.42,
    status: 'weak',
    masteryText: '待巩固',
  },
]

const DEMO_DASHBOARD_SUMMARY = {
  quickStats: [
    { label: '学习中知识点', value: '5' },
    { label: '已掌握知识点', value: '2' },
    { label: '薄弱知识点', value: '1' },
    { label: '总体掌握进度', value: '63%' },
  ],
  knowledgeCards: [
    {
      id: 'overfitting',
      type: '优先补强',
      progress: 28,
      title: '过拟合与正则化',
      domain: '模型泛化',
      tag: '先区分训练误差和测试误差，再补齐正则化方法。',
      route: '/knowledge/overfitting',
    },
    {
      id: 'attention_mechanism',
      type: '继续推进',
      progress: 43,
      title: '注意力机制',
      domain: '大模型基础',
      tag: '建议结合 Transformer 主线理解权重分配过程。',
      route: '/knowledge/attention_mechanism',
    },
    {
      id: 'backpropagation',
      type: '强化理解',
      progress: 47,
      title: '反向传播',
      domain: '深度学习',
      tag: '把链式法则、损失函数和参数更新连起来复盘。',
      route: '/knowledge/backpropagation',
    },
    {
      id: 'knowledge_graph',
      type: '应用迁移',
      progress: 58,
      title: '知识图谱',
      domain: '知识工程',
      tag: '重点理解节点、关系和路径推荐之间的对应关系。',
      route: '/knowledge/knowledge_graph',
    },
  ],
  recommendedPath: [
    {
      knowledgeId: 'overfitting',
      title: '过拟合与正则化',
      detail: '先补齐泛化能力判断，再学习正则化和数据划分策略。',
      reasonLabel: '薄弱优先',
      stage: '第 1 步',
      score: 0.91,
      route: '/knowledge/overfitting',
    },
    {
      knowledgeId: 'backpropagation',
      title: '反向传播',
      detail: '沿着损失函数、梯度下降和链式法则把训练闭环串起来。',
      reasonLabel: '前置衔接',
      stage: '第 2 步',
      score: 0.84,
      route: '/knowledge/backpropagation',
    },
    {
      knowledgeId: 'attention_mechanism',
      title: '注意力机制',
      detail: '在掌握基础神经网络后，继续进入 Transformer 关键结构。',
      reasonLabel: '路径推进',
      stage: '第 3 步',
      score: 0.78,
      route: '/knowledge/attention_mechanism',
    },
  ],
  weeklyTrend: [
    { label: '周一', value: 36 },
    { label: '周二', value: 48 },
    { label: '周三', value: 42 },
    { label: '周四', value: 63 },
    { label: '周五', value: 57 },
    { label: '周六', value: 72 },
    { label: '周日', value: 66 },
  ],
}

const DEMO_PROFILE = {
  summary: '当前已形成机器学习主线的基础掌握，建议继续补强过拟合、反向传播和注意力机制，把知识点从单点理解推进到路径化应用。',
  goals: [
    '优先补齐“过拟合与正则化”，降低后续模型评估中的理解断点',
    '复盘“损失函数—梯度下降—反向传播”的训练闭环',
    '完成“注意力机制”相关练习，为 Transformer 学习做准备',
  ],
  recentRecords: [
    '完成“梯度下降”练习 12 次，掌握度提升至 88%。',
    '“损失函数”答题正确率稳定在 89%，已进入已掌握状态。',
    '“过拟合与正则化”正确率偏低，已被系统列为优先补强项。',
  ],
}

const DEMO_RADAR_AXES = [
  { name: '数学基础', value: 72 },
  { name: '编程数据', value: 64 },
  { name: '机器学习', value: 81 },
  { name: '深度学习', value: 58 },
  { name: '语言知识', value: 51 },
  { name: '工程治理', value: 66 },
]

const DEMO_HEATMAP_DOMAINS = ['数学基础', '机器学习', '深度学习', '知识工程']
const DEMO_HEATMAP_ROWS = [
  { label: '梯度下降', values: [76, 88, 42, 0] },
  { label: '损失函数', values: [68, 82, 38, 0] },
  { label: '反向传播', values: [54, 47, 62, 0] },
  { label: '注意力机制', values: [36, 43, 58, 28] },
  { label: '知识图谱', values: [0, 34, 22, 58] },
  { label: '过拟合与正则化', values: [42, 28, 24, 0] },
]

const DEMO_GROWTH_INSIGHTS = [
  { label: '机器学习', value: 81, note: '基础训练流程掌握较稳，可以继续向模型泛化和评估迁移。' },
  { label: '数学基础', value: 72, note: '矩阵、梯度和损失函数理解较好，适合作为后续推导支撑。' },
  { label: '深度学习', value: 58, note: '反向传播和注意力机制仍需连续练习，建议放在下一轮重点。' },
]

function hasMeaningfulProgress(item = {}) {
  return Number(item.answeredCount || 0) > 0 || Number(item.masteryScore ?? item.mastery ?? 0) > 0
}

onMounted(async () => {
  await Promise.all([
    userStore.loadProgress().catch(() => null),
    userStore.loadDashboard().catch(() => null),
    userStore.loadCompetency().catch(() => null),
  ])

  window.addEventListener('resize', handleRadarResize)
  requestAnimationFrame(() => { ready.value = true })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleRadarResize)
})

const shouldUseDemoData = computed(() => {
  const list = Array.isArray(userStore.progressList) ? userStore.progressList : []
  return !list.some((item) => hasMeaningfulProgress(item))
})

const dashboardSummary = computed(() => (shouldUseDemoData.value ? DEMO_DASHBOARD_SUMMARY : userStore.dashboardSummary || null))
const quickStats = computed(() => dashboardSummary.value?.quickStats || dashboardSummary.value?.stats || [])
const knowledgeCards = computed(() => dashboardSummary.value?.knowledgeCards || [])
const recommendedPath = computed(() => dashboardSummary.value?.recommendedPath || [])

const profile = computed(() => {
  const current = userStore.competencyProfile?.profile || null
  if (!shouldUseDemoData.value) return current
  return {
    ...(current || {}),
    ...DEMO_PROFILE,
    name: current?.name || userStore.user?.name || userStore.user?.username || '测试学生',
  }
})
const rawRadarAxes = computed(() => {
  const current = userStore.competencyProfile?.radarAxes || []
  if (current.length || !shouldUseDemoData.value) return current
  return DEMO_RADAR_AXES
})
const growthInsights = computed(() => {
  const current = userStore.competencyProfile?.growthInsights || []
  if (current.length || !shouldUseDemoData.value) return current
  return DEMO_GROWTH_INSIGHTS
})
const growthCards = computed(() => userStore.competencyProfile?.growthSummaryCards || [])
const heatmapDomains = computed(() => {
  const current = userStore.competencyProfile?.heatmapDomains || []
  if (current.length || !shouldUseDemoData.value) return current
  return DEMO_HEATMAP_DOMAINS
})
const heatmapRows = computed(() => {
  const current = userStore.competencyProfile?.heatmapRows || []
  if (current.length || !shouldUseDemoData.value) return current
  return DEMO_HEATMAP_ROWS
})

const RADAR_DIMENSION_GROUPS = [
  {
    key: 'math',
    name: '数学基础',
    keywords: ['数学', '代数', '矩阵', '概率', '统计', '微积分', '数值', '最优化', '凸优化', '信息论'],
  },
  {
    key: 'data',
    name: '编程数据',
    keywords: ['python', '编程', '代码', '数据结构', '科学计算', 'numpy', 'pandas', 'sql', 'linux', 'git', '脚本'],
  },
  {
    key: 'ml',
    name: '机器学习',
    keywords: ['机器学习', '回归', '分类', '聚类', 'svm', '决策树', '集成', '降维', '贝叶斯', '强化学习', '推荐'],
  },
  {
    key: 'dl',
    name: '深度学习',
    keywords: ['深度学习', '神经网络', '卷积', '训练', '反向传播', 'transformer', '注意力', '预训练', '微调', '参数高效', '大模型'],
  },
  {
    key: 'vision',
    name: '视觉感知',
    keywords: ['视觉', '图像', '检测', '分割', '识别', 'opencv', '多模态', '感知'],
  },
  {
    key: 'language',
    name: '语言知识',
    keywords: ['自然语言', 'nlp', '文本', '语言模型', '知识图谱', '知识表示', '信息抽取', '语义', '问答', '检索', '图神经'],
  },
  {
    key: 'engineering',
    name: '工程治理',
    keywords: ['工程', '部署', '推理', 'mlops', '服务', '治理', '合规', '伦理', '智能体', '应用', '导论'],
  },
]

function normalizeRadarText(value = '') {
  return String(value || '').trim().toLowerCase()
}

function countKeywordHits(normalized = '', keywords = []) {
  if (!normalized) return 0
  return (Array.isArray(keywords) ? keywords : []).reduce((total, keyword) => {
    const current = normalizeRadarText(keyword)
    if (!current) return total
    return normalized.includes(current) ? total + 1 : total
  }, 0)
}

function countSpecialRadarHits(normalized = '', groupKey = '') {
  const specialKeywords = {
    math: ['梯度下降', 'adagrad', 'rmsprop', 'adam', '优化器', '最优化'],
    data: ['pytorch', 'tensorflow', '训练', '数据集', '实验'],
    ml: ['梯度下降', 'adagrad', 'rmsprop', 'adam', '目标检测', '分类', '回归'],
    dl: ['adagrad', 'rmsprop', 'adam', '优化器', 'senet', 'vision transformer', 'transformer'],
    vision: ['计算机视觉', '目标检测', '视觉问答', '图像分类', 'senet', 'vision transformer', 'vit'],
    language: ['视觉问答', '问答', '对话', '语言模型'],
    engineering: ['导论', '迁移', '路径', '推荐动作', '学习规划'],
  }
  return countKeywordHits(normalized, specialKeywords[groupKey] || [])
}

function clampRadarScore(value, min = 0, max = 100) {
  return Math.max(min, Math.min(max, Math.round(Number(value || 0))))
}

function resolveRadarGroups(name = '') {
  const normalized = normalizeRadarText(name)
  if (!normalized) return []

  const matched = new Map()
  RADAR_DIMENSION_GROUPS.forEach((group) => {
    if (countKeywordHits(normalized, group.keywords) > 0) {
      matched.set(group.key, group)
    }
  })

  return [...matched.values()]
}

function resolveRadarGroup(name = '') {
  return resolveRadarGroups(name)[0] || null
}

function truncateRadarLabel(name = '') {
  const text = String(name || '').trim()
  if (text.length <= 6) return text
  return `${text.slice(0, 6)}…`
}

function buildDisplayRadarAxes(items = []) {
  if (!Array.isArray(items) || !items.length) return []

  const grouped = new Map()
  const unmatched = []

  items.forEach((item) => {
    const name = String(item?.name || '').trim()
    const value = Number(item?.value || 0)
    if (!name || !Number.isFinite(value)) return

    const group = resolveRadarGroup(name)
    if (!group) {
      unmatched.push({ name, value })
      return
    }

    const current = grouped.get(group.key) || { total: 0, count: 0 }
    current.total += value
    current.count += 1
    grouped.set(group.key, current)
  })

  const result = RADAR_DIMENSION_GROUPS
    .map((group) => {
      const current = grouped.get(group.key)
      if (!current?.count) return null
      return {
        name: group.name,
        value: Math.round(current.total / current.count),
      }
    })
    .filter(Boolean)

  if (unmatched.length) {
    const total = unmatched.reduce((sum, item) => sum + item.value, 0)
    result.push({
      name: '综合拓展',
      value: Math.round(total / unmatched.length),
    })
  }

  if (result.length >= 3) return result.slice(0, 8)

  return items.slice(0, 8).map((item) => ({
    name: truncateRadarLabel(item.name),
    value: Math.round(Number(item.value || 0)),
  }))
}

const progressItems = computed(() => {
  const seen = new Set()
  const source = shouldUseDemoData.value ? DEMO_PROGRESS_ITEMS : userStore.progressList || []
  return source.filter((item) => {
    const key = String(item.knowledgeId || item.routeKnowledgeId || item.canonicalName || item.name || '').trim()
    if (!key || seen.has(key)) return false
    seen.add(key)
    return true
  })
})

const progressStats = computed(() => {
  let mastered = 0
  let learning = 0
  let weak = 0
  let total = 0

  progressItems.value.forEach((item) => {
    total += 1
    const mastery = Number(item.masteryScore || item.mastery || 0)
    const answered = Number(item.answeredCount || 0)
    if (mastery >= 80) mastered += 1
    else if (mastery >= 40) learning += 1
    if (mastery < 30 && answered > 0) weak += 1
  })

  return { total, mastered, learning, weak }
})

const hasRadarBehaviorEvidence = computed(() =>
  progressItems.value.some((item) => Number(item.answeredCount || 0) > 0 || Number(item.masteryScore || item.mastery || 0) > 0),
)

const radarSignals = computed(() => {
  const signals = []
  const seen = new Set()

  const pushSignal = (textParts = [], rawValue = 0) => {
    const text = textParts
      .map((item) => String(item || '').trim())
      .filter(Boolean)
      .join(' ')
      .trim()
    if (!text || seen.has(text)) return
    seen.add(text)

    signals.push({
      text,
      value: clampRadarScore(rawValue, 36, 92),
    })
  }

  progressItems.value.forEach((item, index) => {
    const mastery = Number(item.masteryScore || item.mastery || 0)
    const answered = Number(item.answeredCount || 0)
    pushSignal(
      [item.canonicalName, item.name, item.chapterTitle, item.knowledgeId, item.routeKnowledgeId],
      44 + mastery * 0.38 + answered * 4 + Math.max(0, 10 - index * 2),
    )
  })

  recommendedPath.value.forEach((item, index) => {
    const score = item.score != null ? Number(item.score || 0) * 100 : 82 - index * 5
    pushSignal(
      [item.title, item.detail, item.reasonLabel, item.stage, item.sourceCourse, item.targetCourse],
      46 + score * 0.18 + Math.max(0, 8 - index * 2),
    )
  })

  knowledgeCards.value.forEach((card, index) => {
    const progress = Number(card.progress || 0)
    pushSignal(
      [card.title, card.domain, card.tag, card.type],
      44 + (progress > 0 ? progress : 76 - index * 6) * 0.16 + Math.max(0, 6 - index * 2),
    )
  })

  return signals
})

function buildFallbackRadarAxes(signals = []) {
  if (!Array.isArray(signals) || !signals.length) return []
  const coldStart = !hasRadarBehaviorEvidence.value && rawRadarAxes.value.length === 0

  const matched = RADAR_DIMENSION_GROUPS
    .map((group) => {
      let weighted = 0
      let hits = 0
      let support = 0

      signals.forEach((signal) => {
        const normalized = normalizeRadarText(signal.text)
        const hitCount = countKeywordHits(normalized, group.keywords) + countSpecialRadarHits(normalized, group.key)
        if (!hitCount) return

        weighted += Number(signal.value || 0) * (1 + Math.min(0.24, (hitCount - 1) * 0.08))
        hits += 1
        support += hitCount
      })

      if (!hits) return null

      return {
        key: group.key,
        name: group.name,
        value: clampRadarScore(
          coldStart
            ? weighted / hits * 0.68 + 6 + Math.min(3, support * 0.8)
            : weighted / hits * 0.84 + 5 + Math.min(6, support * 1.15),
          coldStart ? 44 : 46,
          coldStart ? 72 : 84,
        ),
        support,
      }
    })
    .filter(Boolean)
    .sort((left, right) => {
      if (right.support !== left.support) return right.support - left.support
      return Number(right.value || 0) - Number(left.value || 0)
    })

  if (matched.length > 0 && matched.length < 3) {
    const combinedText = signals.map((item) => normalizeRadarText(item.text)).join(' ')
    const average = matched.reduce((sum, item) => sum + Number(item.value || 0), 0) / matched.length
    const supplement = []

    const pushSupplement = (key, baseValue) => {
      if (matched.some((item) => item.key === key) || supplement.some((item) => item.key === key)) return
      const group = RADAR_DIMENSION_GROUPS.find((item) => item.key === key)
      if (!group) return
      supplement.push({
        key,
        name: group.name,
        value: clampRadarScore(baseValue, 40, 78),
        support: 0,
      })
    }

    if (/梯度下降|adagrad|rmsprop|adam|优化/i.test(combinedText)) {
      pushSupplement('ml', average - (coldStart ? 3 : 2))
      pushSupplement('dl', average - (coldStart ? 9 : 6))
      pushSupplement('data', average - (coldStart ? 13 : 9))
    }

    if (/视觉|图像|检测|问答|vision|senet|vit/i.test(combinedText)) {
      pushSupplement('vision', average - (coldStart ? 4 : 2))
      pushSupplement('language', average - (coldStart ? 10 : 6))
      pushSupplement('engineering', average - (coldStart ? 14 : 9))
    }

    if (/迁移|推荐|路径|导论/i.test(combinedText)) {
      pushSupplement('engineering', average - (coldStart ? 11 : 7))
    }

    pushSupplement('data', average - (coldStart ? 14 : 10))
    pushSupplement('engineering', average - (coldStart ? 16 : 12))

    matched.push(...supplement)
  }

  const limited = coldStart ? matched.slice(0, 3) : matched.slice(0, 5)
  return limited.map(({ name, value }, index) => ({
    name,
    value: coldStart ? clampRadarScore(value - index * 3, 44, 70) : value,
  }))
}

const synthesizedRadarAxes = computed(() => buildFallbackRadarAxes(radarSignals.value))

const radarAxes = computed(() => {
  const live = buildDisplayRadarAxes(rawRadarAxes.value)
  if (live.length >= 3) return live

  const fallback = synthesizedRadarAxes.value
  if (!live.length) return fallback

  const merged = [...live]
  fallback.forEach((item) => {
    if (merged.some((existing) => existing.name === item.name)) return
    if (merged.length >= 8) return
    merged.push(item)
  })
  return merged
})

const totalProgress = computed(() => {
  const { total, mastered, learning } = progressStats.value
  if (!total) return 0
  return Math.round(((mastered + learning * 0.6) / total) * 100)
})

const averageRadarScore = computed(() => {
  const source = radarAxes.value
  if (!source.length) return 0
  const total = source.reduce((sum, item) => sum + Number(item.value || 0), 0)
  return Math.round(total / source.length)
})

const learnerName = computed(() => {
  const profileName = String(profile.value?.name || '').trim()
  if (profileName) return profileName

  const storeName = String(userStore.user?.name || userStore.user?.username || '').trim()
  if (storeName) return storeName

  const email = String(userStore.user?.email || '').trim()
  if (email.includes('@')) return email.split('@')[0]

  return '学习者'
})

const heroSummary = computed(() => {
  if (String(profile.value?.summary || '').trim()) {
    return profile.value.summary
  }

  if (progressStats.value.total > 0) {
    return `已追踪 ${progressStats.value.total} 个知识点，当前完成度 ${totalProgress.value}% 。建议优先处理薄弱点，再把推荐路径串成一条连续主线。`
  }

  return '这里不是简单的数据陈列，而是你的学习驾驶舱。完成几次练习后，这里会开始呈现能力画像、热区变化和下一步最值得投入的方向。'
})

function inferMetricTone(label = '', index = 0) {
  const text = String(label).toLowerCase()
  if (text.includes('掌握')) return 'success'
  if (text.includes('薄弱') || text.includes('风险')) return 'warning'
  if (text.includes('进度') || text.includes('%')) return 'accent'
  if (text.includes('能力')) return 'info'
  return ['accent', 'info', 'warning', 'success'][index % 4]
}

function buildMetricHelper(label = '') {
  const text = String(label)
  if (text.includes('掌握')) return '已经形成稳定输出的知识节点'
  if (text.includes('薄弱')) return '建议优先回到前置链查漏补缺'
  if (text.includes('学习中')) return '继续把相关知识点连成一条主线'
  if (text.includes('进度')) return '综合已掌握与进行中的完成估算'
  if (text.includes('能力')) return '由答题记录与图谱关系共同生成'
  return '仪表盘会随着学习记录持续刷新'
}

const metricCards = computed(() => {
  if (quickStats.value.length) {
    return quickStats.value.slice(0, 4).map((item, index) => ({
      eyebrow: index === 0 ? 'Focus' : `Metric 0${index + 1}`,
      label: item.label,
      value: item.value,
      helper: buildMetricHelper(item.label),
      tone: inferMetricTone(item.label, index),
    }))
  }

  return [
    {
      eyebrow: 'Focus',
      label: '学习中知识点',
      value: progressStats.value.learning,
      helper: '维持理解连续性的主战场',
      tone: 'accent',
    },
    {
      eyebrow: 'Stable',
      label: '已掌握知识点',
      value: progressStats.value.mastered,
      helper: '可以拿来做迁移和类比的基础',
      tone: 'success',
    },
    {
      eyebrow: 'Risk',
      label: '薄弱知识点',
      value: progressStats.value.weak,
      helper: '优先补齐前置关系，避免后续卡住',
      tone: 'warning',
    },
    {
      eyebrow: 'Progress',
      label: '总体掌握进度',
      value: `${totalProgress.value}%`,
      helper: '按掌握与在学状态综合估算',
      tone: 'info',
    },
  ]
})

const sortedAxes = computed(() => [...radarAxes.value].sort((left, right) => Number(right.value || 0) - Number(left.value || 0)))
const strongestAxes = computed(() => sortedAxes.value.slice(0, 3))
const weakestAxes = computed(() => [...sortedAxes.value].reverse().slice(0, 3))

const normalizedWeeklyActivity = computed(() => {
  const raw = dashboardSummary.value?.weeklyTrend
  const fallbackLabels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

  const source = Array.isArray(raw) && raw.length
    ? raw.map((item, index) => ({
        label: String(item.label || fallbackLabels[index] || `第${index + 1}天`),
        value: Number(item.value || 0),
      }))
    : fallbackLabels.map((label, index) => ({
        label,
        value: progressStats.value.total > 0
          ? Math.max(0, Math.min(100, Math.round((progressStats.value.learning * 14 + progressStats.value.mastered * 9 + index * 7) % 100)))
          : 0,
      }))

  const peak = Math.max(0, ...source.map((item) => item.value))
  return source.map((item) => ({
    ...item,
    height: peak > 0 ? Math.max(16, Math.round((item.value / peak) * 100)) : 14,
  }))
})

const weeklyTotals = computed(() => {
  const total = normalizedWeeklyActivity.value.reduce((sum, item) => sum + Number(item.value || 0), 0)
  const peak = Math.max(0, ...normalizedWeeklyActivity.value.map((item) => Number(item.value || 0)))
  const average = normalizedWeeklyActivity.value.length ? Math.round(total / normalizedWeeklyActivity.value.length) : 0
  return { total, peak, average }
})

const weakKnowledge = computed(() =>
  progressItems.value
    .filter((item) => Number(item.answeredCount || 0) > 0)
    .sort((left, right) => {
      const masteryGap = Number(left.masteryScore || 0) - Number(right.masteryScore || 0)
      if (masteryGap !== 0) return masteryGap
      return Number(right.answeredCount || 0) - Number(left.answeredCount || 0)
    })
    .slice(0, 4),
)

function resolveRoute(route = '', fallbackId = '') {
  const normalized = String(route || '').trim()
  if (normalized) return normalized
  const id = String(fallbackId || '').trim()
  return id ? `/knowledge/${encodeURIComponent(id)}` : '/'
}

const focusActions = computed(() => {
  if (recommendedPath.value.length) {
    return recommendedPath.value.slice(0, 4).map((item, index) => ({
      title: item.title || `推荐动作 ${index + 1}`,
      detail: item.detail || '建议从该节点切入，并继续展开它的前置与相邻知识。',
      meta: item.reasonLabel || item.stage || `第 ${index + 1} 步`,
      score: item.score != null ? `${Math.round(Number(item.score || 0) * 100)}%` : '',
      route: resolveRoute(item.route, item.knowledgeId || item.title),
    }))
  }

  if (weakKnowledge.value.length) {
    return weakKnowledge.value.map((item, index) => ({
      title: item.canonicalName || item.name || `薄弱知识点 ${index + 1}`,
      detail: `当前掌握度 ${Math.round(Number(item.masteryScore || 0))}% ，建议先回到它的前置关系补齐基础。`,
      meta: '优先补强',
      score: `${Math.round(Number(item.masteryScore || 0))}%`,
      route: resolveRoute('', item.routeKnowledgeId || item.knowledgeId || item.canonicalName || item.name),
    }))
  }

  return [
    {
      title: '先完成一组知识点练习',
      detail: '答题后这里会自动出现能力画像、热区矩阵和更具体的推荐动作。',
      meta: '启动建议',
      score: '',
      route: '/',
    },
    {
      title: '从图谱中展开一条课程主线',
      detail: '不要同时学很多散点，先沿着一条前置链往下走，效果会更稳定。',
      meta: '学习节奏',
      score: '',
      route: '/',
    },
  ]
})

const displayKnowledgeCards = computed(() => {
  const baseCards = knowledgeCards.value.length
    ? knowledgeCards.value
    : recommendedPath.value.slice(0, 4).map((item, index) => ({
        id: item.knowledgeId || item.title || `path-${index}`,
        type: item.stage || '推荐路径',
        progress: Number(item.score || 0) * 100,
        title: item.title,
        domain: item.reasonLabel || item.stage || '图谱推荐',
        tag: item.detail || '建议结合图谱关系继续深入',
        route: resolveRoute(item.route, item.knowledgeId || item.title),
      }))

  return baseCards.slice(0, 4).map((card) => ({
    ...card,
    progressValue: Math.max(0, Math.min(100, Math.round(Number(card.progress || 0)))),
    palette: getCoursePalette(card.domain || card.tag || card.title),
    route: resolveRoute(card.route, card.id || card.title),
  }))
})

const growthHighlights = computed(() => {
  if (growthInsights.value.length) {
    return growthInsights.value.slice(0, 4).map((item) => ({
      title: item.label,
      value: `${Math.round(Number(item.value || 0))}%`,
      detail: item.note || '当前能力维度正在持续变化。',
      ratio: Math.max(0, Math.min(100, Math.round(Number(item.value || 0)))),
    }))
  }

  if (growthCards.value.length) {
    return growthCards.value.slice(0, 4).map((item) => ({
      title: item.label,
      value: item.value,
      detail: '这是当前画像阶段给出的摘要判断。',
      ratio: 68,
    }))
  }

  if (strongestAxes.value.length) {
    return strongestAxes.value.map((item) => ({
      title: item.name,
      value: `${Math.round(Number(item.value || 0))}%`,
      detail: '当前表现相对稳定，可作为继续迁移的支点。',
      ratio: Math.max(0, Math.min(100, Math.round(Number(item.value || 0)))),
    }))
  }

  return []
})

const goalItems = computed(() => {
  if (profile.value?.goals?.length) {
    return profile.value.goals.slice(0, 4)
  }

  if (focusActions.value.length) {
    return focusActions.value.slice(0, 3).map((item) => `围绕「${item.title}」安排一段连续学习时间`)
  }

  return [
    '先在图谱中完成一次节点展开，建立自己的学习主线',
    '完成几道练习题，让能力画像开始有真实反馈',
    '把薄弱知识点和推荐路径连接起来，而不是零散地看概念',
  ]
})

const recentItems = computed(() => {
  if (profile.value?.recentRecords?.length) {
    return profile.value.recentRecords.slice(0, 4).map((item, index) => ({
      title: `最近记录 ${index + 1}`,
      detail: item,
    }))
  }

  if (progressItems.value.length) {
    return progressItems.value
      .slice()
      .sort((left, right) => Number(right.answeredCount || 0) - Number(left.answeredCount || 0))
      .slice(0, 4)
      .map((item) => ({
        title: item.canonicalName || item.name || item.knowledgeId || '知识点',
        detail: `${Math.round(Number(item.masteryScore || 0))}% 掌握度 · ${Number(item.answeredCount || 0)} 次练习`,
      }))
  }

  return [
    {
      title: '还没有学习记录',
      detail: '完成几次练习之后，这里会自动变成一条清晰的学习时间线。',
    },
  ]
})

const svgRingCircumference = 2 * Math.PI * 80
const svgRingDasharray = computed(() => {
  const pct = Math.max(0, Math.min(100, totalProgress.value)) / 100
  return `${svgRingCircumference * pct} ${svgRingCircumference * (1 - pct)}`
})

function metricIcon(tone) {
  const icons = {
    accent: 'M13 10V3L4 14h7v7l9-11h-7z',
    success: 'M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z',
    warning: 'M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z',
    info: 'M3.5 18.49l6-6.01 4 4L22 6.92l-1.41-1.41-7.09 7.97-4-4L2 16.99l1.5 1.5z',
  }
  return icons[tone] || icons.info
}

const heatmapPeak = computed(() => {
  const values = heatmapRows.value.flatMap((row) => (row.values || []).map((value) => Number(value || 0)))
  return Math.max(1, ...values, 0)
})

function heatValueStyle(value) {
  const ratio = Math.max(0, Math.min(1, Number(value || 0) / heatmapPeak.value))
  const intensity = 0.08 + ratio * 0.24
  const border = 0.12 + ratio * 0.16
  return {
    background: `rgba(59, 130, 246, ${intensity})`,
    borderColor: `rgba(148, 163, 184, ${border})`,
    boxShadow: 'none',
    color: ratio > 0.58 ? '#f8fafc' : 'rgba(226, 232, 240, 0.88)',
  }
}

function handleRadarResize() {
  nextTick(() => {
    if (radarAxes.value.length >= 3) {
      drawRadar()
    }
  })
}

function drawRadar() {
  const canvas = radarCanvas.value
  if (!canvas || radarAxes.value.length < 3) return

  const parentWidth = canvas.parentElement?.clientWidth || 320
  const size = Math.max(280, Math.min(360, parentWidth))
  const dpr = window.devicePixelRatio || 1
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  canvas.width = size * dpr
  canvas.height = size * dpr
  canvas.style.width = `${size}px`
  canvas.style.height = `${size}px`

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, size, size)

  const cx = size / 2
  const cy = size / 2
  const maxRadius = size * 0.31
  const axes = radarAxes.value
  const angleStep = (Math.PI * 2) / axes.length
  const startAngle = -Math.PI / 2
  const LEVELS = 5
  const font = '"IBM Plex Sans", "PingFang SC", sans-serif'
  const labelRadius = maxRadius + 26

  const tracePolygon = (radiusOrGetter) => {
    ctx.beginPath()
    axes.forEach((axis, index) => {
      const angle = startAngle + angleStep * index
      const radius = typeof radiusOrGetter === 'function' ? radiusOrGetter(axis, index) : radiusOrGetter
      const x = cx + radius * Math.cos(angle)
      const y = cy + radius * Math.sin(angle)
      if (index === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })
    ctx.closePath()
  }

  for (let level = 1; level <= LEVELS; level += 1) {
    const radius = (maxRadius / LEVELS) * level
    tracePolygon(radius)
    if (level % 2 === 0) {
      ctx.fillStyle = 'rgba(148, 163, 184, 0.025)'
      ctx.fill()
    }
    ctx.strokeStyle = `rgba(148, 163, 184, ${0.12 + level * 0.025})`
    ctx.lineWidth = 1
    ctx.stroke()
  }

  axes.forEach((_, index) => {
    const angle = startAngle + angleStep * index
    const x = cx + maxRadius * Math.cos(angle)
    const y = cy + maxRadius * Math.sin(angle)
    ctx.beginPath()
    ctx.moveTo(cx, cy)
    ctx.lineTo(x, y)
    ctx.strokeStyle = 'rgba(148, 163, 184, 0.18)'
    ctx.lineWidth = 1
    ctx.stroke()
  })

  tracePolygon((axis) => {
    const ratio = Math.max(0, Math.min(1, Number(axis.value || 0) / 100))
    return maxRadius * ratio
  })
  ctx.fillStyle = 'rgba(59, 130, 246, 0.14)'
  ctx.fill()
  ctx.strokeStyle = 'rgba(37, 99, 235, 0.8)'
  ctx.lineWidth = 2.25
  ctx.stroke()

  axes.forEach((axis, index) => {
    const angle = startAngle + angleStep * index
    const ratio = Math.max(0, Math.min(1, Number(axis.value || 0) / 100))
    const x = cx + maxRadius * ratio * Math.cos(angle)
    const y = cy + maxRadius * ratio * Math.sin(angle)

    ctx.beginPath()
    ctx.arc(x, y, 4.5, 0, Math.PI * 2)
    ctx.fillStyle = '#f8fafc'
    ctx.fill()
    ctx.strokeStyle = 'rgba(37, 99, 235, 0.78)'
    ctx.lineWidth = 1.5
    ctx.stroke()
  })

  ctx.save()
  ctx.font = `600 11px ${font}`
  ctx.fillStyle = 'rgba(203, 213, 225, 0.9)'
  ctx.textBaseline = 'middle'
  axes.forEach((axis, index) => {
    const angle = startAngle + angleStep * index
    const lx = cx + labelRadius * Math.cos(angle)
    const ly = cy + labelRadius * Math.sin(angle)
    const name = String(axis.name || '')
    const cos = Math.cos(angle)
    const sin = Math.sin(angle)
    let offsetX = 0
    let offsetY = 0

    if (cos > 0.28) {
      ctx.textAlign = 'left'
      offsetX = 7
    } else if (cos < -0.28) {
      ctx.textAlign = 'right'
      offsetX = -7
    } else {
      ctx.textAlign = 'center'
    }

    if (sin > 0.55) offsetY = 8
    if (sin < -0.55) offsetY = -8

    ctx.fillText(name, lx + offsetX, ly + offsetY)
  })
  ctx.restore()

  ctx.beginPath()
  ctx.arc(cx, cy, 30, 0, Math.PI * 2)
  ctx.fillStyle = 'rgba(15, 23, 42, 0.94)'
  ctx.fill()
  ctx.strokeStyle = 'rgba(148, 163, 184, 0.2)'
  ctx.lineWidth = 1.2
  ctx.stroke()

  ctx.fillStyle = 'rgba(148, 163, 184, 0.72)'
  ctx.font = `600 10px ${font}`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText('AVG', cx, cy - 8)
  ctx.fillStyle = '#f8fafc'
  ctx.font = `700 20px ${font}`
  ctx.fillText(`${averageRadarScore.value}%`, cx, cy + 12)
}

watch(
  radarAxes,
  () => {
    handleRadarResize()
  },
  { immediate: true, deep: true },
)
</script>

<template>
  <section class="dashboard-view page-frame" :class="{ 'dashboard-view--ready': ready }">
    <!-- Ambient floating orbs -->
    <div class="dashboard-orb dashboard-orb--1" aria-hidden="true"></div>
    <div class="dashboard-orb dashboard-orb--2" aria-hidden="true"></div>
    <div class="dashboard-orb dashboard-orb--3" aria-hidden="true"></div>

    <div class="dashboard-shell">
      <!-- ===== HERO ===== -->
      <header class="page-card dashboard-hero dash-enter" style="--enter-i:0">
        <div class="dashboard-hero__copy">
          <p class="eyebrow">Learning Cockpit</p>
          <h1 class="dashboard-hero__title">{{ learnerName }}的<br>学习概览</h1>
          <p class="dashboard-hero__summary">{{ heroSummary }}</p>

          <div class="dashboard-hero__chips">
            <span class="dashboard-chip"><i class="dashboard-chip__dot"></i>{{ progressStats.total }} 个知识点已纳入追踪</span>
            <span class="dashboard-chip"><i class="dashboard-chip__dot"></i>{{ radarAxes.length || 0 }} 个能力维度已激活</span>
            <span class="dashboard-chip"><i class="dashboard-chip__dot"></i>{{ focusActions.length }} 条当前建议可直接执行</span>
          </div>
        </div>

        <div class="dashboard-hero__side">
          <!-- SVG Progress Ring -->
          <div class="dashboard-ring-wrap">
            <svg class="dashboard-ring-svg" viewBox="0 0 200 200">
              <defs>
                <linearGradient id="dashRingGrad" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stop-color="#818cf8" />
                  <stop offset="50%" stop-color="#a78bfa" />
                  <stop offset="100%" stop-color="#38bdf8" />
                </linearGradient>
              </defs>
              <!-- Glow halo ring -->
              <circle class="dashboard-ring__halo" cx="100" cy="100" r="92" />
              <!-- Background track -->
              <circle class="dashboard-ring__track" cx="100" cy="100" r="80" />
              <!-- Progress arc -->
              <circle
                class="dashboard-ring__fill"
                cx="100" cy="100" r="80"
                :stroke-dasharray="svgRingDasharray"
                :stroke-dashoffset="0"
              />
              <!-- Decorative inner ring -->
              <circle class="dashboard-ring__inner-deco" cx="100" cy="100" r="66" />
            </svg>
            <div class="dashboard-ring__label">
              <strong>{{ totalProgress }}%</strong>
              <span>掌握进度</span>
            </div>
          </div>

          <div class="dashboard-hero__actions">
            <RouterLink class="primary-btn" to="/">返回图谱继续学习</RouterLink>
            <RouterLink class="secondary-btn" to="/">查看知识图谱</RouterLink>
          </div>
        </div>
      </header>

      <!-- ===== METRICS ===== -->
      <section class="dashboard-metrics">
        <article
          v-for="(stat, idx) in metricCards"
          :key="`${stat.label}-${stat.eyebrow}`"
          class="page-card dashboard-metric-card dash-enter"
          :class="`dashboard-metric-card--${stat.tone}`"
          :style="{ '--enter-i': idx + 1 }"
        >
          <div class="dashboard-metric-card__icon">
            <svg viewBox="0 0 24 24"><path :d="metricIcon(stat.tone)" /></svg>
          </div>
          <div class="dashboard-metric-card__body">
            <span class="dashboard-metric-card__eyebrow">{{ stat.eyebrow }}</span>
            <strong>{{ stat.value }}</strong>
            <span class="dashboard-metric-card__label">{{ stat.label }}</span>
            <p>{{ stat.helper }}</p>
          </div>
        </article>
      </section>

      <!-- ===== LAYOUT GRID ===== -->
      <section class="dashboard-layout">
        <!-- Radar -->
        <article class="page-card dashboard-panel dashboard-panel--radar dash-enter" style="--enter-i:5">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Ability Graph</p>
              <h2>能力图谱</h2>
            </div>
            <span class="dashboard-section-badge">{{ averageRadarScore }}% 平均表现</span>
          </div>

          <div class="dashboard-radar-block">
            <div class="dashboard-radar-stage">
              <div v-if="radarAxes.length >= 3" class="dashboard-radar-canvas-wrap">
                <canvas ref="radarCanvas"></canvas>
              </div>
              <div v-else class="dashboard-empty-state dashboard-empty-state--compact">
                <strong>能力图谱还在生成</strong>
                <p>当前答题和学习记录还不够，所以能力雷达暂未展开。完成几次练习后，这里会显示各能力维度的真实表现与补强方向。</p>
                <RouterLink class="primary-btn" to="/">去知识图谱开始学习</RouterLink>
              </div>
            </div>

            <div class="dashboard-axis-column">
              <div class="dashboard-axis-list">
                <div class="dashboard-axis-list__head">
                  <span class="eyebrow">优势轴</span>
                  <strong>当前最稳的部分</strong>
                </div>
                <div v-if="strongestAxes.length" class="dashboard-axis-pills">
                  <div v-for="axis in strongestAxes" :key="axis.name" class="dashboard-axis-pill dashboard-axis-pill--strong">
                    <span>{{ axis.name }}</span>
                    <div class="dashboard-axis-pill__bar">
                      <span :style="{ width: `${Math.round(axis.value)}%` }"></span>
                    </div>
                    <strong>{{ Math.round(axis.value) }}%</strong>
                  </div>
                </div>
                <p v-else class="dashboard-muted-copy">还没有足够数据来判断稳定优势。</p>
              </div>

              <div class="dashboard-axis-list">
                <div class="dashboard-axis-list__head">
                  <span class="eyebrow">补强轴</span>
                  <strong>接下来最值得投入</strong>
                </div>
                <div v-if="weakestAxes.length" class="dashboard-axis-pills">
                  <div v-for="axis in weakestAxes" :key="axis.name" class="dashboard-axis-pill dashboard-axis-pill--weak">
                    <span>{{ axis.name }}</span>
                    <div class="dashboard-axis-pill__bar">
                      <span :style="{ width: `${Math.round(axis.value)}%` }"></span>
                    </div>
                    <strong>{{ Math.round(axis.value) }}%</strong>
                  </div>
                </div>
                <p v-else class="dashboard-muted-copy">开始练习后，这里会自动生成需要优先补的能力轴。</p>
              </div>
            </div>
          </div>
        </article>

        <!-- Actions -->
        <article class="page-card dashboard-panel dashboard-panel--actions dash-enter" style="--enter-i:6">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Next Step</p>
              <h2>推荐动作</h2>
            </div>
            <span class="dashboard-section-badge">按优先级排序</span>
          </div>

          <div class="dashboard-action-list">
            <RouterLink
              v-for="(item, idx) in focusActions"
              :key="`${item.meta}-${item.title}`"
              class="dashboard-action-card"
              :to="item.route"
            >
              <span class="dashboard-action-card__step">{{ String(idx + 1).padStart(2, '0') }}</span>
              <div class="dashboard-action-card__content">
                <div class="dashboard-action-card__meta">
                  <span>{{ item.meta }}</span>
                  <strong v-if="item.score">{{ item.score }}</strong>
                </div>
                <h3>{{ item.title }}</h3>
                <p>{{ item.detail }}</p>
              </div>
            </RouterLink>
          </div>
        </article>

        <!-- Trend -->
        <article class="page-card dashboard-panel dashboard-panel--trend dash-enter" style="--enter-i:7">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Weekly Rhythm</p>
              <h2>本周趋势</h2>
            </div>
            <span class="dashboard-section-badge">连续观察比单天爆发更重要</span>
          </div>

          <div class="dashboard-trend-summary">
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.total }}</strong>
              <span>本周活跃值</span>
            </div>
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.peak }}</strong>
              <span>单日峰值</span>
            </div>
            <div class="dashboard-trend-summary__item">
              <strong>{{ weeklyTotals.average }}</strong>
              <span>日均节奏</span>
            </div>
          </div>

          <div class="trend-bars trend-bars--large dashboard-trend-bars">
            <div v-for="item in normalizedWeeklyActivity" :key="item.label" class="trend-bar dashboard-trend-bar">
              <span class="trend-bar__col dashboard-trend-bar__col" :style="{ height: `${item.height}%` }">
                <i class="dashboard-trend-bar__glow"></i>
              </span>
              <strong>{{ item.value }}</strong>
              <small>{{ item.label }}</small>
            </div>
          </div>
        </article>

        <!-- Heatmap -->
        <article v-if="heatmapRows.length && heatmapDomains.length" class="page-card dashboard-panel dashboard-panel--heatmap dash-enter" style="--enter-i:8">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Heat Matrix</p>
              <h2>知识热区</h2>
            </div>
            <span class="dashboard-section-badge">颜色越亮，当前关注度越高</span>
          </div>

          <div class="dashboard-heatmap">
            <div class="dashboard-heatmap__head">
              <span class="dashboard-heatmap__corner">知识点</span>
              <span v-for="domain in heatmapDomains" :key="domain" class="dashboard-heatmap__domain">{{ domain }}</span>
            </div>

            <div v-for="row in heatmapRows" :key="row.label" class="dashboard-heatmap__row">
              <span class="dashboard-heatmap__label">{{ row.label }}</span>
              <span
                v-for="(value, index) in row.values"
                :key="`${row.label}-${index}`"
                class="dashboard-heatmap__cell"
                :class="{ 'dashboard-heatmap__cell--hot': (value / heatmapPeak) > 0.7 }"
                :style="heatValueStyle(value)"
              >
                {{ Math.round(value) }}
              </span>
            </div>
          </div>
        </article>

        <!-- Knowledge Cards -->
        <article class="page-card dashboard-panel dashboard-panel--cards dash-enter" style="--enter-i:9">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Focus Cards</p>
              <h2>值得马上处理的知识点</h2>
            </div>
            <span class="dashboard-section-badge">{{ displayKnowledgeCards.length }} 张卡片</span>
          </div>

          <div v-if="displayKnowledgeCards.length" class="dashboard-knowledge-grid">
            <RouterLink
              v-for="card in displayKnowledgeCards"
              :key="card.id || card.title"
              class="dashboard-knowledge-card"
              :to="card.route"
              :style="{
                '--card-primary': card.palette.primary,
                '--card-soft': card.palette.light,
                '--card-ring': card.palette.ring,
              }"
            >
              <div class="dashboard-knowledge-card__meta">
                <span class="dashboard-knowledge-card__tag">{{ card.type || card.domain || '推荐' }}</span>
                <strong>{{ card.progressValue }}%</strong>
              </div>
              <h3>{{ card.title }}</h3>
              <p>{{ card.tag || card.domain || '建议结合图谱关系继续推进' }}</p>
              <div class="dashboard-knowledge-card__progress">
                <span :style="{ width: `${card.progressValue}%` }"></span>
              </div>
            </RouterLink>
          </div>

          <div v-else class="dashboard-empty-state">
            <strong>推荐卡片还没生成</strong>
            <p>开始在图谱里学习后，这里会自动整理出最适合你当下推进的知识卡片。</p>
            <RouterLink class="secondary-btn" to="/">去图谱里找一个起点</RouterLink>
          </div>
        </article>

        <!-- Insights -->
        <article class="page-card dashboard-panel dashboard-panel--insights dash-enter" style="--enter-i:10">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Growth Insights</p>
              <h2>近期画像解读</h2>
            </div>
            <span class="dashboard-section-badge">从趋势看，而不是只看一组分数</span>
          </div>

          <div v-if="growthHighlights.length" class="dashboard-insight-grid">
            <article v-for="item in growthHighlights" :key="item.title" class="dashboard-insight-card">
              <div class="dashboard-insight-card__head">
                <span>{{ item.title }}</span>
                <strong>{{ item.value }}</strong>
              </div>
              <div class="dashboard-insight-card__bar">
                <span :style="{ width: `${item.ratio}%` }"></span>
              </div>
              <p>{{ item.detail }}</p>
            </article>
          </div>

          <div v-else class="dashboard-empty-state dashboard-empty-state--compact">
            <strong>成长洞察会逐步形成</strong>
            <p>先做几次练习，系统才能判断你的稳定强项、薄弱区和近期变化趋势。</p>
          </div>
        </article>

        <!-- Records -->
        <article class="page-card dashboard-panel dashboard-panel--records dash-enter" style="--enter-i:11">
          <div class="dashboard-section-head">
            <div>
              <p class="eyebrow">Memo</p>
              <h2>目标与记录</h2>
            </div>
            <span class="dashboard-section-badge">让当前任务和最近变化同屏出现</span>
          </div>

          <div class="dashboard-records-layout">
            <div class="dashboard-note-block">
              <h3>当前目标</h3>
              <ul class="dashboard-list">
                <li v-for="goal in goalItems" :key="goal">{{ goal }}</li>
              </ul>
            </div>

            <div class="dashboard-note-block">
              <h3>最近记录</h3>
              <div class="dashboard-record-list dashboard-record-list--timeline">
                <article v-for="item in recentItems" :key="`${item.title}-${item.detail}`" class="dashboard-record-item">
                  <strong>{{ item.title }}</strong>
                  <p>{{ item.detail }}</p>
                </article>
              </div>
            </div>
          </div>
        </article>
      </section>
    </div>
  </section>
</template>
