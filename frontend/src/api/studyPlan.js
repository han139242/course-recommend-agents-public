import { request } from './request'

export function fetchGoalProfile() {
  return request('/study-plan/profile')
}

export function saveGoalProfile(payload) {
  return request('/study-plan/profile', {
    method: 'PUT',
    body: payload,
  })
}

export function fetchCurrentStudyPlan() {
  return request('/study-plan/current')
}

export function generateStudyPlan() {
  return request('/study-plan/generate', {
    method: 'POST',
    body: {},
  })
}

// adjustStudyPlan 根据最新掌握度与练习表现重新规划学习计划（自适应闭环）。
export function adjustStudyPlan() {
  return request('/study-plan/adjust', {
    method: 'POST',
    body: {},
  })
}
