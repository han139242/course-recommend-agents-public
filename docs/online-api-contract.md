# 在线接口契约（第一批）

本文件对应第一阶段“静态页面 -> 真实在线数据”的最小契约，服务端先由 Node 提供，接口形态向未来 Go BFF 靠拢。

## 统一响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

## 页面到接口映射

| 页面 | 原静态来源 | 新接口 |
| --- | --- | --- |
| 首页 | `frontend/src/data/learning.js` | `GET /api/dashboard/summary` |
| 知识图谱 | `graphNodes` / `graphEdges` | `GET /api/graph/subgraph` |
| 知识点详情 | `knowledgeDetails` | `GET /api/knowledge/:id` |
| 成长分析 | `growthRadarAxes` / `heatmapRows` | `GET /api/competency/profile` |
| 推荐路径 | `recommendedPath` | `GET /api/recommend/path` |
| 课后扩展 | 无 | `GET /api/frontier/briefing` |

## 接口摘要

### `GET /api/dashboard/summary`

返回首页所需的统计、知识卡片、推荐路径和成长摘要。

### `GET /api/graph/subgraph?root={nodeId}`

返回一个以指定知识点为中心的一跳子图：

- `root`
- `nodes`
- `edges`

### `GET /api/knowledge/:id`

返回知识点详情：

- 基础信息
- 先修知识点
- 关联知识点
- 应用场景

### `GET /api/competency/profile`

返回成长分析页所需画像数据：

- `profile`
- `radarAxes`
- `growthInsights`
- `growthSummaryCards`
- `heatmapDomains`
- `heatmapRows`

### `GET /api/recommend/path`

返回推荐学习路径和当前推荐策略。

### `GET /api/frontier/briefing?knowledge={id}`

为“课后扩展/前沿知识”预留的统一入口。第一阶段可返回占位结构，第二阶段接入论文/前沿知识链路。
