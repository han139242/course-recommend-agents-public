# Frontier Knowledge Pipeline

`pipelines/txt2neo4j` 现在除了教材 TXT 抽取，还支持一条可落库的前沿知识扩展链路：

1. 读取本地论文元数据 `JSON / JSONL`
2. 基于 `curated_batches + term_dictionary + synonym_map` 匹配课程知识点
3. 生成 `Paper / ResearchTopic / FrontierDigest / Venue / Author`
4. 写出 JSON、CSV 产物，并可直接导入 Neo4j

## 输入格式

示例文件：`pipelines/txt2neo4j/config/frontier_seed_metadata.json`

核心字段：

- `source`
- `generatedAt`
- `papers[]`
- `papers[].title`
- `papers[].abstract`
- `papers[].keywords`
- `papers[].year`
- `papers[].venue`
- `papers[].url`
- `papers[].authors`
- `papers[].knowledgeHints`
- `papers[].topicHints`

其中：

- `knowledgeHints` 用来和课程知识点建立主映射
- `topicHints` 用来生成更前沿的 `ResearchTopic`

## 运行方式

只生成产物：

```bash
cd pipelines/txt2neo4j
txt2neo4j --frontier-input "config/frontier_seed_metadata.json" --frontier-output-dir "artifacts/frontier-demo"
```

生成后直接导入 Neo4j：

```bash
cd pipelines/txt2neo4j
txt2neo4j ^
  --frontier-input "config/frontier_seed_metadata.json" ^
  --frontier-output-dir "artifacts/frontier-demo" ^
  --import-frontier-neo4j ^
  --neo4j-uri "bolt://127.0.0.1:7687" ^
  --neo4j-user "neo4j" ^
  --neo4j-password "your-password"
```

## 产物

- `knowledge_nodes.json`
- `papers.json`
- `topics.json`
- `digests.json`
- `relationships.json`
- `report.json`
- `neo4j/*.csv`

## 与后端接口的对接

导入后的图结构会补齐这些关系：

- `(:Paper)-[:MENTIONS_TOPIC]->(:ResearchTopic)`
- `(:Paper)-[:PUBLISHED_IN]->(:Venue)`
- `(:Paper)-[:AUTHORED_BY]->(:Author)`
- `(:ResearchTopic)-[:EXTENDS_KNOWLEDGE]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:CURATED_FOR]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:SUMMARIZES_PAPER]->(:Paper)`

因此现有后端 `GET /api/frontier/briefing` 可以直接读取这些节点和关系，不需要再补中间层。
