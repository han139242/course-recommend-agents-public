# 课程推荐智能体

面向人工智能专业课程学习场景的知识图谱推荐系统，目标是打通“课内知识图谱 + 个性化学习路径 + 课后前沿扩展”。

## 当前系统组成

- `frontend/`：Vue 3 前端，提供知识学习首页、知识图谱、知识点详情、成长分析、登录注册等页面
- `backend/`：Node/Express 在线 API，负责鉴权、MySQL、Neo4j 只读查询、首页摘要、成长画像与推荐路径
- `db/`：MySQL 初始化 SQL
- `pipelines/txt2neo4j/`：教材/课程资料到 Neo4j 的图谱抽取与导入管线
- `docs/`：在线 API 契约、前沿知识更新设计等文档

## 当前已打通的在线闭环

- 登录 / 注册 / 个人档案
- 从 Neo4j 读取真实知识图谱子图
- 知识点详情页读取真实先修/关联/应用场景
- 首页与成长分析页读取真实在线摘要接口
- 推荐路径与课后扩展入口的统一 API 结构

## 本地启动

### 1. 环境变量

复制 `.env.example` 为 `.env`，填写 MySQL 与 Neo4j 连接信息。

### 2. 启动后端

```bash
cd backend
npm install
npm run dev
```

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

### 4. 初始化图谱

参考 `pipelines/txt2neo4j/README.md` 使用 `txt2neo4j` 将课内知识导入 Neo4j。

## 关键接口

- `GET /api/health`
- `GET /api/dashboard/summary`
- `GET /api/graph/subgraph`
- `GET /api/knowledge/:id`
- `GET /api/competency/profile`
- `GET /api/recommend/path`
- `GET /api/frontier/briefing`

完整契约见：

- `docs/online-api-contract.md`
- `docs/frontier-knowledge-pipeline.md`

## 演示建议

- 先登录一个测试账号，确保 MySQL 中有用户记录
- 在 Neo4j 中提前导入 curated 知识图谱批次
- 首页 -> 图谱 -> 知识点详情 -> 成长分析 -> 推荐路径，构成一条完整答辩演示链路