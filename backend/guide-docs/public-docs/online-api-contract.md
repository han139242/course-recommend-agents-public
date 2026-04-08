# 在线 API 契约

本文件以 `course-recommend-agents.wiki/设计文档.md` 为上位设计依据，定义当前系统对前端暴露的正式 HTTP API 契约。

目标不是复刻当前 Node.js 演示实现，而是为后续 Go 在线服务落地提供稳定边界，同时兼顾当前前端页面的接入现实，并补充教师端、班级管理、题库管理等新增需求。

## 适用架构边界

本契约只描述**前端展示层 -> Go API Server** 之间的外部 HTTP / WebSocket 边界，不描述 Go 内部模块调用细节，也不描述 Python 离线流水线内部任务编排。

系统当前分为三个可部署单元：

- 前端展示层：Vue 3 页面、路由、交互与可视化。
- 在线 API 服务：Go 单体服务，对外统一暴露 `/api` 接口。
- 离线数据流水线：Python 进程，负责爬虫、语料处理、LLM 抽取、知识图谱写入。

边界约束：

- 对外只有 Go API Server 提供 HTTP API。
- Python 离线流水线**不直接**对前端暴露接口。
- Neo4j、MySQL、Redis 都属于内部基础设施，前端不得直连。
- Go 在线服务具备 Neo4j 读写能力，但读写职责在服务内部仍应隔离：图谱查询走只读查询模块，用户能力图谱写回走专用异步写入链路。
- Python 离线流水线负责知识图谱实体与关系的批量构建与更新。
- 用户答题后产生的用户能力图谱写回由 Go 在线侧异步完成，不阻塞前端请求返回。
- Go 与 Python 之间当前通过 `Redis Pub/Sub` 的 `topology_updates` 频道做拓扑热更新通知。

## 契约设计原则

- 外部 API 按领域能力组织：认证、教师端管理、图谱查询、能力画像、推荐、聚合接口、扩展接口。
- 用户态接口默认从 JWT 推导当前用户身份，不向前端开放任意 `uid` 查询参数。
- 主键优先使用稳定 ID，例如 `knowledgeId`、`target`、`classId`、`studentId`。
- 文件导入解析工作由前端完成，后端只接收**批量结构化数据**，不负责解析 Excel / CSV 原始文件。
- 当前阶段**不开放学生自主注册**，账号创建由教师端或管理员端统一完成。
- 若前端保留 `/register` 页面，它仅作为“账号开通说明页”，不是有效业务入口。
- 当前阶段教师与管理员同责，统一使用教师端 API 即可。
- 行为埋点接口暂不实现，不写入当前最小可行契约。
- 在线请求只承诺可稳定返回的能力；离线生成的扩展内容属于最终一致数据。
- 聚合接口可以保留，但必须明确它们是 BFF 视图聚合，不是底层领域模型本身。
- 首页知识目录、项目推荐、图谱跳转 ID 等前端已消费但后端尚未完全在线化的能力，需要在主契约中显式声明，不能默认依赖前端静态兜底长期存在。
- 前端允许在接口失败时回退到静态展示内容，但这不代表后端字段可以缺失；静态兜底只用于演示降级，不用于放宽契约要求。

## 统一请求约定

### Base URL

前端当前按以下优先级解析 API 地址：

1. `VITE_API_BASE_URL`
2. `http://localhost:3000/api`
3. `http://localhost:3002/api`
4. `http://localhost:3001/api`

因此 Go API Server 应继续保持 `/api` 作为统一前缀。

### 认证方式

需要登录的接口通过请求头传递 Bearer Token：

```http
Authorization: Bearer <token>
```

### 角色约定

当前阶段仅保留两类系统身份：

- `teacher`：教师，具备教师端全部能力。
- `student`：学生，只能访问学生端能力。

说明：

- 当前阶段教师与管理员同责，因此不单独拆分 `admin` 角色。
- 未来若确需拆分权限，可在 JWT 中追加更细粒度角色或权限声明。

### 通用成功响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

### 通用失败响应

为了兼容当前前端实现，失败响应至少必须返回：

```json
{
  "message": "请求失败，请稍后重试。"
}
```

Go 在线服务建议统一为：

```json
{
  "code": 10001,
  "message": "请求失败，请稍后重试。",
  "data": null
}
```

### 分页约定

列表接口统一使用以下分页参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | `number` | 否 | `1` | 页码，从 1 开始 |
| `pageSize` | `number` | 否 | `20` | 每页数量，上限 100 |

分页响应统一包含：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [],
    "total": 120,
    "page": 1,
    "pageSize": 20
  }
}
```

## 一致性语义

### 在线实时返回

以下接口的主要结果应在当前请求内同步返回：

- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/teacher/teachers`
- `POST /api/teacher/classes`
- `POST /api/teacher/classes/:classId/students`
- `POST /api/teacher/questions/batch`
- `GET /api/graph/subgraph`
- `GET /api/knowledge/:knowledgeId`
- `POST /api/learning/answers`
- `GET /api/learning/questions`
- `GET /api/competency/profile`
- `GET /api/recommend/path`
- `GET /api/dashboard/summary`
- `GET /api/teacher/students/:studentId/knowledge-status`

### 异步最终一致副作用

以下效果不要求在当前请求中完成，但允许随后收敛：

- 学生答题后，Go 在线服务异步写回 Neo4j 的用户能力图谱。
- Python 离线侧完成知识图谱写入后，经 `Redis Pub/Sub` 通知 Go 侧热更新权重矩阵。
- 前沿扩展类内容的补齐与更新。

因此：

- `POST /api/learning/answers` 成功，表示在线状态更新已被接受，**不表示** Go 异步写入 Neo4j 的用户能力图谱已经完成。
- `GET /api/frontier/briefing` 即使返回成功，也可能表示“当前内容尚未就绪”。

## 数据模型

### AuthUser

```json
{
  "id": 1,
  "name": "张老师",
  "email": "teacher@example.com",
  "role": "teacher",
  "title": "副教授",
  "studentNo": null,
  "classId": null,
  "className": null,
  "target": "",
  "status": "active",
  "createdAt": "2026-04-03T00:00:00.000Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | `number` | 是 | 用户主键 |
| `name` | `string` | 是 | 用户姓名 |
| `email` | `string` | 是 | 登录邮箱 |
| `role` | `string` | 是 | `teacher` 或 `student` |
| `title` | `string \| null` | 否 | 教师职称，学生为 null |
| `studentNo` | `string \| null` | 否 | 学号，教师为 null |
| `classId` | `number \| null` | 否 | 学生所属班级 ID，教师可为空 |
| `className` | `string \| null` | 否 | 学生所属班级名称，教师可为空 |
| `target` | `string` | 是 | 学生目标方向文本，教师可为空字符串 |
| `status` | `string` | 是 | `active` / `disabled`，登录时须校验 |
| `createdAt` | `string` | 是 | ISO 8601 时间字符串 |

### Classroom

```json
{
  "id": 101,
  "name": "2026 春人工智能 1 班",
  "grade": "2026 春",
  "studentCount": 36,
  "status": "active",
  "createdAt": "2026-04-03T00:00:00.000Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | `number` | 是 | 班级主键 |
| `name` | `string` | 是 | 班级名称 |
| `grade` | `string` | 否 | 学期 / 年级描述 |
| `studentCount` | `number` | 否 | 班级学生数 |
| `status` | `string` | 是 | `active` / `archived`，当前阶段固定返回 `active` |
| `createdAt` | `string` | 是 | 创建时间 |

### TeacherCreateItem

```json
{
  "name": "张老师",
  "email": "teacher@example.com",
  "password": "123456",
  "title": "副教授"
}
```

### StudentBatchItem

```json
{
  "name": "李四",
  "studentNo": "20260001",
  "email": "student01@example.com",
  "password": "123456",
  "target": "算法工程师"
}
```

### QuestionOption

```json
{
  "key": "A",
  "text": "沿负梯度方向更新参数"
}
```

### ChoiceQuestion

```json
{
  "questionId": "q_001",
  "type": "single_choice",
  "knowledgeId": "kp_001",
  "stem": "以下关于梯度下降的说法，哪一项正确？",
  "options": [
    { "key": "A", "text": "沿负梯度方向更新参数" },
    { "key": "B", "text": "随机删除损失函数" }
  ],
  "correctOption": "A",
  "difficulty": 0.4,
  "explanation": "梯度下降通过负梯度方向逼近最优解。"
}
```

说明：

- 当前阶段只支持 `single_choice`。
- 每道题只绑定一个主知识点 `knowledgeId`，暂不支持多知识点权重映射。
- **映射说明**：API 层的 `questionId` 对应 DB 层的 `question_code`，是题目的业务唯一标识（如 `q_001`），不是数据库自增主键。

### StudentKnowledgeStatus

```json
{
  "student": {
    "id": 201,
    "name": "李四",
    "classId": 101,
    "className": "2026 春人工智能 1 班"
  },
  "knowledge": {
    "id": "kp_001",
    "canonicalName": "梯度下降"
  },
  "masteryScore": 52,
  "status": "progressing",
  "correctRate": 0.63,
  "answeredCount": 19,
  "lastAnsweredAt": "2026-04-03T10:00:00.000Z"
}
```

### StatItem

```json
{
  "label": "学习中知识点",
  "value": "6"
}
```

### GrowthInsightItem

```json
{
  "label": "数学基础",
  "value": 46,
  "note": "当前数学基础仍有明显提升空间，建议结合先修关系持续补强。"
}
```

### KnowledgeCard

```json
{
  "id": "node-1",
  "type": "补弱推荐",
  "progress": 49,
  "title": "梯度下降",
  "domain": "数学基础",
  "tag": "知识图谱中的关键能力节点",
  "route": "/knowledge/node-1"
}
```

### RecommendedPathItem

```json
{
  "stage": "补弱推荐",
  "title": "梯度下降",
  "detail": "建议先补偏导数，再进入梯度下降。",
  "route": "/knowledge/node-1",
  "reasonCode": "missing_prerequisite",
  "reasonLabel": "先修缺口",
  "score": 0.87
}
```

说明：

- 推荐结果不能只依赖 Neo4j 邻接点。
- 推荐服务应基于：图谱关系候选召回 + 用户掌握度状态过滤 + 目标方向重排 + 规则打分。
- `reasonCode`、`reasonLabel`、`score` 用于支持解释性展示，前端当前可按需消费。

### GraphNode

```json
{
  "id": "node-1",
  "routeKnowledgeId": "gradient-descent",
  "name": "梯度下降",
  "canonicalName": "梯度下降",
  "entityType": "concept",
  "description": "用于优化目标函数的经典迭代方法。",
  "chapterTitle": "优化基础",
  "sourcePath": "machine-learning/chapter-1"
}
```

字段补充说明：

- `id` 表示图谱内部节点 ID。
- `routeKnowledgeId` 表示前端路由层使用的稳定知识点 ID，推荐优先用于跳转 `/knowledge/:knowledgeId`。
- 前端当前兼容读取顺序建议为：`routeKnowledgeId` -> `knowledgeId` -> `slug` -> `id`。

### GraphEdge

```json
{
  "relationId": "rel-1",
  "relationType": "PREREQUISITE_OF",
  "sourceId": "node-a",
  "targetId": "node-b",
  "sourceName": "偏导数",
  "targetName": "梯度下降"
}
```

### KnowledgeDetail

```json
{
  "id": "node-1",
  "name": "梯度下降",
  "canonicalName": "梯度下降",
  "entityType": "concept",
  "description": "用于优化目标函数的经典迭代方法。",
  "chapterTitle": "优化基础",
  "sourcePath": "machine-learning/chapter-1",
  "summary": "梯度下降是当前知识图谱中的重点知识点。",
  "prerequisites": ["偏导数", "微积分"],
  "related": ["随机梯度下降"],
  "scenarios": ["模型训练"],
  "parentTopics": ["机器学习"],
  "masteryScore": 52,
  "status": "progressing",
  "masteryText": "建议先梳理 2 个关键先修知识点后再深入学习。",
  "recommendReason": "该知识点与偏导数等先修内容直接相关。"
}
```

### CompetencyProfile

```json
{
  "profile": {
    "name": "李四",
    "role": "student",
    "classId": 101,
    "className": "2026 春人工智能 1 班",
    "target": "算法工程师",
    "summary": "当前处于基础补强阶段。",
    "stats": [
      {
        "label": "学习中知识点",
        "value": "6"
      }
    ],
    "goals": ["补齐 梯度下降，提升相关先修理解"],
    "recentRecords": ["当前薄弱：梯度下降（掌握度 46%）"]
  },
  "radarAxes": [
    {
      "name": "数学基础",
      "value": 46
    }
  ],
  "growthInsights": [
    {
      "label": "数学基础",
      "value": 46,
      "note": "当前数学基础仍有明显提升空间，建议结合先修关系持续补强。"
    }
  ],
  "growthSummaryCards": [
    {
      "label": "已形成优势",
      "value": "深度学习"
    }
  ],
  "heatmapDomains": ["数学基础", "机器学习", "深度学习", "知识图谱", "场景应用"],
  "heatmapRows": [
    {
      "label": "梯度下降",
      "values": [46, 8, 8, 8, 8]
    }
  ]
}
```

### FrontierBriefing

```json
{
  "knowledgeId": "gradient-descent",
  "status": "ready",
  "knowledge": "梯度下降",
  "message": "已为该知识点准备第一版课后扩展条目。",
  "digestSummary": "梯度下降近期在大模型训练稳定性与自适应优化方向上仍是核心基础。",
  "topics": [
    {
      "topicId": "adaptive-optimization",
      "name": "自适应优化",
      "reason": "该主题直接扩展了梯度下降在训练策略上的应用边界。"
    }
  ],
  "papers": [
    {
      "paperId": "paper_001",
      "title": "A Survey on Adaptive Optimization",
      "venue": "NeurIPS",
      "year": 2025,
      "url": "https://example.com/paper"
    }
  ],
  "generatedAt": "2026-04-08T00:00:00Z"
}
```

## 接口定义

## 1. 认证

当前阶段仅允许教师端封闭创建账号，**不允许学生独立注册**。

### `POST /api/auth/login`

请求体：

```json
{
  "email": "teacher@example.com",
  "password": "123456"
}
```

成功响应：

```json
{
  "code": 0,
  "message": "登录成功。",
  "data": {
    "token": "jwt-token",
    "user": {
      "id": 1,
      "name": "张老师",
      "email": "teacher@example.com",
      "role": "teacher",
      "title": "副教授",
      "studentNo": null,
      "classId": null,
      "className": null,
      "target": "",
      "status": "active",
      "createdAt": "2026-04-03T00:00:00.000Z"
    }
  }
}
```

说明：

- 后端须校验用户 `status`，`disabled` 状态用户登录应返回错误。
- 登录成功后返回的 `user` 对象遵循 `AuthUser` 数据模型。

### `GET /api/auth/me`

请求头：

```http
Authorization: Bearer <token>
```

说明：

- 当前用户信息从 JWT 推导，不接受前端通过查询参数传 `uid`。

## 2. 教师端管理

该组接口统一要求教师身份。当前阶段教师与管理员同责，因此不再拆分单独管理员接口。

### `POST /api/teacher/teachers`

教师创建教师账号。

说明：

- 文件解析由前端完成，本接口只接受结构化 JSON。
- 当前阶段教师与管理员同责，因此允许教师创建教师。

请求体：

```json
{
  "items": [
    {
      "name": "张老师",
      "email": "teacher@example.com",
      "password": "123456",
      "title": "副教授"
    },
    {
      "name": "李老师",
      "email": "teacher2@example.com",
      "password": "123456",
      "title": "讲师"
    }
  ]
}
```

成功响应：

```json
{
  "code": 0,
  "message": "教师账号创建成功。",
  "data": {
    "createdCount": 2
  }
}
```

### `POST /api/teacher/classes`

教师创建班级。

请求体：

```json
{
  "name": "2026 春人工智能 1 班",
  "grade": "2026 春"
}
```

成功响应：

```json
{
  "code": 0,
  "message": "班级创建成功。",
  "data": {
    "classroom": {
      "id": 101,
      "name": "2026 春人工智能 1 班",
      "grade": "2026 春",
      "studentCount": 0,
      "status": "active",
      "createdAt": "2026-04-03T00:00:00.000Z"
    }
  }
}
```

### `GET /api/teacher/classes`

返回当前教师创建（或被授权管理）的班级列表。

说明：

- 班级作用域：仅返回 `teacher_id = 当前教师 ID` 的班级，不支持跨教师查看。
- 支持分页，使用统一分页参数。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 101,
        "name": "2026 春人工智能 1 班",
        "grade": "2026 春",
        "studentCount": 36,
        "status": "active",
        "createdAt": "2026-04-03T00:00:00.000Z"
      }
    ],
    "total": 3,
    "page": 1,
    "pageSize": 20
  }
}
```

### `POST /api/teacher/classes/:classId/students`

向某个班级批量导入学生账号。

说明：

- 前端负责解析 Excel / CSV，并转换为结构化数组。
- 后端只接收批量结构化数据并进行校验与落库。
- 学生账号由教师统一创建，学生不能自行注册。

请求体：

```json
{
  "items": [
    {
      "name": "李四",
      "studentNo": "20260001",
      "email": "student01@example.com",
      "password": "123456",
      "target": "算法工程师"
    },
    {
      "name": "王五",
      "studentNo": "20260002",
      "email": "student02@example.com",
      "password": "123456",
      "target": "知识图谱工程师"
    }
  ]
}
```

成功响应：

```json
{
  "code": 0,
  "message": "学生导入成功。",
  "data": {
    "classId": 101,
    "createdCount": 2
  }
}
```

### `GET /api/teacher/classes/:classId/students`

查看班级学生列表。

说明：

- 支持分页，使用统一分页参数。
- 仅允许查看自己管理的班级下的学生。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 201,
        "name": "李四",
        "studentNo": "20260001",
        "email": "student01@example.com",
        "target": "算法工程师",
        "createdAt": "2026-04-03T00:00:00.000Z"
      }
    ],
    "total": 36,
    "page": 1,
    "pageSize": 20
  }
}
```

### `GET /api/teacher/students/:studentId`

查看单个学生基础档案。

说明：

- 仅允许查看自己管理的班级下的学生。后端须校验该 `studentId` 对应的 `class_id` 是否归属当前教师。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 201,
    "name": "李四",
    "email": "student01@example.com",
    "role": "student",
    "title": null,
    "studentNo": "20260001",
    "classId": 101,
    "className": "2026 春人工智能 1 班",
    "target": "算法工程师",
    "status": "active",
    "createdAt": "2026-04-03T00:00:00.000Z"
  }
}
```

### `GET /api/teacher/students/:studentId/knowledge-status`

查看学生知识点掌握情况。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeId` | `string` | 否 | 传入则返回单个知识点详情，不传则返回列表 |
| `page` | `number` | 否 | 页码，从 1 开始（列表模式，`knowledgeId` 不传时生效） |
| `pageSize` | `number` | 否 | 每页数量，默认 20，上限 100（列表模式） |

成功响应示例（列表模式，不传 `knowledgeId`）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "student": {
      "id": 201,
      "name": "李四",
      "classId": 101,
      "className": "2026 春人工智能 1 班"
    },
    "list": [
      {
        "knowledgeId": "kp_001",
        "canonicalName": "梯度下降",
        "masteryScore": 52,
        "status": "progressing",
        "correctRate": 0.63,
        "answeredCount": 19,
        "lastAnsweredAt": "2026-04-03T10:00:00.000Z"
      }
    ],
    "total": 15,
    "page": 1,
    "pageSize": 20
  }
}
```

成功响应示例（单个知识点，传入 `knowledgeId`）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "student": {
      "id": 201,
      "name": "李四",
      "classId": 101,
      "className": "2026 春人工智能 1 班"
    },
    "knowledge": {
      "id": "kp_001",
      "canonicalName": "梯度下降"
    },
    "masteryScore": 52,
    "status": "progressing",
    "correctRate": 0.63,
    "answeredCount": 19,
    "lastAnsweredAt": "2026-04-03T10:00:00.000Z"
  }
}
```

### `POST /api/teacher/questions/batch`

教师批量导入题目。

说明：

- 前端负责文件解析，后端只接收结构化 JSON。
- 当前阶段只支持单选题。
- 每道题只绑定一个主知识点。

请求体：

```json
{
  "items": [
    {
      "questionId": "q_001",
      "type": "single_choice",
      "knowledgeId": "kp_001",
      "stem": "以下关于梯度下降的说法，哪一项正确？",
      "options": [
        { "key": "A", "text": "沿负梯度方向更新参数" },
        { "key": "B", "text": "随机删除损失函数" },
        { "key": "C", "text": "固定权重不变" },
        { "key": "D", "text": "只在测试集上训练" }
      ],
      "correctOption": "A",
      "difficulty": 0.4,
      "explanation": "梯度下降通过负梯度方向逼近最优解。"
    }
  ]
}
```

说明：

- `questionId` 为可选字段。若提供，直接作为题目业务 ID（`question_code`）存入；若不提供，后端自动生成。

成功响应：

```json
{
  "code": 0,
  "message": "题目导入成功。",
  "data": {
    "createdCount": 1
  }
}
```

### `GET /api/teacher/questions`

查询题目列表。

说明：

- 题目全局共享，所有教师均可查看全部题目。
- 支持分页，使用统一分页参数。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeId` | `string` | 否 | 按知识点筛选 |
| `type` | `string` | 否 | 按题型筛选，当前仅 `single_choice` |
| `keyword` | `string` | 否 | 题干关键字搜索 |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "questionId": "q_001",
        "type": "single_choice",
        "knowledgeId": "kp_001",
        "stem": "以下关于梯度下降的说法，哪一项正确？",
        "options": [
          { "key": "A", "text": "沿负梯度方向更新参数" },
          { "key": "B", "text": "随机删除损失函数" }
        ],
        "correctOption": "A",
        "difficulty": 0.4,
        "explanation": "梯度下降通过负梯度方向逼近最优解。"
      }
    ],
    "total": 50,
    "page": 1,
    "pageSize": 20
  }
}
```

### `GET /api/teacher/questions/:questionId`

查看单题详情。

路径参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `questionId` | `string` | 是 | 题目业务 ID（`question_code`） |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "questionId": "q_001",
    "type": "single_choice",
    "knowledgeId": "kp_001",
    "stem": "以下关于梯度下降的说法，哪一项正确？",
    "options": [
      { "key": "A", "text": "沿负梯度方向更新参数" },
      { "key": "B", "text": "随机删除损失函数" },
      { "key": "C", "text": "固定权重不变" },
      { "key": "D", "text": "只在测试集上训练" }
    ],
    "correctOption": "A",
    "difficulty": 0.4,
    "explanation": "梯度下降通过负梯度方向逼近最优解。",
    "createdBy": 1,
    "createdAt": "2026-04-03T00:00:00.000Z"
  }
}
```

## 3. 图谱查询

该组接口对应 Go 在线服务中的 `GraphQuery` 模块，所有查询都应通过只读 Neo4j 查询模块完成。

### `GET /api/graph/subgraph`

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `root` | `string` | 否 | 建议传知识点 ID，名称查询仅用于兼容 |
| `limit` | `number` | 否 | 默认 `12`，建议限制在 `1-50` |

说明：此处 `limit` 为子图规模上限（最大返回节点数），非分页参数。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "nodes": [
      {
        "id": "node-1",
        "name": "梯度下降",
        "canonicalName": "梯度下降",
        "entityType": "concept",
        "description": "用于优化目标函数的经典迭代方法。",
        "chapterTitle": "优化基础",
        "sourcePath": "machine-learning/chapter-1"
      }
    ],
    "edges": [
      {
        "relationId": "rel-1",
        "relationType": "PREREQUISITE_OF",
        "sourceId": "node-a",
        "targetId": "node-b",
        "sourceName": "偏导数",
        "targetName": "梯度下降"
      }
    ]
  }
}
```

### `GET /api/knowledge/:knowledgeId`

路径参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeId` | `string` | 是 | 知识点 ID |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "node-1",
    "name": "梯度下降",
    "canonicalName": "梯度下降",
    "entityType": "concept",
    "description": "用于优化目标函数的经典迭代方法。",
    "chapterTitle": "优化基础",
    "sourcePath": "machine-learning/chapter-1",
    "summary": "梯度下降是当前知识图谱中的重点知识点。",
    "prerequisites": ["偏导数", "微积分"],
    "related": ["随机梯度下降"],
    "scenarios": ["模型训练"],
    "parentTopics": ["机器学习"],
    "masteryScore": 52,
    "status": "progressing",
    "masteryText": "建议先梳理 2 个关键先修知识点后再深入学习。",
    "recommendReason": "该知识点与偏导数等先修内容直接相关。"
  }
}
```

说明：返回结构对应数据模型 `KnowledgeDetail`。其中 `masteryScore`、`status`、`masteryText` 等用户态字段在登录态下基于当前用户填充，匿名态下为默认值。

字段来源说明：

- **图谱数据**（`name`、`canonicalName`、`entityType`、`description`、`chapterTitle`、`sourcePath`、`prerequisites`、`related`、`scenarios`、`parentTopics`）：来自 Neo4j `GraphQuery` 模块只读查询。
- **用户态数据**（`masteryScore`、`status`）：来自 Redis 掌握度向量或 MySQL `learning_progress` 表。
- **实时生成数据**（`summary`、`masteryText`、`recommendReason`）：由 Go `Recommend` / `Competency` 模块根据图谱关系与用户掌握度实时计算生成。

## 4. 学习作答

当前阶段先只保留**答题结果提交**，不实现通用行为埋点接口。

### `POST /api/learning/answers`

提交一次选择题作答结果，驱动知识点掌握度的在线更新。

请求头：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "questionId": "q_001",
  "knowledgeId": "kp_001",
  "selectedOption": "B",
  "isCorrect": false,
  "durationMs": 18000,
  "answeredAt": "2026-04-03T10:00:00.000Z"
}
```

成功响应：

```json
{
  "code": 0,
  "message": "答题结果已记录。",
  "data": {
    "accepted": true,
    "questionId": "q_001",
    "knowledgeId": "kp_001",
    "degraded": false
  }
}
```

说明：

- 当前用户身份从 JWT 推导。
- 成功返回表示在线状态更新已被接受。
- Go 在线服务会在主响应返回后异步写入 Neo4j 用户能力图谱，该写回属于异步副作用，不阻塞本接口成功返回。
- 当 Redis 不可用且系统进入降级路径时，可返回 `degraded: true` 供前端提示。

### `GET /api/learning/questions`

获取当前学生可用的练习题目。

请求头：

```http
Authorization: Bearer <token>
```

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeId` | `string` | 否 | 按知识点筛选 |
| `page` | `number` | 否 | 页码，从 1 开始 |
| `pageSize` | `number` | 否 | 每页数量，默认 20，上限 100 |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "questionId": "q_001",
        "type": "single_choice",
        "knowledgeId": "kp_001",
        "stem": "以下关于梯度下降的说法，哪一项正确？",
        "options": [
          { "key": "A", "text": "沿负梯度方向更新参数" },
          { "key": "B", "text": "随机删除损失函数" },
          { "key": "C", "text": "固定权重不变" },
          { "key": "D", "text": "只在测试集上训练" }
        ],
        "correctOption": "A",
        "difficulty": 0.4,
        "explanation": "梯度下降通过负梯度方向逼近最优解。"
      }
    ],
    "total": 50,
    "page": 1,
    "pageSize": 20
  }
}
```

说明：

- 学生端只读访问题库，返回完整题目内容（含正确答案与解析），供前端答题交互使用。
- `isCorrect` 由前端本地判定后通过 `POST /api/learning/answers` 提交。
- 当前用户身份从 JWT 推导，仅返回 `status = active` 的题目。
- 支持分页，使用统一分页参数。

## 5. 能力画像

该组接口对应 Go 在线服务中的 `Competency` 模块，在线读 Redis 掌握度向量并进行内存矩阵计算。

### `GET /api/competency/profile`

请求头：

```http
Authorization: Bearer <token>
```

说明：

- 当前用户身份从 JWT 推导，不接受前端传 `uid` 查询他人画像。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "profile": {
      "name": "李四",
      "role": "student",
      "classId": 101,
      "className": "2026 春人工智能 1 班",
      "target": "算法工程师",
      "summary": "当前处于基础补强阶段。",
      "stats": [
        { "label": "学习中知识点", "value": "6" }
      ],
      "goals": ["补齐 梯度下降，提升相关先修理解"],
      "recentRecords": ["当前薄弱：梯度下降（掌握度 46%）"]
    },
    "radarAxes": [
      { "name": "数学基础", "value": 46 }
    ],
    "growthInsights": [
      { "label": "数学基础", "value": 46, "note": "当前数学基础仍有明显提升空间。" }
    ],
    "growthSummaryCards": [
      { "label": "已形成优势", "value": "深度学习" }
    ],
    "heatmapDomains": ["数学基础", "机器学习", "深度学习", "知识图谱", "场景应用"],
    "heatmapRows": [
      { "label": "梯度下降", "values": [46, 8, 8, 8, 8] }
    ]
  }
}
```

说明：返回结构对应数据模型 `CompetencyProfile`。

## 6. 推荐

该组接口对应 Go 在线服务中的 `Recommend` 模块，其核心输入来自：

- 当前登录用户的掌握度向量（Redis）
- 图谱查询模块返回的先修关系或目标节点相关子图（Neo4j 只读）

推荐策略说明：

- 不能只依赖 Neo4j 相邻节点做推荐。
- 推荐应采用“三阶段”：候选召回、状态过滤、规则打分。
- 候选召回依赖 Neo4j 中的先修、关联、主题等关系。
- 状态过滤依赖 Redis / MySQL 中的掌握度、答题记录、历史学习状态。
- 最终排序至少应考虑：先修缺口、当前薄弱度、目标方向匹配度、重复惩罚、难度跳跃惩罚。

### `GET /api/recommend/path`

请求头：

```http
Authorization: Bearer <token>
```

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `target` | `string` | 否 | 推荐目标知识点 ID，不传时由系统按首页主线生成推荐 |
| `limit` | `number` | 否 | 返回推荐项数量上限，非分页参数 |
| `scene` | `string` | 否 | 推荐发生场景，如 `dashboard`、`detail`、`after_wrong_answer` |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "path": [
      {
        "stage": "补弱推荐",
        "title": "梯度下降",
        "detail": "建议先补偏导数，再进入梯度下降。",
        "route": "/knowledge/node-1",
        "reasonCode": "missing_prerequisite",
        "reasonLabel": "先修缺口",
        "score": 0.87
      }
    ]
  }
}
```

说明：返回结构中 `path` 为 `RecommendedPathItem[]` 数组。

## 7. 聚合接口

该组接口是 **BFF 视图聚合接口**，用于减少首页请求数，方便前端一次性拉取渲染所需数据。

### `GET /api/catalog/domains`

该接口用于替代当前首页仍在使用的前端本地静态知识目录数据。

建议用途：

- 首页知识域导航。
- 首页知识点卡片列表。
- 知识点搜索的数据源。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "domains": [
      {
        "id": "ml",
        "name": "机器学习",
        "desc": "经典模型与训练方法论",
        "points": [
          {
            "id": "gradient-descent",
            "name": "梯度下降",
            "level": "core",
            "levelLabel": "核心",
            "mastery": 52,
            "description": "模型训练的核心优化算法。",
            "recommend": "补弱推荐",
            "route": "/knowledge/gradient-descent"
          }
        ]
      }
    ]
  }
}
```

说明：

- 这是建议新增接口，当前前端在后端未提供该接口时可继续使用静态兜底数据。
- 一旦后端落地，首页应优先消费在线目录结果，而不是长期依赖本地常量。

### `GET /api/dashboard/summary`

说明：

- 用于首页、学习仪表盘等聚合视图。
- 匿名态可以返回公共默认内容；登录态则基于当前用户返回个性化结果。
- 不接受前端显式传 `uid`，登录态身份仍从 JWT 推导。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "quickStats": [
      { "label": "学习中知识点", "value": "6" },
      { "label": "已掌握知识点", "value": "12" }
    ],
    "stats": [
      { "label": "学习中知识点", "value": "6" },
      { "label": "已掌握知识点", "value": "12" }
    ],
    "knowledgeCards": [
      {
        "id": "node-1",
        "type": "补弱推荐",
        "progress": 49,
        "title": "梯度下降",
        "domain": "数学基础",
        "tag": "知识图谱中的关键能力节点",
        "route": "/knowledge/node-1"
      }
    ],
    "recommendedPath": [
      {
        "stage": "补弱推荐",
        "title": "梯度下降",
        "detail": "建议先补偏导数，再进入梯度下降。",
        "route": "/knowledge/node-1",
        "reasonCode": "missing_prerequisite",
        "reasonLabel": "先修缺口",
        "score": 0.87
      }
    ]
  }
}
```

说明：

- `quickStats` 为首页主用字段名。
- `stats` 作为兼容别名可短期保留。
- `knowledgeCards` 对应 `KnowledgeCard[]`。
- `recommendedPath` 对应 `RecommendedPathItem[]`。
- 匿名态返回公共默认内容；登录态返回当前用户个性化结果。

### `GET /api/recommend/projects`

该接口用于承接首页“行业实践实时推荐”模块，替代当前前端本地排序逻辑。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "image-classify",
        "title": "图像分类项目",
        "domain": "行业实践",
        "fitPercent": 78,
        "fitHint": "侧重 深度学习、Python 工程化 · 匹配度 78%",
        "reasonCode": "ability_match",
        "reasonLabel": "能力匹配",
        "route": "/knowledge/image-classify"
      }
    ]
  }
}
```

说明：

- 这是建议新增接口，用于把首页项目推荐从静态卡片升级为在线个性化结果。
- 若当前阶段暂未落地，前端可继续使用静态展示，但主契约应提前保留该接口位置。

## 8. 扩展接口

该组接口不属于当前在线核心闭环，数据往往来自离线产物或二期能力，语义上允许“已成功响应但内容未就绪”。

### `GET /api/frontier/briefing`

该接口用于知识点课后扩展与前沿阅读摘要。

定位说明：

- 它是扩展接口，不属于当前在线推荐主链路。
- 当前仓库里“前沿知识域”仍处于本体与文档先行阶段，完整的论文抓取、主题抽取、digest 生成与 Go 路由尚未全部落地。
- 因此这里描述的是**目标契约**，不是“当前后端已全部实现完成”的既成事实。
- 数据通常来自离线爬虫、语料处理和内容整理的产物。
- 因此接口成功并不代表每个知识点都已有完整内容。

查询参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeId` | `string` | 否 | 推荐主用字段 |
| `knowledge` | `string` | 否 | 当前兼容字段，表示知识点名称 |

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "knowledgeId": "gradient-descent",
    "status": "ready",
    "knowledge": "梯度下降",
    "message": "已为该知识点准备第一版课后扩展条目。",
    "digestSummary": "梯度下降近期在大模型训练稳定性与自适应优化方向上仍是核心基础。",
    "topics": [
      {
        "topicId": "adaptive-optimization",
        "name": "自适应优化",
        "reason": "该主题直接扩展了梯度下降在训练策略上的应用边界。"
      }
    ],
    "papers": [
      {
        "paperId": "paper_001",
        "title": "A Survey on Adaptive Optimization",
        "venue": "NeurIPS",
        "year": 2025,
        "url": "https://example.com/paper"
      }
    ],
    "generatedAt": "2026-04-08T00:00:00Z"
  }
}
```

说明：

- `status` 建议至少支持：`ready`、`not_ready`、`stale`、`failed`。
- 当内容尚未就绪时，建议仍返回 HTTP 200，但 `status = not_ready`，同时给出可读 `message`。
- `digestSummary` 是前端知识点详情页最稳定可消费的主字段，不建议只返回自由文本 `digest` 而没有状态字段。
- 数据通常来自离线流水线产物，允许最终一致。

## 页面消费映射（附录）

下表仅用于说明当前前端页面的消费关系，不作为 API 主契约的组织方式。

| 页面 | 接口 | 说明 |
| --- | --- | --- |
| 登录页 | `POST /api/auth/login` | 创建登录态，返回 token 和用户信息 |
| 教师端班级管理 | `POST /api/teacher/classes`、`GET /api/teacher/classes` | 创建并查看班级 |
| 教师端学生导入 | `POST /api/teacher/classes/:classId/students` | 批量创建学生账号 |
| 教师端教师导入 | `POST /api/teacher/teachers` | 批量创建教师账号 |
| 教师端题目导入 | `POST /api/teacher/questions/batch` | 批量导入结构化题目 |
| 教师端学生知识状态 | `GET /api/teacher/students/:studentId/knowledge-status` | 查看学生知识点掌握情况 |
| 全局鉴权启动 | `GET /api/auth/me` | 校验 token 是否仍有效 |
| 首页 | `GET /api/dashboard/summary`、`GET /api/catalog/domains`、`GET /api/recommend/projects` | 首页摘要、知识目录、项目推荐 |
| 知识图谱页 | `GET /api/graph/subgraph` | 加载一跳子图 |
| 知识点详情页 | `GET /api/knowledge/:knowledgeId`、`GET /api/graph/subgraph`、`GET /api/competency/profile` | 加载知识点详情、局部子图、侧边画像 |
| 成长分析页 | `GET /api/competency/profile` | 能力画像、热力图 |
| 详情页侧栏推荐 | `GET /api/recommend/path` | 推荐学习路径 |
| 课后扩展 | `GET /api/frontier/briefing` | 扩展内容摘要 |
| 答题 / 练习 | `GET /api/learning/questions`、`POST /api/learning/answers` | 拉取练习题目、驱动掌握度在线更新 |

## 当前不实现的接口

以下需求后续可能恢复，但当前阶段暂不实现，不纳入最小可行契约：

- `POST /api/auth/register`：学生自主注册关闭，前端若保留 `/register` 页面，仅作说明页。
- `POST /api/behavior/event`：通用行为埋点后置。
- 复杂文件上传解析接口：由前端自行完成文件解析，后端只接受结构化 JSON。
- `PUT/DELETE /api/teacher/questions/:questionId`：题目修改与删除。
- `PUT/DELETE /api/teacher/classes/:classId/students/:studentId`：学生信息修改与移除。
- `PUT/PATCH /api/teacher/classes/:classId`：班级信息修改与归档。
- `PUT/PATCH /api/teacher/teachers/:teacherId`：教师信息修改。
- `PUT /api/student/profile`：学生自行修改目标方向（`target`）。
- `POST /api/assistant/chat`、`POST /api/assistant/generate-content`：当前前端仍可通过开发代理直连模型，后端统一接管属于扩展能力，不纳入本阶段最小闭环。

## 兼容说明

为兼顾当前前端实现，以下兼容策略暂时保留：

- `GET /api/recommend/path` 返回字段名继续使用 `path`。
- `GET /api/dashboard/summary` 返回字段名继续使用 `recommendedPath`、`knowledgeCards`，并短期同时兼容 `quickStats` / `stats`。
- `GET /api/graph/subgraph` 仍兼容通过名称查询 `root`，但主契约推荐使用稳定 ID。
- 图谱节点跳转字段短期兼容 `routeKnowledgeId`、`knowledgeId`、`slug`、`id`，但推荐后端优先返回 `routeKnowledgeId`。
- `GET /api/frontier/briefing` 仍兼容通过 `knowledge` 名称查询，但长期建议迁移为 `knowledgeId`。
- 首页知识目录与项目推荐在后端完全落地前，前端允许保留静态兜底内容，但这不改变主契约中对 `GET /api/catalog/domains`、`GET /api/recommend/projects` 的建议方向。

这些兼容字段不代表长期最优设计，后续如需统一命名，应通过版本化或前后端联动改造完成。
