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

const GENERIC_COURSE_HINT_PATTERNS = [
  /人工智能基础知识骨架/i,
  /人工智能课程知识图谱/i,
  /curated_ai_professional_ontology/i,
]

const LEGACY_COURSE_TITLE_ALIASES = {
  人工智能在游戏设计中的应用: '人工智能导论',
  智能体与多智能体系统: '人工智能导论',
  智能计算及其应用: '人工智能导论',
  绪论: '人工智能导论',
  人工神经网络及其应用: '深度学习',
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
  {
    pattern: /知识图谱|knowledge[_\s-]*graph|semantic[_\s-]*web|rdf|owl|graph[_\s-]*database|neo4j|本体|语义网/i,
    title: '知识图谱',
  },
  {
    pattern: /强化学习|reinforcement[_\s-]*learning|\brl\b|马尔可夫决策过程|mdp|策略梯度|policy[_\s-]*gradient|actor[_\s-]*critic|q[_\s-]*learning|q学习|dqn|ppo|off[_\s-]*policy|on[_\s-]*policy|回报|动作价值函数|状态价值函数|价值函数|状态表示|智能体-环境交互/i,
    title: '强化学习',
  },
  {
    pattern: /计算机视觉|computer[_\s-]*vision|\bcv\b|vision[_\s-]*transformer|visual[_\s-]*transformer|\bvit\b|clip|swin|deit|mae|image|video|图像|视觉|目标检测|图像分割|图像分类|图像描述|多模态/i,
    title: '计算机视觉',
  },
  {
    pattern: /自然语言处理|nlp|bert|machine[_\s-]*translation|translation|speech|question[_\s-]*answering|信息抽取|机器翻译|语音识别|问答系统|文本|语言|序列标注|文本分类/i,
    title: '自然语言处理',
  },
  {
    pattern: /大模型|\bllm\b|transformer|rag|prompt|language[_\s-]*model|instruction[_\s-]*tuning|alignment|in[_\s-]*context/i,
    title: '大模型',
  },
  {
    pattern: /深度学习|deep[_\s-]*learning|neural[_\s-]*network|cnn|rnn|gan|autoencoder|感知机|表示学习|自监督学习|迁移学习|神经网络/i,
    title: '深度学习',
  },
  {
    pattern: /ai[_\s-]*engineering|mlops|serving|deployment|compression|edge[_\s-]*ai|onnx|inference/i,
    title: 'AI工程',
  },
  {
    pattern: /机器学习|machine[_\s-]*learning|automl|regression|classification|clustering|decision[_\s-]*tree|bayes|k[-_\s]*means|svm|判别式模型|生成式模型|回归|聚类|朴素贝叶斯|决策树|最优化|数据集划分|模型评估/i,
    title: '机器学习',
  },
  {
    pattern: /线性代数|linear[_\s-]*algebra|矩阵|向量|特征值|特征向量/i,
    title: '线性代数',
  },
  {
    pattern: /概率.*统计|统计.*概率|概率统计|probability|statistics|贝叶斯|随机变量|概率分布/i,
    title: '概率与统计',
  },
  {
    pattern: /高等数学|微积分|数学分析|calculus|导数|梯度/i,
    title: '高等数学',
  },
  {
    pattern: /python/i,
    title: 'Python编程',
  },
  {
    pattern: /数据结构|data[_\s-]*structure|链表|栈|队列|树|图算法/i,
    title: '数据结构',
  },
  {
    pattern: /ai[_\s-]*ethics|可信ai|负责任ai|算法治理|ethics|fairness|privacy|safety|xai|governance/i,
    title: 'AI伦理',
  },
  {
    pattern: /人工智能导论|知识表示|推理|搜索|规划|专家系统|多智能体|博弈搜索|启发式搜索|A\*|reasoning|search|planning|expert[_\s-]*system|multi[_\s-]*agent|game[_\s-]*ai|intelligent[_\s-]*computing/i,
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

function isGenericCourseHint(value = '') {
  const normalized = normalizeText(value)
  if (!normalized) return false
  return GENERIC_COURSE_HINT_PATTERNS.some((pattern) => pattern.test(normalized))
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

  if (isGenericCourseHint(normalized)) {
    return ''
  }

  if (LEGACY_COURSE_TITLE_ALIASES[normalized]) {
    return LEGACY_COURSE_TITLE_ALIASES[normalized]
  }

  return matchRule(normalized, COURSE_TITLE_KEYWORD_RULES)
}

function buildNodeKeywordText(node = {}) {
  return [
    node.canonicalName,
    node.name,
    node.description,
    isGenericCourseHint(node.sourcePath) ? '' : node.sourcePath,
    isGenericCourseHint(node.chapterTitle) ? '' : node.chapterTitle,
    isGenericCourseHint(node.courseTitle) ? '' : node.courseTitle,
  ]
    .map((value) => normalizeText(value))
    .filter(Boolean)
    .join(' ')
}

export function getCourseTitleFromSource(value = '') {
  const sourceKey = normalizeCourseSource(value)
  return COURSE_SOURCE_TITLE_MAP[sourceKey] || ''
}

export function normalizeCourseTitle(value = '', fallback = DEFAULT_COURSE_TITLE) {
  const normalized = normalizeText(value)
  if (!normalized) return fallback
  if (isGenericCourseHint(normalized)) return fallback

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

  const nodeKeywordText = buildNodeKeywordText(node)
  const keywordTitle = matchRule(nodeKeywordText, COURSE_TITLE_KEYWORD_RULES)
  if (keywordTitle) return keywordTitle

  const normalizedCourseTitle = normalizeText(node.courseTitle)
  if (normalizedCourseTitle && !isGenericCourseHint(normalizedCourseTitle)) {
    return normalizedCourseTitle
  }

  const normalizedChapterTitle = normalizeText(node.chapterTitle)
  if (normalizedChapterTitle && !isGenericCourseHint(normalizedChapterTitle)) {
    return normalizedChapterTitle
  }

  return fallback
}
