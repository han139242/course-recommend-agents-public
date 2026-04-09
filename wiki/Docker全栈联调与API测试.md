# Docker 全栈联调与 API 测试说明

本文档说明如何通过 `docker compose` 一键启动 **MySQL、Redis、Neo4j、后端 API**，以及如何用种子账号与示例数据逐项验证当前已实现的 HTTP 接口。

## 1. 启动方式

在仓库根目录执行：

```bash
docker compose -f docker/neo4j/docker-compose.yml --env-file docker/neo4j/.env.example up -d --build
```

首次使用可复制环境模板：

```bash
copy docker\neo4j\.env.example docker\neo4j\.env
```

按需编辑 `docker/neo4j/.env`（尤其是 `JWT_SECRET`、`NEO4J_PASSWORD`、`MYSQL_*`）。

### 1.1 服务与端口（默认）

| 服务 | 容器名 | 宿主机端口 |
|------|--------|------------|
| 后端 API | course-recommend-backend | 8080 |
| MySQL | course-recommend-mysql | 3306 |
| Redis | course-recommend-redis | 6379 |
| Neo4j Browser / Bolt | course-recommend-neo4j | 7474 / 7687 |

### 1.2 数据库初始化

- **MySQL**：首次启动时自动执行 `backend/migrations` 下 `000001`～`000006` 迁移，并执行 `docker/neo4j/mysql/init/000007_seed_test_data.sql` 写入联调种子数据。
- **Neo4j**：`neo4j-import` 服务在 Neo4j 就绪后执行 `db/neo4j/bootstrap/knowledge_graph_seed.cypher`。该文件较大，**导入可能持续较长时间**；后端**不阻塞**等待该任务结束即可启动，图谱相关接口在数据尚未导入完成时可能返回空图或 `not_ready` 状态。

### 1.3 健康检查

```bash
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
```

`readyz` 会汇总 MySQL、Redis、Neo4j、GraphWriter、Graph 等子项状态。

### 1.4 重置数据

需要彻底重建数据库与卷时：

```bash
docker compose -f docker/neo4j/docker-compose.yml down -v
```

然后重新 `up`。

---

## 2. 种子账号（MySQL）

| 角色 | 邮箱 | 密码 | 说明 |
|------|------|------|------|
| 教师 | teacher@test.local | TestPass123! | `users.id = 1` |
| 学生 | student@test.local | TestPass123! | `users.id = 2`，班级 `classrooms.id = 1` |

密码为 bcrypt 哈希（与 `000007_seed_test_data.sql` 一致）。

---

## 3. 逐项 API 测试（curl 示例）

以下假设后端在 `http://localhost:8080`。统一响应封装为 `code` / `message` / `data`（见 `pkg/result`）。

### 3.1 无需登录

| 说明 | 方法 | 路径 |
|------|------|------|
| 存活 | GET | `/healthz` |
| 就绪 | GET | `/readyz` |
| Ping | GET | `/api/ping` |
| 登录 | POST | `/api/auth/login` |

登录示例：

```bash
curl -s -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d "{\"email\":\"teacher@test.local\",\"password\":\"TestPass123!\"}"
```

从返回 JSON 中取出 `data.accessToken` 作为 `ACCESS_TOKEN`（Bearer）。

### 3.2 图谱与知识点（公开，无需 Token）

| 说明 | 方法 | 路径 | 示例 |
|------|------|------|------|
| 子图 | GET | `/api/graph/subgraph` | `?root=002edb92a1990f62&limit=12` |
| 知识点详情 | GET | `/api/knowledge/:knowledgeId` | `/api/knowledge/002edb92a1990f62` |

```bash
curl -s "http://localhost:8080/api/graph/subgraph?root=002edb92a1990f62&limit=12"
curl -s "http://localhost:8080/api/knowledge/002edb92a1990f62"
```

### 3.3 前沿扩展（公开，无需 Token）

| 说明 | 方法 | 路径 |
|------|------|------|
| 摘要 | GET | `/api/frontier/briefing` |

```bash
curl -s "http://localhost:8080/api/frontier/briefing?knowledgeId=002edb92a1990f62"
```

若图谱中尚无对应前沿离线数据，返回 `status: not_ready` 属预期行为。

---

### 3.4 需登录（教师 Token）

将 `ACCESS_TOKEN` 替换为教师登录返回的 `accessToken`。

| 说明 | 方法 | 路径 |
|------|------|------|
| 当前用户 | GET | `/api/auth/me` |
| 创建班级 | POST | `/api/teacher/classes` |
| 班级列表 | GET | `/api/teacher/classes` |
| 批量导入学生 | POST | `/api/teacher/classes/:classId/students` |
| 班级学生列表 | GET | `/api/teacher/classes/:classId/students` |
| 学生详情 | GET | `/api/teacher/students/:studentId` |
| 学生知识状态 | GET | `/api/teacher/students/:studentId/knowledge-status` |
| 批量导题 | POST | `/api/teacher/questions/batch` |
| 题目列表 | GET | `/api/teacher/questions` |
| 题目详情 | GET | `/api/teacher/questions/:questionId` |

示例：

```bash
curl -s http://localhost:8080/api/auth/me -H "Authorization: Bearer ACCESS_TOKEN"
curl -s "http://localhost:8080/api/teacher/classes?page=1&pageSize=20" -H "Authorization: Bearer ACCESS_TOKEN"
curl -s "http://localhost:8080/api/teacher/questions?page=1&pageSize=20" -H "Authorization: Bearer ACCESS_TOKEN"
curl -s "http://localhost:8080/api/teacher/questions/1" -H "Authorization: Bearer ACCESS_TOKEN"
```

---

### 3.5 需登录（学生 Token）

使用 `student@test.local` 登录获取 Token。

| 说明 | 方法 | 路径 |
|------|------|------|
| 当前用户 | GET | `/api/auth/me` |
| 学生拉题 | GET | `/api/learning/questions` |
| 提交答案 | POST | `/api/learning/answers` |
| 学习进度 | GET | `/api/learning/progress` |
| 能力画像 | GET | `/api/competency/profile` |
| 推荐路径 | GET | `/api/recommend/path` |
| 仪表盘摘要 | GET | `/api/dashboard/summary` |

答题示例（种子题目 `Q-DEMO-001`，知识点 `002edb92a1990f62`）：

```bash
curl -s -X POST http://localhost:8080/api/learning/answers \
  -H "Authorization: Bearer STUDENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"questionId\":\"Q-DEMO-001\",\"knowledgeId\":\"002edb92a1990f62\",\"selectedOption\":\"A\",\"answeredAt\":\"2026-04-09T12:00:00Z\"}"
```

```bash
curl -s "http://localhost:8080/api/recommend/path?limit=5&scene=study" -H "Authorization: Bearer STUDENT_TOKEN"
curl -s http://localhost:8080/api/dashboard/summary -H "Authorization: Bearer STUDENT_TOKEN"
curl -s http://localhost:8080/api/competency/profile -H "Authorization: Bearer STUDENT_TOKEN"
```

---

## 4. 自动化验证（本仓库执行记录）

- **Go 单元测试**：在 `backend` 目录执行 `go test ./... -count=1`，当前全部通过。
- **Docker 镜像**：`backend/Dockerfile` 多阶段构建（运行阶段为 **Alpine**，避免 Debian `apt` 在部分网络环境下失败）；在仓库根目录执行  
  `docker compose -f docker/neo4j/docker-compose.yml build backend`。

联调时若修改了 MySQL 密码，请同步修改 `docker-compose.yml` 中 `MYSQL_DSN` 的 `${MYSQL_PASSWORD}` 与 `.env` 中 `MYSQL_PASSWORD`。

---

## 5. 一键 curl 全量探测（PowerShell）

仓库提供脚本与配套 JSON 请求体（避免 Windows PowerShell 对 JSON 引号与 UTF-8 响应处理踩坑）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run_api_curl_tests.ps1
```

可选环境变量：`API_BASE`（默认 `http://127.0.0.1:8080`）。

| 文件 | 说明 |
|------|------|
| `scripts/run_api_curl_tests.ps1` | 顺序请求全部已挂载路由并打印 `HTTP_CODE` |
| `scripts/login_teacher.json` / `login_student.json` | 登录体 |
| `scripts/post_class.json` | 创建班级 |
| `scripts/post_batch_students.json` | 批量创建学生 |
| `scripts/post_import_questions.json` | 批量导题 |
| `scripts/post_answer.json` | 学生提交答案 |

**注意**：`POST /api/auth/login` 在 PowerShell 中请使用 `curl.exe -d @文件.json`，勿用内联 JSON 字符串，否则易出现 `参数错误`（10001）。

**教师单题详情**路径参数为题目业务编号 `question_code`（如 `Q-DEMO-001`），不是数据库自增 `id`。

---

## 6. 全接口 curl 实测结果（2026-04-09）

在 **MySQL + Redis + Neo4j + 后端** 联调环境下执行 `scripts/run_api_curl_tests.ps1`，结果如下（Neo4j 未导入图谱种子时，子图为空、知识点详情为 404 属预期）。

| 接口 | 方法 | 路径 | HTTP |
|------|------|------|------|
| 存活 | GET | `/healthz` | 200 |
| 就绪 | GET | `/readyz` | 200 |
| Ping | GET | `/api/ping` | 200 |
| 登录（教师/学生） | POST | `/api/auth/login` | 200 |
| 当前用户 | GET | `/api/auth/me` | 200 |
| 创建班级 | POST | `/api/teacher/classes` | 200 |
| 班级列表 | GET | `/api/teacher/classes` | 200 |
| 批量创建学生 | POST | `/api/teacher/classes/:classId/students` | 200 |
| 班级学生列表 | GET | `/api/teacher/classes/:classId/students` | 200 |
| 学生详情 | GET | `/api/teacher/students/:studentId` | 200 |
| 学生知识状态 | GET | `/api/teacher/students/:studentId/knowledge-status` | 200 |
| 批量导题 | POST | `/api/teacher/questions/batch` | 200 |
| 题目列表 | GET | `/api/teacher/questions` | 200 |
| 题目详情 | GET | `/api/teacher/questions/:questionId` | 200 |
| 学生拉题 | GET | `/api/learning/questions` | 200 |
| 提交答案 | POST | `/api/learning/answers` | 200 |
| 学习进度 | GET | `/api/learning/progress` | 200 |
| 能力画像 | GET | `/api/competency/profile` | 200 |
| 推荐路径 | GET | `/api/recommend/path` | 200 |
| 仪表盘摘要 | GET | `/api/dashboard/summary` | 200 |
| 前沿摘要 | GET | `/api/frontier/briefing` | 200 |
| 子图 | GET | `/api/graph/subgraph` | 200 |
| 知识点详情 | GET | `/api/knowledge/:knowledgeId` | 404（图中无该节点时） |

未默认开启 **`/debug/pprof`**（需设置 `ENABLE_PPROF=true` 且 `APP_ENV` 非 `production`），本次未测。

---

## 7. 相关文件

| 文件 | 作用 |
|------|------|
| `docker/neo4j/docker-compose.yml` | 全栈编排 |
| `docker/neo4j/.env.example` | 环境变量模板 |
| `docker/neo4j/mysql/init/000007_seed_test_data.sql` | MySQL 联调种子 |
| `backend/Dockerfile` | 后端镜像构建 |
| `backend/migrations/*.sql` | 表结构迁移 |
| `scripts/run_api_curl_tests.ps1` | 全接口 curl 探测脚本 |
