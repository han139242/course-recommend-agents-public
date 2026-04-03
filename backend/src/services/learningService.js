const { pool } = require('../db')
const {
  getGraphStats,
  getKnowledgeByNames,
  getKnowledgeDetail,
  listFeaturedKnowledge,
} = require('./graphService')

const DOMAIN_ORDER = ['数学基础', '机器学习', '深度学习', '知识图谱', '场景应用']
const DEMO_SEED_MASTERY = {
  偏导数: 46,
  梯度下降: 52,
  卷积神经网络: 66,
  知识图谱: 41,
  检索增强生成: 34,
}

function toNumber(value, digits = 0) {
  const number = Number(value || 0)
  if (digits <= 0) {
    return Math.round(number)
  }

  return Number(number.toFixed(digits))
}

function masteryStatus(mastery) {
  if (mastery >= 75) {
    return 'solid'
  }
  if (mastery >= 45) {
    return 'progressing'
  }
  return 'weak'
}

function inferDomainBucket(name = '', chapterTitle = '', sourcePath = '') {
  const text = `${name} ${chapterTitle} ${sourcePath}`

  if (/(导数|偏导|矩阵|概率|优化|线性代数|微积分)/.test(text)) {
    return '数学基础'
  }
  if (/(卷积|神经网络|深度学习|Transformer|反向传播|CNN)/i.test(text)) {
    return '深度学习'
  }
  if (/(知识图谱|图谱|图神经|推理|RAG|检索增强|向量数据库)/i.test(text)) {
    return '知识图谱'
  }
  if (/(项目|应用|场景|机器人|推荐系统|视觉|问答|智能体)/.test(text)) {
    return '场景应用'
  }
  return '机器学习'
}

function buildStatCards(progressRows, graphStats) {
  const masteredCount = progressRows.filter((row) => row.mastery >= 60).length
  const total = progressRows.length || 1
  const avgMastery = progressRows.reduce((sum, row) => sum + row.mastery, 0) / total

  return [
    { label: '学习中知识点', value: `${progressRows.length}` },
    { label: '已掌握知识点', value: `${masteredCount} / ${progressRows.length}` },
    { label: '平均掌握度', value: `${toNumber(avgMastery)}%` },
    { label: '图谱节点规模', value: `${graphStats.totalNodes}` },
  ]
}

function buildRecentRecords(progressRows) {
  return progressRows.slice(0, 3).map((row) => {
    const prefix =
      row.mastery >= 70 ? '已形成优势' : row.mastery >= 45 ? '正在提升' : '当前薄弱'
    return `${prefix}：${row.canonical_name}（掌握度 ${toNumber(row.mastery)}%）`
  })
}

function buildGoals(target, progressRows) {
  const weakest = [...progressRows].sort((a, b) => a.mastery - b.mastery).slice(0, 2)
  const goals = weakest.map((row) => `补齐 ${row.canonical_name}，提升相关先修理解`)

  if (target) {
    goals.push(`围绕“${target}”完成一条可解释学习路径`)
  }

  return goals.slice(0, 3)
}

function buildDomainScores(progressRows) {
  const bucket = Object.fromEntries(DOMAIN_ORDER.map((domain) => [domain, []]))

  for (const row of progressRows) {
    const domain = inferDomainBucket(row.canonical_name, row.chapter_title, row.source_path)
    bucket[domain].push(row.mastery)
  }

  return DOMAIN_ORDER.map((domain) => {
    const values = bucket[domain]
    const score = values.length
      ? values.reduce((sum, value) => sum + value, 0) / values.length
      : 35

    return {
      name: domain,
      value: toNumber(score),
    }
  })
}

function buildHeatmapRows(progressRows) {
  return progressRows.slice(0, 6).map((row) => {
    const domain = inferDomainBucket(row.canonical_name, row.chapter_title, row.source_path)

    return {
      label: row.canonical_name,
      values: DOMAIN_ORDER.map((item) => {
        if (item === domain) {
          return toNumber(row.mastery)
        }
        return item === '场景应用' && domain === '深度学习'
          ? Math.max(8, toNumber(row.mastery * 0.35))
          : Math.max(6, toNumber(row.mastery * 0.18))
      }),
    }
  })
}

async function ensureDemoLearningProgress(user) {
  if (!user?.id) {
    return
  }

  const [existingRows] = await pool.query(
    'SELECT id FROM learning_progress WHERE user_id = ? LIMIT 1',
    [user.id],
  )

  if (existingRows.length > 0) {
    return
  }

  const demoNodes = await getKnowledgeByNames(Object.keys(DEMO_SEED_MASTERY))

  if (demoNodes.length === 0) {
    return
  }

  for (const node of demoNodes) {
    const mastery = DEMO_SEED_MASTERY[node.canonicalName] || 40
    await pool.query(
      `
      INSERT INTO learning_progress
        (user_id, knowledge_node_id, canonical_name, mastery, status, last_event_type)
      VALUES (?, ?, ?, ?, ?, ?)
      ON DUPLICATE KEY UPDATE
        canonical_name = VALUES(canonical_name),
        mastery = VALUES(mastery),
        status = VALUES(status),
        last_event_type = VALUES(last_event_type)
      `,
      [user.id, node.id, node.canonicalName, mastery, masteryStatus(mastery), 'seeded'],
    )
  }
}

async function loadProgressRows(userId) {
  const [rows] = await pool.query(
    `
    SELECT knowledge_node_id, canonical_name, mastery, status, updated_at
    FROM learning_progress
    WHERE user_id = ?
    ORDER BY mastery ASC, updated_at DESC
    `,
    [userId],
  )

  return rows
}

async function buildRecommendedPath(progressRows) {
  const candidates = progressRows.slice(0, 4)
  const detailList = await Promise.all(
    candidates.map((row) => getKnowledgeDetail(row.knowledge_node_id)),
  )

  return detailList
    .filter(Boolean)
    .map((item, index) => ({
      stage:
        index === 0
          ? '补弱推荐'
          : index === 1
            ? '进阶推荐'
            : index === 2
              ? '跨课程推荐'
              : '拓展推荐',
      title: item.canonicalName,
      detail:
        item.prerequisites.length > 0
          ? `建议先补 ${item.prerequisites.slice(0, 2).join('、')}，再进入 ${item.canonicalName}。`
          : item.summary,
      route: `/knowledge/${item.id}`,
    }))
}

async function buildFeatureCards() {
  const featured = await listFeaturedKnowledge(6)

  return featured.map((item, index) => ({
    id: item.id,
    title: item.canonicalName,
    progress: 40 + ((index + 1) * 9) % 35,
    tag: item.description || '知识图谱中的关键能力节点',
    type: index < 2 ? '补弱推荐' : index < 4 ? '进阶推荐' : '延展推荐',
    domain: inferDomainBucket(item.canonicalName, item.chapterTitle, item.sourcePath),
    route: `/knowledge/${item.id}`,
  }))
}

async function buildLearningSnapshot(user) {
  if (user?.id) {
    await ensureDemoLearningProgress(user)
  }

  const [graphStats, featureCards] = await Promise.all([getGraphStats(), buildFeatureCards()])

  const progressRows = user?.id
    ? await loadProgressRows(user.id)
    : featureCards.map((item) => ({
        knowledge_node_id: item.id,
        canonical_name: item.title,
        mastery: item.progress,
        status: masteryStatus(item.progress),
        updated_at: new Date(),
        chapter_title: item.domain,
        source_path: item.domain,
      }))

  const domainScores = buildDomainScores(progressRows)
  const recommendedPath = await buildRecommendedPath(progressRows)
  const avgMastery =
    progressRows.reduce((sum, row) => sum + Number(row.mastery || 0), 0) /
    Math.max(progressRows.length, 1)
  const strongestDomain = [...domainScores].sort((a, b) => b.value - a.value)[0]
  const weakestDomain = [...domainScores].sort((a, b) => a.value - b.value)[0]

  const target = user?.target || '算法工程师 / 智能体应用开发'

  return {
    quickStats: buildStatCards(progressRows, graphStats).slice(0, 3),
    featureCards,
    recommendedPath,
    growthSummaryCards: [
      { label: '已形成优势', value: strongestDomain?.name || '算法理解' },
      { label: '当前短板', value: weakestDomain?.name || '数学基础' },
      { label: '推荐策略', value: avgMastery >= 60 ? '进阶优先' : '补弱优先' },
    ],
    growthInsights: domainScores.map((item) => ({
      label: item.name,
      value: item.value,
      note:
        item.value >= 70
          ? `你在${item.name}方向已经形成相对稳定的能力优势，可继续承接更高阶知识点。`
          : `当前${item.name}仍有明显提升空间，建议结合先修关系持续补强。`,
    })),
    radarAxes: domainScores,
    heatmapDomains: DOMAIN_ORDER,
    heatmapRows: buildHeatmapRows(progressRows),
    profile: {
      name: user?.name || '访客同学',
      role: user?.role || '人工智能专业学生',
      target: `目标方向：${target}`,
      summary:
        avgMastery >= 60
          ? '当前处于“基础逐步稳固 + 进阶方向展开”阶段，推荐策略以进阶主线和应用延展为主。'
          : '当前处于“基础补强 + 关键知识打通”阶段，推荐策略以薄弱点优先和先修回补为主。',
      stats: buildStatCards(progressRows, graphStats),
      goals: buildGoals(target, progressRows),
      recentRecords: buildRecentRecords(progressRows),
    },
  }
}

module.exports = {
  buildLearningSnapshot,
}
