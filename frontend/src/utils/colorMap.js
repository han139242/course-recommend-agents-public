const COURSE_COLOR_MAP = {
  '机器学习': { primary: '#6C8EEF', light: '#E8EDFC', ring: '#B0C4F5', dark: '#3656B7' },
  '深度学习': { primary: '#9B7FE6', light: '#EDE8FB', ring: '#C4B3F0', dark: '#6347B8' },
  '数学基础': { primary: '#4DB8C7', light: '#E3F4F7', ring: '#A0D8E1', dark: '#227C89' },
  '线性代数': { primary: '#5BA3D9', light: '#E5F0FA', ring: '#A5CBE8', dark: '#2D709F' },
  '概率与统计': { primary: '#4DB8A5', light: '#E3F5F0', ring: '#A0D8CA', dark: '#267B6C' },
  '数据结构': { primary: '#4DB87A', light: '#E3F5EA', ring: '#A0D8B2', dark: '#25774C' },
  '自然语言处理': { primary: '#E0A542', light: '#FCF1DE', ring: '#EFC98A', dark: '#9F6C19' },
  '计算机视觉': { primary: '#E07272', light: '#FCE6E6', ring: '#EFA5A5', dark: '#A94444' },
  '大模型': { primary: '#E06090', light: '#FCE1EB', ring: '#EFA0BC', dark: '#A93C69' },
  '知识图谱': { primary: '#A570E0', light: '#EFE4FB', ring: '#C9A8EF', dark: '#6B38A8' },
  '其他': { primary: '#8C9BB5', light: '#EBEEF4', ring: '#B5C0D5', dark: '#55627B' },
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

export function getCoursePalette(courseTitle = '其他') {
  const key = String(courseTitle || '其他').trim() || '其他'
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
  const courses = new Set(['全部'])
  nodes.forEach((node) => {
    courses.add(String(node.chapterTitle || '其他').trim() || '其他')
  })
  return [...courses]
}

export { COURSE_COLOR_MAP, FALLBACK_PALETTE }
