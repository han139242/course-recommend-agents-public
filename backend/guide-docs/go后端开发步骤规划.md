# Go 后端开发步骤规划

本文基于 `backend/guide-docs` 下现有文档整理，目标是为本项目的 Go 在线服务提供一份可直接执行的开发路线图。规划范围严格遵循以下设计前提：

- 架构形态：Go 单体在线服务 + Python 离线流水线 + Vue 前端。
- 对外边界：前端只访问 Go API Server，统一前缀 `/api`。
- 存储分工：MySQL 承担强一致业务数据，Redis 承担实时缓存，Neo4j 承担知识图谱与用户能力图谱。
- Go 侧职责：承接在线低延迟请求，读取 MySQL / Redis / Neo4j，只异步写入用户能力图谱域，不写知识图谱域。
- Python 侧职责：离线抓取、处理、抽取与批量写入知识图谱，并通过 Redis Pub/Sub 通知 Go 热更新拓扑。
- 工程原则：清晰边界、手动依赖注入、统一错误模型、统一响应结构、统一日志字段、优先做扎实单体，不照搬微服务复杂度。

---

# 一、先给结论：推荐开发顺序

推荐按照以下顺序推进，而不是按接口列表平铺开发：

1. **先搭工程骨架和基础设施接入**。
2. **再打通认证、配置、日志、错误、健康检查这些横切能力**。
3. **再落强一致主链路：教师端基础数据管理 + 学生答题落库**。
4. **再落实时状态链路：Redis 掌握度向量、能力画像聚合、推荐算法**。
5. **再落图谱链路：Neo4j 只读查询 + Go 异步写用户能力图谱**。
6. **最后补齐聚合接口、前沿扩展接口、可观测性、性能压测与上线准备**。

这样安排的原因是：

- 先把工程底座搭稳，后续业务不会反复返工。
- 先做 MySQL 主真相链路，再做 Redis/Neo4j 加速与关系补充，更符合文档约束。
- 先做同步主流程，再做异步最终一致副作用，便于定位问题。
- 先保证系统“可运行、可观测、可回滚”，再追求“更快、更智能”。

---

# 二、开发边界与必须遵守的约束

## 2.1 架构边界

## 推荐方案

- Go 在线服务作为唯一对外 HTTP API 提供方。
- Python 离线流水线不直接暴露给前端。
- Neo4j、MySQL、Redis 都是内部基础设施。
- Go 只写用户能力图谱域。
- Python 只写知识图谱域。

## 不推荐方案

- 前端直连 Neo4j / Redis。
- Go 同时维护知识图谱实体与用户能力图谱两个写入口。
- Python 直接承接前端在线请求。
- 为了“未来可拆服务”提前引入 gRPC、消息中间件、复杂补偿框架。

## 原因

这会破坏现有文档明确的职责边界，使一致性、调试路径和后续维护成本迅速失控。

## 2.2 数据主真相约束

- **MySQL 是强一致业务数据主真相**：用户、班级、题目、答题记录、学习进度快照、能力快照。
- **Redis 不是唯一真相**：只做实时读写加速、短期缓存与状态聚合。
- **Neo4j 是关系补充与图查询载体**：不是登录认证、用户基础资料、核心事务数据的主库。

## 2.3 响应与错误约束

- HTTP 状态码要语义化使用，不能把所有业务错误都包成 HTTP 200。
- Body 统一使用：`code`、`message`、`data`，建议补充 `trace_id`、`timestamp`。
- 错误必须分层翻译：repository 翻译底层错误，service 翻译业务语义，handler 翻译协议语义。

## 2.4 工程组织约束

- 推荐目录组织：按业务模块聚合，再在模块内分 handler / service / repository / dto / entity。
- `main.go` 不承担全部装配逻辑。
- 依赖注入采用手动构造，不引入复杂 DI 框架。
- Redis key、TTL、状态值、业务枚举必须集中治理，不允许业务里到处散落魔法字符串和魔法数字。

---

# 三、推荐里程碑规划

建议将 Go 后端开发拆成 8 个里程碑，每个里程碑都能形成一个可验收的阶段成果。

| 里程碑 | 目标 | 是否可独立验收 |
|---|---|---|
| M1 | 工程骨架与基础设施接入 | 是 |
| M2 | 认证、统一响应、日志、错误、健康检查 | 是 |
| M3 | 教师端基础主数据管理 | 是 |
| M4 | 题库与答题主链路 | 是 |
| M5 | Redis 实时状态与学习进度落盘 | 是 |
| M6 | 能力画像、图谱只读查询、推荐路径 | 是 |
| M7 | Go 异步写用户能力图谱 + Python 拓扑热更新协同 | 是 |
| M8 | 聚合接口、前沿扩展、可观测性、压测与上线 | 是 |

下面按里程碑展开。

---

# 四、里程碑 M1：工程骨架与基础设施接入

## 4.1 目标

建立一个符合文档规范的 Go 单体服务骨架，让项目具备以下能力：

- 能启动。
- 能加载配置。
- 能初始化日志。
- 能连接 MySQL / Redis / Neo4j。
- 能暴露基础 HTTP 服务和健康检查接口。
- 能优雅停机。

## 4.2 目录结构建议

推荐第一版直接建立如下目录：

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── bootstrap/
│   │   ├── app.go
│   │   ├── config.go
│   │   ├── logger.go
│   │   ├── mysql.go
│   │   ├── redis.go
│   │   ├── neo4j.go
│   │   ├── router.go
│   │   └── shutdown.go
│   ├── config/
│   │   ├── config.go
│   │   ├── app.go
│   │   ├── http.go
│   │   ├── mysql.go
│   │   ├── redis.go
│   │   ├── neo4j.go
│   │   ├── auth.go
│   │   └── log.go
│   ├── middleware/
│   ├── router/
│   └── module/
│       ├── auth/
│       ├── teacher/
│       ├── learning/
│       ├── competency/
│       ├── recommend/
│       ├── graph/
│       ├── dashboard/
│       └── frontier/
├── pkg/
│   ├── apperr/
│   ├── ctxmeta/
│   ├── logger/
│   ├── result/
│   ├── key/
│   └── constant/
├── migrations/
└── test/
    ├── integration/
    ├── e2e/
    └── fixture/
```

## 4.3 本阶段开发任务

### 任务 1：初始化 Go 模块与基础依赖

建议依赖：

- HTTP 框架：`gin`
- MySQL ORM：`gorm` + MySQL driver
- Redis：`go-redis`
- Neo4j：官方 Go driver
- 配置：轻量 env + struct 方案
- 日志：`slog` 或 `zap`
- JWT：成熟稳定库
- 参数校验：Gin 默认绑定 + validator

### 任务 2：建立配置模型

建议配置结构至少包括：

- `AppConfig`
- `HTTPConfig`
- `MySQLConfig`
- `RedisConfig`
- `Neo4jConfig`
- `AuthConfig`
- `LogConfig`

要求：

- 启动时统一读取。
- 关键字段启动时校验。
- 打印脱敏后的配置摘要。
- 业务代码禁止直接 `os.Getenv()`。

### 任务 3：完成基础设施初始化

初始化顺序严格按文档建议：

1. 加载配置。
2. 初始化日志。
3. 初始化 MySQL。
4. 初始化 Neo4j。
5. 初始化 Redis。
6. 初始化 repository。
7. 初始化 service。
8. 初始化 handler。
9. 初始化 router。
10. 初始化 HTTP Server。
11. 注册优雅停机钩子。
12. 启动服务。

### 任务 4：提供健康检查与版本信息

建议最早实现：

- `GET /healthz`
- `GET /readyz`
- 构建版本信息注入

其中：

- `healthz` 侧重进程活性。
- `readyz` 侧重依赖可用性。

## 4.4 本阶段产出物

- 可启动的服务骨架。
- 配置加载与校验机制。
- 基础依赖连接初始化代码。
- 健康检查接口。
- 优雅停机机制。

## 4.5 验收标准

- 本地启动成功。
- MySQL / Redis / Neo4j 可连通。
- 服务可响应健康检查。
- 停机时连接和后台任务能按顺序释放。
- 日志中可看到启动摘要和关闭阶段日志。

---

# 五、里程碑 M2：认证、统一响应、日志、错误、健康检查增强

## 5.1 目标

把横切能力一次打牢，避免后面每个业务模块重复造轮子。

## 5.2 本阶段必须优先落地的公共能力

### 1. 请求链路元数据

统一建立请求上下文字段：

- `trace_id`
- `request_id`
- `user_id`
- `role`
- `device_id`（如未来需要）

要求：

- 每个请求自动生成 `trace_id`。
- 日志自动带 `trace_id`。
- 响应头回写 `X-Request-ID` 或等价字段。

### 2. 中间件体系

按推荐顺序接入：

1. `RequestID`
2. `RealIP`
3. `Logger`
4. `Recovery`
5. `CORS`
6. `Auth`
7. `RateLimit`

### 3. 统一响应结构

统一成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "trace_id": "xxx",
  "timestamp": "2026-04-04T00:00:00Z"
}
```

统一失败响应：

```json
{
  "code": 10001,
  "message": "参数错误",
  "data": null,
  "trace_id": "xxx",
  "timestamp": "2026-04-04T00:00:00Z"
}
```

### 4. 应用错误模型

建议先定义稳定的错误分类：

- 参数错误
- 未认证
- 无权限
- 资源不存在
- 资源冲突
- 依赖不可用
- 系统异常
- 限流

建议 repository 统一翻译：

- MySQL 唯一键冲突
- Redis 连接错误
- Neo4j 会话错误

### 5. JWT 认证模块

优先支持：

- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`（如当前只做前端删 token，可先保持幂等空实现）

要求：

- token 中写入：`user_id`、`role`。
- token 校验通过后，将上下文元数据注入 request context。
- 权限控制在 service 层再次校验，不仅依赖路由中间件。

## 5.3 本阶段产出物

- 统一响应工具。
- 统一错误模型。
- JWT 认证中间件。
- 请求日志、恢复、链路字段。
- `login` / `me` 主接口。

## 5.4 验收标准

- 登录成功可返回 token 与用户信息。
- 非法 token、过期 token、无 token 均能稳定返回错误。
- 关键日志都能带 trace 信息。
- 各类错误响应结构统一。

---

# 六、里程碑 M3：教师端基础主数据管理

## 6.1 目标

打通教师端最基础的主数据管理能力，让系统具备“建班、看班、导入学生、看学生”的可用闭环。

## 6.2 涉及的数据表

本里程碑优先依赖：

- `users`
- `classrooms`

后续里程碑会继续接入：

- `questions`
- `question_options`
- `student_answers`
- `learning_progress`
- `competency_snapshots`

本里程碑先重点使用：

- `users`
- `classrooms`

## 6.3 先做哪些 API

### 教师端用户与班级管理

建议优先实现：

- `POST /api/teacher/classes`
- `GET /api/teacher/classes`
- `POST /api/teacher/classes/:classId/students`
- `GET /api/teacher/classes/:classId/students`

说明：

- 当前代码事实已经落地的是班级创建、班级分页查询、批量创建学生、班级学生分页查询。
- `POST /api/teacher/teachers` 仍属于规划能力，不应与上述 4 个已落地接口混写为“当前优先实现完成项”。
- 当前阶段教师与管理员同责，统一使用 `teacher` 角色。
- 学生不能自主注册，必须由教师创建或导入。
- 教师只能管理自己创建的班级与班级内学生。
- 分页参数建议统一为 `page`、`pageSize`，默认值分别为 `1`、`20`，`pageSize` 最大为 `100`。

## 6.4 关键实现点

### 1. 用户与角色约束

- `email` 全局唯一。
- `student_no` 对学生唯一。
- 学生必须绑定 `class_id`。
- 登录时校验 `status`。

### 2. 班级作用域控制

- 班级查询默认限定 `teacher_id = 当前登录教师 ID`。
- 教师不能跨班级写入学生。

### 3. DTO 与实体分离

不能直接把 GORM Model 暴露给前端，必须设计：

- 请求 DTO
- 响应 DTO
- 持久化 Model

### 4. 批量导入约束

遵守文档要求：

- 前端解析 Excel / CSV。
- 后端只接收结构化 JSON。
- 后端只负责业务校验与批量入库。

## 6.5 本阶段产出物

- 教师端基础主数据管理 API。
- 对应的 MySQL migration。
- 教师作用域权限控制。
- 基础分页能力。

## 6.6 验收标准

- 教师可创建班级、批量导入学生并查看班级列表、学生列表。
- 列表接口分页稳定。
- 同名班级、重复邮箱、重复学号能返回稳定冲突语义。
- 非本教师班级资源不可访问。

---

# 七、里程碑 M4：题库与答题主链路

## 7.1 目标

让教师端可以导入题目，学生端可以取题、答题，完成最核心的“学习行为落库”闭环。

## 7.2 涉及的数据表

- `questions`
- `question_options`
- `student_answers`

## 7.3 优先实现的 API

### 教师端

- `POST /api/teacher/questions/batch`
- `GET /api/teacher/questions`

### 学生端

- `GET /api/learning/questions`
- `POST /api/learning/answers`

## 7.4 关键实现点

### 1. 题目导入责任边界

- 前端负责文件解析。
- 后端接收标准 JSON。
- 后端做业务校验、去重、批量写库。

### 2. 题目与知识点绑定

第一阶段保持简单：

- 一题只绑定一个 `knowledge_id`。
- 不做多知识点映射。

### 3. 学生答题写入顺序

必须保证：

1. 校验题目存在与可用。
2. 校验学生身份与班级归属。
3. 计算正误。
4. 记录 `student_answers`。
5. 再触发后续掌握度更新流程。

## 7.5 验收标准

- 教师可批量导入题目。
- 学生可拉取题目并提交作答。
- 答题记录准确落库。
- 异常题目、无权限、非法参数有稳定响应。

---

# 八、里程碑 M5：Redis 实时状态与学习进度落盘

## 8.1 目标

把学生知识点掌握度的实时更新链路跑通，形成“Redis 实时态 + MySQL 落盘态”的双层模型。

## 8.2 本阶段关键设计

### 1. Redis key 先统一落地

必须先建立 `pkg/key`：

- `uid:{studentId}:kp`
- `uid:{studentId}:competency`
- `uid:{studentId}:recommend:{targetId}`

并建立 TTL 常量与 key builder，禁止业务手写 key。

### 2. UserState 模块落地

建议单独建模块：

- `internal/module/learning` 负责答题主流程。
- `internal/module/userstate` 或在学习模块内拆出实时状态服务，负责掌握度增量更新。

### 3. 答题后实时更新策略

根据文档，推荐流程：

1. 答题记录先写 MySQL。
2. 根据题目所属 `knowledge_id` 计算掌握度增量。
3. 使用 Redis `HINCRBYFLOAT` 原子更新 `uid:{studentId}:kp`。
4. 按批次或周期将结果回写 `learning_progress`。

### 4. 降级策略

必须明确：

- Redis 读失败：优先回源 MySQL。
- Redis 写失败：记录日志并决定是否短重试。
- 不能把 Redis 当成唯一数据来源。

## 8.3 需要补齐的表与逻辑

- `learning_progress`
- 周期性快照/回写任务
- 掌握度状态映射：`not_started` / `progressing` / `solid`

## 8.4 推荐实现顺序

1. `pkg/key`
2. `pkg/constant`
3. 学习进度 repository
4. Redis repository
5. 答题后增量更新服务
6. 快照回写任务
7. 查询学生知识点掌握状态接口

## 8.5 本阶段产出物

- Redis 实时掌握度向量。
- MySQL 学习进度快照。
- Redis 失败回源与日志策略。

## 8.6 验收标准

- 学生答题后 Redis 掌握度能实时变化。
- 周期性回写后 MySQL 快照正确。
- Redis 不可用时系统可降级，不致主链路完全不可用。

---

# 九、里程碑 M6：能力画像、图谱只读查询、推荐路径

## 9.1 目标

打通系统“智能化输出”主能力：

- 图谱查询
- 能力画像
- 推荐路径
- 学习仪表盘基础聚合

## 9.2 图谱查询模块落地

建议建立：

- `internal/module/graph`
- `internal/module/competency`
- `internal/module/recommend`
- `internal/module/dashboard`

其中：

- `graph` 只负责 Neo4j 只读查询。
- `competency` 负责画像聚合。
- `recommend` 负责路径推荐。
- `dashboard` 负责 BFF 视图聚合。

## 9.3 Graph 模块职责

优先实现：

- `GET /api/graph/subgraph`
- `GET /api/knowledge/:knowledgeId`

要求：

- 只读 session。
- 拒绝混入写语句。
- 返回稳定的图查询 DTO，而不是原始 Neo4j 结果。
- DTO 中显式区分图节点内部 ID 与前端路由知识点 ID，建议优先返回 `routeKnowledgeId`，并兼容 `knowledgeId` / `slug` / `id`。

## 9.4 Competency 模块职责

优先实现：

- `GET /api/competency/profile`

处理流程：

1. 从 Redis 读取 `uid:{studentId}:kp`。
2. 读取或持有内存中的知识点到能力维度映射矩阵。
3. 执行矩阵计算得到能力分数。
4. 缓存结果到 Redis。
5. 返回前端雷达图数据。

约束：

- 在线链路不能做大规模 Neo4j 遍历。
- 能力分值与掌握度分值值域不同，必须明确区分：
  - 掌握度：0-100
  - 能力分：0-1

## 9.5 Recommend 模块职责

优先实现：

- `GET /api/recommend/path`

推荐流程：

1. 调用 Graph 模块获取目标节点先修闭包子图。
2. 批量读取用户掌握度向量。
3. 基于先修缺口、薄弱项、目标匹配、重复惩罚、难度惩罚做路径评分。
4. 输出推荐顺序与推荐原因。

建议先做规则型推荐，不引入向量检索和复杂模型。

## 9.6 Dashboard 聚合接口

优先实现：

- `GET /api/dashboard/summary`
- `GET /api/teacher/students/:studentId/knowledge-status`

说明：

- 这是 BFF 聚合视图，不应反向污染底层领域模型。
- 若首页仍依赖前端静态课程目录与项目推荐兜底，建议在本里程碑中同步规划以下接口，而不是默认前端继续长期本地硬编码：
  - `GET /api/catalog/domains`：替代首页静态知识目录数据。
  - `GET /api/recommend/projects`：承接首页“行业实践实时推荐”的在线排序结果。
- 图谱子图与知识点详情接口建议在 DTO 中显式区分“图节点内部 ID”和“前端路由知识点 ID”，避免图节点能展示但无法稳定跳转到 `/knowledge/:knowledgeId`。

## 9.7 本阶段产出物

- 图谱只读查询接口。
- 学生能力画像接口。
- 推荐路径接口。
- 教师查看学生知识状态接口。
- 基础仪表盘聚合接口。

## 9.8 验收标准

- 能从 Redis + Neo4j 组合得到稳定画像结果。
- 推荐路径结果符合先修约束。
- 推荐链路和画像链路满足基础性能目标。
- Graph 查询与写入边界严格隔离。

---

# 十、里程碑 M7：Go 异步写用户能力图谱 + Python 拓扑热更新协同

## 10.1 目标

把系统中“最终一致”的异步链路补齐，让在线服务与离线流水线真正形成闭环。

## 10.2 Go 侧要做什么

### 1. GraphWriter 异步写入器

建议建立独立模块：

- `internal/module/graphwriter`

职责：

- 接收来自 Competency 模块的用户画像更新事件。
- 按批次或时间窗口聚合。
- 使用 `UNWIND + MERGE` 批量写入 Neo4j。

约束：

- 仅写 `User` 与 `HAS_COMPETENCY` 关系域。
- 不写知识图谱实体域。
- 失败重试 3 次，之后写死信日志。

### 2. 后台 worker 生命周期管理

必须接入优雅停机流程：

- 停止接收新任务。
- 等待在途批次完成。
- 必要时刷新缓冲。
- 关闭 Neo4j / Redis 连接。

## 10.3 Python 协同点

Go 侧需要配合 Python 离线链路：

- 订阅 Redis Pub/Sub 的 `topology_updates`。
- 收到更新后重新加载权重矩阵。
- 采用双缓冲 + 原子切换，避免更新过程中阻塞在线请求。

## 10.4 为什么这一阶段放在后面

因为这是典型的“副作用异步一致性链路”：

- 它不应阻塞主请求返回。
- 它依赖前面画像、图谱和实时状态已经稳定。
- 如果前面底座没打牢，异步链路非常难调试。

## 10.5 本阶段产出物

- Go 内部异步写用户能力图谱能力。
- Redis Pub/Sub 拓扑更新订阅器。
- 热更新矩阵加载机制。

## 10.6 验收标准

- 画像更新后可异步写回 Neo4j。
- Python 发布拓扑更新通知后，Go 可无损热更新矩阵。
- 在线请求在更新窗口仍可正常服务。

---

# 十一、里程碑 M8：聚合接口、前沿扩展、可观测性、压测与上线准备

## 11.1 目标

在主链路稳定后，补全产品可交付性与运维可交付性。

## 11.2 前沿扩展接口

优先实现：

- `GET /api/frontier/briefing?knowledgeId={id}`

要求：

- 这里要明确区分“目标态”与“现状”：当前仓库中已存在前沿知识本体定义与文档设计，但还没有完整的论文抓取、digest 生成与 Go 路由落地。
- 因此前沿接口开发应拆成两个子阶段：
  1. 先补齐离线数据落点：`Paper`、`ResearchTopic`、`FrontierDigest` 及其与 `KnowledgeNode` 的关系。
  2. 再实现 Go 侧只读查询、状态判断与缓存聚合。
- 对接的是已被 Python 写入的前沿知识域。
- 允许最终一致，不要要求实时强一致。
- 支持“当前内容尚未就绪”的稳定返回语义。
- 建议响应至少包含：`knowledgeId`、`status`、`digestSummary`、`topics`、`papers`、`generatedAt`。
- 若在线层先于离线层开发，必须返回可区分的未就绪状态，而不是假装返回空成功数据。

## 11.3 可观测性增强

建议补齐：

### 第一优先级

- 高质量访问日志
- Panic 日志
- 慢查询日志
- Redis 异常日志
- 构建版本信息
- 健康检查

### 第二优先级

- Prometheus metrics
- pprof

### 第三优先级

- OpenTelemetry tracing

对于当前体量，建议先确保日志质量，再逐步加 metrics 和 tracing。

## 11.4 性能压测

建议至少压以下链路：

- 登录
- 获取题目
- 提交答案
- 获取能力画像
- 获取推荐路径
- 获取图谱子图

重点观察：

- P95 / P99 延迟
- Redis 命中率
- Neo4j 查询耗时
- MySQL 慢查询
- GraphWriter 队列积压

## 11.5 上线前检查项

- migration 可重复执行与回滚策略明确。
- 配置项说明完整。
- 环境变量模板完整。
- 密钥不入库。
- 健康检查与 readiness 可被部署系统使用。
- 关键链路有最小冒烟用例。
- 异步 worker 可控停机。

## 11.6 本阶段产出物

- 前沿扩展查询接口。
- 指标与压测报告。
- 上线清单。
- 运维启动说明。

## 11.7 验收标准

- 核心接口延迟在目标范围内。
- 可观测性足够支持问题定位。
- 上线环境具备最小运维保障。

---

# 十二、推荐模块拆分与负责人视角开发顺序

结合文档中的团队分工，可以把 Go 开发进一步拆成“模块落地顺序”。

## 12.1 第一批必须先落地的模块

- `config`
- `bootstrap`
- `middleware`
- `auth`
- `pkg/apperr`
- `pkg/result`
- `pkg/logger`
- `pkg/ctxmeta`
- `pkg/key`

## 12.2 第二批业务主链路模块

- `teacher`
- `learning`
- `userstate`

## 12.3 第三批智能化输出模块

- `graph`
- `competency`
- `recommend`
- `dashboard`

## 12.4 第四批异步与扩展模块

- `graphwriter`
- `frontier`

建议：

1. 先由 1 人打底座。
2. 然后从 `auth + teacher + learning` 三个核心模块开始实现。
3. 再补 `userstate + competency + graph + recommend`。
4. 最后再接 `graphwriter + frontier`。

---

# 十三、最后给一份可执行的最小任务清单

如果按“本周就开始开发”的视角，建议顺序是：

## 第一周

- 完成 M1
- 完成 M2
- 打通登录
- 打通统一错误和统一响应

## 第二周

- 完成 M3
- 打通教师建班、查班、导入学生、查学生

## 第三周

- 完成 M4
- 打通题目导入、拉题、答题落库

## 第四周

- 完成 M5
- 打通 Redis 掌握度实时更新与 MySQL 快照回写

## 第五周

- 完成 M6
- 打通画像、图谱查询、推荐路径、仪表盘聚合
- 同步补齐首页目录接口与项目推荐接口的在线替代方案

## 第六周

- 完成 M7
- 补 Go 异步写用户能力图谱与 Python 热更新协同

## 第七周

- 完成 M8
- 补前沿扩展接口、压测、监控、上线准备

---

# 十四、文档对齐建议

为避免后续继续出现“文档目标态”和“代码现状”混淆，建议开发过程中同步遵守以下规则：

- 对教师端接口，明确区分“已落地接口”和“规划接口”。
- 对前沿知识链路，明确区分“本体已定义”“离线已落地”“在线已可查”三个阶段。
- 对知识图谱节点，尽量统一 `KnowledgeNode` 命名；若暂时保留 `KnowledgePoint`，必须在接口层写清主键映射关系。
- 对首页知识目录、项目推荐、图谱跳转 ID 等前端已依赖但后端尚未完全在线化的能力，文档中要明确说明是否仍由前端静态数据兜底。

如果严格按这个顺序推进，Go 后端会先形成一个稳定、可联调、可上线的主闭环，再逐步扩展到图谱增强与前沿知识能力。