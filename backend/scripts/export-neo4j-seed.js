const fs = require('fs/promises')
const path = require('path')
const neo4j = require('neo4j-driver')

const { closeNeo4j, runRead } = require('../src/services/neo4j')

const OUTPUT_PATH = path.resolve(__dirname, '../../db/neo4j/bootstrap/knowledge_graph_seed.cypher')
const BATCH_SIZE = 200

function chunk(items, size) {
  const result = []
  for (let index = 0; index < items.length; index += size) {
    result.push(items.slice(index, index + size))
  }
  return result
}

function escapeKey(key) {
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(key) ? key : `\`${String(key).replace(/`/g, '``')}\``
}

function normalizeValue(value) {
  if (neo4j.isInt(value)) {
    return value.toNumber()
  }

  if (Array.isArray(value)) {
    return value.map((item) => normalizeValue(item))
  }

  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, normalizeValue(item)]),
    )
  }

  return value
}

function toCypherValue(value) {
  if (value === null || value === undefined) {
    return 'null'
  }

  if (typeof value === 'string') {
    return JSON.stringify(value)
  }

  if (typeof value === 'number') {
    return Number.isFinite(value) ? String(value) : 'null'
  }

  if (typeof value === 'boolean') {
    return value ? 'true' : 'false'
  }

  if (Array.isArray(value)) {
    return `[${value.map((item) => toCypherValue(item)).join(', ')}]`
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value).map(
      ([key, item]) => `${escapeKey(key)}: ${toCypherValue(item)}`,
    )
    return `{ ${entries.join(', ')} }`
  }

  return JSON.stringify(String(value))
}

function renderUnwindBlock(rows, body) {
  return `UNWIND [\n${rows.map((row) => `  ${toCypherValue(row)}`).join(',\n')}\n] AS row\n${body}\n`
}

async function exportNodes() {
  const result = await runRead(
    `
    MATCH (n:KnowledgeNode)
    RETURN properties(n) AS props
    ORDER BY props.node_id ASC
    `,
  )

  return result.records.map((record) => normalizeValue(record.get('props')))
}

async function exportRelationships() {
  const result = await runRead(
    `
    MATCH (a:KnowledgeNode)-[r]->(b:KnowledgeNode)
    RETURN type(r) AS relationType, properties(r) AS props
    ORDER BY relationType ASC, props.relation_id ASC, props.source_id ASC, props.target_id ASC
    `,
  )

  return result.records.reduce((acc, record) => {
    const relationType = record.get('relationType')
    if (!acc[relationType]) {
      acc[relationType] = []
    }
    acc[relationType].push(normalizeValue(record.get('props')))
    return acc
  }, {})
}

async function main() {
  try {
    const [nodes, relationshipsByType] = await Promise.all([exportNodes(), exportRelationships()])
    const lines = [
      '// Auto-generated from the current Neo4j graph.',
      '// Regenerate with: node backend/scripts/export-neo4j-seed.js',
      '',
      'CREATE CONSTRAINT knowledge_id_unique IF NOT EXISTS FOR (n:KnowledgeNode) REQUIRE n.node_id IS UNIQUE;',
      '',
    ]

    for (const batch of chunk(nodes, BATCH_SIZE)) {
      lines.push(
        renderUnwindBlock(
          batch,
          `MERGE (n:KnowledgeNode {node_id: row.node_id})
SET n += row;`,
        ),
      )
    }

    for (const [relationType, rows] of Object.entries(relationshipsByType)) {
      for (const batch of chunk(rows, BATCH_SIZE)) {
        lines.push(
          renderUnwindBlock(
            batch,
            `MATCH (a:KnowledgeNode {node_id: row.source_id})
MATCH (b:KnowledgeNode {node_id: row.target_id})
MERGE (a)-[r:${relationType}]->(b)
SET r += row;`,
          ),
        )
      }
    }

    await fs.mkdir(path.dirname(OUTPUT_PATH), { recursive: true })
    await fs.writeFile(OUTPUT_PATH, `${lines.join('\n')}\n`, 'utf8')
    console.log(`Neo4j seed exported to ${OUTPUT_PATH}`)
    console.log(`Nodes: ${nodes.length}`)
    console.log(
      `Relationships: ${Object.values(relationshipsByType).reduce((sum, items) => sum + items.length, 0)}`,
    )
  } finally {
    await closeNeo4j().catch(() => null)
  }
}

main().catch((error) => {
  console.error(error.stack || error.message)
  process.exit(1)
})
