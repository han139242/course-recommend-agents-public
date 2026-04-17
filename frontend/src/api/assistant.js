import { request, streamRequest } from './request'

export function streamAssistantChat(messages, { authenticated = true, signal, onMessage, onEvent } = {}) {
  return streamRequest(authenticated ? '/assistant/chat' : '/assistant/public-chat', {
    method: 'POST',
    auth: authenticated,
    body: { messages },
    signal,
    onMessage,
    onEvent,
  })
}

export function generateKnowledgeCard(payload) {
  return request('/assistant/knowledge-card', {
    method: 'POST',
    body: payload,
  })
}
