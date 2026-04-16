import { request } from './request'

export function loginRequest(payload) {
  return request('/auth/login', {
    method: 'POST',
    auth: false,
    body: payload,
  })
}

export function registerRequest(payload) {
  return request('/auth/register', {
    method: 'POST',
    auth: false,
    body: payload,
  })
}

export function fetchCurrentUser() {
  return request('/auth/me')
}
