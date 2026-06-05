# Neo4j 图谱快照：2780 节点版

本目录保存当前 Docker 容器 `course-recommend-neo4j` 中 Neo4j 数据库的图谱快照。

## 快照内容

- 节点数：2780
- 关系数：5376
- 数据库：`neo4j`
- 导出文件：`data/neo4j/neo4j.dump`
- 统计文件：
  - `data/neo4j/graph_summary.txt`
  - `data/neo4j/label_summary.txt`
  - `data/neo4j/relationship_summary.txt`

## 恢复方式

恢复前需要先停止 Neo4j 容器：

```powershell
docker stop course-recommend-neo4j
```

然后使用 Neo4j 镜像加载 dump：

```powershell
docker run --rm --volumes-from course-recommend-neo4j -v "$PWD/handoff/neo4j_snapshot_20260605_2780_nodes/data/neo4j:/backups" neo4j:5.26.24 neo4j-admin database load neo4j --from-path=/backups --overwrite-destination=true
```

加载完成后重新启动容器：

```powershell
docker start course-recommend-neo4j
```

验证节点与关系数量：

```powershell
docker exec course-recommend-neo4j cypher-shell -u neo4j -p 123456789 "MATCH (n) RETURN count(n) AS nodes; MATCH ()-[r]->() RETURN count(r) AS relationships;"
```
