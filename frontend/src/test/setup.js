import { afterEach, beforeEach, vi } from 'vitest'

const canvasContextStub = {
  arc: vi.fn(),
  beginPath: vi.fn(),
  clearRect: vi.fn(),
  closePath: vi.fn(),
  fill: vi.fn(),
  fillText: vi.fn(),
  lineTo: vi.fn(),
  measureText: vi.fn((text = '') => ({ width: String(text).length * 8 })),
  moveTo: vi.fn(),
  restore: vi.fn(),
  save: vi.fn(),
  scale: vi.fn(),
  setLineDash: vi.fn(),
  setTransform: vi.fn(),
  stroke: vi.fn(),
  translate: vi.fn(),
}

function installCanvasMock() {
  if (!window.HTMLCanvasElement) return
  Object.defineProperty(window.HTMLCanvasElement.prototype, 'getContext', {
    configurable: true,
    writable: true,
    value: vi.fn(() => canvasContextStub),
  })
}

installCanvasMock()

beforeEach(() => {
  window.localStorage.clear()
  installCanvasMock()
})

afterEach(() => {
  vi.restoreAllMocks()
  installCanvasMock()
})

if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

if (!window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

if (!globalThis.requestAnimationFrame) {
  globalThis.requestAnimationFrame = (callback) => setTimeout(() => callback(Date.now()), 16)
}

if (!globalThis.cancelAnimationFrame) {
  globalThis.cancelAnimationFrame = (handle) => clearTimeout(handle)
}
