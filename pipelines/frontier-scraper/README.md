# Frontier Scraper：AI 前沿信息同步器

这个目录负责把外部持续更新的信息源同步成本项目可消费的前沿知识元数据，再交给 `txt2neo4j` 生成 `Paper / ResearchTopic / FrontierDigest` 和 Neo4j CSV 产物。

默认流程不会写数据库，只生成文件；导入 Neo4j 必须显式开启并设置确认环境变量。

## 数据源

- arXiv 官方 API：主论文源，按 AI 分类、关键词和日期窗口抓取最新论文。
- Hugging Face Daily Papers：可选社区趋势源，用于补充近期热门论文线索。
- GitHub Releases：可选开源观察源，用于监测持续更新的 AI 开源项目和论文索引项目。

GitHub 源不是正式论文库，默认仅作为“前沿线索”。论文入库仍优先依赖 arXiv、会议论文元数据和人工/AI 校验后的标准字段。

## 知识点匹配

前沿资料需要匹配到已有课程知识点，否则只能成为孤立资讯。当前实现采用“候选召回 + AI 选择”的闭环：

1. `knowledge_catalog.py` 从 `pipelines/txt2neo4j/config/curated_batches` 读取课程知识点、课程名、描述和别名，构建候选目录。
2. 同步器先根据论文标题、摘要、关键词和已有 `knowledgeHints` 做关键词/别名召回。
3. `ai_summarizer.py` 调用 DeepSeek/OpenAI 时，把候选知识点作为白名单传入，只允许模型从候选集合中选择最终 `knowledgeHints`，并生成 `topicHints`。
4. 使用 `--skip-ai` 或 `--skip-ai-mapping` 时会退回规则召回结果，仍能生成标准 metadata 和 `txt2neo4j` 前沿产物。

## 快速运行

```powershell
cd pipelines/frontier-scraper
python main.py --skip-ai --max-per-category 5 --run-txt2neo4j
```

输出：

- `pipelines/txt2neo4j/config/frontier_seed_metadata.json`
- `pipelines/txt2neo4j/artifacts/frontier/*.json`
- `pipelines/txt2neo4j/artifacts/frontier/neo4j/*.csv`

## 常用命令

只抓 arXiv，跳过 AI 摘要：

```powershell
python main.py --source arxiv --skip-ai --max-per-category 10
```

抓最近 7 天的 LLM/RAG 相关论文：

```powershell
python main.py --source arxiv --query """large language model"" OR RAG" --since-days 7 --skip-ai
```

组合多个源，并保留旧 metadata：

```powershell
python main.py --source arxiv huggingface github --keep-existing --skip-ai
```

生成 `txt2neo4j` 前沿产物：

```powershell
python main.py --skip-ai --run-txt2neo4j
```

启用 DeepSeek/OpenAI 结构化摘要增强：

```powershell
python main.py --max-per-category 3 --ai-delay 1.5 --run-txt2neo4j
```

需要 `.env` 或系统环境变量包含：

```env
DEEPSEEK_API_KEY=sk-xxxx
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-chat
```

## Neo4j 导入

写库是高风险操作，默认关闭。确认数据库连接、备份和影响范围后再执行：

```powershell
$env:FRONTIER_IMPORT_NEO4J_CONFIRMED="true"
python main.py --skip-ai --run-txt2neo4j --import-neo4j
```

导入后，后端 `GET /api/frontier/briefing` 会读取图谱中的 `FrontierDigest / ResearchTopic / Paper` 关系，知识点详情页即可展示前沿扩展；`GET /api/frontier/library` 会返回当代前沿资料列表及其关联知识点，前端 `/frontier` 页面会展示独立资料中心。

## 输出字段契约

`frontier_seed_metadata.json` 核心结构：

```json
{
  "source": "frontier_sync",
  "generatedAt": "2026-05-31T12:00:00Z",
  "sources": ["arXiv"],
  "papers": [
    {
      "title": "Paper title",
      "abstract": "Paper abstract",
      "keywords": ["Transformer"],
      "year": 2026,
      "venue": "arXiv",
      "venueType": "preprint",
      "url": "https://arxiv.org/abs/2605.12345",
      "authors": [{"name": "Author", "affiliation": ""}],
      "updatedAt": "2026-05-31T10:00:00Z",
      "knowledgeHints": ["Transformer"],
      "topicHints": ["高效推理"],
      "source": "arxiv",
      "sourceId": "2605.12345",
      "sourceName": "arXiv",
      "category": "cs.AI"
    }
  ]
}
```

`knowledgeHints` 用于匹配课程知识点，`topicHints` 用于生成前沿研究主题。AI 映射阶段只在候选知识点白名单内选择，避免产生图谱里不存在的伪知识点。

## 测试

```powershell
cd pipelines/frontier-scraper
python -m unittest discover -s tests
```

建议完整验证：

```powershell
cd pipelines/txt2neo4j
python -m unittest discover -s tests
python -m txt2neo4j.graph_audit
```

## 调度建议

Windows 任务计划程序可以每天执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/sync-frontier.ps1
```

默认调度脚本只生成文件，不写 Neo4j。需要自动写库时必须显式传 `-ImportNeo4j`，并完成数据库风险确认。
