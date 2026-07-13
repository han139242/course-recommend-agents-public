// Poll provider-backed asynchronous image/video tasks through the backend.
// Credentials remain server-side and the backend persists the final artifact.
export function useMediaStatusPolling(fetchStatus, options = {}) {
  const intervalMs = options.intervalMs || 4000
  const maxAttempts = options.maxAttempts || 30
  const timers = new Map()
  let stopped = false

  function clear(resourceId) {
    const timer = timers.get(resourceId)
    if (timer) clearTimeout(timer)
    timers.delete(resourceId)
  }

  async function poll(resourceId, callbacks = {}, attempt = 0) {
    if (!resourceId || stopped) return
    clear(resourceId)
    try {
      const status = await fetchStatus(resourceId)
      callbacks.onUpdate?.(status)
      if (status?.ready) {
        await callbacks.onReady?.(status)
        return
      }
      if (['failed', 'error', 'canceled', 'cancelled'].includes(String(status?.status || '').toLowerCase())) {
        callbacks.onError?.(new Error(status?.hint || '媒体生成失败'))
        return
      }
      if (attempt + 1 >= maxAttempts) {
        callbacks.onError?.(new Error('媒体仍在生成，可稍后在资源详情页继续刷新。'))
        return
      }
      timers.set(resourceId, setTimeout(() => poll(resourceId, callbacks, attempt + 1), intervalMs))
    } catch (error) {
      if (attempt + 1 >= maxAttempts) {
        callbacks.onError?.(error)
        return
      }
      timers.set(resourceId, setTimeout(() => poll(resourceId, callbacks, attempt + 1), intervalMs))
    }
  }

  function stopAll() {
    stopped = true
    for (const timer of timers.values()) clearTimeout(timer)
    timers.clear()
  }

  return { poll, stopAll }
}
