const fs = require('fs/promises')
const path = require('path')
const mysql = require('mysql2/promise')

const config = require('./config')

const pool = mysql.createPool({
  host: config.db.host,
  port: config.db.port,
  user: config.db.user,
  password: config.db.password,
  database: config.db.database,
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0,
  multipleStatements: true,
})

async function initializeDatabase() {
  const schemaPath = path.resolve(__dirname, '../../db/schema.sql')
  const schemaSql = await fs.readFile(schemaPath, 'utf-8')
  await pool.query(schemaSql)
}

module.exports = {
  pool,
  initializeDatabase,
}
