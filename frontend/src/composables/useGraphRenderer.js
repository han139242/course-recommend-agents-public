import { drawScene } from '../utils/graphDraw'
import { MAX_ZOOM, MIN_ZOOM } from '../utils/graphLayout'

function easeInOutCubic(t) {
  return t < 0.5 ? 4 * t * t * t : 1 - ((-2 * t + 2) ** 3) / 2
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

export function useGraphRenderer() {
  let canvas = null
  let container = null
  let ctx = null
  let resizeObserver = null
  let frameId = 0
  let sceneGetter = () => ({})
  let transform = { x: 0, y: 0, k: 1 }
  let animationFrame = 0
  let qualityLevel = 'high'
  let frameSample = []

  function updateQuality(now) {
    frameSample.push(now)
    frameSample = frameSample.filter((time) => now - time <= 1000)
    const fps = frameSample.length
    if (fps >= 45) {
      qualityLevel = 'high'
      return
    }
    if (fps >= 25) {
      qualityLevel = 'medium'
      return
    }
    qualityLevel = 'low'
  }

  function adjustDPI() {
    if (!canvas || !ctx || !container) return
    const width = Math.max(container.clientWidth || 0, 320)
    const height = Math.max(container.clientHeight || 0, 240)
    const ratio = window.devicePixelRatio || 1
    canvas.width = Math.round(width * ratio)
    canvas.height = Math.round(height * ratio)
    canvas.style.width = `${width}px`
    canvas.style.height = `${height}px`
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.scale(ratio, ratio)
  }

  function renderFrame() {
    if (!ctx) return
    updateQuality(performance.now())
    const scene = sceneGetter() || {}
    drawScene(ctx, { ...scene, qualityLevel }, transform)
    frameId = window.requestAnimationFrame(renderFrame)
  }

  function init(nextCanvas, nextContainer, getScene) {
    canvas = nextCanvas
    container = nextContainer
    sceneGetter = typeof getScene === 'function' ? getScene : () => ({})
    ctx = canvas?.getContext('2d')
    if (!ctx) {
      throw new Error('Canvas 2D context unavailable.')
    }
    frameSample = []
    qualityLevel = 'high'
    adjustDPI()
    frameId = window.requestAnimationFrame(renderFrame)

    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => adjustDPI())
      resizeObserver.observe(container)
    } else {
      window.addEventListener('resize', adjustDPI)
    }
  }

  function destroy() {
    if (frameId) {
      window.cancelAnimationFrame(frameId)
      frameId = 0
    }
    if (animationFrame) {
      window.cancelAnimationFrame(animationFrame)
      animationFrame = 0
    }
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    } else {
      window.removeEventListener('resize', adjustDPI)
    }
    canvas = null
    container = null
    ctx = null
    frameSample = []
    qualityLevel = 'high'
  }

  function setTransform(nextTransform = {}) {
    transform = {
      x: Number(nextTransform.x || 0),
      y: Number(nextTransform.y || 0),
      k: clamp(Number(nextTransform.k || 1), MIN_ZOOM, MAX_ZOOM),
    }
  }

  function getTransform() {
    return { ...transform }
  }

  function getQualityLevel() {
    return qualityLevel
  }

  function getViewportSize() {
    return {
      width: container?.clientWidth || 0,
      height: container?.clientHeight || 0,
    }
  }

  function screenToWorld(clientX, clientY) {
    const bounds = container?.getBoundingClientRect()
    if (!bounds) return [0, 0]
    const localX = clientX - bounds.left
    const localY = clientY - bounds.top
    return [(localX - transform.x) / transform.k, (localY - transform.y) / transform.k]
  }

  function worldToScreen(x, y) {
    return {
      x: x * transform.k + transform.x,
      y: y * transform.k + transform.y,
    }
  }

  function animateTo(nextTransform, duration = 0) {
    if (animationFrame) {
      window.cancelAnimationFrame(animationFrame)
      animationFrame = 0
    }

    if (!duration) {
      setTransform(nextTransform)
      return Promise.resolve()
    }

    const from = getTransform()
    const to = {
      x: Number(nextTransform.x || 0),
      y: Number(nextTransform.y || 0),
      k: clamp(Number(nextTransform.k || 1), MIN_ZOOM, MAX_ZOOM),
    }
    const start = performance.now()

    return new Promise((resolve) => {
      function step(now) {
        const progress = clamp((now - start) / duration, 0, 1)
        const eased = easeInOutCubic(progress)
        setTransform({
          x: from.x + (to.x - from.x) * eased,
          y: from.y + (to.y - from.y) * eased,
          k: from.k + (to.k - from.k) * eased,
        })
        if (progress >= 1) {
          animationFrame = 0
          resolve()
          return
        }
        animationFrame = window.requestAnimationFrame(step)
      }

      animationFrame = window.requestAnimationFrame(step)
    })
  }

  return {
    init,
    destroy,
    setTransform,
    getTransform,
    animateTo,
    screenToWorld,
    worldToScreen,
    getViewportSize,
    getQualityLevel,
  }
}
