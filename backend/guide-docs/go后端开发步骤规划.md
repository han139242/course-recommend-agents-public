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

- `ErrNotFound`
- `ErrConflict`
- `ErrConstraint`
- `ErrCacheMiss`
- `ErrDependencyUnavailable`

### 5. JWT 认证链路

本阶段先打通：

- `POST /api/auth/login`
- `GET /api/auth/me`

并补齐：

- Bearer Token 提取
- JWT 校验
- 用户状态校验（`active` / `disabled`）
- 用户身份注入上下文

### 6. 限流能力

优先对以下接口做限流：

- 登录
- 验证码类接口（如果后续有）
- 高频查询接口

## 5.3 推荐实现顺序

1. `pkg/result`
2. `pkg/apperr`
3. `pkg/ctxmeta`
4. `middleware/request_id`
5. `middleware/logger`
6. `middleware/recovery`
7. `middleware/auth`
8. `middleware/ratelimit`
9. `module/auth`

## 5.4 本阶段产出物

- 请求级 trace 能力。
- 统一响应封装。
- 统一错误码体系。
- JWT 登录与当前用户接口。
- 中间件框架。

## 5.5 验收标准

- 登录成功可拿到 token。
- 带 token 可访问受保护接口。
- 无 token、过期 token、非法 token 能得到稳定错误响应。
- 所有错误都有统一格式。
- Panic 会被 Recovery 捕获并脱敏输出。

---

# 六、里程碑 M3：教师端基础主数据管理

## 6.1 目标

优先完成系统的“基础数据供给能力”，为后续答题、画像、推荐提供前置数据。

## 6.2 推荐先落地的数据模型

按文档优先建立 MySQL 表及迁移：

- `users`
- `classrooms`
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

- `POST /api/teacher/teachers`
- `POST /api/teacher/classes`
- `GET /api/teacher/classes`
- `POST /api/teacher/classes/:classId/students`
- `GET /api/teacher/classes/:classId/students`

说明：

- 当前阶段教师与管理员同责，统一使用 `teacher` 角色。
- 学生不能自主注册，必须由教师创建或导入。
- 教师只能管理自己创建的班级与班级内学生。

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

- 教师可创建教师账号、班级、学生。
- 列表接口分页稳定。
- 权限边界正确。
- 唯一约束冲突能稳定返回 409。

---

# 七、里程碑 M4：题库与答题主链路

## 7.1 目标

把“题库供给 → 学生做题 → 答题记录落库”主链路打通。

## 7.2 推荐先做的 API

### 教师端题库管理

- `POST /api/teacher/questions/batch`
- `GET /api/teacher/questions`

### 学生端答题主链路

- `GET /api/learning/questions`
- `POST /api/learning/answers`

## 7.3 核心实现要点

### 1. 题库导入

约束遵循文档：

- 当前阶段仅支持单选题。
- `knowledge_id` 先只支持单知识点。
- `question_code` 作为稳定业务 ID。
- 选项表使用 `(question_id, option_key)` 唯一约束。

### 2. 学生答题记录

答题成功时至少需要完成：

- 校验学生身份。
- 校验题目存在且可用。
- 记录答题结果到 `student_answers`。
- 生成知识点答题影响数据。

### 3. 幂等与重复提交策略

当前文档没有要求答题接口强幂等，但建议提前设计：

- 如果前端存在重试，建议使用请求 ID 或幂等键。
- 如果暂不做完整幂等，至少保证重复提交不会导致系统异常。

### 4. 事务边界

以下场景建议由 service 控制事务：

- 答题记录写入 + 统计字段更新。
- 题目批量导入 + 选项批量写入。

## 7.4 本阶段产出物

- 题库导入与查询接口。
- 学生拉题与提交答案接口。
- `student_answers` 数据链路。

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

- 对接的是已被 Python 写入的前沿知识域。
- 允许最终一致，不要求实时强一致。
- 支持“当前内容尚未就绪”的稳定返回语义。

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

## 12.3 第三批智能输出模块

- `graph`
- `competency`
- `recommend`
- `dashboard`

## 12.4 第四批异步与扩展模块

- `graphwriter`
- `frontier`
- `topology subscriber`

这样分层后，开发过程更接近：

- 先做“底座”。
- 再做“业务真相”。
- 再做“智能计算”。
- 最后做“异步协同与体验增强”。

---

# 十三、建议每个模块的标准交付内容

为了避免模块只交一半，建议每个模块都按同一交付模板推进。

每个模块至少交付：

1. `handler`
2. `service`
3. `repository`
4. `dto`
5. `entity/model`
6. 路由注册
7. 单元测试
8. 关键集成测试
9. 错误码
10. 日志字段约定

例如 `auth` 模块至少应交付：

- 登录请求 DTO
- 当前用户响应 DTO
- 用户查询 repository
- JWT service
- 登录 handler
- `RegisterAuthRoutes`
- 登录单元测试
- token 校验测试

---

# 十四、推荐先建的 migration 顺序

建议 migration 按以下顺序建立：

1. `users`
2. `classrooms`
3. `questions`
4. `question_options`
5. `student_answers`
6. `learning_progress`
7. `competency_snapshots`

原因：

- 先有用户与班级，认证和教师端才有基础。
- 先有题库，答题链路才成立。
- 学习进度与能力快照应建立在答题链路之后。

不建议一开始就把所有“二期可能用到”的表全部建出来。

---

# 十五、测试推进计划

## 15.1 测试金字塔建议

- 单元测试：70% ~ 80%
- 集成测试：15% ~ 25%
- e2e / 冒烟测试：5%

## 15.2 各阶段测试重点

### M1-M2

- 配置校验测试
- 中间件测试
- 错误响应映射测试
- JWT 测试

### M3-M4

- 教师权限边界测试
- 班级作用域测试
- 题库批量导入测试
- 学生答题事务测试

### M5-M6

- Redis 掌握度更新测试
- 学习进度回写测试
- 能力矩阵计算测试
- 推荐路径算法测试

### M7-M8

- 异步写图谱测试
- 拓扑热更新测试
- 停机等待 worker 测试
- 冒烟测试与压测脚本

## 15.3 特别强调

这个项目的高风险点不在“页面能不能显示”，而在：

- 事务边界是否正确。
- Redis 与 MySQL 是否一致。
- Neo4j 查询与写入边界是否混乱。
- 异步链路在失败时是否可恢复。

因此必须把这些作为重点测试对象。

---

# 十六、每个阶段的完成定义（Definition of Done）

为了避免“功能写了但不能交付”，建议每个阶段都满足以下完成定义。

## 通用完成定义

- 代码已按目录规范落位。
- 配置项已补齐。
- 错误码已补齐。
- 日志字段符合统一约定。
- 至少有最小单元测试。
- 关键链路有集成测试。
- 无明显新增 lint / 编译错误。
- 接口文档已同步更新。

## 能力型阶段额外要求

对于推荐、画像、图谱类功能，额外要求：

- 输入输出 DTO 明确。
- 值域约束明确。
- 性能目标明确。
- 降级策略明确。
- 最终一致语义明确。

---

# 十七、推荐排期视角

如果按实际开发节奏推进，建议可以这么排：

## 第一周

- 完成 M1
- 完成 M2

## 第二周

- 完成 M3
- 完成 M4

## 第三周

- 完成 M5
- 完成 M6 的 Graph / Competency 基础版

## 第四周

- 完成 M6 的 Recommend / Dashboard
- 完成 M7
- 启动 M8

## 第五周

- 完成 M8
- 压测
- 联调
- 上线准备

如果人力不足，也建议遵守一个原则：

**先把 M1-M5 做扎实，再考虑 M6-M8 的高级能力。**

因为前五个里程碑决定的是系统能不能稳定工作，后面三个里程碑决定的是系统好不好用、聪不聪明。

---

# 十八、明确哪些经验应该继承，哪些不该照搬

## 18.1 应该继承的经验

- 统一上下文元信息模型。
- 统一结构化日志。
- 统一错误码与统一响应。
- 手动依赖注入。
- repository 层错误翻译。
- Redis key 与 TTL 集中治理。
- 优雅停机。
- 以 service 为核心测试主战场。

## 18.2 不该照搬的微服务遗产

- 代理式 service。
- 过细的接口层拆分。
- protobuf 转换链。
- 复杂熔断、补偿队列、异步编排框架。
- 为未来可能拆服务而提前做大量抽象。
- 所有业务错误统一 HTTP 200 的旧习惯。

---

# 十九、最终推荐执行方案

如果现在立刻开始干，我建议严格按下面顺序执行：

## 第 1 步

完成工程骨架、配置、日志、基础设施初始化、优雅停机。

## 第 2 步

完成统一响应、统一错误模型、中间件、认证。

## 第 3 步

完成教师端教师/班级/学生管理。

## 第 4 步

完成题库导入、学生答题、答题记录落库。

## 第 5 步

完成 Redis 掌握度向量与 MySQL 学习进度快照。

## 第 6 步

完成 Neo4j 只读图查询、能力画像、推荐路径。

## 第 7 步

完成 Go 异步写用户能力图谱、Redis Pub/Sub 拓扑热更新协同。

## 第 8 步

完成仪表盘聚合、前沿扩展、可观测性、压测与上线。

---

# 二十、文档结论

本项目的 Go 后端开发，最重要的不是一次性把所有接口写完，而是按正确顺序逐步搭建：

- **先底座，后业务。**
- **先主真相，后缓存。**
- **先同步主链路，后异步副作用。**
- **先稳定可维护，后智能可扩展。**

只要按这份顺序推进，既能遵守现有设计文档和 API 契约，也能最大程度避免后期因架构边界不清、缓存与图谱职责混乱而返工。

如果下一步要继续落地，最合适的后续动作是：

1. 先创建 Go 服务目录骨架。
2. 再补第一版 migration。
3. 然后从 `auth + teacher + learning` 三个核心模块开始实现。
