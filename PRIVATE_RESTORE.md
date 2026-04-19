# 私有仓库完整恢复说明

这个仓库当前已经包含：

- 当前代码
- 根目录 `.env`
- `docker/neo4j/.env`
- 已构建好的 `backend/web/dist`
- 一份可恢复的运行快照：
  `handoff/mid_version_20260419_125921`

如果你想把环境恢复到接近当前开发机的状态，按下面步骤执行。

## 1. 拉取仓库

```powershell
git clone <private-repo-url>
cd course-recommend-agents
```

## 2. 直接执行恢复脚本

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1
```

脚本会自动完成这些事情：

- 启动 `mysql` / `redis` / `neo4j`
- 恢复 `MySQL` 快照
- 恢复 `Neo4j` dump
- 恢复 `Redis` dump
- 启动 `backend`

## 3. 打开页面

- 应用主页：`http://127.0.0.1:8080`
- Neo4j Browser：`http://127.0.0.1:7474`

## 4. 如果只想恢复数据库，不启动后端

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1 -SkipBackend
```

## 5. 如果不想恢复 Redis

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1 -SkipRedis
```

## 6. 说明

- 这份仓库是按“私有仓库内部协作”方式提交的，已包含本地环境配置与当前运行快照。
- `handoff/mid_version_20260419_125921/data/neo4j/neo4j.dump` 是当前图谱快照。
- `handoff/mid_version_20260419_125921/data/mysql/course_recommend.sql` 是当前 MySQL 快照。
- `handoff/mid_version_20260419_125921/data/redis/dump.rdb` 是当前 Redis 快照。
IMPORTANT:
- Plain `docker compose -f docker/neo4j/docker-compose.yml up -d --build` only starts the infrastructure containers and does not import any fallback graph.
- For the exact private environment snapshot, always run `powershell -ExecutionPolicy Bypass -File .\scripts\restore_private_snapshot.ps1`.
- The latest exact snapshot committed in this repo is `handoff/mid_version_20260419_125921`.
- Expected restored graph summary: `KnowledgeNode = 1805`, `relations = 4241`.
