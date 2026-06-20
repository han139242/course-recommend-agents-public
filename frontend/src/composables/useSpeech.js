import { onBeforeUnmount, ref } from 'vue'

// useSpeech 封装浏览器原生语音合成（Web Speech API），为资源讲解、练习题
// 提供语音朗读这一多模态输出形态。生产环境可替换为科大讯飞 TTS，接口保持一致。
export function useSpeech() {
  const supported = typeof window !== 'undefined' && 'speechSynthesis' in window
  const speakingId = ref('')

  // stripMarkup 去除 Markdown/代码标记，让朗读内容更自然。
  function stripMarkup(text) {
    return String(text || '')
      .replace(/```[\s\S]*?```/g, '（此处为代码片段，略）')
      .replace(/[#>*_`~|-]+/g, ' ')
      .replace(/\[(.*?)\]\(.*?\)/g, '$1')
      .replace(/\s+/g, ' ')
      .trim()
  }

  function stop() {
    if (!supported) return
    window.speechSynthesis.cancel()
    speakingId.value = ''
  }

  // speak 朗读一段文本；再次点击同一条目则停止，便于在卡片上切换。
  function speak(text, id = 'default') {
    if (!supported) return
    if (speakingId.value === id) {
      stop()
      return
    }
    window.speechSynthesis.cancel()
    const content = stripMarkup(text)
    if (!content) return
    const utterance = new SpeechSynthesisUtterance(content.slice(0, 4000))
    utterance.lang = 'zh-CN'
    utterance.rate = 1
    utterance.onend = () => {
      if (speakingId.value === id) speakingId.value = ''
    }
    utterance.onerror = () => {
      if (speakingId.value === id) speakingId.value = ''
    }
    speakingId.value = id
    window.speechSynthesis.speak(utterance)
  }

  onBeforeUnmount(stop)

  return { supported, speakingId, speak, stop }
}
