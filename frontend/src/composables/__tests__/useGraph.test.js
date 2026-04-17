import { __graphViewTestUtils } from '../useGraph'

describe('useGraph viewport constraints', () => {
  it('recenters small graphs instead of letting them drift outside the viewport', () => {
    const nodes = [
      {
        id: 'A',
        x: -40,
        y: -30,
        view: { style: { collisionRadius: 60 } },
      },
      {
        id: 'B',
        x: 60,
        y: 40,
        view: { style: { collisionRadius: 60 } },
      },
    ]

    const constrained = __graphViewTestUtils.constrainTransformToViewport(
      { x: 420, y: 320, k: 1.08 },
      {
        nodes,
        viewport: { width: 900, height: 560 },
      },
    )

    const bounds = __graphViewTestUtils.getBounds(nodes)
    const centerX = (bounds.centerX * constrained.k) + constrained.x
    const centerY = (bounds.centerY * constrained.k) + constrained.y

    expect(Math.round(centerX)).toBe(Math.round(900 / 2))
    expect(Math.round(centerY)).toBe(Math.round(560 / 2))
  })

  it('clamps large graphs so panning cannot throw all nodes outside the canvas', () => {
    const nodes = [
      {
        id: 'left',
        x: -380,
        y: 0,
        view: { style: { collisionRadius: 72 } },
      },
      {
        id: 'right',
        x: 380,
        y: 0,
        view: { style: { collisionRadius: 72 } },
      },
    ]

    const constrained = __graphViewTestUtils.constrainTransformToViewport(
      { x: 900, y: -240, k: 1.2 },
      {
        nodes,
        viewport: { width: 720, height: 420 },
      },
    )

    const bounds = __graphViewTestUtils.getBounds(nodes)
    const screenLeft = bounds.minX * constrained.k + constrained.x
    const screenRight = bounds.maxX * constrained.k + constrained.x

    expect(screenLeft).toBeLessThanOrEqual(56)
    expect(screenRight).toBeGreaterThanOrEqual(720 - 56)
  })

  it('nudges expanded clusters back into the safe viewport instead of letting children spill outside', () => {
    const bounds = {
      minX: 120,
      maxX: 480,
      minY: -40,
      maxY: 180,
      width: 360,
      height: 220,
      centerX: 300,
      centerY: 70,
    }

    const transform = __graphViewTestUtils.buildTransformToKeepBoundsVisible(
      { x: 220, y: 140, k: 1.1 },
      bounds,
      {
        viewport: { width: 760, height: 520 },
        padding: { left: 90, right: 90, top: 80, bottom: 120 },
        boundsPadding: 0,
      },
    )

    const screenLeft = bounds.minX * transform.k + transform.x
    const screenRight = bounds.maxX * transform.k + transform.x
    const screenTop = bounds.minY * transform.k + transform.y
    const screenBottom = bounds.maxY * transform.k + transform.y

    expect(screenLeft).toBeGreaterThanOrEqual(90)
    expect(screenRight).toBeLessThanOrEqual(760 - 90)
    expect(screenTop).toBeGreaterThanOrEqual(80)
    expect(screenBottom).toBeLessThanOrEqual(520 - 120)
  })

  it('zooms out slightly when an expanded cluster is too large for the safe viewport', () => {
    const bounds = {
      minX: -240,
      maxX: 480,
      minY: -120,
      maxY: 140,
      width: 720,
      height: 260,
      centerX: 120,
      centerY: 10,
    }

    const transform = __graphViewTestUtils.buildTransformToKeepBoundsVisible(
      { x: 180, y: 130, k: 1.08 },
      bounds,
      {
        viewport: { width: 680, height: 440 },
        padding: { left: 100, right: 100, top: 80, bottom: 120 },
        boundsPadding: 0,
      },
    )

    const screenLeft = bounds.minX * transform.k + transform.x
    const screenRight = bounds.maxX * transform.k + transform.x

    expect(transform.k).toBeLessThan(1.08)
    expect(screenLeft).toBeGreaterThanOrEqual(100)
    expect(screenRight).toBeLessThanOrEqual(680 - 100)
  })
})
