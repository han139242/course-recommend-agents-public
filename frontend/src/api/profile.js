import { request } from './request'

export function fetchLearningProfile() {
  return request('/profile/current')
}

export function extractLearningProfile(message) {
  return request('/profile/extract', {
    method: 'POST',
    body: { message },
  })
}

export function saveLearningProfile(payload) {
  return request('/profile/current', {
    method: 'PUT',
    body: payload,
  })
}

export function refreshLearningProfile() {
  return request('/profile/refresh', {
    method: 'POST',
    body: {},
  })
}
