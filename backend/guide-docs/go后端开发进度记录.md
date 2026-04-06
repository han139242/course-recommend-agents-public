# Go 后端开发进度记录

本文用于记录 Go 后端按《`go后端开发步骤规划.md`》推进时的实际落地进度。要求是每完成一步，就把已完成内容记录在这里，便于回溯。

---

# 当前执行阶段

- 当前里程碑：`M2 认证、统一响应、日志、错误、健康检查增强`
- 当前目标：补齐统一响应结构、应用错误模型、trace_id/request_id 注入中间件、请求日志中间件与恢复中间件，夯实后续认证与业务模块的公共底座。

---

# M1 拆解步骤

## 步骤 1：初始化 Go 模块与基础依赖

- 状态：已完成
- 目标：创建 `go.mod`，引入 HTTP、MySQL、Redis、Neo4j 等基础依赖。
- 已完成：
  - 创建 `go.mod`，模块名为 `course-recommend-agents/backend`。
  - 引入基础依赖：`gin`、`gorm`、`mysql driver`、`go-redis/v9`、`neo4j-go-driver/v5`、`jwt/v5`。
  - 创建 M1 所需的目录骨架：`cmd`、`internal`、`pkg`、`migrations`、`test` 等。

## 步骤 2：建立配置模型

- 状态：已完成
- 目标：建立 `AppConfig`、`HTTPConfig`、`MySQLConfig`、`RedisConfig`、`Neo4jConfig`、`AuthConfig`、`LogConfig`。
- 已完成：
  - 建立 `internal/config` 配置结构体。
  - 实现 `config.Load()`，支持从环境变量加载配置。
  - 补充开发环境默认值。
  - 增加基础配置校验逻辑。

## 步骤 3：完成基础设施初始化与应用装配

- 状态：已完成
- 目标：补齐 `bootstrap` 层，完成配置加载、日志初始化、MySQL、Redis、Neo4j、路由、HTTP Server、关闭钩子装配。
- 已完成：
  - 建立 `internal/bootstrap` 目录。
  - 实现日志初始化。
  - 实现 MySQL、Redis、Neo4j 初始化入口。
  - Redis、Neo4j 在缺失配置或连通失败时支持降级启动。
  - 建立 `App` 结构，统一聚合运行期依赖。
  - 建立关闭任务注册与统一关闭流程。

## 步骤 4：提供健康检查与启动入口

- 状态：已完成
- 目标：补齐 `cmd/server/main.go`、`/healthz`、`/readyz` 与优雅停机流程。
- 已完成：
  - 建立 `internal/router/router.go`。
  - 提供 `/healthz`、`/readyz`、`/api/ping` 最小接口。
  - 建立 `cmd/server/main.go` 启动入口。
  - 补齐信号监听与优雅停机流程。
  - 服务已具备最小可运行骨架。

---

# 变更记录

## 2026-04-04

- 初始化进度记录文档。
- 确认 `backend` 目录当前尚未有 Go 工程代码，适合从 M1 全量开始搭建。
- 根据仓库远程地址，统一将 Go module 与内部 import 路径调整为 GitHub 格式：`github.com/han139242/course-recommend-agents/backend/...`。
- 日志底层实现从 `slog` 切换为 `zap`，但对外接口保持统一，继续使用 `logger.WithCtx(ctx).Info(...)` 形式。
- 日志字段构造方式统一收口到日志包，继续隐藏底层实现细节，便于后续替换实现而不影响业务层。
- 建立统一响应结构 `pkg/result`，统一返回 `code`、`message`、`data`、`trace_id`、`timestamp`。
- 建立统一应用错误模型 `pkg/apperr`，为后续 handler 与 service 层错误映射提供稳定边界。
- 建立 `RequestID`、`RealIP`、请求日志、恢复中间件，并接入 Gin 全局路由链。
- 请求进入后会由后端自动生成 `request_id` 与 `trace_id`，不再依赖前端透传，也不再通过响应头回写给客户端。
- 前端如果需要链路追踪标识，统一从 JSON 响应体中的 `trace_id` 字段获取。
- 路由层改为使用统一响应封装，基础接口已支持走公共中间件链与 panic 恢复链路。
