const CARD_TYPE_META = {
  concept: {
    label: '概念卡',
    graphRoleTitle: '图谱负责把概念放回知识链',
    graphRoleDescription: '概念型知识点最容易出现“定义看懂了，但放回路径里又不会用”的问题，所以图谱重点负责先修和后续连接。',
  },
  method: {
    label: '方法卡',
    graphRoleTitle: '图谱负责串起方法的前提和步骤',
    graphRoleDescription: '方法型知识点不只看定义，更要看它依赖什么条件、适用于什么场景，以及做完后能通向哪些能力。',
  },
  tool: {
    label: '工具卡',
    graphRoleTitle: '图谱负责把工具接到方法和任务',
    graphRoleDescription: '工具型知识点如果只记 API 很快会碎掉，图谱会告诉学生这个工具服务于哪条方法链、最终落在哪类任务上。',
  },
  application: {
    label: '应用卡',
    graphRoleTitle: '图谱负责把任务拆回能力路径',
    graphRoleDescription: '应用型知识点本质上是多个概念、方法、工具的汇合点，图谱负责把任务拆回可补的知识路径。',
  },
}

const KNOWLEDGE_META = {
  derivative: {
    cardType: 'concept',
    learningGoal: '能解释变化率含义，并看懂它为什么会出现在梯度下降和反向传播里。',
  },
  'partial-derivative': {
    cardType: 'concept',
    learningGoal: '能区分偏导、梯度和方向变化，并知道它们怎样驱动参数更新。',
  },
  'matrix-ops': {
    cardType: 'tool',
    learningGoal: '能读懂向量和矩阵计算过程，并在模型计算里定位维度与变换关系。',
  },
  probability: {
    cardType: 'concept',
    learningGoal: '能用概率视角理解不确定性、分布和模型输出。',
  },
  optimization: {
    cardType: 'method',
    learningGoal: '能把训练问题看成优化问题，知道目标、约束和常见求解思路。',
  },
  'python-data': {
    cardType: 'tool',
    learningGoal: '能用 Pandas / NumPy 处理模型输入输出，并整理成稳定的数据流程。',
  },
  'numpy-sci': {
    cardType: 'tool',
    learningGoal: '能用 NumPy 完成基础张量与数组计算，并理解底层运算如何影响性能和调试。',
  },
  'dl-framework': {
    cardType: 'tool',
    learningGoal: '能在框架里搭起最小训练流程，把理论推导落到实验代码上。',
  },
  'model-deploy': {
    cardType: 'application',
    learningGoal: '能把训练好的模型变成可部署的服务，并理解线上推理链路。',
  },
  'gradient-descent': {
    cardType: 'method',
    learningGoal: '能说清更新公式、学习率作用和它与损失函数、偏导数的关系。',
  },
  'loss-function': {
    cardType: 'concept',
    learningGoal: '能解释损失函数衡量什么，以及它为什么决定训练方向。',
  },
  'linear-logistic': {
    cardType: 'method',
    learningGoal: '能跑通最基础监督学习流程，并用它理解训练闭环。',
  },
  'decision-tree': {
    cardType: 'method',
    learningGoal: '能说明树模型怎样划分特征空间，以及集成方法为什么有效。',
  },
  'model-eval': {
    cardType: 'method',
    learningGoal: '能判断一个模型是否真的学会了，而不是只在训练集上表现好。',
  },
  'nn-basic': {
    cardType: 'concept',
    learningGoal: '能解释神经网络的基本结构、激活函数和前向计算流程。',
  },
  backprop: {
    cardType: 'method',
    learningGoal: '能顺着计算图解释梯度是怎么反向传播回每一层参数的。',
  },
  cnn: {
    cardType: 'method',
    learningGoal: '能说清卷积、池化和特征提取为什么适合图像任务。',
  },
  rnn: {
    cardType: 'method',
    learningGoal: '能理解序列状态如何传递，并知道它为什么会遇到长依赖问题。',
  },
  attention: {
    cardType: 'concept',
    learningGoal: '能解释注意力如何重分配信息权重，以及它为何成为 Transformer 核心。',
  },
  transformer: {
    cardType: 'method',
    learningGoal: '能把自注意力、多头机制和编码结构串成一套完整理解。',
  },
  'pretrain-finetune': {
    cardType: 'method',
    learningGoal: '能解释大模型能力从哪里来，以及预训练和微调分别解决什么问题。',
  },
  rag: {
    cardType: 'application',
    learningGoal: '能把检索、知识组织和生成串成一个可执行应用链路。',
  },
  'prompt-eng': {
    cardType: 'tool',
    learningGoal: '能设计一组稳定提示词，让模型在特定任务里输出更可控。',
  },
  'agent-tool': {
    cardType: 'application',
    learningGoal: '能理解智能体何时调用工具、如何规划步骤以及如何闭环反馈。',
  },
  'graph-basic': {
    cardType: 'concept',
    learningGoal: '能看懂节点、边、路径和遍历在知识组织里的作用。',
  },
  'entity-relation': {
    cardType: 'method',
    learningGoal: '能把文本内容抽成实体、关系和三元组结构。',
  },
  'knowledge-graph': {
    cardType: 'application',
    learningGoal: '能理解从知识抽取、关系组织到图查询的完整闭环。',
  },
  'image-classify': {
    cardType: 'application',
    learningGoal: '能完成一个最小图像分类项目，并把模型链路讲清楚。',
  },
  sentiment: {
    cardType: 'application',
    learningGoal: '能把文本分类任务从数据处理、建模到评估完整走通一次。',
  },
  recsys: {
    cardType: 'application',
    learningGoal: '能理解推荐系统从召回、排序到反馈闭环的基本结构。',
  },
  'llm-app': {
    cardType: 'application',
    learningGoal: '能把提示词、检索和工具调用组合成一个可交付的大模型应用。',
  },
}

function buildMasteryChecklist(pointName, cardType, learningGoal) {
  switch (cardType) {
    case 'concept':
      return [
        { title: '能解释概念', description: `能用自己的话说明「${pointName}」是什么、解决什么问题。`, milestone: '先把定义和作用讲明白' },
        { title: '能辨认关键点', description: `看到公式、图示或术语时，能认出它和「${pointName}」的关系。`, milestone: '基础判断要稳定' },
        { title: '能举最小例子', description: `能用一个最小例子说明「${pointName}」怎么出现、为什么重要。`, milestone: '开始具备应用感' },
        { title: '能连到上下游', description: `能说出它依赖什么、学完后能接到哪里。`, milestone: '图谱关系开始打通' },
      ]
    case 'tool':
      return [
        { title: '知道工具价值', description: `知道「${pointName}」主要解决哪类实现问题。`, milestone: '先建立用途感' },
        { title: '能完成基础操作', description: `能独立完成最常见的基础 API / 操作。`, milestone: '会做最小操作' },
        { title: '能接进流程', description: `能把「${pointName}」放进一个最小数据或训练流程里。`, milestone: '开始服务真实任务' },
        { title: '能说明和方法关系', description: `能说清它服务于哪条方法链和哪类任务。`, milestone: '不再只记工具本身' },
      ]
    case 'application':
      return [
        { title: '知道任务目标', description: `知道「${pointName}」要完成什么任务、用什么指标判断好坏。`, milestone: '先明确任务边界' },
        { title: '能拆出所需模块', description: `能说出这个任务依赖哪些概念、方法和工具。`, milestone: '知道自己缺哪一块' },
        { title: '能做最小版本', description: `能搭出一个最小可运行版本，而不是只会看案例。`, milestone: '开始真正落地' },
        { title: '能复盘与优化', description: `能解释结果为什么好/不好，并知道下一步优化方向。`, milestone: '进入可迭代阶段' },
      ]
    default:
      return [
        { title: '知道方法目标', description: `知道「${pointName}」的目标、输入和输出是什么。`, milestone: '先弄清方法边界' },
        { title: '能走通关键步骤', description: `能说清方法的核心步骤，而不是只记名字。`, milestone: '核心流程要顺' },
        { title: '能做简单应用', description: `能在一个简单例子里把它实际用出来。`, milestone: '开始脱离纸面理解' },
        { title: '能迁移到路径', description: `${learningGoal || `能把「${pointName}」和前置、后续知识连接起来。`}`, milestone: '能接进整条学习路径' },
      ]
  }
}

function enrichPoint(domain, point) {
  const meta = KNOWLEDGE_META[point.id] || {}
  const cardType = meta.cardType || 'concept'
  const typeMeta = CARD_TYPE_META[cardType]

  return {
    ...point,
    cardType,
    cardTypeLabel: typeMeta.label,
    learningGoal: meta.learningGoal || `能完成和「${point.name}」相关的最小学习目标。`,
    graphRoleTitle: typeMeta.graphRoleTitle,
    graphRoleDescription: typeMeta.graphRoleDescription,
    masteryChecklist: buildMasteryChecklist(point.name, cardType, meta.learningGoal),
    domainName: domain.name,
  }
}

const rawDomains = [
  {
    id: 'math',
    name: '数学基础',
    desc: '模型训练和算法理解的底层数学工具',
    points: [
      { id: 'derivative', name: '导数与微分', level: 'foundation', levelLabel: '基础', mastery: 72, description: '变化率的数学工具，梯度下降和反向传播的底层支撑。', route: '/knowledge/derivative' },
      { id: 'partial-derivative', name: '偏导数与梯度', level: 'foundation', levelLabel: '基础', mastery: 48, description: '多元函数的方向变化率，模型参数更新的关键。', recommend: '补弱推荐', route: '/knowledge/partial-derivative' },
      { id: 'matrix-ops', name: '矩阵运算与线性代数', level: 'foundation', levelLabel: '基础', mastery: 65, description: '向量空间与线性变换，神经网络计算的数学框架。', route: '/knowledge/matrix-ops' },
      { id: 'probability', name: '概率论基础', level: 'foundation', levelLabel: '基础', mastery: 58, description: '随机变量与分布，贝叶斯推断和生成模型的基石。', route: '/knowledge/probability' },
      { id: 'optimization', name: '最优化方法', level: 'core', levelLabel: '核心', mastery: 40, description: '凸优化、约束优化，模型训练本质上是一个优化问题。', recommend: '补弱推荐', route: '/knowledge/optimization' },
    ],
  },
  {
    id: 'python',
    name: 'Python 工程化',
    desc: '从脚本到工程化的编程能力',
    points: [
      { id: 'python-data', name: 'Python 数据处理', level: 'foundation', levelLabel: '基础', mastery: 70, description: 'Pandas/NumPy 数据操作，模型输入输出的标准处理流程。', route: '/knowledge/python-data' },
      { id: 'numpy-sci', name: 'NumPy 与科学计算', level: 'foundation', levelLabel: '基础', mastery: 62, description: '高效矩阵运算库，理解底层运算对调试模型很重要。', route: '/knowledge/numpy-sci' },
      { id: 'dl-framework', name: '深度学习框架入门', level: 'core', levelLabel: '核心', mastery: 45, description: 'PyTorch/TensorFlow 基本用法，从理论到实验的桥梁。', route: '/knowledge/dl-framework' },
      { id: 'model-deploy', name: '模型工程化部署', level: 'advanced', levelLabel: '进阶', mastery: 18, description: 'ONNX、TorchServe 等方案，让模型从实验走向生产。', route: '/knowledge/model-deploy' },
    ],
  },
  {
    id: 'ml',
    name: '机器学习',
    desc: '经典模型与训练方法论',
    points: [
      { id: 'gradient-descent', name: '梯度下降', level: 'core', levelLabel: '核心', mastery: 52, description: '模型训练的核心优化算法，连接数学基础与深度学习。', recommend: '补弱推荐', route: '/knowledge/gradient-descent' },
      { id: 'loss-function', name: '损失函数', level: 'core', levelLabel: '核心', mastery: 56, description: '衡量模型预测与真实值差距的标准，驱动参数更新方向。', route: '/knowledge/loss-function' },
      { id: 'linear-logistic', name: '线性回归与逻辑回归', level: 'foundation', levelLabel: '基础', mastery: 68, description: '最基础的监督学习模型，理解模型训练全流程的起点。', route: '/knowledge/linear-logistic' },
      { id: 'decision-tree', name: '决策树与集成方法', level: 'core', levelLabel: '核心', mastery: 42, description: '树模型与 Bagging/Boosting，工业界常用的非深度学习方案。', route: '/knowledge/decision-tree' },
      { id: 'model-eval', name: '模型评估与调参', level: 'core', levelLabel: '核心', mastery: 38, description: '交叉验证、超参搜索，确保模型泛化能力的关键环节。', route: '/knowledge/model-eval' },
    ],
  },
  {
    id: 'dl',
    name: '深度学习',
    desc: '神经网络架构与训练技术',
    points: [
      { id: 'nn-basic', name: '神经网络基础', level: 'foundation', levelLabel: '基础', mastery: 55, description: '前馈网络、激活函数，所有深度模型的起点。', route: '/knowledge/nn-basic' },
      { id: 'backprop', name: '反向传播算法', level: 'core', levelLabel: '核心', mastery: 44, description: '链式求导自动求梯度，神经网络训练的计算引擎。', route: '/knowledge/backprop' },
      { id: 'cnn', name: '卷积神经网络 CNN', level: 'core', levelLabel: '核心', mastery: 66, description: '图像任务的核心架构，从数学到视觉任务的完整主线。', recommend: '进阶推荐', route: '/knowledge/cnn' },
      { id: 'rnn', name: '循环神经网络 RNN', level: 'core', levelLabel: '核心', mastery: 35, description: '序列建模的经典方案，理解时序数据处理的基础。', route: '/knowledge/rnn' },
      { id: 'attention', name: '注意力机制', level: 'core', levelLabel: '核心', mastery: 30, description: '从 RNN 到 Transformer 的关键突破，大模型的核心组件。', route: '/knowledge/attention' },
    ],
  },
  {
    id: 'llm',
    name: '大模型',
    desc: '预训练语言模型与应用技术',
    points: [
      { id: 'transformer', name: 'Transformer 架构', level: 'core', levelLabel: '核心', mastery: 25, description: '自注意力驱动的序列模型，当代 NLP 和多模态的基础。', route: '/knowledge/transformer' },
      { id: 'pretrain-finetune', name: '预训练与微调', level: 'core', levelLabel: '核心', mastery: 20, description: '大规模预训练 + 下游任务微调的范式，理解大模型能力来源。', route: '/knowledge/pretrain-finetune' },
      { id: 'rag', name: '检索增强生成 RAG', level: 'advanced', levelLabel: '进阶', mastery: 34, description: '结合检索与生成的应用模式，知识密集型任务的主流方案。', recommend: '场景推荐', route: '/knowledge/rag' },
      { id: 'prompt-eng', name: 'Prompt Engineering', level: 'advanced', levelLabel: '进阶', mastery: 28, description: '通过提示词设计引导模型输出，低成本利用大模型能力。', route: '/knowledge/prompt-eng' },
      { id: 'agent-tool', name: 'Agent 与工具调用', level: 'frontier', levelLabel: '前沿', mastery: 12, description: '让大模型调用外部工具完成复杂任务，智能体应用的前沿方向。', route: '/knowledge/agent-tool' },
    ],
  },
  {
    id: 'kg',
    name: '知识图谱',
    desc: '结构化知识表示与推理',
    points: [
      { id: 'graph-basic', name: '图结构与图算法', level: 'foundation', levelLabel: '基础', mastery: 45, description: '节点-边数据结构与遍历算法，理解知识图谱的数据基础。', route: '/knowledge/graph-basic' },
      { id: 'entity-relation', name: '实体关系建模', level: 'core', levelLabel: '核心', mastery: 38, description: '三元组抽取与关系分类，知识图谱构建的核心技术。', route: '/knowledge/entity-relation' },
      { id: 'knowledge-graph', name: '知识图谱构建与查询', level: 'core', levelLabel: '核心', mastery: 41, description: '从原始文本到结构化图谱的完整流程，本项目的底层能力。', route: '/knowledge/knowledge-graph' },
    ],
  },
  {
    id: 'practice',
    name: '行业实践',
    desc: '从理论到真实场景的落地',
    points: [
      {
        id: 'image-classify',
        name: '图像分类项目',
        level: 'practice',
        levelLabel: '实践',
        mastery: 22,
        description: '用 CNN 完成端到端图像分类，串联数据处理到模型部署。',
        route: '/knowledge/image-classify',
        projectFit: { dl: 0.42, ml: 0.28, python: 0.22, math: 0.08 },
      },
      {
        id: 'sentiment',
        name: '文本情感分析',
        level: 'practice',
        levelLabel: '实践',
        mastery: 15,
        description: '用 NLP 模型判断文本情感，理解文本分类任务全流程。',
        route: '/knowledge/sentiment',
        projectFit: { llm: 0.38, ml: 0.28, python: 0.24, math: 0.1 },
      },
      {
        id: 'recsys',
        name: '推荐系统实践',
        level: 'practice',
        levelLabel: '实践',
        mastery: 10,
        description: '协同过滤到深度推荐，理解个性化推荐的工程实现。',
        route: '/knowledge/recsys',
        projectFit: { ml: 0.36, kg: 0.32, math: 0.14, python: 0.18 },
      },
      {
        id: 'llm-app',
        name: '大模型应用开发',
        level: 'practice',
        levelLabel: '实践',
        mastery: 8,
        description: '调用大模型 API 构建实际应用，RAG + Agent 的工程落地。',
        route: '/knowledge/llm-app',
        projectFit: { llm: 0.4, kg: 0.26, python: 0.22, ml: 0.12 },
      },
    ],
  },
]

export const domains = rawDomains.map((domain) => ({
  ...domain,
  points: domain.points.map((point) => enrichPoint(domain, point)),
}))
