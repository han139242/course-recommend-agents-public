# 突破课程边界的人工智能专业知识点推荐智能体系统

面向人工智能专业课程学习场景的知识点推荐智能体系统，当前已整合为：

- `frontend/`: Vue 3 + Vite 前端
- `backend/`: Go + Gin 在线 API 与统一入口
- `pipelines/txt2neo4j/`: 教材文本到 Neo4j 的知识图谱构建流水线
- `docker/neo4j/`: MySQL、Redis、Neo4j、后端的联调环境

## 当前整合状态

- 前端业务 API 与 AI 对话/内容生成统一走 Go 后端的 `/api/*`
- 前端生产构建产物默认输出到 `backend/web/dist`
- Go 后端在检测到 `backend/web/dist` 存在时，会直接托管前端页面
- 知识图谱页、知识点详情页、成长分析页优先使用 Go 后端真实数据
- 前端本地静态课程数据仍保留为降级兜底，便于后端未就绪时演示

## 本地开发

### 1. 准备环境变量

复制 `.env.example` 为 `.env`，补齐：

- `JWT_SECRET`
- `MYSQL_DSN`
- `REDIS_ADDR`
- `NEO4J_URI`
- `NEO4J_USERNAME`
- `NEO4J_PASSWORD`
- 可选 AI 代理配置：`OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL`，或兼容的 `DEEPSEEK_API_KEY`、`DEEPSEEK_BASE_URL`、`DEEPSEEK_MODEL`
- AI 密钥只放后端 `.env`，不要放前端，也不能直接复用当前 Codex/助手会话的内部密钥

如果你直接复用仓库内 `docker/neo4j` 启动的 MySQL，推荐使用：

```text
MYSQL_DSN=app:app_dev_pass@tcp(127.0.0.1:3307)/course_recommend?parseTime=true&charset=utf8mb4&loc=Local
```

### 2. 启动 Go 后端

后端需要本机可用的 Go 环境。

```bash
cd backend
go run ./cmd/server
```

默认监听：

```text
http://localhost:8080
```

健康检查：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

### 3. 启动前端开发服务

```bash
cd frontend
npm install
npm run dev
```

开发模式下，Vite 会把 `/api/*` 统一代理到 Go 后端，默认目标是：

```text
http://localhost:8080
```

如果你要改代理目标，可以设置：

```text
VITE_API_PROXY_TARGET=http://localhost:8080
```

### 4. 构建前端并交给 Go 托管

```bash
cd frontend
npm run build
```

构建结果会输出到：

```text
backend/web/dist
```

随后只启动 Go 后端，也可以直接访问前端页面。

## Docker 联调

仓库提供了包含 MySQL、Redis、Neo4j、Neo4j 种子导入、Go 后端的联调方案：

```bash
docker compose -f docker/neo4j/docker-compose.yml up -d --build
```

默认后端地址：

```text
http://localhost:8080
```

说明：

- Neo4j 默认账号：`neo4j`
- Neo4j 默认密码：`please-change-me`
- Docker 场景下如果希望 Go 后端直接托管前端，请先执行一次 `frontend` 构建，让 `backend/web/dist` 存在

## 主要接口

- `GET /healthz`
- `GET /readyz`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `GET /api/dashboard/summary`
- `GET /api/graph/subgraph`
- `GET /api/knowledge/:knowledgeId`
- `GET /api/competency/profile`
- `GET /api/recommend/path`
- `GET /api/frontier/briefing`
- `POST /api/chat`
- `POST /api/generate-content`

## 知识图谱流水线

参考：

- `pipelines/txt2neo4j/README.md`

用于把教材文本抽取为结构化知识点并导入 Neo4j。

## 中间版本交付

如果你要把当前项目作为“中间版本”完整发给别人，尤其包含 MySQL / Neo4j 数据快照，参考：

- `MID_VERSION_HANDOFF.md`
- `scripts/export_handoff_bundle.ps1`

## 私有仓库完整恢复

如果你需要把另一个人的环境恢复到当前这份私有仓库对应的运行状态，请直接参考：

- `PRIVATE_RESTORE.md`
- `scripts/restore_private_snapshot.ps1`

IMPORTANT:
- Plain `docker compose up` only starts the infrastructure containers and does not import any graph snapshot.
- For the exact private environment snapshot, including the full Neo4j graph, run `powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1`.

Exact restore note:
- `docker compose -f docker/neo4j/docker-compose.yml up -d --build` now starts infrastructure only and does not import any fallback Neo4j graph.
- The historical smaller bootstrap seed graph has been removed from this repo.
- To reproduce the same graph as the current private environment, always use `powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1`.
