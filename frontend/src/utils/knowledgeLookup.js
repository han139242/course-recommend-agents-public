import { domains } from '../data/courseData'

const MANUAL_ALIASES = {
  derivative: ['导数', '导数与微分'],
  'partial-derivative': ['偏导数', '梯度', '偏导数与梯度'],
  'matrix-ops': ['矩阵', '矩阵乘法', '矩阵运算与线性代数'],
  optimization: ['最优化问题', '最优化方法', '优化方法'],
  cnn: ['卷积神经网络', '卷积神经网络 CNN'],
  rag: ['检索增强生成', 'RAG', '检索增强生成 RAG'],
  'knowledge-graph': ['知识图谱', '知识图谱构建与查询', '知识图谱建模'],
  'graph-basic': ['图结构', '图结构与图算法'],
  'entity-relation': ['实体关系', '实体关系建模'],
  'python-data': ['Python 数据处理', 'Python数据处理'],
  'numpy-sci': ['NumPy 与科学计算', 'NumPy科学计算', '科学计算'],
  'dl-framework': ['深度学习框架入门', 'PyTorch', 'TensorFlow'],
  'model-deploy': ['模型工程化部署', '模型部署'],
  'gradient-descent': ['梯度下降'],
  'loss-function': ['损失函数'],
  'linear-logistic': ['线性回归与逻辑回归', '线性回归', '逻辑回归'],
  'decision-tree': ['决策树与集成方法', '决策树', '集成方法'],
  'model-eval': ['模型评估与调参', '模型评估', '调参'],
  'nn-basic': ['神经网络基础'],
  backprop: ['反向传播', '反向传播算法'],
  rnn: ['循环神经网络', '循环神经网络 RNN'],
  attention: ['注意力机制'],
  transformer: ['Transformer', 'Transformer 架构'],
  'pretrain-finetune': ['预训练与微调'],
  'prompt-eng': ['Prompt Engineering', '提示词工程'],
  'agent-tool': ['Agent 与工具调用', 'Agent', '工具调用'],
  'image-classify': ['图像分类项目'],
  sentiment: ['文本情感分析'],
  recsys: ['推荐系统实践', '推荐系统'],
  'llm-app': ['大模型应用开发'],
}

const EXTERNAL_NODE_ALIASES = [
  { routeKnowledgeId: 'llm-app', name: '大语言模型', canonicalName: '大语言模型', chapterTitle: '大模型', terms: ['large language model', 'llm', 'foundation model'] },
  { routeKnowledgeId: 'transformer', name: 'Transformer 架构', canonicalName: 'Transformer 架构', chapterTitle: '大模型', terms: ['transformer'] },
  { routeKnowledgeId: 'attention', name: '注意力机制', canonicalName: '注意力机制', chapterTitle: '深度学习', terms: ['attention mechanism', 'self attention', 'self-attention'] },
  { routeKnowledgeId: 'rag', name: '检索增强生成 RAG', canonicalName: '检索增强生成 RAG', chapterTitle: '大模型', terms: ['retrieval augmented generation', 'retrieval-augmented generation', 'rag'] },
  { routeKnowledgeId: 'agent-tool', name: 'Agent 与工具调用', canonicalName: 'Agent 与工具调用', chapterTitle: '大模型', terms: ['agent', 'tool use', 'multi agent', 'multi-agent'] },
  { routeKnowledgeId: 'cnn', name: '卷积神经网络 CNN', canonicalName: '卷积神经网络 CNN', chapterTitle: '深度学习', terms: ['convolutional neural network', 'cnn'] },
  { routeKnowledgeId: 'rnn', name: '循环神经网络 RNN', canonicalName: '循环神经网络 RNN', chapterTitle: '深度学习', terms: ['recurrent neural network', 'long short term memory', 'lstm'] },
  { routeKnowledgeId: 'nn-basic', name: '神经网络基础', canonicalName: '神经网络基础', chapterTitle: '深度学习', terms: ['artificial neural network', 'deep neural network', 'neural network'] },
  { routeKnowledgeId: 'decision-tree', name: '决策树与集成方法', canonicalName: '决策树与集成方法', chapterTitle: '机器学习', terms: ['decision tree', 'ensemble learning'] },
  { routeKnowledgeId: 'linear-logistic', name: '线性回归与逻辑回归', canonicalName: '线性回归与逻辑回归', chapterTitle: '机器学习', terms: ['linear regression', 'logistic regression'] },
  { routeKnowledgeId: 'gradient-descent', name: '梯度下降', canonicalName: '梯度下降', chapterTitle: '机器学习', terms: ['gradient descent', 'optimization'] },
  { routeKnowledgeId: 'loss-function', name: '损失函数', canonicalName: '损失函数', chapterTitle: '机器学习', terms: ['softmax function', 'sigmoid function', 'relu function', 'activation function'] },
  { routeKnowledgeId: 'knowledge-graph', name: '知识图谱构建与查询', canonicalName: '知识图谱构建与查询', chapterTitle: '知识图谱', terms: ['knowledge graph', 'graph reasoning'] },
  { routeKnowledgeId: 'sentiment', name: '自然语言处理', canonicalName: '自然语言处理', chapterTitle: '行业实践', terms: ['natural language processing', 'nlp'] },
]

const DISPLAY_NAME_OVERRIDES = {
  a2c: '优势 Actor-Critic（A2C）',
  a3c: '异步优势 Actor-Critic（A3C）',
  'actor critic': '演员-评论家算法（Actor-Critic）',
  'actor-critic': '演员-评论家算法（Actor-Critic）',
  adaboost: '自适应提升算法（AdaBoost）',
  adagrad: '自适应梯度算法（AdaGrad）',
  adam: 'Adam 优化器',
  adamw: 'AdamW 优化器',
  adapter: '适配器微调（Adapter）',
  'ai for science': 'AI for Science 科学智能',
  auc: 'AUC 评估指标',
  bagging: '装袋集成（Bagging）',
  bleu: 'BLEU 机器翻译指标',
  boosting: '提升集成（Boosting）',
  'constitutional ai': '宪法式 AI（Constitutional AI）',
  dataframe: '数据表对象（DataFrame）',
  dbscan: '密度聚类算法（DBSCAN）',
  'deep rl': '深度强化学习（Deep RL）',
  'double dqn': '双重 DQN（Double DQN）',
  dropout: '随机失活（Dropout）',
  'dueling dqn': '竞争网络 DQN（Dueling DQN）',
  'elastic net': '弹性网络回归（Elastic Net）',
  'eu ai act': '欧盟人工智能法案（EU AI Act）',
  gil: '全局解释器锁（GIL）',
  'gradient descent': '梯度下降',
  'large language model': '大语言模型',
  'ml pipeline': '机器学习流水线（ML Pipeline）',
  mlops: '机器学习运维（MLOps）',
  momentum: '动量优化（Momentum）',
  ndarray: '多维数组（ndarray）',
  noisynet: '噪声网络（NoisyNet）',
  pagedattention: '分页注意力（PagedAttention）',
  'plan and execute': '规划执行范式（Plan-and-Execute）',
  'plan-and-execute': '规划执行范式（Plan-and-Execute）',
  ppo: '近端策略优化（PPO）',
  'p tuning': '提示调优（P-Tuning）',
  'p-tuning': '提示调优（P-Tuning）',
  reinforce: 'REINFORCE 策略梯度',
  rlaif: 'AI 反馈强化学习（RLAIF）',
  rmsprop: 'RMSProp 优化器',
  rouge: 'ROUGE 文本生成指标',
  sac: '软 Actor-Critic（SAC）',
  sarsa: 'SARSA 强化学习算法',
  'self consistency': '自洽推理（Self-Consistency）',
  'self-consistency': '自洽推理（Self-Consistency）',
  series: '序列对象（Series）',
  shap: 'SHAP 可解释性方法',
  'stable diffusion': '稳定扩散模型（Stable Diffusion）',
  stacking: '堆叠集成（Stacking）',
  't sne': 't-SNE 降维可视化',
  't-sne': 't-SNE 降维可视化',
  'tf idf': 'TF-IDF 词项权重',
  'tf-idf': 'TF-IDF 词项权重',
  transformer: 'Transformer 架构',
  trpo: '信赖域策略优化（TRPO）',
  warmup: '学习率预热（Warmup）',
  'world model': '世界模型（World Model）',
  zero: 'ZeRO 分布式优化',
}

function normalizeLookupKey(value = '') {
  return String(value || '').trim().toLowerCase().replace(/\s+/g, '')
}

function normalizeExternalNodeKey(value = '') {
  return String(value || '')
    .trim()
    .replace(/^aio:/i, '')
    .toLowerCase()
    .replace(/[_\-:（）(),./]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function hasChinese(value = '') {
  return /[\u4e00-\u9fff]/.test(String(value || ''))
}

function isPlainEnglishName(value = '') {
  const text = String(value || '').trim()
  return Boolean(text && !hasChinese(text) && /[A-Za-z]/.test(text) && /^[A-Za-z0-9\s\-+_./'&]+$/.test(text))
}

function displayNameSuffix(node = {}) {
  const title = String(node.chapterTitle || '').trim()
  if (/python/i.test(title)) return 'Python 知识点'
  if (title.includes('机器学习')) return '机器学习方法'
  if (title.includes('深度学习')) return '深度学习模型'
  if (title.includes('自然语言')) return '自然语言处理技术'
  if (title.includes('计算机视觉')) return '计算机视觉方法'
  if (title.includes('强化学习')) return '强化学习算法'
  if (title.includes('知识图谱')) return '知识图谱技术'
  if (title.includes('大模型')) return '大模型技术'
  if (title.includes('AI工程') || title.includes('AI 工程')) return 'AI 工程工具'
  if (title.includes('AI伦理') || title.includes('AI 伦理')) return 'AI 伦理主题'
  if (/ai for science/i.test(title)) return '科学智能技术'
  if (title.includes('人工智能')) return '人工智能方法'
  return '课程知识点'
}

export function localizeKnowledgeDisplayName(value = '', node = {}) {
  const text = String(value || '').trim()
  if (!text || hasChinese(text)) return text
  const override = DISPLAY_NAME_OVERRIDES[normalizeExternalNodeKey(text)]
  if (override) return override
  if (!isPlainEnglishName(text)) return text
  return `${text}（${displayNameSuffix(node)}）`
}

function isExternalOntologyNode(node = {}) {
  const text = [
    node.id,
    node.routeKnowledgeId,
    node.name,
    node.canonicalName,
    node.sourcePath,
    node.curatedSource,
    node.chapterTitle,
  ].join(' ').toLowerCase()
  return text.includes('aio:') || text.includes('aio ontology')
}

export function canonicalizeExternalKnowledgeNode(node = {}) {
  if (!isExternalOntologyNode(node)) return { hidden: false }

  const candidates = [node.id, node.routeKnowledgeId, node.name, node.canonicalName, node.description]
    .map((item) => normalizeExternalNodeKey(item))
    .filter(Boolean)

  const alias = EXTERNAL_NODE_ALIASES.find((rule) =>
    rule.terms.some((term) => {
      const key = normalizeExternalNodeKey(term)
      return candidates.some((candidate) => candidate === key || candidate.includes(key))
    }),
  )

  if (!alias) return { hidden: true }
  return {
    hidden: false,
    routeKnowledgeId: alias.routeKnowledgeId,
    name: alias.name,
    canonicalName: alias.canonicalName,
    chapterTitle: alias.chapterTitle,
  }
}

function buildKnowledgeCatalog() {
  const byRouteId = {}
  const aliasToRouteId = {}

  for (const domain of domains) {
    for (const point of domain.points) {
      const entry = {
        id: point.id,
        route: point.route,
        name: point.name,
        domainId: domain.id,
        domainName: domain.name,
        level: point.level,
        levelLabel: point.levelLabel,
        description: point.description,
        mastery: point.mastery,
      }

      byRouteId[point.id] = entry

      const aliases = new Set([
        point.id,
        point.name,
        point.route?.replace('/knowledge/', ''),
        ...(MANUAL_ALIASES[point.id] || []),
      ])

      for (const alias of aliases) {
        const key = normalizeLookupKey(alias)
        if (!key) continue
        aliasToRouteId[key] = point.id
      }
    }
  }

  return { byRouteId, aliasToRouteId }
}

const KNOWLEDGE_CATALOG = buildKnowledgeCatalog()

export function resolveKnowledgeCatalogEntry(value = '') {
  const raw = String(value || '').trim()
  if (!raw) return null

  return (
    KNOWLEDGE_CATALOG.byRouteId[raw] ||
    KNOWLEDGE_CATALOG.byRouteId[KNOWLEDGE_CATALOG.aliasToRouteId[normalizeLookupKey(raw)]] ||
    null
  )
}

export function buildKnowledgeLookupCandidates({ knowledgeId = '', currentPoint = null, detailSnapshot = null } = {}) {
  const entry = resolveKnowledgeCatalogEntry(knowledgeId) || resolveKnowledgeCatalogEntry(currentPoint?.name)
  const aliases = new Set([
    detailSnapshot?.routeKnowledgeId,
    detailSnapshot?.id,
    detailSnapshot?.name,
    detailSnapshot?.canonicalName,
    currentPoint?.id,
    currentPoint?.name,
    entry?.id,
    entry?.name,
    ...(MANUAL_ALIASES[entry?.id] || MANUAL_ALIASES[String(knowledgeId || '').trim()] || []),
    knowledgeId,
  ])

  return [...aliases]
    .map((item) => String(item || '').trim())
    .filter(Boolean)
}

export function resolveFriendlyKnowledgeRouteId(node = {}) {
  const routeCandidates = [
    node?.routeKnowledgeId,
    node?.knowledgeId,
    node?.slug,
    node?.id,
    node?.name,
    node?.canonicalName,
  ]

  for (const candidate of routeCandidates) {
    const entry = resolveKnowledgeCatalogEntry(candidate)
    if (entry) {
      return entry.id
    }
  }

  return String(node?.routeKnowledgeId || node?.knowledgeId || node?.slug || node?.id || '').trim()
}

export function resolveFriendlyKnowledgeRoute(route = '', title = '') {
  const routeId = resolveFriendlyKnowledgeRouteId({
    routeKnowledgeId: String(route || '').replace('/knowledge/', ''),
    name: title,
  })
  return routeId ? `/knowledge/${routeId}` : route
}
