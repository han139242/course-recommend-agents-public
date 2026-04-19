export const DEFAULT_COURSE_TITLE = '其他'
export const ALL_COURSE_TITLE = '全部'
export const COURSE_TITLE_ORDER = [
  '高等数学',
  '线性代数',
  '概率与统计',
  '数据结构',
  'Python编程',
  '人工智能导论',
  '机器学习',
  '深度学习',
  '自然语言处理',
  '计算机视觉',
  '强化学习',
  '知识图谱',
  '大模型',
  'AI工程',
  'AI伦理',
  DEFAULT_COURSE_TITLE,
]

export const COURSE_SOURCE_TITLE_MAP = {
  course_ai_engineering: 'AI工程',
  course_ai_ethics: 'AI伦理',
  course_ai_intro: '人工智能导论',
  course_calculus: '高等数学',
  course_cv: '计算机视觉',
  course_data_structure: '数据结构',
  course_deep_learning: '深度学习',
  course_knowledge_graph: '知识图谱',
  course_linear_algebra: '线性代数',
  course_llm: '大模型',
  course_machine_learning: '机器学习',
  course_nlp: '自然语言处理',
  course_probability: '概率与统计',
  course_python: 'Python编程',
  course_rl: '强化学习',
}

const LEGACY_COURSE_TITLE_ALIASES = {
  人工智能基础知识骨架: '人工智能导论',
  人工智能课程知识图谱: '知识图谱',
  人工智能在游戏设计中的应用: '人工智能导论',
  智能体与多智能体系统: '人工智能导论',
  智能计算及其应用: '人工智能导论',
  绪论: '人工智能导论',
  人工神经网络及其应用: '深度学习',
  专家系统与机器学习: '人工智能导论',
  搜索求解策略: '人工智能导论',
  知识表示: '人工智能导论',
  不确定性推理方法: '人工智能导论',
  确定性推理方法: '人工智能导论',
  自然语言处理及其应用: '自然语言处理',
  线性代数基础: '线性代数',
  概率统计: '概率与统计',
  Python程序设计: 'Python编程',
}

const COURSE_TITLE_KEYWORD_RULES = [
  { pattern: /自然语言处理|nlp/i, title: '自然语言处理' },
  { pattern: /知识图谱|knowledge[_\s-]*graph/i, title: '知识图谱' },
  { pattern: /强化学习|reinforcement[_\s-]*learning|\brl\b/i, title: '强化学习' },
  { pattern: /计算机视觉|computer[_\s-]*vision|\bcv\b|image|video/i, title: '计算机视觉' },
  { pattern: /深度学习|deep[_\s-]*learning|neural[_\s-]*network/i, title: '深度学习' },
  { pattern: /ai[_\s-]*engineering|mlops|serving|deployment|compression|edge[_\s-]*ai|inference/i, title: 'AI工程' },
  { pattern: /机器学习|machine[_\s-]*learning|automl/i, title: '机器学习' },
  { pattern: /线性代数|linear[_\s-]*algebra/i, title: '线性代数' },
  { pattern: /概率.*统计|统计.*概率|概率统计|probability|statistics/i, title: '概率与统计' },
  { pattern: /高等数学|微积分|数学分析|calculus/i, title: '高等数学' },
  { pattern: /python/i, title: 'Python编程' },
  { pattern: /数据结构|data[_\s-]*structure/i, title: '数据结构' },
  { pattern: /ai[_\s-]*ethics|可信ai|负责任ai|算法治理|ethics|fairness|privacy|safety|xai|governance/i, title: 'AI伦理' },
  { pattern: /大模型|\bllm\b|transformer|rag|prompt|language[_\s-]*model/i, title: '大模型' },
  {
    pattern: /人工智能导论|人工智能基础|知识表示|ontology|reasoning|search|planning|agent|multi[_\s-]*agent|expert[_\s-]*system|game[_\s-]*ai|intelligent[_\s-]*computing|artificial[_\s-]*intelligence/i,
    title: '人工智能导论',
  },
]

function normalizeText(value = '') {
  return String(value || '').trim()
}

function normalizeCourseSource(value = '') {
  const normalized = normalizeText(value)
  if (!normalized) return ''
  const match = normalized.match(/(course_[a-z_]+)/i)
  return match ? match[1].toLowerCase() : normalized.toLowerCase()
}

function matchRule(value = '', rules = []) {
  const normalized = normalizeText(value)
  if (!normalized) return ''
  const hit = rules.find(({ pattern }) => pattern.test(normalized))
  return hit?.title || ''
}

function resolveCourseHint(value = '') {
  const normalized = normalizeText(value)
  if (!normalized) return ''

  const sourceTitle = getCourseTitleFromSource(normalized)
  if (sourceTitle) return sourceTitle

  if (Object.values(COURSE_SOURCE_TITLE_MAP).includes(normalized)) {
    return normalized
  }

  if (LEGACY_COURSE_TITLE_ALIASES[normalized]) {
    return LEGACY_COURSE_TITLE_ALIASES[normalized]
  }

  return matchRule(normalized, COURSE_TITLE_KEYWORD_RULES)
}

export function getCourseTitleFromSource(value = '') {
  const sourceKey = normalizeCourseSource(value)
  return COURSE_SOURCE_TITLE_MAP[sourceKey] || ''
}

export function normalizeCourseTitle(value = '', fallback = DEFAULT_COURSE_TITLE) {
  const normalized = normalizeText(value)
  if (!normalized) return fallback

  const hint = resolveCourseHint(normalized)
  if (hint) return hint

  return normalized || fallback
}

export function resolveNodeCourseTitle(node = {}, fallback = DEFAULT_COURSE_TITLE) {
  const sourceCandidates = [node.curatedSource, node.sourcePath, node.source, node.courseSource]

  for (const candidate of sourceCandidates) {
    const title = resolveCourseHint(candidate)
    if (title) return title
  }

  const fieldCandidates = [node.courseTitle, node.chapterTitle]
  for (const candidate of fieldCandidates) {
    const title = resolveCourseHint(candidate)
    if (title) return title
  }

  const normalized = normalizeText(node.courseTitle || node.chapterTitle)
  return normalized || fallback
}
