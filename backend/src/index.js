const bcrypt = require('bcryptjs')
const cors = require('cors')
const express = require('express')

const { signToken, sanitizeUser, requireAuth, optionalAuth } = require('./auth')
const config = require('./config')
const { initializeDatabase, pool } = require('./db')
const { getGraphStats, getKnowledgeDetail, getSubgraph } = require('./services/graphService')
const { buildLearningSnapshot } = require('./services/learningService')
const { getFrontierBriefing } = require('./data/frontierDigest')
const { closeNeo4j } = require('./services/neo4j')
const { fail, ok } = require('./utils/response')

const app = express()

app.use(
  cors({
    origin: config.corsOrigin,
    credentials: false,
  }),
)
app.use(express.json())

async function loadUserById(userId) {
  if (!userId) {
    return null
  }

  const [rows] = await pool.query(
    `
      SELECT id, name, email, role, target, created_at
      FROM users
      WHERE id = ?
      LIMIT 1
    `,
    [userId],
  )

  return rows[0] ? sanitizeUser(rows[0]) : null
}

app.get('/api/health', async (req, res) => {
  try {
    await pool.query('SELECT 1')
    const graphStats = await getGraphStats()
    res.json(
      ok({
        mysql: 'ok',
        neo4j: 'ok',
        graphStats,
      }),
    )
  } catch (error) {
    res.status(500).json(fail('服务连接失败，请检查 MySQL 或 Neo4j 配置。'))
  }
})

app.post('/api/auth/register', async (req, res) => {
  const {
    name = '',
    email = '',
    password = '',
    role = '人工智能专业学生',
    target = '',
  } = req.body || {}

  const normalizedName = String(name).trim()
  const normalizedEmail = String(email).trim().toLowerCase()
  const normalizedPassword = String(password)
  const normalizedRole = String(role).trim() || '人工智能专业学生'
  const normalizedTarget = String(target).trim()

  if (!normalizedName || !normalizedEmail || !normalizedPassword) {
    return res.status(400).json({
      message: '姓名、邮箱和密码不能为空。',
    })
  }

  if (normalizedPassword.length < 6) {
    return res.status(400).json({
      message: '密码至少需要 6 位。',
    })
  }

  try {
    const [existingRows] = await pool.query('SELECT id FROM users WHERE email = ? LIMIT 1', [
      normalizedEmail,
    ])

    if (existingRows.length > 0) {
      return res.status(409).json({
        message: '该邮箱已注册，请直接登录。',
      })
    }

    const passwordHash = await bcrypt.hash(normalizedPassword, 10)
    const [result] = await pool.query(
      `
        INSERT INTO users (name, email, password_hash, role, target)
        VALUES (?, ?, ?, ?, ?)
      `,
      [normalizedName, normalizedEmail, passwordHash, normalizedRole, normalizedTarget],
    )

    const [rows] = await pool.query(
      `
        SELECT id, name, email, role, target, created_at
        FROM users
        WHERE id = ?
        LIMIT 1
      `,
      [result.insertId],
    )

    const user = sanitizeUser(rows[0])
    const token = signToken(user)

    return res.status(201).json(ok({ token, user }, '注册成功。'))
  } catch (error) {
    return res.status(500).json(fail('注册失败，请稍后重试。'))
  }
})

app.post('/api/auth/login', async (req, res) => {
  const { email = '', password = '' } = req.body || {}
  const normalizedEmail = String(email).trim().toLowerCase()
  const normalizedPassword = String(password)

  if (!normalizedEmail || !normalizedPassword) {
    return res.status(400).json({
      message: '请输入邮箱和密码。',
    })
  }

  try {
    const [rows] = await pool.query(
      `
        SELECT id, name, email, password_hash, role, target, created_at
        FROM users
        WHERE email = ?
        LIMIT 1
      `,
      [normalizedEmail],
    )

    const userRow = rows[0]

    if (!userRow) {
      return res.status(401).json({
        message: '邮箱或密码错误。',
      })
    }

    const matched = await bcrypt.compare(normalizedPassword, userRow.password_hash)

    if (!matched) {
      return res.status(401).json({
        message: '邮箱或密码错误。',
      })
    }

    const user = sanitizeUser(userRow)
    const token = signToken(user)

    return res.json(ok({ token, user }, '登录成功。'))
  } catch (error) {
    return res.status(500).json(fail('登录失败，请稍后重试。'))
  }
})

app.get('/api/auth/me', requireAuth, async (req, res) => {
  try {
    const [rows] = await pool.query(
      `
        SELECT id, name, email, role, target, created_at
        FROM users
        WHERE id = ?
        LIMIT 1
      `,
      [req.auth.sub],
    )

    const userRow = rows[0]

    if (!userRow) {
      return res.status(404).json({
        message: '用户不存在。',
      })
    }

    return res.json(ok({ user: sanitizeUser(userRow) }))
  } catch (error) {
    return res.status(500).json(fail('获取用户信息失败。'))
  }
})

app.get('/api/graph/subgraph', async (req, res) => {
  try {
    const data = await getSubgraph(req.query.root || '', Number(req.query.limit || 12))
    return res.json(ok(data))
  } catch (error) {
    return res.status(500).json(fail('获取知识图谱子图失败。'))
  }
})

app.get('/api/knowledge/:knowledgeId', async (req, res) => {
  try {
    const detail = await getKnowledgeDetail(req.params.knowledgeId)

    if (!detail) {
      return res.status(404).json(fail('未找到对应知识点。'))
    }

    return res.json(ok(detail))
  } catch (error) {
    return res.status(500).json(fail('获取知识点详情失败。'))
  }
})

app.get('/api/dashboard/summary', optionalAuth, async (req, res) => {
  try {
    const user = await loadUserById(req.auth?.sub)
    const snapshot = await buildLearningSnapshot(user)
    return res.json(
      ok({
        quickStats: snapshot.quickStats,
        growthInsights: snapshot.growthInsights,
        growthSummaryCards: snapshot.growthSummaryCards,
        knowledgeCards: snapshot.featureCards,
        recommendedPath: snapshot.recommendedPath,
        profile: snapshot.profile,
      }),
    )
  } catch (error) {
    return res.status(500).json(fail('获取首页摘要失败。'))
  }
})

app.get('/api/competency/profile', optionalAuth, async (req, res) => {
  try {
    const user = await loadUserById(req.auth?.sub)
    const snapshot = await buildLearningSnapshot(user)
    return res.json(
      ok({
        profile: snapshot.profile,
        radarAxes: snapshot.radarAxes,
        growthInsights: snapshot.growthInsights,
        growthSummaryCards: snapshot.growthSummaryCards,
        heatmapDomains: snapshot.heatmapDomains,
        heatmapRows: snapshot.heatmapRows,
      }),
    )
  } catch (error) {
    return res.status(500).json(fail('获取能力画像失败。'))
  }
})

app.get('/api/recommend/path', optionalAuth, async (req, res) => {
  try {
    const user = await loadUserById(req.auth?.sub)
    const snapshot = await buildLearningSnapshot(user)
    return res.json(
      ok({
        path: snapshot.recommendedPath,
        strategy: snapshot.growthSummaryCards[2]?.value || '补弱优先',
      }),
    )
  } catch (error) {
    return res.status(500).json(fail('获取推荐路径失败。'))
  }
})

app.get('/api/frontier/briefing', async (req, res) => {
  try {
    return res.json(ok(getFrontierBriefing(req.query.knowledge || '')))
  } catch (error) {
    return res.status(500).json(fail('获取前沿知识摘要失败。'))
  }
})

async function start() {
  try {
    await initializeDatabase()
    app.listen(config.port, () => {
      console.log(`Auth backend running at http://localhost:${config.port}`)
    })
  } catch (error) {
    console.error('Backend startup failed:', error.message)
    process.exit(1)
  }
}

start()

async function shutdown() {
  await closeNeo4j().catch(() => null)
  await pool.end().catch(() => null)
}

process.on('SIGINT', () => {
  shutdown().finally(() => process.exit(0))
})
