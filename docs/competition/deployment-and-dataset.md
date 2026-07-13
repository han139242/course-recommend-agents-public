# 运行说明与数据集说明

> 中国软件杯 A3 赛题提交材料。配套：`a3-architecture.md`（架构）、`a3-implemented-features-summary.md`（功能）、`ai-tools-and-licenses.md`（工具与协议）。

## 一、技术栈与端口

| 层 | 技术 | 默认端口 |
| --- | --- | --- |
| 后端 | Go + Gin | 8080 |
| 前端 | Vue 3 + Vite | 5173（dev，代理 /api 到 8080）|
| 关系库 | MySQL | 3306 |
| 缓存 | Redis | 6379 |
| 知识图谱 | Neo4j | 7474 / 7687 |
| 大模型 | DeepSeek / OpenAI 兼容（可选）| — |

生产模式下前端构建产物输出到 `backend/web/dist`，由 Go 服务统一托管，单端口 8080 即可访问完整系统。

## 二、环境变量（`.env`，项目根目录）

参考 `.env.example`，关键变量：

| 变量 | 说明 | 必填 |
| --- | --- | --- |
| `MYSQL_DSN` | MySQL 连接串 | 是 |
| `REDIS_ADDR` | Redis 地址 | 是 |
| `NEO4J_URI` / `NEO4J_USER` / `NEO4J_PASSWORD` | Neo4j 连接 | 是 |
| `JWT_SECRET` | 登录令牌密钥 | 是 |
| `DEEPSEEK_API_KEY` / `BaseURL` / `ChatModel` / `VisionModel` | 主大模型 | 否（未配置时画像/资源/审查走本地模板兜底）|
| `FallbackBaseURL` / `FallbackChatModel` / `FallbackVisionModel` | 备用大模型 | 否 |
| `OPENAI_*_MODEL` / `OPENAI_*_ENDPOINT` | 真实图片、视频生成模型及创建任务端点 | 否 |
| `OPENAI_IMAGE_STATUS_ENDPOINT` / `OPENAI_VIDEO_STATUS_ENDPOINT` | 异步媒体状态查询端点，支持 `{id}` 任务号占位符 | 异步媒体模型必填 |

> 未配置大模型时系统仍可完整演示：画像抽取、资源生成、安全审查、练习出题均有本地模板兜底。

## 三、快速启动

### 方式 A：Docker 全栈（推荐）

```bash
docker compose -f docker/neo4j/docker-compose.yml up -d --build
# 启动 MySQL、Redis、Neo4j（含种子数据）、Go 后端
# 健康检查：GET /healthz、GET /readyz
```

`docker-compose.yml` 已按 `000001`～`000011` 顺序初始化全部业务表；`scripts/start-demo.ps1` 还会调用 `scripts/apply-migrations.ps1`，为已有 MySQL 数据卷补齐缺失的 A3 迁移。若 8080 被占用，可使用 `-HttpPort 18080` 指定演示端口。

### 方式 B：本地分别启动

```bash
# 1) 准备 .env，并启动 MySQL / Redis / Neo4j

# 2) 后端（数据库迁移由 Docker 初始化脚本或手动按序执行 backend/migrations/*.sql）
cd backend
go run ./cmd/server           # 默认 :8080

# 3) 前端（开发模式，热更新，代理 /api 到 8080）
cd frontend
npm install
npm run dev                   # 默认 :5173

# 生产构建（产物输出到 backend/web/dist，由后端托管）
npm run build
```

### 健康检查与登录

- `GET /healthz`、`GET /readyz` 验证服务与依赖可用。
- 首次使用可通过 `/register` 自助注册学生账号并自动建立登录会话；公开注册固定创建 `student` 角色，不允许提升为教师或管理员。

## 四、构建与测试

```bash
# 后端
cd backend
go test ./...                 # 全部单元测试
go build ./cmd/server         # 编译二进制

# 前端
cd frontend
npm run build                 # 生产构建（含编译期校验）
```

## 五、A3 演示主线路径

1. 登录后进入 `/a3-agent`（首页「学习智能体」入口）。
2. 输入一段自然语言学习情况 → 生成并确认六维画像。
3. 一键生成三阶段学习计划与带推荐依据的下一步知识点。
4. 选知识点（可上传图片）→ 生成个性化资源包（SSE 实时进度 + 多智能体步骤）。
5. 查看资源依据来源与安全审查；被判定为 `blocked` 的资源不会进入推送流；可语音朗读；一键导出 PPT 大纲 / 演示脚本 / Markdown。
6. 图片/视频模型若返回异步任务号，前端自动轮询后端，后端向模型提供商查询状态并把最终 URL/Base64 结果回写资源库。
7. 做随堂练习 → 回写画像易错点与真实掌握度 → 提交反馈 → 学习效果评估更新 → 按最新掌握度重新规划。

## 六、数据集说明

### 1. 知识图谱数据（Neo4j）

- 仓库内含 Neo4j 快照（约 2780 个知识点节点，见提交历史 `data: add Neo4j snapshot with 2780 nodes`），Docker 启动时作为种子数据导入。
- 数据内容：人工智能相关课程的知识点、先修关系、跨课程关联等。
- 生产/扩充来源：`pipelines/txt2neo4j/`（Python 管线）从教材文本抽取结构化知识点并导入 Neo4j。

### 2. 关系库数据（MySQL）

- 通过 `backend/migrations/`（按 `000001`…顺序编号）建表，Docker 初始化脚本自动执行：
  - 用户与班级、题库与选项、学生答题、学习进度
  - 学习画像（`student_learning_profiles` 等）
  - 资源生成任务/智能体步骤/生成资源、资源反馈
  - 学习事件与资源推送（`learning_events`、`resource_push_items`）
- 题库支持教师批量导入单选题；AI 随堂练习则由大模型/模板即时生成，作答经 `/learning/practice` 写入真实掌握度。

### 3. 缓存数据（Redis）

- 掌握度向量与学习状态快照、运行期缓存与发布订阅。

### 4. 隐私与合规

- 大模型密钥仅存于后端 `.env`，前端不持有。
- 画像、反馈、学习事件仅在登录用户范围内查询；上传图片仅用于当次生成，不长期留存第三方。
