# TXT to Neo4j Pipeline

这是一套面向教材和课程资料的高精度抽取工具链，用来把 `txt` 文本解析成结构化知识图谱，并以幂等方式导入 `Neo4j`。

## 目标

- 先清洗文本，再抽取结构，最后导入图库
- 优先保证精度和可回溯性，而不是一次性追求最大召回
- 每个节点和关系都保留来源路径、来源行号、来源文本
- 输出中间产物，支持抽样复核和重复执行

## 能力

- 文本清洗：去除大量元数据、页码、URL、纯数字噪声
- 结构解析：识别 `第N章`、`1.1`、`1.1.1` 等层级标题
- 知识点抽取：从章节标题自动生成高置信知识点
- 关系抽取：生成结构关系和少量高置信语义关系
- 语义图导入：默认只导入 `KnowledgeNode` 及其语义关系，不把教材目录树直接塞进图库
- 质量报告：输出低置信节点、孤立节点、关系分布
- Neo4j 导入：通过 `MERGE` 幂等写入，避免重复节点

## 目录结构

- `config/pipeline_config.json`：抽取阈值、过滤规则、`RELATED_TO` 压缩配置
- `config/frontier_ontology.json`：课内知识图谱与前沿论文知识域的分域建模方案
- `dictionaries/term_dictionary.json`：专业术语词典
- `dictionaries/synonym_map.json`：同义词归一表
- `src/txt2neo4j/pipeline.py`：核心清洗、解析、抽取、导出逻辑
- `src/txt2neo4j/neo4j_import.py`：Neo4j 约束创建与批量导入
- `src/txt2neo4j/cli.py`：命令行入口
- `artifacts/`：推荐放生成结果的位置

## 安装

```bash
cd pipelines/txt2neo4j
py -m pip install -e .
```

## 用法

只做抽取，不写入 Neo4j：

```bash
txt2neo4j --input "C:\path\to\book.txt" --output-dir "C:\path\to\artifacts"
```

抽取后直接导入 Neo4j：

```bash
txt2neo4j --input "C:\path\to\book.txt" --output-dir "C:\path\to\artifacts" --import-neo4j
```

也可以显式指定 Neo4j 参数：

```bash
txt2neo4j \
  --input "C:\path\to\book.txt" \
  --output-dir "C:\path\to\artifacts" \
  --import-neo4j \
  --neo4j-uri "bolt://127.0.0.1:7687" \
  --neo4j-user "neo4j" \
  --neo4j-password "your-password"
```

对已有 Neo4j 图谱做工业级规范化清洗：

```bash
cd pipelines/txt2neo4j
python -m txt2neo4j.normalize_graph
```

规范化脚本会执行：

- 用 curated registry 覆盖同名旧节点的属性
- 合并仅有空格差异的近重复节点
- 删除明显题目式/练习式脏节点
- 为孤立节点补充关键 `SUBTOPIC_OF` 关系
- 写入 `normalized_name`、`source_kind`、`quality_tier`、`taxonomy_domain` 等工业化元数据
- 输出 `artifacts/neo4j_normalization/normalization_report.json`

## 输出产物

- `cleaned_lines.json`
- `structure_nodes.json`
- `knowledge_nodes.json`
- `relations.json`
- `quality_report.json`
- `review/review_related_edges.json`
- `review/review_related_edges.csv`
- `review/review_nodes.json`
- `neo4j/structure_nodes.csv`
- `neo4j/knowledge_nodes.csv`
- `neo4j/relations.csv`

## 工业化建议

- 先看 `quality_report.json`，不要直接把低质量结果写到生产图库
- 先看 `review/review_related_edges.csv`，低价值 `RELATED_TO` 会先被压到人工复核清单
- 每次处理一批文本时保留独立 `output-dir`
- 先抽取高置信主干图谱，再逐步增加正文语义关系
- 在正式环境里结合人工同义词表和审核流程
- 如果扩展到多本教材，建议增加跨书归一词表和审核白名单
- 可视化时优先查询 `MATCH (a:KnowledgeNode)-[r]->(b:KnowledgeNode) RETURN a,r,b LIMIT 50`

## 前沿知识扩展

当前已经补充第一版前沿知识域设计：

- `config/frontier_ontology.json`：定义 `Paper`、`ResearchTopic`、`FrontierDigest` 等节点与关系
- `../docs/frontier-knowledge-pipeline.md`：说明如何把论文元数据、摘要与课内知识点连接起来

建议第二阶段按以下顺序扩展：

1. 抓取论文元数据与摘要
2. 清洗文本并抽取主题
3. 将主题映射到 `KnowledgeNode`
4. 生成面向学生的课后扩展摘要
