import { request, streamRequest } from './request'

export function streamAssistantChat(messages, { authenticated = true, signal, onMessage } = {}) {
  return streamRequest(authenticated ? '/assistant/chat' : '/chat', {
    method: 'POST',
    auth: authenticated,
    body: { messages },
    signal,
    onMessage,
  })
}

export function generateKnowledgeCard(payload) {
  return request('/assistant/knowledge-card', {
    method: 'POST',
    body: payload,
  })
}
