const neo4j = require('neo4j-driver')

const config = require('../config')

const driver = neo4j.driver(
  config.neo4j.uri,
  neo4j.auth.basic(config.neo4j.user, config.neo4j.password),
)

async function runRead(query, params = {}) {
  const session = driver.session({
    database: config.neo4j.database,
    defaultAccessMode: neo4j.session.READ,
  })

  try {
    return await session.run(query, params)
  } finally {
    await session.close()
  }
}

async function closeNeo4j() {
  await driver.close()
}

module.exports = {
  runRead,
  closeNeo4j,
}
