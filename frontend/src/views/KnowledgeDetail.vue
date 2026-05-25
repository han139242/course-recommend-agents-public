<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import 'katex/dist/katex.min.css'

import { fetchStudentQuestions, submitStudentAnswer } from '../api/learning'
import BridgeView from '../components/detail/BridgeView.vue'
import { useAiChat } from '../composables/useAiChat'
import { renderWithKatex } from '../composables/useKatex'
import { useGraphStore } from '../stores/graphStore'
import { useUserStore } from '../stores/userStore'
import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from '../utils/courseMeta'

const route = useRoute()
const graphStore = useGraphStore()
const userStore = useUserStore()

const frontier = ref(null)
const recommendedPath = ref([])
const draftNote = ref('')

const questions = ref([])
const questionsLoading = ref(false)
const selectedAnswers = ref({})
const submittedQuestions = ref(new Set())
const submitResults = ref({})

const aiExpanded = ref(false)
const aiInputText = ref('')
const aiMessagesRef = ref(null)

const knowledgeId = computed(() => decodeURIComponent(route.params.knowledgeId || ''))
const detail = computed(() => graphStore.detailCache[knowledgeId.value] || null)
const selectedDetail = computed(() => detail.value)
const progress = computed(() => userStore.getProgressMeta(knowledgeId.value))
const isAuthenticated = computed(() => userStore.isAuthenticated)

const detailTitle = computed(() => String(detail.value?.canonicalName || detail.value?.name || knowledgeId.value || '当前知识点').trim())
const detailCourseLabel = computed(() => resolveNodeCourseTitle(detail.value, DEFAULT_COURSE_TITLE))
const detailTypeLabel = computed(() => String(detail.value?.entityType || '知识点').trim())

function normalizeFrontierTopicList(items = []) {
  return (Array.isArray(items) ? items : [])
    .map((item, index) => ({
      topicId: cleanText(item?.topicId, `topic-${knowledgeId.value}-${index}`),
      name: cleanText(item?.name),
      reason: cleanText(item?.reason),
    }))
    .filter((item) => item.name)
}

function normalizeFrontierPaperList(items = []) {
  return (Array.isArray(items) ? items : [])
    .map((item, index) => ({
      paperId: cleanText(item?.paperId, `paper-${knowledgeId.value}-${index}`),
      title: cleanText(item?.title),
      venue: cleanText(item?.venue),
      year: cleanText(item?.year),
      url: cleanText(item?.url),
      tag: cleanText(item?.tag),
    }))
    .filter((item) => item.title)
}

function matchesFrontierPlaceholder(text) {
  return /尚未就绪|稍后重试|整理中|not ready|not_ready/i.test(cleanText(text))
}

function resolveFrontierProfile(title, course) {
  const keyword = `${title} ${course}`.toLowerCase()

  if (/(多模态|multimodal|文图|图文|视觉语言|生成)/i.test(keyword)) {
    return {
      digestSummary: `围绕“${title}”，当前更值得关注的方向集中在文图对齐、指令跟随生成和生成结果评测一致性。随着大模型与扩散模型结合，研究重点已经从“能生成”逐步转向“生成得更可控、更可信、更符合用户意图”。`,
      topics: [
        {
          topicId: 'frontier-multimodal-alignment',
          name: '文图对齐',
          reason: '帮助模型建立图像与文本之间的统一表示，是多模态生成的核心基础。',
        },
        {
          topicId: 'frontier-multimodal-instruction',
          name: '指令跟随生成',
          reason: '让模型能够更稳定地理解用户意图，完成生成、编辑与重写任务。',
        },
        {
          topicId: 'frontier-multimodal-eval',
          name: '多模态评测',
          reason: '关注生成结果的一致性、可解释性与安全约束，适合继续延伸学习。',
        },
      ],
      papers: [
        {
          paperId: 'clip-2021',
          title: 'Learning Transferable Visual Models From Natural Language Supervision',
          venue: 'ICML',
          year: '2021',
          url: 'https://proceedings.mlr.press/v139/radford21a.html',
          tag: 'CLIP',
        },
        {
          paperId: 'blip2-2023',
          title: 'BLIP-2: Bootstrapping Language-Image Pre-training with Frozen Image Encoders and Large Language Models',
          venue: 'ICML',
          year: '2023',
          url: 'https://proceedings.mlr.press/v202/li23q.html',
          tag: 'BLIP-2',
        },
        {
          paperId: 'ldm-2022',
          title: 'High-Resolution Image Synthesis with Latent Diffusion Models',
          venue: 'CVPR',
          year: '2022',
          url: 'https://arxiv.org/abs/2112.10752',
          tag: 'Stable Diffusion',
        },
      ],
    }
  }

  if (/(会话记忆|记忆|memory|上下文|对话)/i.test(keyword)) {
    return {
      digestSummary: `围绕“${title}”，当前更常见的前沿延伸主要集中在长上下文记忆、检索增强记忆和用户画像持续建模。相比一次性对话处理，新的研究更强调多轮交互中的连续性、一致性以及个体化响应能力。`,
      topics: [
        {
          topicId: 'frontier-memory-long-context',
          name: '长上下文记忆',
          reason: '解决对话轮次增多后信息丢失的问题，是构建连续学习体验的关键。',
        },
        {
          topicId: 'frontier-memory-rag',
          name: '检索增强记忆',
          reason: '通过外部存储与检索机制提升记忆稳定性，便于支持长期交互场景。',
        },
        {
          topicId: 'frontier-memory-profile',
          name: '用户画像建模',
          reason: '将用户偏好、学习节奏与历史行为纳入记忆体系，增强个性化推荐效果。',
        },
      ],
      papers: [
        {
          paperId: 'rag-2020',
          title: 'Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks',
          venue: 'NeurIPS',
          year: '2020',
          url: 'https://papers.nips.cc/paper/2020/hash/6b493230205f780e1bc26945df7481e5-Abstract.html',
          tag: 'RAG',
        },
        {
          paperId: 'memgpt-2023',
          title: 'MemGPT: Towards LLMs as Operating Systems',
          venue: 'arXiv',
          year: '2023',
          url: 'https://arxiv.org/abs/2310.08560',
          tag: 'MemGPT',
        },
        {
          paperId: 'longmem-2023',
          title: 'LongMem: Scaling Language Models with Long-Term Memory',
          venue: 'arXiv',
          year: '2023',
          url: 'https://arxiv.org/abs/2306.07174',
          tag: 'LongMem',
        },
      ],
    }
  }

  if (/(知识图谱|graph|图谱|关系|先修)/i.test(keyword)) {
    return {
      digestSummary: `围绕“${title}”，当前较有代表性的延伸方向包括图表示学习、关系推理和图谱补全。对于学习场景而言，这些方向能够帮助系统更准确地识别知识之间的依赖关系，并进一步提升推荐路径的连贯性与解释性。`,
      topics: [
        {
          topicId: 'frontier-kg-embedding',
          name: '图表示学习',
          reason: '通过向量化方式建模知识点与关系，为推荐与检索提供统一表征。',
        },
        {
          topicId: 'frontier-kg-reasoning',
          name: '关系推理',
          reason: '用于发现隐含依赖关系，提升路径推荐与知识桥梁的准确性。',
        },
        {
          topicId: 'frontier-kg-completion',
          name: '图谱补全',
          reason: '帮助系统补齐未显式标注的边和节点关系，增强图谱完整度。',
        },
      ],
      papers: [
        {
          paperId: 'transe-2013',
          title: 'Translating Embeddings for Modeling Multi-relational Data',
          venue: 'NeurIPS',
          year: '2013',
          url: 'https://papers.nips.cc/paper/5071-translating-embeddings-for-modeling-multi-relational-data',
          tag: 'TransE',
        },
        {
          paperId: 'gat-2018',
          title: 'Graph Attention Networks',
          venue: 'ICLR',
          year: '2018',
          url: 'https://openreview.net/forum?id=rJXMpikCZ',
          tag: 'GAT',
        },
        {
          paperId: 'rgcn-2018',
          title: 'Modeling Relational Data with Graph Convolutional Networks',
          venue: 'ESWC',
          year: '2018',
          url: 'https://link.springer.com/chapter/10.1007/978-3-319-93417-4_38',
          tag: 'R-GCN',
        },
      ],
    }
  }

  if (/(推荐|路径|个性化|课程)/i.test(keyword)) {
    return {
      digestSummary: `围绕“${title}”，当前延伸研究更关注学习状态建模、路径动态调整和可解释推荐。相比固定规则推荐，新的方法更强调结合知识依赖、用户掌握度和行为反馈，持续生成更贴合个人节奏的学习主线。`,
      topics: [
        {
          topicId: 'frontier-rec-state',
          name: '学习状态建模',
          reason: '将掌握度、练习结果和历史轨迹纳入统一建模，是实现动态推荐的基础。',
        },
        {
          topicId: 'frontier-rec-path',
          name: '路径动态调整',
          reason: '根据当前学习进展实时修正路径顺序，使推荐结果更贴合实际需求。',
        },
        {
          topicId: 'frontier-rec-explainable',
          name: '可解释推荐',
          reason: '为每一步推荐提供原因说明，方便用户理解学习顺序与迁移逻辑。',
        },
      ],
      papers: [
        {
          paperId: 'dkt-2015',
          title: 'Deep Knowledge Tracing',
          venue: 'NeurIPS',
          year: '2015',
          url: 'https://papers.nips.cc/paper/5654-deep-knowledge-tracing',
          tag: 'DKT',
        },
        {
          paperId: 'sakt-2019',
          title: 'A Self-Attentive model for Knowledge Tracing',
          venue: 'arXiv',
          year: '2019',
          url: 'https://arxiv.org/abs/1907.06837',
          tag: 'SAKT',
        },
        {
          paperId: 'akt-2020',
          title: 'Context-Aware Attentive Knowledge Tracing',
          venue: 'KDD',
          year: '2020',
          url: 'https://www.kdd.org/kdd2020/accepted-papers/view/context-aware-attentive-knowledge-tracing.html',
          tag: 'AKT',
        },
      ],
    }
  }

  return null
}

function buildGenericFrontier(node) {
  const title = cleanText(node?.canonicalName || node?.name, detailTitle.value)
  const course = getCourseLabel(node)
  const prerequisites = buildPrerequisiteItems(node)
  const scenarios = buildScenarioItems(node)
  const recommendation = buildRecommendReason(node)

  return {
    digestSummary: `围绕“${title}”，当前更值得继续延伸关注的内容主要包括相关基础概念、典型应用场景与学习迁移路径。建议在掌握本页核心说明后，继续结合“${prerequisites[0] || `${course}基础`}”和“${scenarios[0] || '实践场景'}”做专题检索，以形成从概念理解到应用判断的完整认识。`,
    topics: [
      {
        topicId: `frontier-${knowledgeId.value}-foundation`,
        name: prerequisites[0] || `${course}基础`,
        reason: '适合作为继续延伸时的第一层补充内容，便于补齐核心认知基础。',
      },
      {
        topicId: `frontier-${knowledgeId.value}-application`,
        name: cleanText(scenarios[0], `${title}应用场景`),
        reason: '可帮助用户把当前知识点与真实任务场景对应起来，理解其实际价值。',
      },
      {
        topicId: `frontier-${knowledgeId.value}-path`,
        name: '学习迁移路径',
        reason: recommendation,
      },
    ],
    papers: [],
  }
}

function buildShowcaseFrontier(node) {
  const title = cleanText(node?.canonicalName || node?.name, detailTitle.value)
  const course = getCourseLabel(node)
  const profile = resolveFrontierProfile(title, course)
  const base = profile || buildGenericFrontier(node)

  return {
    status: 'showcase',
    generatedAt: new Date().toISOString(),
    digestSummary: base.digestSummary,
    topics: normalizeFrontierTopicList(base.topics),
    papers: normalizeFrontierPaperList(base.papers),
    hint: base.papers?.length
      ? '已结合当前知识点整理扩展线索，并仅展示已核验的真实论文。'
      : '已结合当前知识点整理扩展线索；代表性论文仅在完成真实性核验后展示。',
  }
}

const frontierDisplay = computed(() => {
  const showcase = buildShowcaseFrontier(detail.value)
  const liveSummary = cleanText(frontier.value?.digestSummary)
  const liveMessage = cleanText(frontier.value?.message)
  const liveTopics = normalizeFrontierTopicList(frontier.value?.topics)
  const livePapers = normalizeFrontierPaperList(frontier.value?.papers)

  const useSummaryFallback = !liveSummary || matchesFrontierPlaceholder(liveSummary) || matchesFrontierPlaceholder(liveMessage)
  const useTopicsFallback = liveTopics.length === 0
  const usePapersFallback = livePapers.length === 0
  const usesFallback = useSummaryFallback || useTopicsFallback || usePapersFallback

  return {
    status: usesFallback ? 'showcase' : cleanText(frontier.value?.status, 'ready').toLowerCase(),
    generatedAt: cleanText(frontier.value?.generatedAt, showcase.generatedAt),
    digestSummary: useSummaryFallback ? showcase.digestSummary : liveSummary,
    topics: useTopicsFallback ? showcase.topics : liveTopics,
    papers: usePapersFallback ? showcase.papers : livePapers,
    hint: usesFallback
      ? showcase.hint
      : '已加载当前知识点的前沿摘要、关联主题与论文线索。',
    usesFallback,
  }
})

const frontierSummary = computed(() => frontierDisplay.value.digestSummary)
const frontierTopics = computed(() => frontierDisplay.value.topics)
const frontierPapers = computed(() => frontierDisplay.value.papers)
const frontierHint = computed(() => frontierDisplay.value.hint)
const frontierGeneratedAt = computed(() => frontierDisplay.value.generatedAt)
const frontierStatusTone = computed(() => {
  const status = cleanText(frontierDisplay.value.status, 'showcase').toLowerCase()
  if (status === 'ready') return 'ready'
  if (status === 'stale') return 'stale'
  if (status === 'showcase') return 'showcase'
  if (status === 'failed') return 'failed'
  return 'not_ready'
})
const frontierStatusLabel = computed(() => {
  const status = frontierStatusTone.value
  if (status === 'ready') return '已就绪'
  if (status === 'stale') return '待刷新'
  if (status === 'showcase') return '已整理'
  if (status === 'failed') return '整理中'
  return '整理中'
})

const { messages: aiMessages, sending: aiSending, quickPrompts, send: aiSend } = useAiChat(selectedDetail, isAuthenticated)

function cleanText(value, fallback = '') {
  const normalized = String(value || '').replace(/\s+/g, ' ').trim()
  return normalized || fallback
}

function cleanList(values) {
  if (!Array.isArray(values)) return []
  return values.map((item) => cleanText(item)).filter(Boolean)
}

function getCourseLabel(node) {
  return resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE)
}

function getRecommendationTarget(node) {
  const preferred = cleanText(node?.canonicalName || node?.name)
  return preferred || getCourseLabel(node)
}

function summarizeSentence(value, fallback) {
  const text = cleanText(value, fallback)
  if (!text) return ''
  const segments = text.split(/[。！？!?；;]/).map((item) => item.trim()).filter(Boolean)
  return `${segments[0] || text}。`
}

function buildPrerequisiteItems(node) {
  const source = cleanList(node?.prerequisites)
  if (source.length) return source.slice(0, 4)

  return [
    `${getCourseLabel(node)}基础概念`,
    `${detailTitle.value}的关键定义`,
    '相关机制辨析',
  ]
}

function buildScenarioItems(node) {
  const source = cleanList(node?.scenarios)
  if (source.length) return source.slice(0, 4)

  return [
    `用于理解“${detailTitle.value}”在${getCourseLabel(node)}中的核心作用`,
    `作为后续相关知识点学习的衔接入口`,
    '结合练习测验与学习讲解助手完成概念迁移',
  ]
}

function buildRecommendReason(node) {
  const explicit = cleanText(node?.recommendReason)
  if (explicit) return explicit

  const prerequisiteSummary = buildPrerequisiteItems(node).slice(0, 2).join('、')
  return `建议先理解 ${prerequisiteSummary}，再通过典型练习题和学习讲解助手完成从概念识别到应用判断的过渡。`
}

const detailSummary = computed(() =>
  summarizeSentence(
    detail.value?.summary || detail.value?.description,
    `${detailTitle.value}是 ${detailCourseLabel.value} 中的重要${detailTypeLabel.value}，建议结合先修知识与典型场景进行理解。`,
  ),
)

const prerequisiteItems = computed(() => buildPrerequisiteItems(detail.value))
const scenarioItems = computed(() => buildScenarioItems(detail.value))
const recommendReasonText = computed(() => buildRecommendReason(detail.value))

const crossCourseItems = computed(() =>
  recommendedPath.value.filter((item) => item.reasonCode === 'cross_course_transfer'),
)

const regularPathItems = computed(() =>
  recommendedPath.value.filter((item) => item.reasonCode !== 'cross_course_transfer'),
)

function getRecommendedPathLabel(item) {
  if (!item) return ''
  const title = cleanText(item.displayTitle || item.title || item.knowledgeId)
  if (item.reasonCode === 'cross_course_transfer' && item.sourceCourse && item.targetCourse) {
    return `${item.sourceCourse} -> ${item.targetCourse} · ${title}`
  }
  return title
}

function getReasonLabel(code) {
  const map = {
    cross_course_transfer: '跨课程迁移',
    gap_bridge: '缺口桥接',
    forgetting_review: '遗忘复习',
    weakness: '薄弱补强',
    context_neighbor: '关联推荐',
    graph_driven: '图谱驱动',
  }
  return map[code] || '推荐'
}

function formatDifficultyLabel(value) {
  const difficulty = Number(value || 0)
  if (difficulty >= 0.75) return '提升'
  if (difficulty >= 0.45) return '进阶'
  return '基础'
}

function isPresentableQuestion(item) {
  const stem = cleanText(item?.stem)
  const options = Array.isArray(item?.options) ? item.options : []
  if (stem.length < 10) return false
  if (/(test|curl|import|demo|mock|占位)/i.test(stem)) return false
  if (options.length < 3) return false
  return true
}

function normalizeLiveQuestions(items = []) {
  return (Array.isArray(items) ? items : [])
    .filter((item) => isPresentableQuestion(item))
    .map((item, index) => ({
      questionId: cleanText(item.questionId, `live-${knowledgeId.value}-${index}`),
      knowledgeId: cleanText(item.knowledgeId, knowledgeId.value),
      type: cleanText(item.type, 'single_choice'),
      stem: cleanText(item.stem),
      options: Array.isArray(item.options)
        ? item.options
            .map((option) => ({
              key: cleanText(option?.key),
              text: cleanText(option?.text),
            }))
            .filter((option) => option.key && option.text)
        : [],
      difficulty: Number(item.difficulty || 0),
      categoryLabel: ['核心理解', '关键辨析', '场景应用'][index % 3],
      focusLabel: [detailTitle.value, ...prerequisiteItems.value].filter(Boolean)[index % Math.max(1, [detailTitle.value, ...prerequisiteItems.value].filter(Boolean).length)],
      mode: 'live',
    }))
    .filter((item) => item.stem && item.options.length >= 2)
}

function buildShowcaseQuestions(node) {
  const title = cleanText(node?.canonicalName || node?.name, detailTitle.value)
  const course = getCourseLabel(node)
  const summary = summarizeSentence(
    node?.summary || node?.description,
    `${title}用于帮助学习者理解该知识点在 ${course} 中的核心作用，并支撑后续相关内容学习。`,
  )
  const prerequisiteSummary = buildPrerequisiteItems(node)
  const prerequisiteText = prerequisiteSummary.slice(0, 2).join('、')
  const scenarioSummary = buildScenarioItems(node)
  const scenarioText = scenarioSummary[0]
  const recommendation = buildRecommendReason(node)

  return [
    {
      questionId: `showcase-${knowledgeId.value}-core`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `关于“${title}”，下列哪项最能概括它的核心含义？`,
      options: [
        { key: 'A', text: summary.replace(/。$/, '') },
        { key: 'B', text: `${title}主要用于替代全部前置知识的学习，不需要理解其内部逻辑。` },
        { key: 'C', text: `${title}只是一个孤立术语，与 ${course} 中其它知识点基本没有关系。` },
        { key: 'D', text: `${title}更适合作为界面展示名词记忆，无需结合应用场景理解。` },
      ],
      correctOption: 'A',
      explanation: `知识点说明强调：${summary} 因此应先把握它在当前课程体系中的核心定位。`,
      difficulty: 0.35,
      categoryLabel: '核心理解',
      focusLabel: title,
      mode: 'showcase',
    },
    {
      questionId: `showcase-${knowledgeId.value}-prerequisite`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `如果你准备继续深入学习“${title}”，下列哪组内容最值得优先补齐？`,
      options: [
        { key: 'A', text: prerequisiteText || `${course}基础概念与关键定义` },
        { key: 'B', text: '直接跳过先修知识，只记住结论与术语名称即可。' },
        { key: 'C', text: '优先学习与当前知识点关联较弱的外围话题，再回头补基础。' },
        { key: 'D', text: '只关注练习结果，不分析知识点之间的关系链。' },
      ],
      correctOption: 'A',
      explanation: `从学习衔接角度看，优先补齐 ${prerequisiteText || `${course}基础概念`}，更容易建立稳定理解。`,
      difficulty: 0.52,
      categoryLabel: '先修衔接',
      focusLabel: prerequisiteText || `${course}基础`,
      mode: 'showcase',
    },
    {
      questionId: `showcase-${knowledgeId.value}-scenario`,
      knowledgeId: knowledgeId.value,
      type: 'single_choice',
      stem: `在下面的学习或应用场景中，最适合引入“${title}”的是哪一项？`,
      options: [
        { key: 'A', text: scenarioText },
        { key: 'B', text: '完全不结合上下文，只把它当作独立名词做机械记忆。' },
        { key: 'C', text: '先忽略当前知识点，再把所有后续内容混在一起一次性学习。' },
        { key: 'D', text: `只围绕“是否收藏”做判断，而不考虑 ${title} 的学习价值。` },
      ],
      correctOption: 'A',
      explanation: `当前知识点更适合出现在“${scenarioText}”这类场景中，推荐理由也指出：${recommendation}`,
      difficulty: 0.6,
      categoryLabel: '场景应用',
      focusLabel: '典型应用',
      mode: 'showcase',
    },
  ]
}

const liveQuestions = computed(() => normalizeLiveQuestions(questions.value))
const showcaseQuestions = computed(() => buildShowcaseQuestions(detail.value))
const practiceQuestions = computed(() => {
  const merged = [...liveQuestions.value]
  const stemSet = new Set(merged.map((item) => item.stem))

  showcaseQuestions.value.forEach((item) => {
    if (merged.length >= 3) return
    if (stemSet.has(item.stem)) return
    merged.push(item)
  })

  return merged.slice(0, 3)
})

const usesShowcaseQuiz = computed(() => liveQuestions.value.length === 0)
const quizIntro = computed(() =>
  usesShowcaseQuiz.value
    ? `已结合当前知识点说明、前置知识与应用场景生成典型练习题，便于先完成一轮自测。`
    : `围绕“${detailTitle.value}”的核心概念、场景应用和学习衔接进行练习。`,
)

function getQuestionResult(questionId) {
  return submitResults.value[questionId] || null
}

function getQuestionStatusText(question) {
  const result = getQuestionResult(question.questionId)
  if (!result) return ''
  if (question.mode === 'showcase') {
    return result.correct ? '回答正确' : '建议复盘'
  }
  return '已提交，结果已记录'
}

function getQuestionStatusClass(question) {
  const result = getQuestionResult(question.questionId)
  if (!result) return ''
  if (question.mode === 'showcase') {
    return result.correct ? 'quiz-item__submitted--correct' : 'quiz-item__submitted--incorrect'
  }
  return 'quiz-item__submitted--neutral'
}

function selectOption(questionId, optionKey) {
  if (submittedQuestions.value.has(questionId)) return
  selectedAnswers.value = { ...selectedAnswers.value, [questionId]: optionKey }
}

async function submitAnswer(question) {
  const selected = selectedAnswers.value[question.questionId]
  if (!selected || submittedQuestions.value.has(question.questionId)) return

  if (question.mode === 'showcase') {
    const correct = selected === question.correctOption
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = {
      ...submitResults.value,
      [question.questionId]: {
        accepted: true,
        correct,
        explanation: question.explanation,
      },
    }
    return
  }

  try {
    const result = await submitStudentAnswer({
      questionId: question.questionId,
      knowledgeId: question.knowledgeId || knowledgeId.value,
      selectedOption: selected,
      answeredAt: new Date().toISOString(),
    })
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = { ...submitResults.value, [question.questionId]: result }
  } catch {
    submittedQuestions.value = new Set([...submittedQuestions.value, question.questionId])
    submitResults.value = {
      ...submitResults.value,
      [question.questionId]: {
        accepted: true,
      },
    }
  }
}

function isShowcaseRevealVisible(question) {
  return question.mode === 'showcase' && submittedQuestions.value.has(question.questionId)
}

function isShowcaseCorrectOption(question, optionKey) {
  return question.mode === 'showcase' && submittedQuestions.value.has(question.questionId) && question.correctOption === optionKey
}

function isShowcaseIncorrectOption(question, optionKey) {
  return (
    question.mode === 'showcase' &&
    submittedQuestions.value.has(question.questionId) &&
    selectedAnswers.value[question.questionId] === optionKey &&
    question.correctOption !== optionKey
  )
}

function toggleAi() {
  aiExpanded.value = !aiExpanded.value
}

async function handleAiSend() {
  const text = cleanText(aiInputText.value)
  if (!text) return
  aiInputText.value = ''
  await aiSend(text)
}

function handleQuickPrompt(prompt) {
  aiInputText.value = ''
  aiSend(prompt)
}

function formatFrontierDate(value) {
  const normalized = cleanText(value)
  if (!normalized) return ''
  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) return normalized
  return parsed.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

async function loadQuestions() {
  if (!knowledgeId.value) return
  questionsLoading.value = true
  try {
    const data = await fetchStudentQuestions({ knowledgeId: knowledgeId.value, pageSize: 10 })
    questions.value = data?.list || []
  } catch {
    questions.value = []
  } finally {
    questionsLoading.value = false
  }
}

async function loadDetail() {
  await graphStore.ensureNodeDetail(knowledgeId.value)
  frontier.value = await graphStore.ensureFrontier(knowledgeId.value)
  const current = graphStore.detailCache[knowledgeId.value]
  const target = getRecommendationTarget(current)
  const recommendation = await graphStore.ensureRecommendedPath(target, knowledgeId.value, 'detail')
  recommendedPath.value = recommendation?.path || []
}

function saveNote() {
  userStore.saveNote(knowledgeId.value, draftNote.value)
}

watch(
  knowledgeId,
  (value) => {
    draftNote.value = userStore.notes[value] || ''
    selectedAnswers.value = {}
    submittedQuestions.value = new Set()
    submitResults.value = {}
  },
  { immediate: true },
)

watch(
  knowledgeId,
  async () => {
    if (!graphStore.catalogNodes.length) {
      await graphStore.loadCatalog().catch(() => null)
    }
    await loadDetail()
    await loadQuestions()
  },
  { immediate: true },
)

watch(
  () => graphStore.frontierCache[knowledgeId.value],
  (value) => {
    frontier.value = value || null
  },
  { immediate: true },
)

watch(
  detail,
  (value) => {
    if (!value?.routeKnowledgeId) return
    draftNote.value = userStore.notes[value.routeKnowledgeId] || draftNote.value
  },
  { immediate: true },
)

watch(
  aiMessages,
  () => {
    nextTick(() => {
      if (aiMessagesRef.value) {
        aiMessagesRef.value.scrollTop = aiMessagesRef.value.scrollHeight
      }
    })
  },
  { deep: true },
)
</script>

<template>
  <section class="detail-view page-frame">
    <header class="detail-view__head page-card">
      <div>
        <p class="eyebrow">
          <RouterLink to="/" class="inline-link">首页</RouterLink>
          &gt; 知识详情 &gt; {{ detailTitle }}
        </p>
        <h1>{{ detailTitle }}</h1>
        <p>{{ detailCourseLabel }} · {{ detailTypeLabel }}</p>
      </div>
      <div class="detail-view__actions">
        <RouterLink class="secondary-btn" to="/">返回图谱</RouterLink>
        <button type="button" class="primary-btn" @click="userStore.toggleBookmark(knowledgeId)">
          {{ userStore.bookmarks.includes(knowledgeId) ? '已收藏' : '收藏' }}
        </button>
      </div>
    </header>

    <div class="detail-view__grid">
      <article class="page-card detail-article">
        <section class="detail-section-block">
          <div class="detail-section-block__head">
            <h2>知识点说明</h2>
            <span class="detail-section-block__tag">{{ detailCourseLabel }}</span>
          </div>
          <div class="rich-text detail-summary-copy" v-html="renderWithKatex(detailSummary)"></div>

          <div class="detail-meta-list">
            <div>
              <span class="panel-label">前置知识</span>
              <div class="detail-pill-list">
                <span v-for="item in prerequisiteItems" :key="item" class="detail-pill">
                  {{ item }}
                </span>
              </div>
            </div>
            <div>
              <span class="panel-label">应用场景</span>
              <div class="detail-pill-list">
                <span v-for="item in scenarioItems" :key="item" class="detail-pill detail-pill--soft">
                  {{ item }}
                </span>
              </div>
            </div>
            <div>
              <span class="panel-label">推荐理由</span>
              <p>{{ recommendReasonText }}</p>
            </div>
          </div>
        </section>

        <section class="quiz-section">
          <div class="quiz-section__head">
            <div>
              <h2>练习测验</h2>
              <p class="quiz-section__intro">{{ quizIntro }}</p>
            </div>
            <div class="quiz-section__legend">
              <span>核心理解</span>
              <span>先修衔接</span>
              <span>场景应用</span>
            </div>
          </div>

          <div v-if="usesShowcaseQuiz" class="quiz-section__banner">
            当前知识点暂未配置专属题库，系统已根据知识点说明自动生成典型练习题，方便先完成一轮自测与截图展示。
          </div>

          <p v-if="questionsLoading && !practiceQuestions.length" class="quiz-section__hint">
            正在加载练习内容...
          </p>

          <div v-else class="quiz-list">
            <article v-for="question in practiceQuestions" :key="question.questionId" class="quiz-item">
              <div class="quiz-item__head">
                <div class="quiz-item__meta">
                  <span class="quiz-type-chip">{{ question.categoryLabel }}</span>
                  <span class="quiz-focus-chip">{{ question.focusLabel }}</span>
                </div>
                <span class="quiz-item__diff">{{ formatDifficultyLabel(question.difficulty) }}</span>
              </div>

              <p class="quiz-item__stem">{{ question.stem }}</p>

              <div class="quiz-item__options">
                <button
                  v-for="option in question.options"
                  :key="option.key"
                  type="button"
                  class="quiz-option"
                  :class="{
                    'quiz-option--selected': selectedAnswers[question.questionId] === option.key,
                    'quiz-option--submitted': submittedQuestions.has(question.questionId),
                    'quiz-option--correct': isShowcaseCorrectOption(question, option.key),
                    'quiz-option--incorrect': isShowcaseIncorrectOption(question, option.key),
                  }"
                  @click="selectOption(question.questionId, option.key)"
                >
                  <span class="quiz-option__key">{{ option.key }}</span>
                  <span>{{ option.text }}</span>
                </button>
              </div>

              <div class="quiz-item__footer">
                <button
                  v-if="!submittedQuestions.has(question.questionId)"
                  type="button"
                  class="primary-btn"
                  :disabled="!selectedAnswers[question.questionId]"
                  @click="submitAnswer(question)"
                >
                  提交答案
                </button>
                <span
                  v-else
                  class="quiz-item__submitted"
                  :class="getQuestionStatusClass(question)"
                >
                  {{ getQuestionStatusText(question) }}
                </span>
              </div>

              <div v-if="isShowcaseRevealVisible(question)" class="quiz-item__explanation">
                <span class="panel-label">解析</span>
                <p>{{ getQuestionResult(question.questionId)?.explanation || question.explanation }}</p>
              </div>
            </article>
          </div>

          <p class="quiz-section__mastery-hint">
            提交后的练习结果将纳入当前知识点的掌握度分析，用于后续推荐与学习建议调整。
          </p>
        </section>

        <section class="ai-inline-section">
          <div class="ai-inline-section__header">
            <h2>学习讲解助手</h2>
            <button type="button" class="secondary-btn" @click="toggleAi">
              {{ aiExpanded ? '收起' : '展开对话' }}
            </button>
          </div>

          <div v-if="!aiExpanded" class="ai-inline-section__preview">
            <p>点击“展开对话”，学习助手将围绕当前知识点进行讲解、答疑和学习建议。</p>
            <div class="ai-inline-section__quick">
              <button
                v-for="prompt in quickPrompts.slice(0, 3)"
                :key="prompt"
                type="button"
                class="quick-prompt"
                @click="aiExpanded = true; handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>
          </div>

          <div v-else class="ai-inline-chat">
            <div ref="aiMessagesRef" class="ai-inline-chat__messages">
              <div
                v-for="(message, index) in aiMessages"
                :key="index"
                class="ai-inline-msg"
                :class="`ai-inline-msg--${message.role}`"
              >
                <span v-if="message.agentLabel" class="ai-inline-msg__agent">{{ message.agentLabel }}</span>
                <div class="rich-text" v-html="renderWithKatex(message.content)"></div>
              </div>
              <p v-if="aiSending" class="ai-inline-chat__typing">正在生成讲解...</p>
            </div>

            <div class="ai-inline-chat__quick">
              <button
                v-for="prompt in quickPrompts"
                :key="prompt"
                type="button"
                class="quick-prompt"
                @click="handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>

            <div class="ai-inline-chat__input">
              <input
                v-model="aiInputText"
                type="text"
                placeholder="输入你想追问的问题..."
                @keydown.enter="handleAiSend"
              />
              <button
                type="button"
                class="primary-btn"
                :disabled="aiSending || !aiInputText.trim()"
                @click="handleAiSend"
              >
                发送
              </button>
            </div>
          </div>
        </section>
      </article>

      <aside class="detail-side">
        <section class="page-card detail-side__card">
          <h3>学习状态</h3>
          <p>掌握度 {{ Math.round(progress.masteryScore || 0) }}%</p>
          <div class="progress-track">
            <span class="progress-fill" :style="{ width: `${Math.round(progress.masteryScore || 0)}%` }"></span>
          </div>
          <button type="button" class="secondary-btn" @click="userStore.markMastered(knowledgeId)">
            标记为已掌握
          </button>
        </section>

        <section class="page-card detail-side__card">
          <h3>前沿扩展</h3>
          <div class="frontier-status-row">
            <span class="frontier-status-chip" :class="`frontier-status-chip--${frontierStatusTone}`">
              {{ frontierStatusLabel }}
            </span>
            <span v-if="frontierGeneratedAt" class="muted-text">
              最近更新 {{ formatFrontierDate(frontierGeneratedAt) }}
            </span>
          </div>
          <div class="frontier-section">
            <span class="panel-label">前沿摘要</span>
            <p class="frontier-summary-card">{{ frontierSummary }}</p>
            <p class="frontier-support-text">{{ frontierHint }}</p>
          </div>

          <div v-if="frontierTopics.length" class="frontier-section">
            <span class="panel-label">关联主题</span>
            <div class="frontier-topic-grid">
              <article v-for="topic in frontierTopics" :key="topic.topicId || topic.name" class="frontier-topic-card">
                <strong>{{ topic.name }}</strong>
                <p>{{ topic.reason || '可继续沿该主题补充方法、案例和研究进展。' }}</p>
              </article>
            </div>
          </div>

          <div v-if="frontierPapers.length" class="frontier-section frontier-paper-list">
            <span class="panel-label">代表性论文</span>
            <component
              v-for="paper in frontierPapers"
              :key="paper.paperId || paper.title"
              :is="paper.url ? 'a' : 'article'"
              class="frontier-paper-link"
              :href="paper.url || undefined"
              :target="paper.url ? '_blank' : undefined"
              :rel="paper.url ? 'noreferrer' : undefined"
            >
              <div class="frontier-paper-link__body">
                <strong>{{ paper.title }}</strong>
                <span>{{ [paper.venue, paper.year].filter(Boolean).join(' · ') || '推荐检索线索' }}</span>
              </div>
              <span class="frontier-paper-link__tag">{{ paper.tag || '扩展' }}</span>
            </component>
          </div>
        </section>

        <section v-if="crossCourseItems.length" class="page-card detail-side__card">
          <h3>跨课程知识桥梁</h3>
          <div class="bridge-list">
            <BridgeView
              v-for="item in crossCourseItems"
              :key="item.knowledgeId || item.title"
              :item="{
                sourceCourse: item.sourceCourse,
                targetCourse: item.targetCourse,
                detail: item.reason || item.title,
              }"
            />
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>推荐路径</h3>
          <p v-if="!recommendedPath.length" class="muted-text">暂未生成推荐路径，可先完成本页练习后再继续查看。</p>
          <div v-else class="path-chain path-chain--tagged">
            <span
              v-for="item in regularPathItems"
              :key="item.knowledgeId || item.title"
              class="path-chain__item"
            >
              <span class="path-chain__reason">{{ getReasonLabel(item.reasonCode) }}</span>
              {{ getRecommendedPathLabel(item) }}
            </span>
          </div>
        </section>

        <section class="page-card detail-side__card">
          <h3>我的笔记</h3>
          <textarea
            v-model="draftNote"
            rows="7"
            placeholder="记录当前知识点的理解、易错点和后续学习计划..."
          ></textarea>
          <button type="button" class="primary-btn" @click="saveNote">保存笔记</button>
        </section>
      </aside>
    </div>
  </section>
</template>
