const neo4j = require('neo4j-driver')

const { runRead } = require('./neo4j')

const DEMO_ROOT_NAMES = ['梯度下降', '卷积神经网络', '检索增强生成', '知识图谱', '偏导数']

function toNumber(value) {
  if (neo4j.isInt(value)) {
    return value.toNumber()
  }

  return Number(value || 0)
}

function mapNode(node) {
  if (!node) {
    return null
  }

  return {
    id: node.properties.node_id,
    name: node.properties.name || node.properties.canonical_name,
    canonicalName: node.properties.canonical_name || node.properties.name,
    entityType: node.properties.entity_type || 'concept',
    description: node.properties.description || '',
    chapterTitle: node.properties.chapter_title || '',
    sourcePath: node.properties.source_path || '',
  }
}

async function resolveRootNode(rootKey = '') {
  const trimmedKey = String(rootKey || '').trim()

  if (trimmedKey) {
    const result = await runRead(
      `
      MATCH (root:KnowledgeNode)
      WHERE root.node_id = $key OR root.canonical_name = $key OR root.name = $key
      RETURN root
      LIMIT 1
      `,
      { key: trimmedKey },
    )

    if (result.records[0]) {
      return mapNode(result.records[0].get('root'))
    }
  }

  const result = await runRead(
    `
    MATCH (root:KnowledgeNode)
    WHERE root.canonical_name IN $preferred
    OPTIONAL MATCH (root)-[r]-()
    WITH root, count(r) AS degree
    RETURN root
    ORDER BY degree DESC, root.canonical_name ASC
    LIMIT 1
    `,
    { preferred: DEMO_ROOT_NAMES },
  )

  if (result.records[0]) {
    return mapNode(result.records[0].get('root'))
  }

  const fallback = await runRead(
    `
    MATCH (root:KnowledgeNode)
    RETURN root
    ORDER BY root.canonical_name ASC
    LIMIT 1
    `,
  )

  return mapNode(fallback.records[0]?.get('root'))
}

async function getSubgraph(rootKey = '', limit = 12) {
  const root = await resolveRootNode(rootKey)

  if (!root) {
    return {
      root: null,
      nodes: [],
      edges: [],
    }
  }

  const result = await runRead(
    `
    MATCH (root:KnowledgeNode {node_id: $rootId})
    OPTIONAL MATCH (root)-[r]-(neighbor:KnowledgeNode)
    WITH root, neighbor, r
    ORDER BY neighbor.canonical_name ASC
    WITH root, collect(DISTINCT neighbor)[..$limit] AS neighbors, collect(DISTINCT r) AS rels
    RETURN
      root,
      [item IN neighbors WHERE item IS NOT NULL | item] AS neighbors,
      [rel IN rels WHERE rel IS NOT NULL | {
        relationId: rel.relation_id,
        relationType: type(rel),
        sourceId: startNode(rel).node_id,
        targetId: endNode(rel).node_id,
        sourceName: startNode(rel).canonical_name,
        targetName: endNode(rel).canonical_name
      }] AS edges
    `,
    { rootId: root.id, limit: neo4j.int(toNumber(limit)) },
  )

  const record = result.records[0]
  const rootNode = mapNode(record.get('root'))
  const neighbors = record.get('neighbors').map(mapNode)

  return {
    root: rootNode,
    nodes: [rootNode, ...neighbors],
    edges: record.get('edges'),
  }
}

async function getKnowledgeDetail(id) {
  const result = await runRead(
    `
    MATCH (root:KnowledgeNode {node_id: $id})
    CALL {
      WITH root
      OPTIONAL MATCH (pre:KnowledgeNode)-[:PREREQUISITE_OF]->(root)
      RETURN collect(DISTINCT pre.canonical_name) AS prerequisites
    }
    CALL {
      WITH root
      OPTIONAL MATCH (root)-[:RELATED_TO]-(rel:KnowledgeNode)
      RETURN collect(DISTINCT rel.canonical_name) AS related
    }
    CALL {
      WITH root
      OPTIONAL MATCH (root)-[:RELATED_TO|APPLIES_TO|USES]-(app:KnowledgeNode)
      WHERE app.entity_type = 'application'
      RETURN collect(DISTINCT app.canonical_name) AS scenarios
    }
    CALL {
      WITH root
      OPTIONAL MATCH (parent:KnowledgeNode)<-[:SUBTOPIC_OF]-(root)
      RETURN collect(DISTINCT parent.canonical_name) AS parentTopics
    }
    RETURN root, prerequisites, related, scenarios, parentTopics
    LIMIT 1
    `,
    { id },
  )

  const record = result.records[0]

  if (!record) {
    return null
  }

  const root = mapNode(record.get('root'))

  return {
    ...root,
    summary: root.description || `${root.canonicalName} 是当前知识图谱中的重点知识点。`,
    prerequisites: record.get('prerequisites').filter(Boolean),
    related: record.get('related').filter(Boolean),
    scenarios: record.get('scenarios').filter(Boolean),
    parentTopics: record.get('parentTopics').filter(Boolean),
  }
}

async function listFeaturedKnowledge(limit = 6) {
  const result = await runRead(
    `
    MATCH (n:KnowledgeNode)
    WHERE coalesce(n.description, '') <> ''
    OPTIONAL MATCH (n)-[r]-()
    WITH n, count(r) AS degree
    RETURN n, degree
    ORDER BY
      CASE WHEN n.canonical_name IN $preferred THEN 0 ELSE 1 END ASC,
      degree DESC,
      n.canonical_name ASC
    LIMIT $limit
    `,
    {
      preferred: DEMO_ROOT_NAMES,
      limit: neo4j.int(toNumber(limit)),
    },
  )

  return result.records.map((record) => ({
    ...mapNode(record.get('n')),
    degree: toNumber(record.get('degree')),
  }))
}

async function getKnowledgeByNames(names) {
  const normalized = names.map((item) => String(item || '').trim()).filter(Boolean)

  if (normalized.length === 0) {
    return []
  }

  const result = await runRead(
    `
    MATCH (n:KnowledgeNode)
    WHERE n.canonical_name IN $names
    RETURN n
    `,
    { names: normalized },
  )

  return result.records.map((record) => mapNode(record.get('n')))
}

async function getGraphStats() {
  const result = await runRead(
    `
    MATCH (n:KnowledgeNode)
    WITH count(n) AS totalNodes
    MATCH ()-[r]->()
    RETURN totalNodes, count(r) AS totalRelationships
    `,
  )

  const record = result.records[0]

  return {
    totalNodes: toNumber(record.get('totalNodes')),
    totalRelationships: toNumber(record.get('totalRelationships')),
  }
}

module.exports = {
  getGraphStats,
  getKnowledgeDetail,
  getKnowledgeByNames,
  getSubgraph,
  listFeaturedKnowledge,
  resolveRootNode,
}
