import { request } from './request'

export function fetchEvaluationDemo() {
  return request('/evaluation/demo')
}
