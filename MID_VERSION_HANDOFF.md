# 中间版本交付说明

这份项目当前适合作为“中间版本”发给别人评审、联调或接手。

## 建议一起交付的内容

1. `project-src.zip`
2. `data/mysql/course_recommend.sql`
3. `data/neo4j/neo4j.dump`
4. `data/neo4j/graph_summary.txt`
5. `data/redis/dump.rdb`
6. `handoff-manifest.json`

说明：

- `project-src.zip` 是当前工作区的源码快照，不包含 `.env`、日志和临时文件。
- `MySQL` dump 保存业务库数据。
- `Neo4j` dump 是完整图库快照，恢复后最接近你当前本机状态。
- `Redis` dump 主要是运行态缓存和学习状态，严格来说不是必须，但如果你想把演示状态一并交给别人，可以一起发。

## 一键导出

在仓库根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\export_handoff_bundle.ps1
```

默认会在：

```text
artifacts/handoff/
```

下生成一个时间戳目录。

## 发送给别人的最小集合

如果你只想发“能跑起来的中间版本”，最少发下面这些：

1. `project-src.zip`
2. `data/mysql/course_recommend.sql`
3. `data/neo4j/neo4j.dump`
4. `handoff-manifest.json`

## 收件人恢复步骤

### 1. 解压源码

把 `project-src.zip` 解压到任意目录。

### 2. 准备配置

复制：

```text
.env.example -> .env
docker/neo4j/.env.example -> docker/neo4j/.env
```

然后按本机环境补齐配置。  
如果对方直接复用仓库内 Docker 环境，MySQL 推荐使用：

```text
MYSQL_DSN=app:app_dev_pass@tcp(127.0.0.1:3307)/course_recommend?parseTime=true&charset=utf8mb4&loc=Local
```

### 3. 启动基础依赖

```powershell
docker compose -f docker/neo4j/docker-compose.yml --env-file docker/neo4j/.env up -d mysql redis neo4j
```

### 4. 恢复 MySQL

```powershell
Get-Content .\artifacts\handoff\<bundle>\data\mysql\course_recommend.sql | docker exec -i course-recommend-mysql sh -lc "mysql -uapp -papp_dev_pass course_recommend"
```

### 5. 恢复 Neo4j

先停止 Neo4j：

```powershell
docker stop course-recommend-neo4j
```

再导入 dump：

```powershell
docker run --rm --volumes-from course-recommend-neo4j -v "<bundle-data-neo4j-absolute-path>:/backups" neo4j:5.26.24 neo4j-admin database load neo4j --from-path=/backups --overwrite-destination=true
```

再启动 Neo4j：

```powershell
docker start course-recommend-neo4j
```

### 6. 可选恢复 Redis

如果你希望把当前运行态一并交给别人：

```powershell
docker cp .\artifacts\handoff\<bundle>\data\redis\dump.rdb course-recommend-redis:/data/dump.rdb
docker exec course-recommend-redis redis-cli SHUTDOWN NOSAVE
docker start course-recommend-redis
```

### 7. 启动项目

后端：

```powershell
cd backend
go run ./cmd/server
```

前端开发模式：

```powershell
cd frontend
npm install
npm run dev
```

或者直接使用后端托管的前端构建产物。

## 当前中间版本的已知状态

- 后端 `go test ./...` 通过
- 后端 `go build ./cmd/server` 通过
- 前端 `npm run build` 通过
- `healthz`、`readyz`、`/api/ping` 正常
- Neo4j 当前图谱已整理到 `864` 个实体、`1615` 条关系
- `frontier briefing` 仍然是 `not_ready`，不应当作为“已经完成”的功能对外承诺

## 交付前注意

1. 不要把你自己的 `.env` 原样发出去，里面可能有真实 API Key。
2. 如果只是给老师看，优先发 dump 和源码快照，不要只发容器卷。
3. 如果要长期协作，最好在发出前补一个 git commit/tag，方便后续追溯。
