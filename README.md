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

### 0. 队友复现同款效果

当前 GitHub 上 `main` 和 `前端修改1` 已同步到同一个提交。队友只需要拉取其中任一分支，即可看到和本机一致的前端构建产物与功能效果。

```bash
git fetch origin
git checkout main
git pull --ff-only origin main
```

如果队友习惯使用前端分支，也可以：

```bash
git fetch origin
git checkout 前端修改1
git pull --ff-only origin 前端修改1
```

推荐统一使用一键脚本启动演示环境：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/start-demo.ps1
```

浏览器统一打开：

```text
http://127.0.0.1:8080
```

比赛展示中心统一打开：

```text
http://127.0.0.1:8080/showcase
```

说明：演示和队友验收都不要打开 `5173`。项目的统一效果以 Go 后端托管的 `backend/web/dist` 为准，入口固定为 `8080`。

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

### 3. 统一入口运行前端页面

```bash
cd frontend
npm install
npm run build
```

构建结果会写入：

```text
backend/web/dist
```

随后由 Go 后端统一托管前端页面。浏览器固定访问：

```text
http://127.0.0.1:8080
```

说明：项目运行入口统一为 `8080`。前端先执行生产构建，Go 后端直接托管 `backend/web/dist`，浏览器不再需要打开 Vite 的 `5173` 端口。

如果只是做前端源码开发，可以临时运行 Vite 开发服务器；但比赛演示、队友验收和最终效果检查仍以 `8080` 为准。

```bash
npm run dev
```

如果你要改 Vite 开发代理目标，可以设置：

```text
VITE_API_PROXY_TARGET=http://localhost:8080
```

### 3.1 比赛演示入口

比赛现场建议只打开一个浏览器入口：

```text
http://127.0.0.1:8080
```

评委展示中心：

```text
http://127.0.0.1:8080/showcase
```

如果需要一键拉起演示环境，可以在项目根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/start-demo.ps1
```

脚本会启动 Docker 依赖，构建前端，并启动 Go 后端作为唯一访问入口，日志写入 `artifacts/logs/`。

推荐演示顺序：

1. 作品展示中心：先按创意、技术、应用、设计、效果说明评分点对齐。
2. 首页：展示课程知识地图、学习导航、当前进度和 AI 助教。
3. 点击一个知识点：展示前置知识、典型场景、练习测验、推荐路径和学习笔记。
4. 进入学习总览：展示总体掌握度、待巩固内容、本周节奏和下一步学习建议。
5. 如需说明后台能力，再进入教研评测页；学生主流程不要从评测页开始。

比赛材料：

- `docs/competition/c4-a-series-score-alignment.md`
- `docs/competition/national-first-prize-checklist.md`

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

## 质量门禁

每次提交前至少执行以下检查，确保后端业务逻辑、前端组件行为和生产构建都处于可交付状态：

```bash
cd backend
go test ./...
```

```bash
cd frontend
npm test -- --run
npm run build
```

当前前端测试环境已经在 `frontend/src/test/setup.js` 中补齐 `ResizeObserver`、`matchMedia`、`requestAnimationFrame` 和 Canvas 2D 上下文 mock，图谱与雷达图相关组件可以在 jsdom 下稳定测试，不应再出现 `HTMLCanvasElement.getContext()` 未实现噪声。

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
- `GET /api/recommend/unified-path`
- `GET /api/evaluation/demo`
- `GET /api/frontier/briefing`
- `POST /api/chat`
- `POST /api/generate-content`

## 推荐算法设计

### KT-KGPath 统一评分公式

系统采用自研的 KT-KGPath（Knowledge-Traced Knowledge Graph Path）统一评分器对候选知识点进行排序，核心公式：

```
Score(k) = α·Weakness + β·PrerequisiteReadiness + γ·TargetRelevance
         + δ·ForgettingRisk + ε·CrossCourseTransfer + θ·FrontierRelevance
         - λ·MasteredPenalty
```

**7 个评分因子：**

| 因子 | 权重 | 说明 |
|------|------|------|
| Weakness（薄弱度） | α=0.30 | 基于掌握度分档 + 正确率 + 答题次数归一化到 [0,1] |
| PrerequisiteReadiness（先修满足度） | β=0.20 | 查询 Neo4j 先修链，平均先修节点掌握度 |
| TargetRelevance（目标相关度） | γ=0.20 | 基于图谱最短路径距离，距离越近越相关 |
| ForgettingRisk（遗忘风险） | δ=0.10 | 指数遗忘模型 `mastery·exp(-days/7)`，掌握度≥60 时激活 |
| CrossCourseTransfer（跨课程迁移） | ε=0.10 | 跨课程关系权重 × 源知识掌握度 |
| FrontierRelevance（前沿关联度） | θ=0.05 | 调用前沿模块查询研究主题和论文数量 |
| MasteredPenalty（已掌握惩罚） | λ=0.05 | 对 mastery≥70 的知识点施加软惩罚，避免推荐已掌握内容 |

**设计原则：**
- 薄弱度和先修满足度权重最高，反映"补短板、打基础"的核心策略
- 遗忘风险基于艾宾浩斯遗忘曲线，识别需要复习的知识点
- 跨课程迁移发现不同课程间的知识关联，促进迁移学习
- 前沿关联度为创新因子，连接课堂知识与前沿研究
- 已掌握惩罚与硬过滤（mastery≥85 且 correctRate≥0.85）互补

### 评测框架

`backend/internal/module/evaluation/` 包含完整的推荐算法评测体系：

- **5 个基线策略**: Random、WeaknessFirst、CourseOrder、PrerequisiteTopo、KTKGPath（可通过评测框架扩展）
- **10 项评测指标**: Hit@1/3/5、NDCG@5/10、MRR、先修违反率、难度平滑度、平均路径长度、覆盖率
- 通过 `/api/evaluation/demo` 接口可在线对比各策略表现

### 比赛评分标准对标

| 评分维度（权重） | 对应实现 |
|-----------------|---------|
| 创意（20%） | KT-KGPath 7因子统一评分公式，跨课程迁移 + 前沿关联为创新因子 |
| 技术（30%） | Go + Gin 模块化后端，Neo4j 知识图谱，Redis 实时学习状态，评测框架 |
| 应用（20%） | 个性化学习路径推荐，薄弱点检测，遗忘曲线提醒 |
| 设计（30%） | Vue 3 可视化仪表盘，AntV G6 知识图谱交互，评分因子分解展示 |
| 效果（20%） | 多基线对比评测，Hit@K/NDCG 量化指标，前端评分分解增强可解释性 |

### 方法对比

KT-KGPath 与现有学习路径推荐方法的关键差异：

| 方法 | 知识追踪 | 图谱约束 | 跨课程迁移 | 前沿关联 | 可解释评分 |
|------|---------|---------|-----------|---------|-----------|
| 传统课程顺序 | ✗ | ✗ | ✗ | ✗ | ✗ |
| 薄弱点优先 | ✗ | ✗ | ✗ | ✗ | 部分 |
| PKT (概率知识追踪) | ✓ | ✗ | ✗ | ✗ | ✗ |
| KG-based 最短路径 | ✗ | ✓ | ✗ | ✗ | 部分 |
| GraphRAG 增强推荐 | ✗ | 部分 | ✗ | ✓ | ✓ |
| **KT-KGPath（本系统）** | **✓** | **✓** | **✓** | **✓** | **✓** |

### 创新点总结

1. **知识追踪融合**：将学生动态掌握状态 (答题正确率、答题次数、掌握度) 融入评分，实现"补短板、打基础"的自适应推荐
2. **跨课程知识图谱**：构建 2,659 个知识点、21 个课程分类的统一图谱，融合 ACM CCS 2012 和 AIO Ontology 两个权威 AI 知识体系，超越单课程推荐的局限
3. **跨课程迁移因子**：发现不同课程间的知识关联（如线性代数对深度学习的支撑），促进迁移学习
4. **前沿关联因子**：将知识点与前沿研究主题关联，连接课堂内容与学术前沿
5. **可解释评分分解**：7 个评分因子可视化展示，学生和教师可清晰理解每个推荐的理由
6. **多基线评测框架**：5 种基线策略 + 10 项量化指标，客观验证推荐算法有效性

### 应用场景

**场景一：学生自主学习**
学生登录后进入知识图谱主页，系统基于薄弱度和先修满足度自动推荐下一步学习知识点。评分分解面板展示每个推荐的详细理由（薄弱环节、前置基础、目标匹配等），学生点击即可进入知识点详情页开始学习。

**场景二：教师教学设计**
教师通过"我的路径"功能创建自定义学习路径，拖拽排序知识点，发布给学生。系统自动计算路径中每个知识点的掌握度状态，教师可据此调整教学计划。

**场景三：前沿研究追踪**
学生完成基础知识点学习后，系统通过前沿关联因子推荐与当前知识相关的最新研究主题和论文，引导学生从课堂学习向科研前沿延伸。

### 用户角色

| 功能 | 学生 | 教师 | 管理员 |
|------|------|------|--------|
| 知识图谱浏览 | ✓ | ✓ | ✓ |
| 个性化推荐路径 | ✓ | - | - |
| 评分因子分解查看 | ✓ | ✓ | - |
| 练习测验 | ✓ | - | - |
| AI 学习助手 | ✓ | - | - |
| 学习仪表盘 | ✓ | ✓ | - |
| 创建学习路径 | - | ✓ | - |
| 班级管理 | - | ✓ | - |
| 评测框架访问 | ✓ | ✓ | ✓ |

## 知识图谱流水线

参考：

- `pipelines/txt2neo4j/README.md`

用于把教材文本抽取为结构化知识点并导入 Neo4j。

### 外部知识库集成

知识图谱融合了两个权威 AI 知识体系，作为课程知识点的补充骨架：

| 来源 | 节点数 | 说明 |
|------|--------|------|
| **ACM CCS 2012** | 82 | 计算机科学分类系统 AI 子树的完整层次结构（8 大分支：NLP、知识表示、规划、搜索、控制方法、哲学基础、分布式 AI、计算机视觉） |
| **AIO Ontology** | 225 | 人工智能本体论（ML 方法、模型、网络架构、激活函数、预处理、训练策略等 6 大子集） |

导入命令：

```bash
cd pipelines/txt2neo4j
python scripts/import_external_knowledge.py
```

## 前沿论文抓取流水线

`pipelines/frontier-scraper/` 提供自动化的前沿论文抓取 + AI 摘要管线：

1. 从 ArXiv API 抓取 AI 各子领域最新论文
2. 调用 DeepSeek 为每篇论文生成 knowledge_hints（课程知识点关联）、topic_hints（前沿研究方向）、中文摘要
3. 输出标准化 JSON，可直接供 `txt2neo4j --frontier-input` 使用

```bash
cd pipelines/frontier-scraper
pip install -r requirements.txt
python main.py --max-per-category 5
```

详细用法参考 `pipelines/frontier-scraper/README.md` 和 `docs/frontier-knowledge-pipeline.md`。

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
