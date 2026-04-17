import { quadtree } from 'd3-quadtree'

function getNodeRadius(node) {
  return Number(node?.view?.style?.collisionRadius || node?.collisionRadius || 36)
}

export function createNodeQuadtree(nodes = []) {
  return quadtree()
    .x((node) => node.x)
    .y((node) => node.y)
    .addAll(nodes.filter((node) => Number.isFinite(node?.x) && Number.isFinite(node?.y)))
}

export function findNodeAtPoint(nodes = [], x = 0, y = 0, searchRadius = 52) {
  const tree = createNodeQuadtree(nodes)
  const found = tree.find(x, y, searchRadius)
  if (!found) return null

  const distance = Math.hypot(x - found.x, y - found.y)
  return distance <= getNodeRadius(found) ? found : null
}
