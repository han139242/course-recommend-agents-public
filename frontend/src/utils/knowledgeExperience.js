function clampScore(value, fallback = 0) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return fallback
  return Math.max(0, Math.min(100, numeric))
}

const MASTERY_STAGE_CONFIG = [
  {
    key: 'not_started',
    label: '未入门',
    min: 0,
    description: '还没有形成稳定认识，先把定义、作用和前置关系看明白。',
  },
  {
    key: 'understanding',
    label: '理解中',
    min: 20,
    description: '已经知道它大概是什么，但遇到变化题或真实场景还容易卡住。',
  },
  {
    key: 'basic_apply',
    label: '能做基础',
    min: 45,
    description: '基础判断和最小例子能完成，但还需要继续巩固迁移能力。',
  },
  {
    key: 'solid',
    label: '基本掌握',
    min: 65,
    description: '已经能稳定使用，下一步重点是和上下游知识点连起来。',
  },
  {
    key: 'transfer',
    label: '可以迁移',
    min: 80,
    description: '不仅会当前点，还能解释它与方法、工具和场景的连接关系。',
  },
]

const CHECKPOINT_THRESHOLDS = [20, 45, 65, 80]

export function getKnowledgeMasteryProfile(point = {}, score = 0) {
  const mastery = clampScore(score, point?.mastery ?? 0)
  const stage = [...MASTERY_STAGE_CONFIG]
    .reverse()
    .find((item) => mastery >= item.min) || MASTERY_STAGE_CONFIG[0]

  const checklist = Array.isArray(point?.masteryChecklist)
    ? point.masteryChecklist.map((item, index) => ({
        ...item,
        done: mastery >= (CHECKPOINT_THRESHOLDS[index] ?? 100),
      }))
    : []

  const completedCount = checklist.filter((item) => item.done).length
  const nextCheckpoint = checklist.find((item) => !item.done) || null

  return {
    score: mastery,
    stageKey: stage.key,
    stageLabel: stage.label,
    stageDescription: stage.description,
    completedCount,
    totalCount: checklist.length,
    nextCheckpoint,
    checklist,
  }
}

export function buildKnowledgeGraphContribution(point = {}, detail = null) {
  const prerequisites = detail?.prerequisites || []
  const related = detail?.related || []
  const extensions = detail?.parentTopics || []

  return {
    title: point?.graphRoleTitle || '知识图谱在这里的作用',
    summary:
      point?.graphRoleDescription ||
      '知识图谱不是拿来单独看图的，而是用来解释当前知识点为什么会卡、应该先补什么、学完接到哪里。',
    items: [
      {
        label: '定位卡点',
        text: prerequisites.length
          ? `图谱会先回看 ${prerequisites.length} 个先修点，帮助你判断当前不会，是概念没懂还是前置链路没通。`
          : '当前先修链还不完整时，会先退回到课程主线，避免学生在“不会为什么”上失去方向。',
      },
      {
        label: '生成路径',
        text: related.length || extensions.length
          ? `当前点会和 ${related.length} 个关联点、${extensions.length} 个主题方向一起组成“先学什么、一起学什么、学完去哪”的路径。`
          : '图谱会把当前点放回所属路径，给出最近的前后知识点，而不是只展示一个孤立页面。',
      },
      {
        label: '汇总能力',
        text: `这个知识点会被归入「${point?.domain || point?.domainName || '当前路径'}」这条能力路径，后续能力图谱就是从这些路径聚合出来的。`,
      },
    ],
  }
}
