import { mount } from '@vue/test-utils'

import StudyGoalWizard from '../StudyGoalWizard.vue'

describe('StudyGoalWizard', () => {
  it('submits normalized goal profile fields', async () => {
    const wrapper = mount(StudyGoalWizard, {
      props: {
        modelValue: true,
      },
    })

    await wrapper.find('textarea').setValue('我是大二学生，想做智能体项目')
    const inputs = wrapper.findAll('input')
    await inputs[2].setValue('Python、机器学习')

    await wrapper.find('.primary-btn').trigger('click')

    const submit = wrapper.emitted('submit')?.[0]?.[0]
    expect(submit).toMatchObject({
      gradeLevel: 'sophomore',
      goalType: 'project_building',
      goalText: '我是大二学生，想做智能体项目',
      foundationLevel: 'intermediate',
      learningPreference: 'balanced',
    })
    expect(submit.learnedTopics).toEqual(['Python', '机器学习'])
  })

  it('preloads AI draft profile for confirmation', async () => {
    const wrapper = mount(StudyGoalWizard, {
      props: {
        modelValue: true,
        draftProfile: {
          gradeLevel: 'junior',
          goalType: 'career_preparation',
          goalText: '想准备算法工程师实习',
          targetDirection: '算法工程师',
          learnedTopics: ['线性代数'],
          foundationLevel: 'beginner',
          weeklyHours: 10,
          learningPreference: 'project_first',
        },
      },
    })

    expect(wrapper.find('textarea').element.value).toBe('想准备算法工程师实习')
    expect(wrapper.text()).toContain('制定你的学习目标')
  })
})
