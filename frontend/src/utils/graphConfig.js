export function createGraphOptions({
  container,
  width,
  height,
  data,
  minimapContainer,
}) {
  return {
    container,
    width,
    height,
    data,
    animation: false,
    autoResize: false,
    zoomRange: [0.08, 4],
    padding: 60,
    layout: {
      type: 'd3-force',
      preventOverlap: true,
      nodeSize: 32,
      alphaDecay: 0.028,
      velocityDecay: 0.4,
      forceSimulation: {
        alphaMin: 0.001,
      },
      manyBody: {
        strength: -280,
        distanceMax: 600,
      },
      link: {
        distance: 100,
        strength: 0.3,
      },
      collide: {
        radius: 30,
        strength: 0.7,
      },
      x: { strength: 0.04 },
      y: { strength: 0.04 },
    },
    behaviors: [
      { type: 'drag-canvas', key: 'drag-canvas' },
      { type: 'zoom-canvas', key: 'zoom-canvas', sensitivity: 0.72 },
      { type: 'drag-element', key: 'drag-element' },
    ],
    plugins: [
      {
        type: 'minimap',
        key: 'minimap',
        size: [180, 130],
        container: minimapContainer,
        className: 'graph-minimap-inner',
        maskStyle: {
          border: '1px solid rgba(99, 102, 241, 0.4)',
          background: 'rgba(99, 102, 241, 0.08)',
        },
        containerStyle: {
          border: '1px solid rgba(255,255,255,0.06)',
          borderRadius: '14px',
          background: 'rgba(10, 15, 28, 0.95)',
          overflow: 'hidden',
        },
      },
    ],
    node: {
      type: 'circle',
      style: {
        labelMaxWidth: 120,
        cursor: 'pointer',
      },
      state: {
        dimmed: {
          opacity: 0.08,
          labelOpacity: 0,
        },
        active: {
          opacity: 1,
          labelOpacity: 1,
          halo: true,
          haloLineWidth: 2,
          haloStroke: 'rgba(129, 140, 248, 0.3)',
        },
        selected: {
          halo: true,
          haloStroke: 'rgba(129, 140, 248, 0.5)',
          haloLineWidth: 3,
          haloOpacity: 0.6,
        },
        path: {
          lineWidth: 4,
          halo: true,
          haloStroke: '#818CF8',
          haloLineWidth: 2,
        },
      },
    },
    edge: {
      type: 'line',
      style: {
        cursor: 'pointer',
      },
      state: {
        dimmed: {
          opacity: 0.04,
          labelOpacity: 0,
        },
        active: {
          opacity: 0.95,
          lineWidth: 3,
          labelOpacity: 1,
          stroke: '#C7D2FE',
        },
        path: {
          opacity: 1,
          lineWidth: 3.2,
          stroke: '#818CF8',
          labelOpacity: 1,
          lineDash: [10, 6],
        },
      },
    },
  }
}
