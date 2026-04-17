import { __graphInteractionTestUtils } from '../useGraphInteraction'

describe('useGraphInteraction drag tolerance', () => {
  it('only treats real dragging as movement that should reheat the simulation', () => {
    expect(__graphInteractionTestUtils.hasExceededDragTolerance({ x: 10, y: 10 }, { x: 20, y: 18 })).toBe(false)
    expect(__graphInteractionTestUtils.hasExceededDragTolerance({ x: 10, y: 10 }, { x: 42, y: 10 })).toBe(true)
  })
})
