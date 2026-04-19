import { DEFAULT_COURSE_TITLE, resolveNodeCourseTitle } from './courseMeta'

function normalizeKeyword(value = '') {
  return String(value || '').trim().toLowerCase().replace(/\s+/g, '')
}

function scoreResult(item, query) {
  const name = normalizeKeyword(item.name)
  const canonical = normalizeKeyword(item.canonicalName)
  const chapter = normalizeKeyword(item.chapterTitle)

  if (!query) return 0
  if (canonical === query || name === query) return 100
  if (canonical.startsWith(query) || name.startsWith(query)) return 80
  if (canonical.includes(query) || name.includes(query)) return 60
  if (chapter.includes(query)) return 30
  return 0
}

export function createSearchIndex(nodes = []) {
  return nodes.map((node) => ({
    id: node.id,
    routeKnowledgeId: node.routeKnowledgeId,
    name: node.name || node.canonicalName || node.routeKnowledgeId || '',
    canonicalName: node.canonicalName || node.name || '',
    chapterTitle: resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE),
    entityType: node.entityType || 'concept',
  }))
}

export function searchInIndex(index = [], keyword = '', limit = 8) {
  const query = normalizeKeyword(keyword)
  if (!query) return []

  return index
    .map((item) => ({ item, score: scoreResult(item, query) }))
    .filter(({ score }) => score > 0)
    .sort((left, right) => {
      if (right.score !== left.score) return right.score - left.score
      return String(left.item.name).localeCompare(String(right.item.name), 'zh-CN')
    })
    .slice(0, limit)
    .map(({ item }) => item)
}
