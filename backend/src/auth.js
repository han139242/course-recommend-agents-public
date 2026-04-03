const jwt = require('jsonwebtoken')

const config = require('./config')

function signToken(user) {
  return jwt.sign(
    {
      sub: user.id,
      email: user.email,
      name: user.name,
    },
    config.jwtSecret,
    {
      expiresIn: '7d',
    },
  )
}

function sanitizeUser(row) {
  return {
    id: row.id,
    name: row.name,
    email: row.email,
    role: row.role,
    target: row.target,
    createdAt: row.created_at,
  }
}

function parseBearerToken(req) {
  const authorization = req.headers.authorization || ''
  const [scheme, token] = authorization.split(' ')

  if (scheme !== 'Bearer' || !token) {
    return null
  }

  return token
}

function verifyToken(token) {
  return jwt.verify(token, config.jwtSecret)
}

function requireAuth(req, res, next) {
  const token = parseBearerToken(req)

  if (!token) {
    return res.status(401).json({
      message: '请先登录后再继续。',
    })
  }

  try {
    req.auth = verifyToken(token)
    return next()
  } catch (error) {
    return res.status(401).json({
      message: '登录状态已失效，请重新登录。',
    })
  }
}

function optionalAuth(req, res, next) {
  const token = parseBearerToken(req)

  if (!token) {
    req.auth = null
    return next()
  }

  try {
    req.auth = verifyToken(token)
  } catch (error) {
    req.auth = null
  }

  return next()
}

module.exports = {
  signToken,
  sanitizeUser,
  requireAuth,
  optionalAuth,
}
