import { reactive } from 'vue'

export function useContextMenu() {
  const state = reactive({
    visible: false,
    x: 0,
    y: 0,
    node: null,
  })

  function open(payload) {
    state.visible = true
    state.x = payload.x
    state.y = payload.y
    state.node = payload.node
  }

  function close() {
    state.visible = false
    state.node = null
  }

  return {
    state,
    open,
    close,
  }
}
