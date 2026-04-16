export const navTabs = [
  { label: '知识学习', to: '/' },
  { label: 'AI 助学室', to: '/assistant' },
  { label: '下一步学习', to: '/graph' },
  { label: '能力主线', to: '/growth' },
]

export const profileData = {
  name: '韩治宇',
  role: '人工智能专业学生',
  target: '目标方向：算法工程师 / 智能体应用开发',
  summary: '当前处于“数学基础补强 + 深度学习进阶”阶段，推荐策略以补弱优先、场景穿插为主。',
  stats: [
    { label: '本周学习时长', value: '6.5h' },
    { label: '已掌握知识点', value: '18 / 26' },
    { label: '推荐命中率', value: '89%' },
    { label: '最近连续学习', value: '5 天' },
  ],
  goals: ['补齐偏导数与优化方法', '进入 CNN 训练主线', '完成图像分类项目实践'],
  recentRecords: [
    '完成“梯度下降”补弱学习，已经能接回损失函数和参数更新',
    '完成 2 组跨课程练习，数学基础错误率下降',
    '收到新的进阶推荐：卷积神经网络 CNN',
  ],
}

export const assistantPrompts = [
  '如果我想做算法工程师，下一步该补哪些知识点？',
  '我梯度下降总学不明白，应该回补哪些先修内容？',
  '怎样从机器学习过渡到大模型应用开发？',
]

export const assistantReplies = {
  '如果我想做算法工程师，下一步该补哪些知识点？':
    '如果你的目标是算法工程师，当前建议先补“梯度下降”和“损失函数”这组基础知识点，再进入“卷积神经网络 CNN”与模型训练，最后通过图像分类项目把模型理解和工程实现串起来。',
  '我梯度下降总学不明白，应该回补哪些先修内容？':
    '梯度下降学不明白，通常不是公式本身的问题，而是先修知识没打通。建议你先补“导数”“偏导数”“函数极值”这三个知识点，再回到损失函数和参数更新逻辑。',
  '怎样从机器学习过渡到大模型应用开发？':
    '你可以先完成机器学习和深度学习主线，再补“向量表示”“检索基础”，然后进入“RAG”与“工具调用”方向，这样从模型理解到大模型应用会更顺。',
}

export const stageTabs = [
  {
    id: 'learn',
    label: '学',
    title: '知识点导学',
    intro: '围绕当前目标知识点，自动串联先修内容、核心概念和关联知识，先学会再继续。',
    panelTitle: '当前学习目标',
    panelContent: '理解梯度下降为什么依赖导数与偏导，并能说明它与模型训练的关系。',
    feedbackLabel: '导学方式',
    feedbackContent: '先修唤醒 + 概念解释 + 关联知识点导航',
    tips: ['先学当前点', '补回缺失先修', '看懂关联关系'],
  },
  {
    id: 'practice',
    label: '练',
    title: '关联练习',
    intro: '练习不再只属于单一课程，而是围绕知识点掌握情况组织跨课程、分层次训练。',
    panelTitle: '当前练习任务',
    panelContent: '同时练习“偏导数计算”和“梯度下降更新公式”，验证数学基础与机器学习理解是否真正打通。',
    feedbackLabel: '练习反馈',
    feedbackContent: '错因自动回流到薄弱知识点，并推荐补弱内容',
    tips: ['基础题稳固', '跨课程联练', '错因回流补弱'],
  },
  {
    id: 'test',
    label: '测',
    title: '跨课程测评',
    intro: '测评聚焦真实掌握，而不是章节完成度，系统会用跨课程题组检查知识链接是否建立。',
    panelTitle: '测评结果',
    panelContent: '导数与梯度关系掌握良好，但优化器选择与学习率调节仍然薄弱，建议回补优化基础。',
    feedbackLabel: '评估输出',
    feedbackContent: '测后结论 + 薄弱点回流 + 推荐调整',
    tips: ['短测不打断学习', '关注真实掌握', '测后立即给建议'],
  },
  {
    id: 'recommend',
    label: '荐',
    title: '智能推荐',
    intro: '系统根据薄弱点、目标方向和兴趣偏好，动态生成补弱、进阶和场景化路径。',
    panelTitle: '推荐安排',
    panelContent: '当前建议优先补“优化方法”相关知识点，再进入“卷积网络训练”，最后完成一个图像分类项目。',
    feedbackLabel: '推荐依据',
    feedbackContent: '先修依赖 + 当前薄弱点 + 目标方向 + 场景偏好',
    tips: ['补弱推荐', '进阶推荐', '场景型推荐'],
  },
]

export const valueCards = [
  { title: '哪里不会，就从哪里开始', description: '系统会直接定位薄弱知识点，并推荐最需要回补的内容。' },
  { title: '不被课程顺序绑住', description: '学习顺序按知识点关系组织，而不是按固定章节线性推进。' },
  { title: '学完马上知道下一步', description: '每次学习后都会给出可执行的补弱、进阶或项目推荐。' },
]

export const abilityAxes = [
  { name: '数学基础', text: '导数、矩阵、概率基础', value: 72 },
  { name: '算法能力', text: '模型理解与训练能力', value: 78 },
  { name: '工程实践', text: '框架使用与项目落地', value: 61 },
]

export const curriculumFilters = [
  '数学基础',
  'Python 工程化',
  '机器学习',
  '深度学习',
  '大模型',
  '知识图谱',
  '行业实践',
]

export const recommendationStories = [
  {
    id: 'gradient-descent',
    title: '梯度下降',
    progress: 52,
    mastery: 52,
    tag: '需要先补偏导数与学习率理解',
    type: '补弱推荐',
    domain: '机器学习',
    storyTitle: '先补回模型训练的数学底层',
    storyDetail: '你的主要卡点不在模型公式本身，而在导数、偏导数与学习率如何共同决定参数更新。',
    nextStep: '先学习偏导数与损失函数，再回到梯度下降本体。',
    route: '/knowledge/gradient-descent',
  },
  {
    id: 'cnn',
    title: '卷积神经网络 CNN',
    progress: 66,
    mastery: 66,
    tag: '可从图像任务进入进阶学习',
    type: '进阶推荐',
    domain: '深度学习',
    storyTitle: '把算法理解推进到图像任务',
    storyDetail: '你已经具备基础模型理解能力，当前最适合通过 CNN 建立从数学到视觉任务的完整主线。',
    nextStep: '先复习梯度更新，再进入卷积、池化和特征提取。',
    route: '/knowledge/cnn',
  },
  {
    id: 'rag',
    title: '检索增强生成 RAG',
    progress: 34,
    mastery: 34,
    tag: '适合大模型应用方向继续探索',
    type: '场景推荐',
    domain: '大模型',
    storyTitle: '从模型学习延展到真实应用场景',
    storyDetail: '如果你的目标是智能体或大模型应用，RAG 是连接知识库、检索和生成能力的关键中间层。',
    nextStep: '补向量表示和检索基础，再进入 RAG 应用链路。',
    route: '/knowledge/rag',
  },
  {
    id: 'knowledge-graph',
    title: '知识图谱建模',
    progress: 41,
    mastery: 41,
    tag: '建议先补图结构与实体关系设计',
    type: '关联知识点',
    domain: '知识图谱',
    storyTitle: '理解系统如何组织跨课程知识',
    storyDetail: '当你开始进入推荐系统和学习安排设计时，知识图谱建模会帮助你看懂知识点之间的连接逻辑。',
    nextStep: '先理解图结构与实体关系，再进入图查询和推荐关联。',
    route: '/knowledge/knowledge-graph',
  },
]

export const knowledgeCards = recommendationStories.map((item) => ({
  id: item.id,
  title: item.title,
  progress: item.progress,
  tag: item.tag,
  type: item.type,
  domain: item.domain,
  route: item.route,
}))

export const quickStats = [
  { label: '本周学习时长', value: '6.5h' },
  { label: '已掌握知识点', value: '18 / 26' },
  { label: '推荐命中率', value: '89%' },
]

export const growthInsights = [
  { label: '数学基础', value: 69, note: '导数和矩阵基础对模型理解影响明显，建议优先补齐。' },
  { label: '算法理解', value: 81, note: '模型原理理解较好，可以增加进阶型推荐。' },
  { label: '场景应用', value: 57, note: '项目落地和行业场景关联还需要继续强化。' },
]

export const recommendedPath = recommendationStories.map((item) => ({
  stage: item.type,
  title: item.title,
  detail: item.storyDetail,
  route: item.route,
}))

export const graphNodes = [
  { id: 'derivative', label: '导数', group: '数学基础', x: 14, y: 40, level: 'base' },
  { id: 'partial-derivative', label: '偏导数', group: '数学基础', x: 30, y: 22, level: 'base' },
  { id: 'gradient-descent', label: '梯度下降', group: '机器学习', x: 42, y: 48, level: 'focus' },
  { id: 'loss-function', label: '损失函数', group: '机器学习', x: 58, y: 24, level: 'core' },
  { id: 'cnn', label: '卷积神经网络', group: '深度学习', x: 62, y: 58, level: 'core' },
  { id: 'knowledge-graph', label: '知识图谱建模', group: '知识图谱', x: 70, y: 78, level: 'extend' },
  { id: 'rag', label: 'RAG', group: '大模型', x: 84, y: 44, level: 'extend' },
  { id: 'vision-project', label: '图像分类项目', group: '行业实践', x: 86, y: 70, level: 'project' },
]

export const graphEdges = [
  { from: 'derivative', to: 'partial-derivative', label: '先修' },
  { from: 'partial-derivative', to: 'gradient-descent', label: '支撑' },
  { from: 'loss-function', to: 'gradient-descent', label: '共同决定' },
  { from: 'gradient-descent', to: 'cnn', label: '训练基础' },
  { from: 'cnn', to: 'vision-project', label: '应用' },
  { from: 'knowledge-graph', to: 'rag', label: '知识组织' },
  { from: 'rag', to: 'vision-project', label: '扩展场景' },
  { from: 'gradient-descent', to: 'knowledge-graph', label: '能力延展' },
]

export const growthRadarAxes = [
  { name: '数学基础', value: 69 },
  { name: '算法理解', value: 81 },
  { name: '工程实践', value: 61 },
  { name: '知识组织', value: 57 },
  { name: '场景应用', value: 64 },
]

export const growthSummaryCards = [
  { label: '已形成优势', value: '算法理解' },
  { label: '当前短板', value: '知识组织' },
  { label: '推荐策略', value: '补弱优先' },
]

export const heatmapDomains = ['数学基础', '机器学习', '深度学习', '大模型', '知识图谱']

export const heatmapRows = [
  {
    label: '导数与偏导',
    values: [82, 46, 24, 8, 10],
  },
  {
    label: '梯度下降',
    values: [71, 52, 36, 12, 10],
  },
  {
    label: '卷积网络',
    values: [25, 58, 66, 14, 8],
  },
  {
    label: '知识图谱建模',
    values: [10, 16, 8, 28, 41],
  },
  {
    label: 'RAG 应用',
    values: [6, 18, 24, 34, 32],
  },
]

export const knowledgeDetails = {
  'gradient-descent': {
    id: 'gradient-descent',
    title: '梯度下降',
    domain: '机器学习',
    summary: '模型训练中的核心优化方法，也是连接高数导数与机器学习训练过程的关键知识点。',
    mastery: '当前处于补基础阶段，建议优先补弱。',
    recommendReason: '你在损失函数理解上表现稳定，但在偏导数计算和学习率选择上出现连续错误。',
    prerequisites: ['导数', '偏导数', '函数极值'],
    related: ['损失函数', '反向传播', '优化器选择'],
    scenarios: ['线性回归训练', '神经网络参数更新', '深度学习调参'],
  },
  cnn: {
    id: 'cnn',
    title: '卷积神经网络 CNN',
    domain: '深度学习',
    summary: '图像任务中的核心模型结构，连接线性代数、特征提取、模型训练与视觉场景。',
    mastery: '当前已经具备继续进阶的基础。',
    recommendReason: '你已具备基本模型理解能力，可以从卷积与池化进入图像任务学习主线。',
    prerequisites: ['矩阵运算', '梯度下降', '神经网络基础'],
    related: ['图像分类', '目标检测', '特征提取'],
    scenarios: ['图像分类项目', '自动驾驶视觉识别', '医学影像分析'],
  },
  rag: {
    id: 'rag',
    title: '检索增强生成 RAG',
    domain: '大模型',
    summary: '把检索系统与生成模型结合的关键能力点，适合智能体与大模型应用方向。',
    mastery: '当前建议先补检索与向量表示。',
    recommendReason: '如果你的目标是大模型应用开发，RAG 是连接知识库、检索和智能体的重要中间层。',
    prerequisites: ['向量表示', 'Embedding', '检索基础'],
    related: ['知识库问答', 'Agent 工具调用', '向量数据库'],
    scenarios: ['校园知识问答', '企业知识库助手', '智能搜索应用'],
  },
  'knowledge-graph': {
    id: 'knowledge-graph',
    title: '知识图谱建模',
    domain: '知识图谱',
    summary: '将知识点、课程、能力和场景组织为结构化关系网络，是推荐智能体的重要底座。',
    mastery: '当前建议先补图结构与实体关系。',
    recommendReason: '你已经开始进入推荐系统设计阶段，知识图谱能帮助你理解跨课程关联如何组织。',
    prerequisites: ['图结构基础', '实体关系', '图查询'],
    related: ['Neo4j', '推荐逻辑', '知识主线'],
    scenarios: ['课程知识网络', '跨课程推荐系统', '个性化学习安排'],
  },
}
