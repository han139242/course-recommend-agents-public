/**
 * 根据各知识域平均掌握度，为「行业实践」项目计算匹配度并排序（可与后端能力画像对齐）。
 */

const DOMAIN_LABEL = {
  math: '数学基础',
  python: 'Python',
  ml: '机器学习',
  dl: '深度学习',
  llm: '大模型',
  kg: '知识图谱',
}

/** @returns {Record<string, number>} 域 id -> 平均掌握度 0–100 */
export function averageMasteryByDomain(allDomains) {
  const out = {}
  for (const d of allDomains) {
    if (d.id === 'practice') continue
    const pts = d.points || []
    if (!pts.length) {
      out[d.id] = 0
      continue
    }
    out[d.id] = pts.reduce((s, p) => s + (p.mastery ?? 0), 0) / pts.length
  }
  return out
}

/**
 * projectFit: 各域权重（反映项目依赖），不必加总为 1
 */
function fitScore(project, masteryByDomain) {
  const fit = project.projectFit || {}
  const entries = Object.entries(fit)
  if (!entries.length) return { score: 0, percent: 0, hint: '' }

  let num = 0
  let den = 0
  const contrib = []
  for (const [domId, w] of entries) {
    const m = (masteryByDomain[domId] ?? 0) / 100
    num += m * w
    den += w
    contrib.push({ domId, w, value: m * w })
  }
  const score = den > 0 ? num / den : 0
  const percent = Math.min(100, Math.round(score * 100))

  contrib.sort((a, b) => b.value - a.value)
  const topNames = contrib
    .slice(0, 2)
    .map((c) => DOMAIN_LABEL[c.domId] || c.domId)
    .filter(Boolean)

  const hint =
    topNames.length > 0
      ? `侧重 ${topNames.join('、')} · 匹配度 ${percent}%`
      : `匹配度 ${percent}%`

  return { score, percent, hint }
}

/** @returns {Array<object>} 带 fitPercent、fitHint、已排序的实践项目 */
export function rankPracticeProjects(allDomains) {
  const practice = allDomains.find((d) => d.id === 'practice')
  if (!practice?.points?.length) return []

  const masteryByDomain = averageMasteryByDomain(allDomains)

  const ranked = practice.points.map((p) => {
    const { score, percent, hint } = fitScore(p, masteryByDomain)
    return {
      ...p,
      fitPercent: percent,
      fitHint: hint,
      fitScore: score,
    }
  })

  ranked.sort((a, b) => {
    if (b.fitScore !== a.fitScore) return b.fitScore - a.fitScore
    return (b.mastery ?? 0) - (a.mastery ?? 0)
  })

  return ranked
}
