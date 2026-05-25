import { describe, expect, it } from 'vitest'

import {
  DEFAULT_COURSE_TITLE,
  getCourseTitleFromSource,
  normalizeCourseTitle,
  resolveNodeCourseTitle,
} from '../courseMeta'

describe('courseMeta', () => {
  it('prefers direct curated course sources when resolving course titles', () => {
    expect(getCourseTitleFromSource('course_ai_intro')).toBe('人工智能导论')
    expect(getCourseTitleFromSource('C:/graph/course_knowledge_graph.json')).toBe('知识图谱')
    expect(
      resolveNodeCourseTitle({
        chapterTitle: '人工智能基础知识骨架',
        curatedSource: 'course_ai_intro',
      }),
    ).toBe('人工智能导论')
  })

  it('keeps generic skeleton labels from collapsing into artificial-intelligence-intro by default', () => {
    expect(normalizeCourseTitle('人工智能基础知识骨架')).toBe(DEFAULT_COURSE_TITLE)
    expect(normalizeCourseTitle('人工智能课程知识图谱')).toBe(DEFAULT_COURSE_TITLE)
    expect(resolveNodeCourseTitle({ curatedSource: 'curated_ai_professional_ontology' })).toBe(DEFAULT_COURSE_TITLE)
  })

  it('folds specific legacy chapter titles back into canonical courses', () => {
    expect(normalizeCourseTitle('自然语言处理及其应用')).toBe('自然语言处理')
    expect(normalizeCourseTitle('概率统计')).toBe('概率与统计')
    expect(normalizeCourseTitle('人工神经网络及其应用')).toBe('深度学习')
    expect(normalizeCourseTitle('智能体与多智能体系统')).toBe('人工智能导论')
  })

  it('infers canonical courses from batch-style and node-specific hints', () => {
    expect(resolveNodeCourseTitle({ sourcePath: 'curated/batch_22_automl_compression_edge_ai' })).toBe('AI工程')
    expect(resolveNodeCourseTitle({ sourcePath: 'curated/batch_07_nlp_foundation' })).toBe('自然语言处理')
    expect(resolveNodeCourseTitle({ curatedSource: 'batch_28_machine_learning_full_route' })).toBe('机器学习')
    expect(resolveNodeCourseTitle({ canonicalName: '图像分割', curatedSource: 'curated_ai_professional_ontology' })).toBe('计算机视觉')
    expect(resolveNodeCourseTitle({ canonicalName: '马尔可夫决策过程', curatedSource: 'curated_ai_professional_ontology' })).toBe('强化学习')
    expect(resolveNodeCourseTitle({ canonicalName: 'BERT', curatedSource: 'curated_ai_professional_ontology' })).toBe('自然语言处理')
    expect(resolveNodeCourseTitle({ canonicalName: '知识表示', chapterTitle: '人工智能基础知识骨架' })).toBe('人工智能导论')
  })

  it('keeps already-normalized titles stable', () => {
    expect(normalizeCourseTitle('机器学习')).toBe('机器学习')
    expect(resolveNodeCourseTitle({ chapterTitle: 'AI工程' })).toBe('AI工程')
  })
})
