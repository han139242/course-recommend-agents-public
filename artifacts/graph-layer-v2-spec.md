# 图谱渲染层 V2 技术方案

> 决策锁定日期：2026-04-17
> 状态：待实施

---

## 1. 决策总览

| 决策项 | 选定方案 | 否决方案 |
|--------|----------|----------|
| 渲染引擎 | **d3-force + Canvas 2D** | vis-network (当前)、WebGL |
| 数据加载模式 | **真按需扩展**（双击拉邻居） | 全图预加载 + 局部显隐 (当前)、混合扩展 |
| 第一版渲染目标 | **Canvas 2D** | SVG、WebGL、Canvas 后升 WebGL |

选择优先级：**只看性能和优化上限，不考虑工作量。**

---

## 2. 为什么弃用 vis-network

### 2.1 当前瓶颈

| 问题 | 根因 |
|------|------|
| 力学参数调不细 | vis-network 物理引擎是封装好的黑箱，只有 `barnesHut` / `forceAtlas2Based` 几个旋钮；alpha 衰减、velocity decay、各力独立配比都不可控 |
| 同对节点多条边无法优雅分弧 | `smooth.type: curvedCW` 只有全局 `roundness`，无法按同一对节点的边数量自动分配弧度 |
| 渲染循环不可控 | 无法做脏区域重绘、离屏节点剔除、zoom 级别裁剪标签等精细优化 |
| 包体积 | vis-network gzip ~150 KB；d3-force 全套 ~30 KB，只用力学模块 ~8 KB |
| 拖拽/碰撞需要大量 workaround | 当前 `useGraph.js`（1622 行）中超过一半代码都在绕 vis-network 的限制：手写碰撞解算、手写簇跟随、手写 wheel 缩放覆盖 |

### 2.2 d3-force 能解决的

- **力的独立组合**：`forceLink`、`forceManyBody`、`forceCollide`、`forceCenter`、`forceRadial` 每个力独立可加减，参数全暴露
- **局部 reheat**：新节点加入时只 `simulation.alphaTarget(0.3).restart()`，不抖动全图
- **渲染层分离**：d3-force 只输出坐标 `(x, y)`，不管渲染；Canvas 2D 渲染循环完全由我们控制
- **未来升级路径**：节点量级突破 5000 时可平滑切 WebGL（pixi.js / regl），力学层代码零改动

---

## 3. 新架构设计

### 3.1 技术栈

```
d3-force          → 力学布局（只算坐标）       ~8 KB gzip
d3-zoom           → 缩放平移                   ~4 KB gzip
d3-drag           → 拖拽                       ~3 KB gzip
d3-quadtree       → hit detection（Canvas 点击判断） ~2 KB gzip
d3-dispatch       → 事件分发                   ~1 KB gzip
───────────────────────────────────────────────────────
总计                                           ~18 KB gzip
```

不引入 d3-selection，直接操作 Canvas API + Vue 响应式。

### 3.2 模块划分

```
frontend/src/
├── composables/
│   ├── useGraphEngine.js       ← 新文件：d3-force simulation 管理
│   ├── useGraphRenderer.js     ← 新文件：Canvas 2D 渲染循环
│   ├── useGraphInteraction.js  ← 新文件：拖拽/缩放/点击/hover/右键
│   └── useGraph.js             ← 改造：薄 orchestrator，组合上面三个
├── utils/
│   ├── graphTransform.js       ← 改造：去掉 vis-network 视觉计算，保留数据变换
│   ├── graphLayout.js          ← 新文件：d3-force 力的配置工厂
│   ├── graphDraw.js            ← 新文件：Canvas 绘制函数（节点/边/标签/箭头）
│   └── graphHitTest.js         ← 新文件：quadtree + 距离判断
├── stores/
│   └── graphStore.js           ← 改造：去掉全图预加载，改为增量图管理
└── api/
    └── graph.js                ← 改造：新增 fetchNeighbors 接口
```

### 3.3 数据流

```
用户打开 /graph
  │
  ▼
graphStore.loadInitialNodes()
  → GET /api/graph/seeds          ← 新接口：只返回种子节点（~14 个）
  → graphStore.nodes / edges 初始化
  │
  ▼
useGraphEngine 创建 d3-force simulation
  → simulation.nodes([...])
  → simulation.force('link', ...).links([...])
  → simulation.on('tick', () => renderer.draw())
  │
  ▼
useGraphRenderer 在 Canvas 上绘制
  → requestAnimationFrame loop
  → 根据 zoom level 决定标签密度
  │
  ▼
用户双击节点 A
  │
  ▼
graphStore.expandNode('A')
  → GET /api/graph/neighbors?nodeId=A&depth=1    ← 新接口
  → 增量追加到 graphStore.nodes / edges
  → 去重、引用计数更新
  │
  ▼
useGraphEngine 增量更新
  → simulation.nodes([...更新后])
  → simulation.force('link').links([...更新后])
  → simulation.alphaTarget(0.3).restart()         ← 局部 reheat
  → 新节点初始位置 = 父节点坐标 ± 随机偏移
  │
  ▼
用户双击已展开的节点 A（收起）
  │
  ▼
graphStore.collapseNode('A')
  → 引用计数清零的子节点移除
  → simulation 更新
```

---

## 4. 后端改造

### 4.1 新接口：邻居查询

**路由**：`GET /api/graph/neighbors`

**请求参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `nodeId` | string | 是 | - | 目标节点 ID（knowledge_id / node_id / id / name） |
| `depth` | int | 否 | 1 | 扩展深度，1 = 直接邻居，最大 2 |
| `direction` | string | 否 | `both` | `outgoing` / `incoming` / `both` |
| `limit` | int | 否 | 20 | 最大返回邻居数，上限 50 |

**响应**（复用现有 `SubgraphResponse` 格式）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "nodes": [ /* GraphNodeView[] - 包含目标节点 + 邻居节点 */ ],
    "edges": [ /* GraphEdgeView[] - 目标节点到邻居的边 */ ]
  }
}
```

**Cypher**：

```cypher
MATCH (n:KnowledgeNode)
WHERE coalesce(n.knowledge_id, n.node_id, n.id, n.name) = $nodeId
MATCH (n)-[r]-(m:KnowledgeNode)
WITH n, r, m
ORDER BY
  CASE type(r)
    WHEN 'PREREQUISITE_OF' THEN 0
    WHEN 'SUBTOPIC_OF' THEN 1
    WHEN 'PART_OF' THEN 2
    WHEN 'RELATED_TO' THEN 3
    ELSE 4
  END,
  coalesce(m.name, m.canonicalName)
LIMIT $limit
RETURN n, collect(DISTINCT m) AS neighbors,
       collect(DISTINCT {
         relation_id: toString(id(r)),
         relation_type: type(r),
         source_id: coalesce(startNode(r).knowledge_id, startNode(r).node_id, startNode(r).id, startNode(r).name),
         target_id: coalesce(endNode(r).knowledge_id, endNode(r).node_id, endNode(r).id, endNode(r).name),
         source_name: coalesce(startNode(r).name, startNode(r).canonicalName, ''),
         target_name: coalesce(endNode(r).name, endNode(r).canonicalName, '')
       }) AS edges
```

### 4.2 新接口：种子节点

**路由**：`GET /api/graph/seeds`

**请求参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `count` | int | 否 | 14 | 返回种子节点数量，上限 30 |

**说明**：当前种子节点选择逻辑在前端 `buildSeedNodeIds()` 中完成。迁移到后端的好处：

- 前端不再需要全量节点来计算 degree / crossCourse 等指标
- 种子选择可以利用 Neo4j 的图算法插件（pageRank / betweenness centrality）做更准确的重要度排序
- 首屏加载量从 ~1000 节点降到 ~14 节点 + 它们的边

**Cypher**（简化版，用 degree 排序）：

```cypher
MATCH (n:KnowledgeNode)
WITH n, size((n)--()) AS degree
ORDER BY degree DESC
LIMIT $count
WITH collect(n) AS seeds
UNWIND seeds AS n
OPTIONAL MATCH (n)-[r]-(m:KnowledgeNode)
WHERE m IN seeds
RETURN seeds AS nodes,
       collect(DISTINCT {
         relation_id: toString(id(r)),
         relation_type: type(r),
         source_id: coalesce(startNode(r).knowledge_id, startNode(r).node_id, startNode(r).id, startNode(r).name),
         target_id: coalesce(endNode(r).knowledge_id, endNode(r).node_id, endNode(r).id, endNode(r).name),
         source_name: coalesce(startNode(r).name, startNode(r).canonicalName, ''),
         target_name: coalesce(endNode(r).name, endNode(r).canonicalName, '')
       }) AS edges
```

### 4.3 后端文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `dto/dto.go` | 修改 | 新增 `NeighborQuery`、`SeedQuery` DTO |
| `entity/entity.go` | 不变 | 复用现有 `Node`、`Edge` |
| `repository/neo4j_repository.go` | 修改 | 新增 `LoadNeighbors()`、`LoadSeeds()` 方法 |
| `service/service.go` | 修改 | 新增 `GetNeighbors()`、`GetSeeds()` 方法 |
| `handler/handler.go` | 修改 | 新增 `GetNeighbors()`、`GetSeeds()` handler |
| `module.go` | 修改 | 注册新路由 |

现有的 `/graph/full` 和 `/graph/subgraph` 保留不删，向后兼容。

---

## 5. 前端核心实现

### 5.1 增量图状态管理（graphStore.js 改造）

```js
// 核心数据结构
const nodes = ref(new Map())           // id → { id, label, x, y, expanded, depth, ... }
const edges = ref(new Map())           // id → { source, target, label, relationType, ... }
const expandedBy = ref(new Map())      // childId → Set<parentId>  （引用计数）
const expandedNodeIds = ref(new Set()) // 已展开的节点 ID 集合

// 展开节点
async function expandNode(nodeId) {
  if (expandedNodeIds.value.has(nodeId)) return

  const { nodes: newNodes, edges: newEdges } = await fetchNeighbors(nodeId)

  for (const n of newNodes) {
    if (nodes.value.has(n.id)) {
      // 已存在 — 只增加引用
      expandedBy.value.get(n.id)?.add(nodeId)
    } else {
      // 新节点 — 初始位置在父节点附近
      const parent = nodes.value.get(nodeId)
      n.x = (parent?.x ?? 0) + (Math.random() - 0.5) * 80
      n.y = (parent?.y ?? 0) + (Math.random() - 0.5) * 80
      nodes.value.set(n.id, n)
      expandedBy.value.set(n.id, new Set([nodeId]))
    }
  }

  for (const e of newEdges) {
    if (!edges.value.has(e.id)) {
      edges.value.set(e.id, e)
    }
  }

  expandedNodeIds.value.add(nodeId)
  // 通知 engine 层增量更新 simulation
}

// 收起节点
function collapseNode(nodeId) {
  if (!expandedNodeIds.value.has(nodeId)) return

  const toRemove = new Set()

  for (const [childId, refs] of expandedBy.value) {
    refs.delete(nodeId)
    if (refs.size === 0) {
      toRemove.add(childId)
      // 递归：如果这个子节点也曾展开，也要收起
      if (expandedNodeIds.value.has(childId)) {
        collapseNode(childId)  // 递归
      }
    }
  }

  for (const childId of toRemove) {
    nodes.value.delete(childId)
    expandedBy.value.delete(childId)
  }

  // 清理失效的边（source 或 target 不在 nodes 中）
  for (const [edgeId, edge] of edges.value) {
    if (!nodes.value.has(edge.source) || !nodes.value.has(edge.target)) {
      edges.value.delete(edgeId)
    }
  }

  expandedNodeIds.value.delete(nodeId)
  // 通知 engine 层更新 simulation
}
```

### 5.2 力学引擎（useGraphEngine.js）

```js
import {
  forceSimulation,
  forceLink,
  forceManyBody,
  forceCollide,
  forceCenter,
  forceX,
  forceY
} from 'd3-force'

export function useGraphEngine(graphStore) {
  let simulation = null

  function createSimulation(nodes, links) {
    simulation = forceSimulation(nodes)
      .force('link', forceLink(links)
        .id(d => d.id)
        .distance(d => getLinkDistance(d))    // 按关系类型动态距离
        .strength(d => getLinkStrength(d)))
      .force('charge', forceManyBody()
        .strength(d => getChargeStrength(d)) // 按节点类型动态斥力
        .distanceMax(400))
      .force('collide', forceCollide()
        .radius(d => getCollisionRadius(d))  // 按节点大小 + label 宽度
        .strength(0.8)
        .iterations(3))
      .force('center', forceCenter(0, 0).strength(0.02))
      .velocityDecay(0.4)
      .alphaDecay(0.02)
      .alphaMin(0.001)

    simulation.on('tick', onTick)
    return simulation
  }

  // 增量更新（展开/收起时调用）
  function updateSimulation(nodes, links) {
    simulation.nodes(nodes)
    simulation.force('link').links(links)
    simulation.alphaTarget(0.3).restart()

    // 稳定后归零
    setTimeout(() => simulation.alphaTarget(0), 2000)
  }

  // 力的参数工厂
  function getLinkDistance(link) {
    const TYPE_DISTANCE = {
      PREREQUISITE_OF: 140,
      SUBTOPIC_OF: 110,
      PART_OF: 110,
      RELATED_TO: 180,
      USES: 160,
      APPLIES_TO: 160,
    }
    return TYPE_DISTANCE[link.relationType] ?? 160
  }

  function getLinkStrength(link) {
    const TYPE_STRENGTH = {
      PREREQUISITE_OF: 0.7,
      SUBTOPIC_OF: 0.8,
      PART_OF: 0.8,
      RELATED_TO: 0.3,
      USES: 0.4,
      APPLIES_TO: 0.4,
    }
    return TYPE_STRENGTH[link.relationType] ?? 0.4
  }

  function getChargeStrength(node) {
    return node.isRoot ? -300 : -150
  }

  function getCollisionRadius(node) {
    return node.visualRadius + 8  // visualRadius = 半径 + label 溢出
  }

  return { createSimulation, updateSimulation, simulation }
}
```

### 5.3 Canvas 渲染器（useGraphRenderer.js）

```js
export function useGraphRenderer(canvasRef, graphStore, engine) {
  let ctx = null
  let transform = d3.zoomIdentity  // { x, y, k } 平移缩放状态
  let frameId = null

  function init() {
    const canvas = canvasRef.value
    ctx = canvas.getContext('2d')
    adjustDPI(canvas, ctx)
    startRenderLoop()
  }

  function startRenderLoop() {
    function frame() {
      draw()
      frameId = requestAnimationFrame(frame)
    }
    frameId = requestAnimationFrame(frame)
  }

  function draw() {
    const { width, height } = ctx.canvas
    ctx.save()
    ctx.clearRect(0, 0, width, height)
    ctx.translate(transform.x, transform.y)
    ctx.scale(transform.k, transform.k)

    // 1. 绘制边（底层）
    drawEdges(ctx, graphStore.edges, graphStore.nodes, transform.k)

    // 2. 绘制节点（中层）
    drawNodes(ctx, graphStore.nodes, transform.k)

    // 3. 绘制标签（顶层，zoom < 0.6 时隐藏非根节点标签）
    drawLabels(ctx, graphStore.nodes, transform.k)

    ctx.restore()
  }

  return { init, draw, setTransform: (t) => { transform = t } }
}
```

### 5.4 Canvas 绘制细节（graphDraw.js）

#### 5.4.1 节点绘制

```js
function drawNode(ctx, node, zoomLevel) {
  const { x, y, visualRadius, fill, stroke, borderWidth, shadowColor, shadowBlur } = node

  // 阴影
  ctx.shadowColor = shadowColor
  ctx.shadowBlur = shadowBlur * Math.min(zoomLevel, 1.2)
  ctx.shadowOffsetX = 0
  ctx.shadowOffsetY = 0

  // 填充
  ctx.beginPath()
  ctx.arc(x, y, visualRadius, 0, Math.PI * 2)
  ctx.fillStyle = fill
  ctx.fill()

  // 边框
  ctx.shadowBlur = 0
  ctx.strokeStyle = stroke
  ctx.lineWidth = borderWidth
  ctx.stroke()
}
```

#### 5.4.2 边绘制 — 成对弧线分配

这是对标 Neo4j Browser 最关键的视觉改进。同一对节点之间有多条边时，自动计算弧度偏移：

```js
function drawEdges(ctx, edgeMap, nodeMap, zoomLevel) {
  // 1. 按 (source, target) 分组（无向：小 ID 在前）
  const pairMap = new Map()
  for (const edge of edgeMap.values()) {
    const pairKey = [edge.source, edge.target].sort().join('::')
    if (!pairMap.has(pairKey)) pairMap.set(pairKey, [])
    pairMap.get(pairKey).push(edge)
  }

  // 2. 对每组边分配弧度
  for (const [, group] of pairMap) {
    const count = group.length
    group.forEach((edge, i) => {
      const curvature = count === 1
        ? 0
        : (i - (count - 1) / 2) * 0.2  // 均匀分布在 -0.2*(n-1)/2 到 +0.2*(n-1)/2

      drawSingleEdge(ctx, edge, nodeMap, curvature, zoomLevel)
    })
  }
}

function drawSingleEdge(ctx, edge, nodeMap, curvature, zoomLevel) {
  const source = nodeMap.get(edge.source)
  const target = nodeMap.get(edge.target)
  if (!source || !target) return

  const dx = target.x - source.x
  const dy = target.y - source.y
  const distance = Math.hypot(dx, dy) || 1

  if (curvature === 0) {
    // 直线
    ctx.beginPath()
    ctx.moveTo(source.x, source.y)
    ctx.lineTo(target.x, target.y)
  } else {
    // 二次贝塞尔弧线
    // 控制点 = 中点 + 法线方向偏移
    const mx = (source.x + target.x) / 2
    const my = (source.y + target.y) / 2
    const nx = -dy / distance  // 法线方向
    const ny = dx / distance
    const offset = distance * curvature
    const cpx = mx + nx * offset
    const cpy = my + ny * offset

    ctx.beginPath()
    ctx.moveTo(source.x, source.y)
    ctx.quadraticCurveTo(cpx, cpy, target.x, target.y)
  }

  ctx.strokeStyle = edge.style.stroke
  ctx.lineWidth = edge.style.lineWidth
  if (edge.style.lineDash?.length) ctx.setLineDash(edge.style.lineDash)
  else ctx.setLineDash([])
  ctx.stroke()

  // 箭头
  if (edge.style.endArrow) {
    drawArrowhead(ctx, source, target, curvature, edge.style)
  }

  // 边标签（zoom > 0.8 时显示）
  if (zoomLevel > 0.8 && edge.data?.label) {
    drawEdgeLabel(ctx, edge, source, target, curvature)
  }
}
```

#### 5.4.3 标签密度控制

```js
function drawLabels(ctx, nodeMap, zoomLevel) {
  for (const node of nodeMap.values()) {
    // zoom < 0.6 且非根节点 → 隐藏标签
    if (zoomLevel < 0.6 && !node.isRoot) continue
    // zoom < 0.4 → 只显示根节点标签
    if (zoomLevel < 0.4 && !node.isRoot) continue

    const fontSize = node.isRoot
      ? Math.max(10, 14 / zoomLevel * 0.8)  // 根节点字号随缩放补偿
      : Math.max(8, 11 / zoomLevel * 0.7)

    ctx.font = `${Math.round(fontSize)}px "IBM Plex Sans SC", "HarmonyOS Sans SC", sans-serif`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillStyle = '#F8FAFC'
    ctx.strokeStyle = 'rgba(15, 23, 42, 0.88)'
    ctx.lineWidth = 4
    ctx.strokeText(node.label, node.x, node.y)
    ctx.fillText(node.label, node.x, node.y)
  }
}
```

### 5.5 交互层（useGraphInteraction.js）

```js
import { zoom as d3Zoom } from 'd3-zoom'
import { drag as d3Drag } from 'd3-drag'
import { select } from 'd3-selection'
import { quadtree } from 'd3-quadtree'

export function useGraphInteraction(canvasRef, engine, renderer, graphStore) {

  // —— Hit Detection ——
  function findNodeAtPoint(canvasX, canvasY) {
    const tree = quadtree()
      .x(d => d.x)
      .y(d => d.y)
      .addAll([...graphStore.nodes.values()])

    const found = tree.find(canvasX, canvasY, 40)  // 40px 搜索半径
    if (!found) return null

    const distance = Math.hypot(canvasX - found.x, canvasY - found.y)
    return distance <= found.visualRadius + 4 ? found : null
  }

  // —— 缩放 ——
  const zoomBehavior = d3Zoom()
    .scaleExtent([0.48, 2.0])
    .on('zoom', (event) => {
      renderer.setTransform(event.transform)
    })

  // —— 拖拽 ——
  const dragBehavior = d3Drag()
    .subject((event) => {
      const [cx, cy] = renderer.screenToCanvas(event.x, event.y)
      return findNodeAtPoint(cx, cy)
    })
    .on('start', (event) => {
      if (!event.subject) return
      // 固定被拖节点
      event.subject.fx = event.subject.x
      event.subject.fy = event.subject.y
      engine.simulation.alphaTarget(0.3).restart()
    })
    .on('drag', (event) => {
      if (!event.subject) return
      event.subject.fx = event.x
      event.subject.fy = event.y
    })
    .on('end', (event) => {
      if (!event.subject) return
      engine.simulation.alphaTarget(0)
      // 保持固定（用户手动拖放的节点不再被力学推动）
      // 如果希望松手后恢复弹性，可以设 fx = fy = null
    })

  // —— 点击/双击判定 ——
  let clickTimer = null
  const DOUBLE_CLICK_DELAY = 250  // ms

  function handleCanvasClick(event) {
    const [cx, cy] = renderer.screenToCanvas(event.clientX, event.clientY)
    const node = findNodeAtPoint(cx, cy)

    if (!node) {
      graphStore.clearSelection()
      return
    }

    if (clickTimer) {
      // 双击
      clearTimeout(clickTimer)
      clickTimer = null
      graphStore.toggleNodeExpansion(node.id)
    } else {
      // 单击 — 延迟判定
      clickTimer = setTimeout(() => {
        clickTimer = null
        graphStore.selectNode(node.id)
      }, DOUBLE_CLICK_DELAY)
    }
  }

  // —— Hover ——
  function handleCanvasMouseMove(event) {
    const [cx, cy] = renderer.screenToCanvas(event.clientX, event.clientY)
    const node = findNodeAtPoint(cx, cy)
    canvasRef.value.style.cursor = node ? 'pointer' : 'default'
    // 触发 tooltip 显示/隐藏
  }

  // —— 右键 ——
  function handleCanvasContextMenu(event) {
    event.preventDefault()
    const [cx, cy] = renderer.screenToCanvas(event.clientX, event.clientY)
    const node = findNodeAtPoint(cx, cy)
    if (node) {
      // 打开 context menu
    }
  }

  return { zoomBehavior, dragBehavior, handleCanvasClick, handleCanvasMouseMove, handleCanvasContextMenu }
}
```

### 5.6 深度限制与展开控制

| 参数 | 值 | 说明 |
|------|----|------|
| `MAX_EXPAND_DEPTH` | 4 | 从种子节点起最多 4 层深度 |
| `MAX_VISIBLE_NODES` | 200 | 超过 200 节点时提示用户收起部分子图 |
| `NEIGHBOR_LIMIT` | 20 | 单次展开最多拉 20 个邻居 |
| `EXPANSION_PRIORITY` | PREREQUISITE_OF > SUBTOPIC_OF > PART_OF > RELATED_TO > USES > APPLIES_TO | 邻居排序：高优先关系的邻居排前面 |

---

## 6. 视觉规格

### 6.1 节点

| 属性 | 种子节点 | 普通节点 | 选中节点 | 已展开节点 |
|------|----------|----------|----------|------------|
| 半径 | 38-46 px | 26-34 px | +4 px border | +3 px border |
| 边框色 | `rgba(196, 181, 253, 1)` | 同左 | `rgba(103, 232, 249, 1)` | `rgba(103, 232, 249, 1)` |
| 阴影 | 14 blur, 0.24 opacity | 8 blur, 0.16 opacity | 24 blur, 0.42 | 20 blur, 0.34 |
| 字号 | 14 px | 11 px | 不变 | 不变 |

学习状态颜色保持不变：

| 状态 | 填充色 | 边框色 |
|------|--------|--------|
| 默认 | `rgba(241, 234, 255, 0.99)` | `rgba(196, 181, 253, 1)` |
| 学习中 | `rgba(230, 236, 255, 0.99)` | `rgba(129, 140, 248, 0.98)` |
| 薄弱 | `rgba(255, 228, 196, 0.99)` | `rgba(251, 146, 60, 0.98)` |
| 已掌握 | `rgba(255, 246, 214, 0.99)` | `rgba(250, 204, 21, 0.98)` |

### 6.2 边

| 关系类型 | 颜色 | 线宽 | 线型 | 箭头 |
|----------|------|------|------|------|
| PREREQUISITE_OF | `#CBD5E1` | 2.1 | 实线 | 有 |
| SUBTOPIC_OF | `#C7D2FE` | 1.7 | `[6,3]` 虚线 | 有 |
| PART_OF | `#BFDBFE` | 1.7 | `[8,4]` 虚线 | 有 |
| RELATED_TO | `#E2E8F0` | 1.5 | 实线 | 有 |
| USES | `#93C5FD` | 1.45 | `[4,4]` 虚线 | 有 |
| APPLIES_TO | `#6EE7B7` | 1.45 | `[6,4]` 虚线 | 有 |

**成对弧线**：同一对节点间多条边自动分弧，弧度系数 `0.2 * (i - (n-1)/2)`。

### 6.3 标签密度策略

| Zoom 范围 | 显示策略 |
|-----------|----------|
| `>= 0.8` | 所有节点标签 + 边标签 |
| `0.6 ~ 0.8` | 所有节点标签，隐藏边标签 |
| `0.4 ~ 0.6` | 仅种子/已展开/选中节点标签 |
| `< 0.4` | 仅种子节点标签 |

---

## 7. 前端依赖变更

### 移除

```json
{
  "vis-network": "^10.0.2",
  "vis-data": "^8.0.3"
}
```

### 新增

```json
{
  "d3-force": "^3.0.0",
  "d3-zoom": "^3.0.0",
  "d3-drag": "^3.0.0",
  "d3-quadtree": "^3.0.0",
  "d3-selection": "^3.0.0",
  "d3-dispatch": "^3.0.0"
}
```

> d3-selection 和 d3-dispatch 是 d3-zoom / d3-drag 的 peer dependency，必须安装，但不直接在业务代码中 import。

---

## 8. 文件变更全清单

### 后端

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/internal/module/graph/dto/dto.go` | 修改 | 新增 `NeighborQuery`、`SeedQuery` |
| `backend/internal/module/graph/repository/neo4j_repository.go` | 修改 | 新增 `LoadNeighbors()`、`LoadSeeds()` |
| `backend/internal/module/graph/service/service.go` | 修改 | 新增 `GetNeighbors()`、`GetSeeds()` |
| `backend/internal/module/graph/handler/handler.go` | 修改 | 新增 `GetNeighbors()`、`GetSeeds()` handler |
| `backend/internal/module/graph/module.go` | 修改 | 注册 `/graph/neighbors`、`/graph/seeds` 路由 |

### 前端

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/package.json` | 修改 | 移除 vis-*，新增 d3-* |
| `frontend/src/api/graph.js` | 修改 | 新增 `fetchNeighbors()`、`fetchSeeds()` |
| `frontend/src/stores/graphStore.js` | **重写** | 去掉全图预加载，改为 Map 增量管理 + 引用计数 |
| `frontend/src/composables/useGraph.js` | **重写** | 薄 orchestrator，组合 engine / renderer / interaction |
| `frontend/src/composables/useGraphEngine.js` | **新建** | d3-force simulation 管理 |
| `frontend/src/composables/useGraphRenderer.js` | **新建** | Canvas 2D 渲染循环 |
| `frontend/src/composables/useGraphInteraction.js` | **新建** | 拖拽/缩放/点击/hover/右键 |
| `frontend/src/utils/graphTransform.js` | **重写** | 去掉 vis-network 视觉计算，保留 RELATION_CONFIG 和数据变换 |
| `frontend/src/utils/graphLayout.js` | **新建** | d3-force 力的配置工厂函数 |
| `frontend/src/utils/graphDraw.js` | **新建** | Canvas 绘制函数 |
| `frontend/src/utils/graphHitTest.js` | **新建** | quadtree hit detection |
| `frontend/src/components/graph/GraphCanvas.vue` | 修改 | template 改为单 `<canvas>` 元素 + 覆盖层 UI |
| `frontend/src/components/graph/GraphControls.vue` | 微调 | 对接新 zoom API |
| `frontend/src/composables/useNodeFocus.js` | 修改 | 对接新 engine API |

### 可删除

| 文件 | 说明 |
|------|------|
| `frontend/src/utils/graphConfig.js` | 遗留的 D3 旧配置，不再需要 |

---

## 9. 实施分期

### Phase 1：后端新接口（1 期）

1. `dto.go` 新增 DTO
2. `neo4j_repository.go` 新增 `LoadNeighbors()` + `LoadSeeds()`
3. `service.go` + `handler.go` + `module.go` 注册路由
4. 手动测试：`curl` 验证接口返回

### Phase 2：前端引擎层（2 期）

1. `npm remove vis-network vis-data && npm install d3-force d3-zoom d3-drag d3-quadtree d3-selection d3-dispatch`
2. 新建 `useGraphEngine.js` — d3-force simulation
3. 新建 `useGraphRenderer.js` — Canvas 2D 渲染
4. 新建 `graphDraw.js` — 节点/边/标签/箭头绘制
5. 新建 `graphHitTest.js` — quadtree
6. 基础可视化可以跑起来（静态节点 + 力学布局 + 缩放拖拽）

### Phase 3：增量图管理（3 期）

1. 重写 `graphStore.js` — Map + 引用计数
2. 对接 `fetchSeeds()` → 首屏只加载种子
3. 对接 `fetchNeighbors()` → 双击拉邻居
4. 展开/收起逻辑 + 深度限制 + 节点数量限制

### Phase 4：交互打磨（4 期）

1. 新建 `useGraphInteraction.js` — 完整交互层
2. 改造 `GraphCanvas.vue` — 单 canvas + 覆盖 UI
3. tooltip / context menu / 高亮路径 / flash 效果
4. 拖拽时簇跟随 / 松手后固定

### Phase 5：视觉对齐（5 期）

1. 成对弧线
2. 标签密度策略
3. 边标签沿弧线排版
4. 箭头沿弧线方向
5. 学习状态颜色 + 选中/展开/高亮效果

---

## 10. 风险与兜底

| 风险 | 影响 | 兜底 |
|------|------|------|
| Canvas 文字渲染质量不如 DOM | 标签在低 DPI 屏幕上可能模糊 | `adjustDPI()` 根据 `devicePixelRatio` 缩放 canvas |
| 节点数量超 200 后力学不稳定 | 展开过多层后抖动 | 前端硬性上限 `MAX_VISIBLE_NODES = 200`，超出提示收起 |
| 后端邻居查询慢（>200ms） | 双击展开有延迟 | 展开时先放 placeholder 节点 + 骨架动画，接口返回后替换 |
| d3-zoom 和 Vue 事件冲突 | zoom / drag 事件被 Vue 拦截 | d3 直接绑定在 canvas DOM 上，Vue 只管覆盖层 UI |
| 迁移过程中功能回退 | 已有交互暂时不可用 | 保留旧 `useGraph.js` 在 git 中，按 Phase 逐步替换，不一次性切 |

---

## 11. 性能指标

| 指标 | 目标值 | 测量方式 |
|------|--------|----------|
| 首屏渲染（种子节点） | < 200 ms | Performance.mark |
| 单次展开（20 个邻居） | < 100 ms（接口 + 渲染） | Network + requestAnimationFrame |
| 稳态帧率（200 节点） | >= 55 fps | requestAnimationFrame 计数 |
| 拖拽帧率 | >= 50 fps | 同上 |
| 包体积（graph 相关） | < 25 KB gzip | webpack-bundle-analyzer |

---

## 12. 不在本次范围内

- WebGL 渲染升级（未来节点 > 5000 时再考虑）
- 3D 图谱
- 服务端图布局计算
- Neo4j graph algorithms 插件集成（pageRank / betweenness centrality 用于种子选择）
- 协同编辑 / 多人实时图谱
