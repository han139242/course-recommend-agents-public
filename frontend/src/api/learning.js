import { request } from './request'

export function fetchLearningProgress() {
  return request('/learning/progress')
}

export function fetchDashboardSummary() {
  return request('/dashboard/summary')
}

export function fetchCompetencyProfile() {
  return request('/competency/profile')
}
