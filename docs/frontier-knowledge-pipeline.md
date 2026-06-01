# 前沿知识更新链路

本项目已经形成一条可执行闭环：

1. `pipelines/frontier-scraper` 从公开源同步 AI 前沿论文和开源线索。
2. 输出标准 `frontier_seed_metadata.json`。
3. 同步器先用课程知识点目录做候选召回，再用 DeepSeek/OpenAI 在候选集合内结构化选择相关知识点和主题。
4. `pipelines/txt2neo4j` 把论文元数据映射到课程知识点，生成 `Paper / ResearchTopic / FrontierDigest`。
5. 经人工确认后，可导入 Neo4j。
6. Go 后端 `GET /api/frontier/briefing`、`GET /api/frontier/library`、知识点详情页和前沿资料页读取这些前沿内容。

## 可靠数据源策略

主源优先级：

- arXiv 官方 API：稳定、无密钥、适合按 `cs.AI / cs.LG / cs.CL / cs.CV / stat.ML` 等分类拉取最新论文。
- Hugging Face Daily Papers：可选趋势源，用于补充社区热度。
- GitHub 开源观察源：可选线索源，用于跟踪持续更新的 AI 论文索引、工具和数据项目。

GitHub 观察源不会替代正式论文源。它更适合提示“哪些方向正在活跃更新”，进入图谱前仍会按统一字段映射和去重。

## 标准输入格式

示例文件：

`pipelines/txt2neo4j/config/frontier_seed_metadata.json`

核心字段：

- `source`
- `generatedAt`
- `sources`
- `papers[].title`
- `papers[].abstract`
- `papers[].keywords`
- `papers[].year`
- `papers[].venue`
- `papers[].venueType`
- `papers[].url`
- `papers[].authors`
- `papers[].updatedAt`
- `papers[].knowledgeHints`
- `papers[].topicHints`
- `papers[].source`
- `papers[].sourceId`
- `papers[].sourceName`
- `papers[].category`

`knowledgeHints` 用于匹配课程知识点，`topicHints` 用于生成更前沿的研究主题。匹配流程不是把论文标题直接塞进图谱，而是分两步：

1. `knowledge_catalog.py` 从 curated graph 汇总课程知识点、课程名、描述和别名，对论文标题、摘要、关键词和已有 hint 做关键词/别名召回，得到最多几十个候选知识点。
2. `ai_summarizer.py` 调用 DeepSeek/OpenAI 时只允许模型在候选集合中选择 `knowledgeHints`，同时生成可解释的 `topicHints`。这样既能利用 AI 语义匹配，也能避免模型编造图谱里不存在的知识点。

如果运行时添加 `--skip-ai` 或 `--skip-ai-mapping`，系统会退回规则召回结果，仍然生成标准 metadata 和 `txt2neo4j` 前沿产物。

## 运行方式

默认只生成文件，不写数据库：

```powershell
cd pipelines/frontier-scraper
python main.py --skip-ai --max-per-category 5 --run-txt2neo4j
```

多源同步：

```powershell
python main.py --source arxiv huggingface github --skip-ai --keep-existing --run-txt2neo4j
```

使用封装脚本：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/sync-frontier.ps1
```

## 产物

同步器输出：

- `pipelines/txt2neo4j/config/frontier_seed_metadata.json`

`txt2neo4j` 输出：

- `artifacts/frontier/knowledge_nodes.json`
- `artifacts/frontier/papers.json`
- `artifacts/frontier/topics.json`
- `artifacts/frontier/digests.json`
- `artifacts/frontier/relationships.json`
- `artifacts/frontier/report.json`
- `artifacts/frontier/neo4j/*.csv`

## Neo4j 导入

数据库导入会写入或更新节点和关系，执行前必须确认数据库环境与备份。

```powershell
$env:FRONTIER_IMPORT_NEO4J_CONFIRMED="true"
cd pipelines/frontier-scraper
python main.py --skip-ai --run-txt2neo4j --import-neo4j
```

导入的图结构：

- `(:Paper)-[:MENTIONS_TOPIC]->(:ResearchTopic)`
- `(:Paper)-[:PUBLISHED_IN]->(:Venue)`
- `(:Paper)-[:AUTHORED_BY]->(:Author)`
- `(:ResearchTopic)-[:EXTENDS_KNOWLEDGE]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:CURATED_FOR]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:SUMMARIZES_PAPER]->(:Paper)`

## 后端与前端联动

导入完成后：

- 后端接口：`GET /api/frontier/briefing?knowledgeId={id}` 或 `?knowledge={name}`
- 后端接口：`GET /api/frontier/library?limit=50` 查看当代前沿资料列表及其关联知识点
- 前端位置：知识点详情页“前沿扩展”
- 前端位置：`/frontier` 独立“前沿资料”模块，首页也有入口
- 推荐算法：`FrontierRelevance` 因子会读取前沿摘要、主题和论文数量

## 质量门禁

建议每次更新后执行：

```powershell
cd pipelines/frontier-scraper
python -m unittest discover -s tests

cd ../txt2neo4j
python -m unittest discover -s tests
python -m txt2neo4j.graph_audit
```

完整项目验证：

```powershell
cd backend
go test ./...

cd ../frontend
npm test -- --run
npm run build
```

## 调度

Windows 任务计划程序建议每日执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/sync-frontier.ps1 -MaxPerCategory 5 -SinceDays 7 -KeepExisting
```

生产环境不建议自动写库。更稳妥的方式是先生成 artifacts，查看 `report.json` 中的 `unmatched_papers` 和 `linked_knowledge_names`，确认质量后再导入 Neo4j。
