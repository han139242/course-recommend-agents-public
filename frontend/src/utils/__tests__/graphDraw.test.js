import { __graphDrawTestUtils } from '../graphDraw'

describe('graphDraw test helpers', () => {
  it('caches repeated text measurements like Neo4j text measurement utils', () => {
    let measureCount = 0
    const ctx = {
      font: '12px Test',
      measureText(text) {
        measureCount += 1
        return { width: String(text || '').length * 8 }
      },
    }

    __graphDrawTestUtils.clearTextWidthCache()

    expect(__graphDrawTestUtils.measureTextWidthCached(ctx, 'PREREQUISITE_OF')).toBe(120)
    expect(__graphDrawTestUtils.measureTextWidthCached(ctx, 'PREREQUISITE_OF')).toBe(120)
    ctx.font = '13px Test'
    expect(__graphDrawTestUtils.measureTextWidthCached(ctx, 'PREREQUISITE_OF')).toBe(120)
    expect(measureCount).toBe(2)
  })

  it('spreads multiple edges between the same node pair into symmetric curvatures', () => {
    const group = [
      { id: 'e1', source: 'A', target: 'B' },
      { id: 'e2', source: 'A', target: 'B' },
      { id: 'e3', source: 'A', target: 'B' },
    ]
    const source = { id: 'A', x: 0, y: 0 }
    const target = { id: 'B', x: 220, y: 0 }

    const curvatureMap = __graphDrawTestUtils.assignPairCurvatures(group, source, target)
    const curvatures = [...curvatureMap.values()]

    expect(curvatureMap.size).toBe(3)
    expect(new Set(curvatures).size).toBeGreaterThan(1)
    expect(Math.min(...curvatures)).toBeLessThan(0)
    expect(Math.max(...curvatures)).toBeGreaterThan(0)
  })

  it('keeps the highest-priority relationship closest to the center line for readability', () => {
    const group = [
      { id: 'related', source: 'A', target: 'B', view: { data: { relationType: 'RELATED_TO' } } },
      { id: 'prerequisite', source: 'A', target: 'B', view: { data: { relationType: 'PREREQUISITE_OF' } } },
      { id: 'uses', source: 'A', target: 'B', view: { data: { relationType: 'USES' } } },
    ]
    const source = { id: 'A', x: 0, y: 0 }
    const target = { id: 'B', x: 260, y: 0 }

    const curvatureMap = __graphDrawTestUtils.assignPairCurvatures(group, source, target, {})

    expect(curvatureMap.get('prerequisite')).toBe(0)
    expect(Math.abs(curvatureMap.get('related'))).toBeGreaterThan(0)
    expect(Math.abs(curvatureMap.get('uses'))).toBeGreaterThan(0)
  })

  it('only prefers centered edge captions when the shaft is long and stable enough', () => {
    expect(
      __graphDrawTestUtils.canPlaceEdgeLabelOnShaft(
        { distance: 240, curvature: 0.04 },
        68,
      ),
    ).toBe(true)

    expect(
      __graphDrawTestUtils.canPlaceEdgeLabelOnShaft(
        { distance: 96, curvature: 0.32 },
        68,
      ),
    ).toBe(false)
  })

  it('applies Neo4j-style label density thresholds by zoom and quality tier', () => {
    expect(__graphDrawTestUtils.shouldShowNodeLabel({ view: { data: { isRoot: false, isExpanded: false } } }, 0.3, new Set(), 'low')).toBe(false)
    expect(__graphDrawTestUtils.shouldShowNodeLabel({ view: { data: { isRoot: true, isExpanded: false } } }, 0.3, new Set(), 'low')).toBe(true)
    expect(__graphDrawTestUtils.shouldShowNodeLabel({ view: { data: { isRoot: false, isExpanded: true } } }, 0.45, new Set(), 'high')).toBe(true)

    expect(__graphDrawTestUtils.shouldShowEdgeLabel(0.7, 8, new Set(), 'high')).toBe(false)
    expect(__graphDrawTestUtils.shouldShowEdgeLabel(0.74, 8, new Set(['path']), 'high')).toBe(true)
    expect(__graphDrawTestUtils.shouldShowEdgeLabel(1.02, 80, new Set(), 'high')).toBe(false)
    expect(__graphDrawTestUtils.shouldShowEdgeLabel(0.92, 12, new Set(['active']), 'medium')).toBe(true)
  })
})
