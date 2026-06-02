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
