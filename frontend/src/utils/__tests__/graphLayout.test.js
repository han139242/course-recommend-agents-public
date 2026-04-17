import {
  createCircularPlacement,
  createExpansionPlacement,
  getLinkDistance,
  getNodeCharge,
  getLinkStrength,
  getNodeCollisionRadius,
  getRelationPriority,
} from '../graphLayout'

function pairwiseDistances(points = []) {
  const distances = []
  for (let left = 0; left < points.length; left += 1) {
    for (let right = left + 1; right < points.length; right += 1) {
      distances.push(Math.hypot(points[left].x - points[right].x, points[left].y - points[right].y))
    }
  }
  return distances
}

describe('graphLayout', () => {
  it('places seed nodes on a stable circular ring', () => {
    const placements = createCircularPlacement(4, { spacing: 120, yScale: 1 })

    expect(placements).toHaveLength(4)
    const radii = placements.map((point) => Math.round(Math.hypot(point.x, point.y)))
    expect(new Set(radii).size).toBe(1)
    expect(Math.min(...pairwiseDistances(placements))).toBeGreaterThan(100)
  })

  it('creates expansion placements outside the anchor radius with usable spacing', () => {
    const anchor = { id: 'anchor', x: 0, y: 0, collisionRadius: 60 }
    const placements = createExpansionPlacement(anchor, 8, 12, {
      occupiedNodes: [
        { id: 'occupied-1', x: 220, y: 0 },
        { id: 'occupied-2', x: -80, y: -160 },
      ],
    })

    expect(placements).toHaveLength(8)
    placements.forEach((point) => {
      expect(Math.hypot(point.x - anchor.x, point.y - anchor.y)).toBeGreaterThanOrEqual(176)
    })
    expect(Math.min(...pairwiseDistances(placements))).toBeGreaterThan(48)
  })

  it('biases expansion placement toward the preferred viewport direction when space is available', () => {
    const anchor = { id: 'anchor', x: 0, y: 0, collisionRadius: 60 }
    const placements = createExpansionPlacement(anchor, 5, 6, {
      occupiedNodes: [
        { id: 'occupied-right', x: 240, y: 10 },
        { id: 'occupied-bottom-right', x: 160, y: 180 },
      ],
      viewportCenterWorld: { x: -260, y: 0 },
    })

    const averageX = placements.reduce((sum, point) => sum + point.x, 0) / placements.length

    expect(placements).toHaveLength(5)
    expect(averageX).toBeLessThan(0)
  })

  it('avoids the preferred direction when a large occupied node blocks that sector', () => {
    const anchor = { id: 'anchor', x: 0, y: 0, collisionRadius: 60 }
    const placements = createExpansionPlacement(anchor, 4, 6, {
      occupiedNodes: [
        { id: 'occupied-left', x: -230, y: 0, collisionRadius: 152 },
      ],
      viewportCenterWorld: { x: -320, y: 0 },
    })

    const averageX = placements.reduce((sum, point) => sum + point.x, 0) / placements.length
    const furthestLeft = Math.min(...placements.map((point) => point.x))

    expect(placements).toHaveLength(4)
    expect(averageX).toBeGreaterThan(0)
    expect(furthestLeft).toBeGreaterThan(-24)
  })

  it('maps relation physics, scales with node size, and clamps collision radii', () => {
    const defaultRelatedDistance = getLinkDistance({
      relationType: 'RELATED_TO',
      source: { collisionRadius: 48 },
      target: { collisionRadius: 48 },
    })
    const largeRelatedDistance = getLinkDistance({
      relationType: 'RELATED_TO',
      source: { collisionRadius: 140 },
      target: { collisionRadius: 132 },
    })

    expect(getLinkDistance({ relationType: 'PREREQUISITE_OF' })).toBeGreaterThan(getLinkDistance({ relationType: 'SUBTOPIC_OF' }))
    expect(largeRelatedDistance).toBeGreaterThan(defaultRelatedDistance)
    expect(getLinkStrength({ relationType: 'PREREQUISITE_OF' })).toBeGreaterThan(getLinkStrength({ relationType: 'RELATED_TO' }))
    expect(getRelationPriority('PREREQUISITE_OF')).toBeLessThan(getRelationPriority('USES'))
    expect(getNodeCharge({ collisionRadius: 140 })).toBeLessThan(getNodeCharge({ collisionRadius: 48 }))
    expect(getNodeCollisionRadius({ collisionRadius: 8 })).toBe(24)
    expect(getNodeCollisionRadius({ collisionRadius: 220 })).toBe(164)
  })
})
