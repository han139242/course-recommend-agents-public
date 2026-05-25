function buildLookupVariants(value = '') {
  const normalized = String(value || '').trim()
  if (!normalized) return []

  const lower = normalized.toLowerCase()
  return lower === normalized ? [normalized] : [normalized, lower]
}

function readLookupLabel(entry) {
  if (!entry) return ''
  if (typeof entry === 'string') {
    return String(entry).trim()
  }

  return String(entry.label || entry.canonicalName || entry.name || '').trim()
}

export function looksLikeMachineKnowledgeId(value = '') {
  const normalized = String(value || '').trim()
  return /^[0-9a-f]{16,}$/i.test(normalized)
}

export function extractKnowledgeIdFromRoute(route = '') {
  const normalized = String(route || '').trim()
  const match = normalized.match(/\/knowledge\/([^/?#]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

export function buildKnowledgeTitleLookup(nodes = []) {
  const lookup = {}

  ;(nodes || []).forEach((node) => {
    const label =
      String(node?.canonicalName || node?.name || node?.routeKnowledgeId || node?.id || '').trim()
    if (!label) return

    const keys = [
      node?.routeKnowledgeId,
      node?.id,
      node?.canonicalName,
      node?.name,
      extractKnowledgeIdFromRoute(node?.route),
    ]

    keys.forEach((key) => {
      buildLookupVariants(key).forEach((variant) => {
        const existing = readLookupLabel(lookup[variant])
        if (!existing || looksLikeMachineKnowledgeId(existing)) {
          lookup[variant] = { label }
        }
      })
    })
  })

  return lookup
}

export function resolveKnowledgeTitle(value = '', lookup = {}) {
  for (const variant of buildLookupVariants(value)) {
    const label = readLookupLabel(lookup?.[variant])
    if (label) return label
  }
  return ''
}

export function replaceKnowledgeIdsInText(text = '', lookup = {}) {
  return String(text || '').replace(/\b[0-9a-f]{16,}\b/gi, (matched) => {
    const label = resolveKnowledgeTitle(matched, lookup)
    return label || matched
  })
}

export function resolveRecommendationTitle(item = {}, lookup = {}) {
  const rawTitle = String(item?.title || '').trim()
  const candidates = [
    item?.knowledgeId,
    item?.routeKnowledgeId,
    extractKnowledgeIdFromRoute(item?.route),
    rawTitle,
  ]

  const resolvedLabel = candidates.map((value) => resolveKnowledgeTitle(value, lookup)).find(Boolean) || ''
  if (resolvedLabel && (!rawTitle || looksLikeMachineKnowledgeId(rawTitle))) {
    return resolvedLabel
  }

  const normalizedTitle = replaceKnowledgeIdsInText(rawTitle, lookup)
  return normalizedTitle || resolvedLabel || String(item?.knowledgeId || '').trim() || '推荐知识点'
}

export function normalizeRecommendationPresentation(item = {}, lookup = {}) {
  const displayTitle = resolveRecommendationTitle(item, lookup)
  const displayDetail = replaceKnowledgeIdsInText(item?.detail || '', lookup)
  const sourceCourse = String(item?.sourceCourse || '').trim()
  const targetCourse = String(item?.targetCourse || '').trim()

  return {
    ...item,
    title: displayTitle,
    detail: displayDetail,
    displayTitle,
    displayDetail,
    displayReasonLabel: String(item?.reasonLabel || '').trim() || '图谱推荐',
    displayCourseLabel:
      sourceCourse && targetCourse && sourceCourse !== targetCourse
        ? `${sourceCourse} → ${targetCourse}`
        : targetCourse || sourceCourse || '',
  }
}
