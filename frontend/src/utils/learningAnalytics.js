import { domains } from '../data/courseData'
import { resolveFriendlyKnowledgeRouteId } from './knowledgeLookup'

function clampScore(value, fallback = 0) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return fallback
  return Math.max(0, Math.min(100, numeric))
}

export function createProgressMasteryMap(progressItems = []) {
  const masteryMap = {}

  for (const item of progressItems || []) {
    const routeId = resolveFriendlyKnowledgeRouteId({
      routeKnowledgeId: item?.knowledgeId,
      knowledgeId: item?.knowledgeId,
      id: item?.knowledgeId,
      name: item?.canonicalName,
      canonicalName: item?.canonicalName,
    })

    if (!routeId) continue
    masteryMap[routeId] = clampScore(item?.masteryScore)
  }

  return masteryMap
}

export function buildDisplayDomains(progressItems = []) {
  const masteryMap = createProgressMasteryMap(progressItems)

  return domains.map((domain) => ({
    ...domain,
    points: domain.points.map((point) => ({
      ...point,
      mastery: masteryMap[point.id] ?? point.mastery,
    })),
  }))
}

export function flattenDisplayPoints(progressItems = []) {
  return buildDisplayDomains(progressItems).flatMap((domain) =>
    domain.points.map((point, index) => ({
      ...point,
      domainId: domain.id,
      domainName: domain.name,
      domainDesc: domain.desc,
      order: index,
    })),
  )
}

export function findDisplayPoint(knowledgeId = '', progressItems = []) {
  return flattenDisplayPoints(progressItems).find((point) => point.id === knowledgeId) || null
}

export function buildAbilitySnapshot(progressItems = []) {
  const displayDomains = buildDisplayDomains(progressItems)
  const coreDomains = displayDomains.filter((domain) => domain.id !== 'practice')

  const radarAxes = coreDomains.map((domain) => {
    const total = domain.points.reduce((sum, point) => sum + clampScore(point.mastery), 0)
    return {
      name: domain.name,
      value: Math.round(total / Math.max(domain.points.length, 1)),
    }
  })

  const sortedAxes = [...radarAxes].sort((left, right) => right.value - left.value)
  const weakestAxes = [...radarAxes].sort((left, right) => left.value - right.value)

  const growthInsights = weakestAxes.slice(0, 3).map((axis) => ({
    label: axis.name,
    value: axis.value,
    note: `由该路径下关键知识点的掌握度聚合得出，当前更适合作为下一轮补强重点。`,
  }))

  const growthSummaryCards = [
    { label: '当前优势', value: sortedAxes[0]?.name || '暂无' },
    { label: '优先补强', value: weakestAxes[0]?.name || '暂无' },
    { label: '生成方式', value: '按知识路径聚合' },
  ]

  const heatmapDomains = radarAxes.map((axis) => axis.name)
  const heatmapRows = coreDomains
    .flatMap((domain) =>
      domain.points.map((point) => ({
        label: point.name,
        values: heatmapDomains.map((axisName) => (axisName === domain.name ? clampScore(point.mastery) : 0)),
      })),
    )
    .sort((left, right) => {
      const leftMax = Math.max(...left.values)
      const rightMax = Math.max(...right.values)
      return leftMax - rightMax
    })
    .slice(0, 8)

  return {
    profileSummary: '能力图谱由知识路径上的关键节点掌握度聚合生成，用来回答“哪条路径已经稳了，哪条路径还需要补”。',
    radarAxes,
    growthInsights,
    growthSummaryCards,
    heatmapDomains,
    heatmapRows,
  }
}
