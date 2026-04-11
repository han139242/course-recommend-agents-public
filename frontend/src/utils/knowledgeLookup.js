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

function normalizeLookupKey(value = '') {
  return String(value || '').trim().toLowerCase().replace(/\s+/g, '')
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
