import {
  ALL_COURSE_TITLE,
  COURSE_TITLE_ORDER,
  DEFAULT_COURSE_TITLE,
  normalizeCourseTitle,
  resolveNodeCourseTitle,
} from './courseMeta'

const COURSE_COLOR_MAP = {
  高等数学: { primary: '#4DB8C7', light: '#E3F4F7', ring: '#A0D8E1', dark: '#227C89' },
  线性代数: { primary: '#5BA3D9', light: '#E5F0FA', ring: '#A5CBE8', dark: '#2D709F' },
  概率与统计: { primary: '#4DB8A5', light: '#E3F5F0', ring: '#A0D8CA', dark: '#267B6C' },
  数据结构: { primary: '#4DB87A', light: '#E3F5EA', ring: '#A0D8B2', dark: '#25774C' },
  Python编程: { primary: '#4F8ACC', light: '#E5EFFB', ring: '#A6C2EB', dark: '#255D96' },
  人工智能导论: { primary: '#7A8DA6', light: '#EDF1F5', ring: '#BEC9D5', dark: '#46586F' },
  机器学习: { primary: '#6C8EEF', light: '#E8EDFC', ring: '#B0C4F5', dark: '#3656B7' },
  深度学习: { primary: '#9B7FE6', light: '#EDE8FB', ring: '#C4B3F0', dark: '#6347B8' },
  自然语言处理: { primary: '#E0A542', light: '#FCF1DE', ring: '#EFC98A', dark: '#9F6C19' },
  计算机视觉: { primary: '#E07272', light: '#FCE6E6', ring: '#EFA5A5', dark: '#A94444' },
  强化学习: { primary: '#D97757', light: '#FAE8E2', ring: '#E8B19E', dark: '#9A4C30' },
  知识图谱: { primary: '#A570E0', light: '#EFE4FB', ring: '#C9A8EF', dark: '#6B38A8' },
  大模型: { primary: '#E06090', light: '#FCE1EB', ring: '#EFA0BC', dark: '#A93C69' },
  AI工程: { primary: '#6D9F5A', light: '#EEF6E9', ring: '#BDD8B1', dark: '#416930' },
  AI伦理: { primary: '#C98B3C', light: '#F8EEDC', ring: '#E6C48F', dark: '#86551A' },
  其他: { primary: '#8C9BB5', light: '#EBEEF4', ring: '#B5C0D5', dark: '#55627B' },
}

const FALLBACK_PALETTE = ['#6C8EEF', '#E06090', '#E0A542', '#4DB87A', '#5BA3D9', '#A570E0']

const ENTITY_LABEL_MAP = {
  algorithm: '算法',
  concept: '概念',
  method: '方法',
  application: '应用',
  topic: '主题',
}

function stringHash(value = '') {
  return [...String(value)].reduce((hash, char) => hash + char.charCodeAt(0), 0)
}

function withAlpha(hex = '#64748B', alpha = '22') {
  const normalized = String(hex || '').trim()
  return normalized.startsWith('#') && normalized.length === 7 ? `${normalized}${alpha}` : normalized
}

export function getCoursePalette(courseTitle = DEFAULT_COURSE_TITLE) {
  const key = normalizeCourseTitle(courseTitle, DEFAULT_COURSE_TITLE)
  if (COURSE_COLOR_MAP[key]) return COURSE_COLOR_MAP[key]

  const seed = stringHash(key)
  const primary = FALLBACK_PALETTE[seed % FALLBACK_PALETTE.length]
  return {
    primary,
    light: withAlpha(primary, '22'),
    ring: withAlpha(primary, '66'),
    dark: primary,
  }
}

export function getEntityLabel(entityType = '') {
  return ENTITY_LABEL_MAP[String(entityType || '').trim()] || '知识点'
}

export function getCourseOptions(nodes = []) {
  const courses = new Set([ALL_COURSE_TITLE])
  nodes.forEach((node) => {
    courses.add(resolveNodeCourseTitle(node, DEFAULT_COURSE_TITLE))
  })

  const orderIndex = new Map(COURSE_TITLE_ORDER.map((title, index) => [title, index]))
  return [...courses].sort((left, right) => {
    if (left === ALL_COURSE_TITLE) return -1
    if (right === ALL_COURSE_TITLE) return 1
    const leftRank = orderIndex.get(left) ?? Number.MAX_SAFE_INTEGER
    const rightRank = orderIndex.get(right) ?? Number.MAX_SAFE_INTEGER
    if (leftRank !== rightRank) return leftRank - rightRank
    return String(left).localeCompare(String(right), 'zh-CN')
  })
}

export { COURSE_COLOR_MAP, FALLBACK_PALETTE }
