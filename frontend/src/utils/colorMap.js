const COURSE_COLOR_MAP = {
  '机器学习': { primary: '#4F46E5', light: '#818CF8', dark: '#3730A3' },
  '深度学习': { primary: '#7C3AED', light: '#A78BFA', dark: '#5B21B6' },
  '数学基础': { primary: '#0891B2', light: '#22D3EE', dark: '#155E75' },
  '线性代数': { primary: '#0284C7', light: '#38BDF8', dark: '#0C4A6E' },
  '概率与统计': { primary: '#0D9488', light: '#2DD4BF', dark: '#115E59' },
  '数据结构': { primary: '#059669', light: '#34D399', dark: '#064E3B' },
  '自然语言处理': { primary: '#D97706', light: '#FBBF24', dark: '#92400E' },
  '计算机视觉': { primary: '#DC2626', light: '#F87171', dark: '#991B1B' },
  '大模型': { primary: '#E11D48', light: '#FB7185', dark: '#9F1239' },
  '知识图谱': { primary: '#9333EA', light: '#C084FC', dark: '#6B21A8' },
  '其他': { primary: '#64748B', light: '#94A3B8', dark: '#334155' },
}

const FALLBACK_PALETTE = ['#6366F1', '#EC4899', '#F59E0B', '#10B981', '#3B82F6', '#8B5CF6']

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

export function getCoursePalette(courseTitle = '其他') {
  const key = String(courseTitle || '其他').trim() || '其他'
  if (COURSE_COLOR_MAP[key]) return COURSE_COLOR_MAP[key]

  const seed = stringHash(key)
  const primary = FALLBACK_PALETTE[seed % FALLBACK_PALETTE.length]
  return {
    primary,
    light: primary,
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
